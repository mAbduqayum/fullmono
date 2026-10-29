package links

import (
	"errors"
	"slices"
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

func TestPostgresStoreSearches(t *testing.T) {
	store := testStore(t)
	for _, link := range [][2]string{
		{"gopher0", "https://go.dev/doc"},
		{"react00", "https://react.dev"},
		{"snake00", "https://example.com/snake_case"},
	} {
		if _, err := store.Create(t.Context(), link[0], link[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		query         string
		limit, offset int
		want          []string
		total         int
	}{
		{"", 2, 0, []string{"snake00", "react00"}, 3},
		{"", 2, 2, []string{"gopher0"}, 3},
		{"DEV", 10, 0, []string{"react00", "gopher0"}, 2},
		// Escaped: a bare _ in ILIKE matches any one character, so every link.
		{"_", 10, 0, []string{"snake00"}, 1},
		{"%", 10, 0, []string{}, 0},
	} {
		links, total, err := store.Search(t.Context(), c.query, c.limit, c.offset)
		if err != nil {
			t.Fatal(err)
		}
		codes := []string{}
		for _, link := range links {
			codes = append(codes, link.Code)
		}
		if !slices.Equal(codes, c.want) || total != c.total {
			t.Errorf("Search(%q, %d, %d) = %v of %d; want %v of %d", c.query, c.limit, c.offset, codes, total, c.want, c.total)
		}
	}
}

func TestPostgresStoreDeletes(t *testing.T) {
	store := testStore(t)
	if _, err := store.Create(t.Context(), "spam000", "https://example.com/spam"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(t.Context(), "spam000"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Visit(t.Context(), "spam000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Visit after Delete: err = %v, want ErrNotFound", err)
	}
	if err := store.Delete(t.Context(), "spam000"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete: err = %v, want ErrNotFound", err)
	}
}

func TestPostgresStoreStats(t *testing.T) {
	store := testStore(t)
	// No links at all: sum() is NULL then, and must still read as 0.
	if stats, err := store.Stats(t.Context(), 5); err != nil || stats.Links != 0 || stats.Clicks != 0 || len(stats.Top) != 0 {
		t.Errorf("empty: %+v, %v", stats, err)
	}
	for code, visits := range map[string]int{"often00": 2, "once000": 1, "never00": 0} {
		if _, err := store.Create(t.Context(), code, "https://example.com/"+code); err != nil {
			t.Fatal(err)
		}
		for range visits {
			if _, err := store.Visit(t.Context(), code); err != nil {
				t.Fatal(err)
			}
		}
	}
	stats, err := store.Stats(t.Context(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Links != 3 || stats.Clicks != 3 || len(stats.Top) != 2 || stats.Top[0].Code != "often00" {
		t.Errorf("stats %+v: want 3 links, 3 clicks, often00 then once000 on top", stats)
	}
}
