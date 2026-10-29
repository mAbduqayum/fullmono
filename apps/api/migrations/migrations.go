// Package migrations holds the database schema as numbered SQL files,
// embedded in the api binary: `api migrate` applies the ones that shipped
// with it, so the schema and the code that reads it are always one version.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // the pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var files embed.FS

// Up applies every migration the database doesn't have yet.
func Up(databaseURL string) error {
	source, err := iofs.New(files, ".")
	if err != nil {
		return err
	}
	// golang-migrate picks its driver by the URL's scheme.
	m, err := migrate.NewWithSourceInstance("iofs", source, strings.Replace(databaseURL, "postgres://", "pgx5://", 1))
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		err = nil
	}
	sourceErr, databaseErr := m.Close()
	if err := errors.Join(err, sourceErr, databaseErr); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
