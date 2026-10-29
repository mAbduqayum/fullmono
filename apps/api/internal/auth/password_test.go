package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// At the real cost each hash takes a quarter of a second, and far longer under
// the race detector; the tests need the logic, not the cost.
func TestMain(m *testing.M) {
	passwordCost = bcrypt.MinCost
	m.Run()
}

func TestCheckUsername(t *testing.T) {
	for name, ok := range map[string]bool{
		"alice":                 true,
		"ops_team-2":            true,
		"al":                    false,
		"Alice":                 false,
		"alice smith":           false,
		strings.Repeat("a", 33): false,
	} {
		if err := CheckUsername(name); (err == nil) != ok {
			t.Errorf("CheckUsername(%q) = %v, want ok=%v", name, err, ok)
		}
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !passwordMatches(hash, "correct horse battery") || passwordMatches(hash, "correct horse battery!") {
		t.Error("the hash must match its password and nothing else")
	}
	for _, password := range []string{"too short", strings.Repeat("x", 73)} {
		if _, err := HashPassword(password); err == nil {
			t.Errorf("HashPassword(%.20q…) took it", password)
		}
	}
}
