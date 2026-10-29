package links

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mAbduqayum/fullmono/apps/api/migrations"
)

// Integration tests: PostgresStore against a real Postgres, which is the only
// way to check its SQL, the migrations and what the unique index does. They're
// slower and need a server, so they skip unless TEST_DATABASE_URL names one:
// `just test-integration` here, a `services:` container in CI.

// testStore returns a PostgresStore on a database of its own, created and
// migrated for this test and dropped after it, so no test sees another's rows
// or the dev data.
func testStore(t *testing.T) *PostgresStore {
	t.Helper()
	serverURL := os.Getenv("TEST_DATABASE_URL")
	if serverURL == "" {
		t.Skip("TEST_DATABASE_URL is unset; `just test-integration` runs this against Postgres")
	}
	admin, err := pgx.Connect(t.Context(), serverURL)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(t.Context(), "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// t.Context() is already cancelled when cleanups run.
		ctx := context.Background()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		_ = admin.Close(ctx)
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
	return NewPostgresStore(pool)
}

func TestPostgresStoreListsNewestFirst(t *testing.T) {
	store := testStore(t)
	for _, code := range []string{"first00", "second0", "third00"} {
		link, err := store.Create(t.Context(), code, "https://example.com/"+code)
		if err != nil {
			t.Fatal(err)
		}
		if link.Clicks != 0 || link.CreatedAt.IsZero() {
			t.Errorf("created %+v: want 0 clicks and a creation time", link)
		}
	}

	links, err := store.List(t.Context(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || links[0].Code != "third00" || links[1].Code != "second0" {
		t.Errorf("List(2) = %+v, want third00 then second0", links)
	}
}

func TestPostgresStoreRejectsATakenCode(t *testing.T) {
	store := testStore(t)
	if _, err := store.Create(t.Context(), "same000", "https://example.com/a"); err != nil {
		t.Fatal(err)
	}
	// The table's unique index answers this, not Go code: a fake can't check it.
	if _, err := store.Create(t.Context(), "same000", "https://example.com/b"); !errors.Is(err, ErrCodeTaken) {
		t.Errorf("second create: err = %v, want ErrCodeTaken", err)
	}
}

func TestPostgresStoreCountsVisits(t *testing.T) {
	store := testStore(t)
	if _, err := store.Create(t.Context(), "visit00", "https://go.dev"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		target, err := store.Visit(t.Context(), "visit00")
		if err != nil || target != "https://go.dev" {
			t.Fatalf("Visit = %q, %v", target, err)
		}
	}

	links, err := store.List(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if links[0].Clicks != 2 {
		t.Errorf("clicks = %d, want 2", links[0].Clicks)
	}
}

func TestPostgresStoreUnknownCode(t *testing.T) {
	if _, err := testStore(t).Visit(t.Context(), "nope000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
