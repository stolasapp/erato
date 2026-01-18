// Package db contains the sqlite database code generation and utilities used
// by the storage package.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pressly/goose/v3"
	"modernc.org/sqlite" // sqlite sql.DB driver initialization
)

//go:embed migrations/*.sql
var migrations embed.FS

// Open initializes a SQLite DB connection to the specified dbPath. If the
// database file does not exist, it attempts to create it, and then migrates the
// database to match the current state expected of the system.
func Open(ctx context.Context, logger *slog.Logger, dbPath string) (*sql.DB, error) {
	if dbPath == ":memory:" { //nolint:revive // for documentation
		// noop
	} else if _, err := os.Stat(dbPath); err != nil {
		const userOnlyDirPerms = 0o700
		if err = os.MkdirAll(filepath.Dir(dbPath), userOnlyDirPerms); err != nil {
			return nil, fmt.Errorf("failed to create db parent directory: %w", err)
		}
	}

	if strings.ContainsRune(dbPath, '?') {
		dbPath += "&"
	} else {
		dbPath += "?"
	}
	dbPath += "_time_format=sqlite"

	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		const initSQL = `
		pragma journal_mode = WAL; -- allow concurrent writes
		pragma synchronous = normal; -- don't wait for fsync except on checkpointing
		pragma temp_store = memory; -- temporary indices
		pragma mmap_size = 1000000000; -- up to 1GB, keep it all in RAM
		`
		_, err := conn.ExecContext(context.Background(), initSQL, nil)
		return err
	})

	handle, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB handler: %w", err)
	} else if err = handle.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}
	handle.SetMaxOpenConns(1)

	logger = logger.With(slog.String("db", dbPath))
	goose.SetLogger(slog.NewLogLogger(logger.Handler(), slog.LevelDebug))
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("failed to set migration dialect: %w", err)
	}
	return handle, goose.UpContext(ctx, handle, "migrations")
}
