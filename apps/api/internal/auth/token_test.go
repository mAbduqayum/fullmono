package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "a test secret that is 32 bytes!!"

func testTokens(t *testing.T) *Tokens {
	t.Helper()
	tokens, err := NewTokens(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return tokens
}

func TestTokenNamesItsAdmin(t *testing.T) {
	tokens := testTokens(t)
	token, expires, err := tokens.Issue("alice")
	if err != nil {
		t.Fatal(err)
	}
	if left := time.Until(expires); left < tokenLifetime-time.Minute || left > tokenLifetime {
		t.Errorf("expires in %v, want %v", left, tokenLifetime)
	}
	if username, err := tokens.Check(token); username != "alice" || err != nil {
		t.Errorf("Check = %q, %v; want alice", username, err)
	}
	// Three base64url parts; anyone can read the first two.
	header, _ := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[0])
	if string(header) != `{"alg":"HS256","typ":"JWT"}` {
		t.Errorf("header %s", header)
	}
}

func TestTokenRejectsForgeries(t *testing.T) {
	tokens := testTokens(t)
	sign := func(method jwt.SigningMethod, claims jwt.RegisteredClaims, key any) string {
		token, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	valid := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   "alice",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	with := func(change func(*jwt.RegisteredClaims)) jwt.RegisteredClaims {
		claims := valid
		change(&claims)
		return claims
	}
	// alice's real token, with mallory's name in place of hers.
	parts := strings.Split(sign(jwt.SigningMethodHS256, valid, tokens.secret), ".")
	mallory, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, with(func(c *jwt.RegisteredClaims) { c.Subject = "mallory" })).SigningString()
	parts[1] = strings.Split(mallory, ".")[1]

	for name, token := range map[string]string{
		"with its claims changed":    strings.Join(parts, "."),
		"signed with another secret": sign(jwt.SigningMethodHS256, valid, []byte("another secret, also 32 bytes!!!")),
		"expired": sign(jwt.SigningMethodHS256, with(func(c *jwt.RegisteredClaims) {
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
		}), tokens.secret),
		"without an expiry":   sign(jwt.SigningMethodHS256, with(func(c *jwt.RegisteredClaims) { c.ExpiresAt = nil }), tokens.secret),
		"from another issuer": sign(jwt.SigningMethodHS256, with(func(c *jwt.RegisteredClaims) { c.Issuer = "elsewhere" }), tokens.secret),
		// The right secret, but not the algorithm this api signs with.
		"signed with HS512": sign(jwt.SigningMethodHS512, valid, tokens.secret),
		// The classic forgery: no signature, and a header saying none is needed.
		"unsigned (alg none)": sign(jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType),
		"not a JWT":           "hello",
	} {
		if username, err := tokens.Check(token); err == nil {
			t.Errorf("%s: accepted, for %q", name, username)
		}
	}
}

func TestNewTokensWantsALongSecret(t *testing.T) {
	if _, err := NewTokens("too short"); err == nil {
		t.Error("took a 9-byte secret")
	}
}
