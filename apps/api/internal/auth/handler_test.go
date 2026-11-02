package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Unit tests: the login routes through httptest, with a fake Admins. That
// the admin's operations refuse a request without a login is the contract's
// doing, which ../../api_test.go tests.

type fakeAdmins map[string][]byte

func (f fakeAdmins) PasswordHash(_ context.Context, username string) ([]byte, error) {
	if hash, ok := f[username]; ok {
		return hash, nil
	}
	return nil, ErrNoAdmin
}

const alicesLogIn = `{"username": "alice", "password": "correct horse battery"}`

func testHandler(t *testing.T) (http.Handler, *Tokens) {
	t.Helper()
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	tokens := testTokens(t)
	return NewHandler(fakeAdmins{"alice": hash}, tokens), tokens
}

func send(t *testing.T, h http.Handler, method, body string, edit func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, "/api/admin/session", strings.NewReader(body))
	if edit != nil {
		edit(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLogInSetsATokenCookie(t *testing.T) {
	h, _ := testHandler(t)
	rec := send(t, h, http.MethodPost, alicesLogIn, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"username":"alice"`) {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	cookie := rec.Result().Cookies()[0]
	if cookie.Name != cookieName || cookie.Path != "/api/admin" || !cookie.HttpOnly ||
		cookie.SameSite != http.SameSiteStrictMode || cookie.Secure {
		t.Errorf("cookie %s", cookie)
	}

	// Behind Caddy on HTTPS, it travels over HTTPS only.
	rec = send(t, h, http.MethodPost, alicesLogIn, func(r *http.Request) {
		r.Header.Set("X-Forwarded-Proto", "https")
	})
	if cookie := rec.Result().Cookies()[0]; !cookie.Secure {
		t.Errorf("cookie %s: want Secure", cookie)
	}
}

func TestLogInRejects(t *testing.T) {
	h, _ := testHandler(t)
	answers := map[string]string{}
	for name, body := range map[string]string{
		"a wrong password": `{"username": "alice", "password": "not her password"}`,
		"an unknown name":  `{"username": "bob", "password": "correct horse battery"}`,
	} {
		rec := send(t, h, http.MethodPost, body, nil)
		if rec.Code != http.StatusUnauthorized || len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s: status %d, cookies %v", name, rec.Code, rec.Result().Cookies())
		}
		answers[rec.Body.String()] = name
	}
	if len(answers) != 1 {
		t.Errorf("the answers differ, so they tell which names exist: %v", answers)
	}
	if rec := send(t, h, http.MethodPost, "not json", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("not json: status %d, want 400", rec.Code)
	}
}

// whoIs runs a request with the cookie through Identify, and says who the
// handler then sees and what CheckLogin says of it.
func whoIs(t *testing.T, tokens *Tokens, cookie *http.Cookie) (admin string, problem error) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/admin/stats", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	Identify(tokens)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		admin, problem = Admin(r.Context()), CheckLogin(r)
	})).ServeHTTP(httptest.NewRecorder(), req)
	return admin, problem
}

func TestOnlyAValidTokenLogsIn(t *testing.T) {
	h, tokens := testHandler(t)
	cookie := send(t, h, http.MethodPost, alicesLogIn, nil).Result().Cookies()[0]

	if admin, err := whoIs(t, tokens, nil); admin != "" || !errors.Is(err, errNoLogin) {
		t.Errorf("no cookie: admin %q, %v", admin, err)
	}
	if admin, err := whoIs(t, tokens, cookie); admin != "alice" || err != nil {
		t.Errorf("logged in: admin %q, %v", admin, err)
	}
	rec := send(t, Identify(tokens)(h), http.MethodGet, "", func(r *http.Request) { r.AddCookie(cookie) })
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"username":"alice"`) {
		t.Errorf("GET session: status %d, body %s", rec.Code, rec.Body)
	}
	tokens.now = func() time.Time { return time.Now().Add(tokenLifetime + time.Minute) }
	if admin, err := whoIs(t, tokens, cookie); admin != "" || !errors.Is(err, errLoginExpired) {
		t.Errorf("expired: admin %q, %v", admin, err)
	}
}

func TestLogOutDropsTheCookie(t *testing.T) {
	h, _ := testHandler(t)
	rec := send(t, h, http.MethodDelete, "", nil)
	cookie := rec.Result().Cookies()[0]
	if rec.Code != http.StatusNoContent || cookie.Name != cookieName || cookie.MaxAge >= 0 {
		t.Errorf("status %d, cookie %s", rec.Code, cookie)
	}
}
