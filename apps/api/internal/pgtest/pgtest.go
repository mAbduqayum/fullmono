// Package pgtest gives integration tests a real Postgres to run against.
// They're slower and need a server, so they skip unless TEST_DATABASE_URL
// names one: `just test-integration` locally, a `services:` container in CI.
package pgtest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mAbduqayum/fullmono/apps/api/migrations"
)

// Pool connects to a database of the test's own, created and migrated for it
// and dropped after it, so no test sees another's rows or the dev data.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	serverURL := os.Getenv("TEST_DATABASE_URL")
	if serverURL == "" {
		t.Skip("TEST_DATABASE_URL is unset; `just test-integration` runs this against Postgres")
	}
	server, err := pgx.Connect(t.Context(), serverURL)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := server.Exec(t.Context(), "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// t.Context() is already cancelled when cleanups run.
		ctx := context.Background()
		if _, err := server.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		_ = server.Close(ctx)
	})

	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	if err := migrations.Up(u.String()); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
