package db

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers "pgx5" driver.
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// pgx5DSN converts a postgres:// or postgresql:// DSN to pgx5:// scheme so that
// golang-migrate uses the pgx/v5 driver (registered as "pgx5").
func pgx5DSN(dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "postgresql://"):
		return "pgx5" + dsn[len("postgresql"):]
	case strings.HasPrefix(dsn, "postgres://"):
		return "pgx5" + dsn[len("postgres"):]
	default:
		return dsn
	}
}

// Migrate runs all pending UP migrations from fsys against the Postgres DSN.
// fsys must be rooted at the migrations directory (*.sql files at the top level of fsys).
// ErrNoChange is treated as success (non-fatal).
func Migrate(dsn string, fsys fs.FS) error {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, pgx5DSN(dsn))
	if err != nil {
		return err
	}
	defer func() {
		// m.Close returns (source error, database error); both are best-effort on cleanup.
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
