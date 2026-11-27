// Package migrations holds the database schema as numbered SQL files,
// embedded in the api binary: `api migrate` applies the ones that shipped
// with it, so the schema and the code that reads it are always one version.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // the pgx5:// driver
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// The numbered migrations only: schema.sql is their sum, for reading.
//
//go:embed *.up.sql *.down.sql
var files embed.FS

// Up applies every migration the database doesn't have yet.
//
// A database ahead of them, which a newer api migrated before a rollback to
// this one, is left as it is: golang-migrate would refuse it, and the api
// would never start. The README's rules keep each schema readable by the code
// before it, and the next deploy of the newer api finds it ready.
func Up(databaseURL string) error {
	return up(databaseURL, files)
}

func up(databaseURL string, migrationFiles fs.FS) (err error) {
	src, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return err
	}
	newest, err := newestVersion(src)
	if err != nil {
		return err
	}
	// golang-migrate picks its driver by the URL's scheme.
	m, err := migrate.NewWithSourceInstance("iofs", src, strings.Replace(databaseURL, "postgres://", "pgx5://", 1))
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if err = errors.Join(err, sourceErr, databaseErr); err != nil {
			err = fmt.Errorf("migrate: %w", err)
		}
	}()

	// A dirty database (a migration that failed halfway) is Up's to report.
	current, dirty, err := m.Version()
	if err == nil && !dirty && current > newest {
		slog.Warn("the database has migrations this api doesn't; leaving it as it is",
			"database_version", current, "api_version", newest)
		return nil
	}
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return err
	}
	if err := m.Up(); !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// newestVersion is the number of the last migration in the source.
func newestVersion(src source.Driver) (uint, error) {
	version, err := src.First()
	if err != nil {
		return 0, fmt.Errorf("no migrations: %w", err)
	}
	for {
		next, err := src.Next(version)
		if errors.Is(err, fs.ErrNotExist) {
			return version, nil
		}
		if err != nil {
			return 0, err
		}
		version = next
	}
}
