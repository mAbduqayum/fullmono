package auth

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mAbduqayum/fullmono/apps/api/internal/pgtest"
)

// Integration tests: PostgresAdmins against a real Postgres (see pgtest).

func TestPostgresAdminsSetCreatesThenUpdates(t *testing.T) {
	admins := NewPostgresAdmins(pgtest.Pool(t))
	for _, hash := range [][]byte{[]byte("first hash"), []byte("second hash")} {
		if err := admins.Set(t.Context(), "alice", hash); err != nil {
			t.Fatal(err)
		}
		got, err := admins.PasswordHash(t.Context(), "alice")
		if err != nil || !bytes.Equal(got, hash) {
			t.Errorf("PasswordHash = %q, %v; want %q", got, err, hash)
		}
	}
}

func TestPostgresAdminsUnknown(t *testing.T) {
	admins := NewPostgresAdmins(pgtest.Pool(t))
	if _, err := admins.PasswordHash(t.Context(), "nobody"); !errors.Is(err, ErrNoAdmin) {
		t.Errorf("err = %v, want ErrNoAdmin", err)
	}
}
