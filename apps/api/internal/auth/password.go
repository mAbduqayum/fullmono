// Package auth logs admins in: their accounts in the admins table, with
// bcrypt hashes of their passwords, and the signed tokens (JWTs) that prove a
// login on every request after it.
package auth

import (
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 12

// passwordCost is bcrypt's work factor. Each step doubles the time one guess
// takes: about a quarter of a second at 12, for a login and for someone trying
// a stolen hash against a word list alike. A var only so tests can lower it.
var passwordCost = 12

var usernamePattern = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)

// CheckUsername says what's wrong with a new admin's name, if anything.
func CheckUsername(username string) error {
	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("the username %q must be 3 to 32 of a-z, 0-9, _ and -", username)
	}
	return nil
}

// HashPassword returns the bcrypt hash to store for a new password.
func HashPassword(password string) ([]byte, error) {
	if len(password) < minPasswordLength {
		return nil, fmt.Errorf("the password must be at least %d characters", minPasswordLength)
	}
	// It refuses a password over bcrypt's 72-byte limit rather than cut it short.
	return bcrypt.GenerateFromPassword([]byte(password), passwordCost)
}

func passwordMatches(hash []byte, password string) bool {
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}
