package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// tokenLifetime is how long a login lasts. A signed JWT can't be taken
	// back, so it's kept short: a leaked one is good for hours, not weeks.
	tokenLifetime = 8 * time.Hour
	// issuer names who signed a token; Check takes only this api's own.
	issuer = "fullmono"
	// HS256's own size: a shorter secret is easier to guess.
	minSecretBytes = 32
)

// Tokens signs and checks the JWTs that say which admin logged in.
//
// A JWT is three base64url parts joined by dots: a header naming how it's
// signed, the claims (who, issued when, expires when), and a signature over
// both, made with the secret. Anyone can read the claims, but only the secret
// makes a signature that fits them, so the api trusts a token it signed
// without having to remember it.
type Tokens struct {
	secret []byte
	now    func() time.Time
}

func NewTokens(secret string) (*Tokens, error) {
	if len(secret) < minSecretBytes {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d bytes; make one with `openssl rand -base64 32`", minSecretBytes)
	}
	return &Tokens{secret: []byte(secret), now: time.Now}, nil
}

// Issue signs a token naming the admin, and says when it expires.
func (t *Tokens) Issue(username string) (string, time.Time, error) {
	now := t.now()
	expires := now.Add(tokenLifetime)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expires),
	}).SignedString(t.secret)
	return token, expires, err
}

// Check returns the admin a token names, if this api signed it and it hasn't
// expired.
func (t *Tokens) Check(token string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		// Whoever sends the token writes its header, algorithm included: take
		// HS256 only, never what the token asks for ("none", say).
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return "", err
	}
	return claims.Subject, nil
}
