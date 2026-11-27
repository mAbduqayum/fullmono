package migrations_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/mAbduqayum/fullmono/apps/api/internal/pgtest"
	"github.com/mAbduqayum/fullmono/apps/api/migrations"
)

// An integration test (see pgtest): after a rollback, the older api's
// `migrate` finds the database a migration ahead of it, and must let it start.
func TestAnOlderAPIStartsOnANewerSchema(t *testing.T) {
	pool := pgtest.Pool(t)
	older, newest := allButTheNewest(t)
	if err := migrations.UpFrom(pool.Config().ConnString(), older); err != nil {
		t.Fatalf("the older api's migrate: %v", err)
	}
	var version int64
	var dirty bool
	if err := pool.QueryRow(t.Context(), "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if want, _ := strconv.ParseInt(newest, 10, 64); version != want || dirty {
		t.Errorf("version %d (dirty %t), want %s left as it was", version, dirty, newest)
	}
}

// allButTheNewest is the migrations the api one migration older shipped, and
// the number of the one it didn't have.
func allButTheNewest(t *testing.T) (fstest.MapFS, string) {
	t.Helper()
	names, err := filepath.Glob("[0-9]*_*.sql")
	if err != nil || len(names) == 0 {
		t.Fatalf("migrations %v: %v", names, err)
	}
	newest, _, _ := strings.Cut(names[len(names)-1], "_")
	older := fstest.MapFS{}
	for _, name := range names {
		if strings.HasPrefix(name, newest+"_") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		older[name] = &fstest.MapFile{Data: data}
	}
	return older, newest
}
