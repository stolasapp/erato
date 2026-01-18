package storage

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math/rand/v2"
	"regexp"

	"github.com/influxdata/influxdb/pkg/snowflake"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/storage/db"
)

// Username validation constraints matching CreateUserRequest proto.
const (
	minUsernameLen = 3
	maxUsernameLen = 64
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// validateUsername validates that a username meets the requirements:
// 3-64 characters, alphanumeric and underscores only.
func validateUsername(name string) bool {
	return len(name) >= minUsernameLen &&
		len(name) <= maxUsernameLen &&
		usernameRegex.MatchString(name)
}

// DB is a [Store] backed by a SQLite database.
type DB struct {
	ids     *snowflake.Generator
	db      *sql.DB
	queries *db.Queries
}

// NewDB initializes a DB with the given config and logger.
func NewDB(ctx context.Context, cfg *eratov1.Config, logger *slog.Logger) (*DB, error) {
	handle, err := db.Open(ctx, logger, cfg.GetDbFilepath())
	if err != nil {
		return nil, err
	}
	return &DB{
		ids:     snowflake.New(rand.IntN(1023)), //nolint:gosec,mnd // this isn't for crypto
		db:      handle,
		queries: db.New(handle),
	}, nil
}

// Close satisfies the [Store] interface.
func (d *DB) Close() error {
	return d.db.Close()
}

// ListResources satisfies the [Resources] interface.
func (d *DB) ListResources(ctx context.Context, userID uint64, paths ...string) ([]db.Resource, error) {
	return d.queries.GetResources(ctx, db.GetResourcesParams{
		User:  userID,
		Paths: paths,
	})
}

// GetResource satisfies the [Resources] interface.
func (d *DB) GetResource(ctx context.Context, userID uint64, path string) (db.Resource, error) {
	res, err := d.queries.GetResource(ctx, db.GetResourceParams{
		User: userID,
		Path: path,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return res, ErrNotFound
	}
	return res, err
}

// UpsertResource satisfies the [Resources] interface.
func (d *DB) UpsertResource(ctx context.Context, resource db.Resource) error {
	_, err := d.queries.UpsertResource(ctx, db.UpsertResourceParams(resource))
	return err
}

// ListUsers satisfies the [Users] interface.
func (d *DB) ListUsers(ctx context.Context, afterName string, limit int32) ([]db.User, error) {
	return d.queries.GetUsers(ctx, db.GetUsersParams{
		AfterName: afterName,
		Limit:     int64(limit),
	})
}

// GetUser satisfies the [Users] interface.
func (d *DB) GetUser(ctx context.Context, userID uint64) (db.User, error) {
	user, err := d.queries.GetUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

// GetUserByName satisfies the [Users] interface.
func (d *DB) GetUserByName(ctx context.Context, name string) (db.User, error) {
	user, err := d.queries.GetUserByName(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return user, ErrNotFound
	}
	return user, err
}

// UpsertUser satisfies the [Users] interface.
func (d *DB) UpsertUser(ctx context.Context, user db.User) error {
	if !validateUsername(user.Name) {
		return ErrInvalidUsername
	}
	if user.ID == 0 {
		user.ID = d.ids.Next()
	}
	switch _, err := d.queries.UpsertUser(ctx, db.UpsertUserParams(user)); {
	case errors.Is(err, sql.ErrNoRows):
		return ErrAlreadyExists
	default:
		return err
	}
}

// DeleteUser satisfies the [Users] interface.
func (d *DB) DeleteUser(ctx context.Context, userID uint64) error {
	return d.queries.DeleteUser(ctx, userID)
}

var _ Store = (*DB)(nil)
