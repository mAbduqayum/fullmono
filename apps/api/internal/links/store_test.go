package links

import (
	"errors"
	"testing"

	"github.com/mAbduqayum/fullmono/apps/api/internal/pgtest"
)

// Integration tests: PostgresStore against a real Postgres, which is the only
// way to check its SQL, the migrations and what the unique index does.

func testStore(t *testing.T) *PostgresStore {
	return NewPostgresStore(pgtest.Pool(t))
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
