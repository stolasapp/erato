// Package storage provides the state management for resources and users.
package storage

import (
	"context"

	"github.com/stolasapp/erato/internal/storage/db"
)

const (
	// ErrNotFound is returned when a resource or user cannot be found.
	ErrNotFound Error = "not found"
	// ErrAlreadyExists is returned if a unique resource or user already exists.
	ErrAlreadyExists Error = "already exists"
	// ErrInvalidUsername is returned when a username fails validation.
	ErrInvalidUsername Error = "username must be 3-64 characters, alphanumeric and underscores only"
	// ErrInternal is returned for any other type of error.
	ErrInternal Error = "internal error"
)

// Error is an error type returned by the storage implementation.
type Error string

// Error satisfies [error].
func (e Error) Error() string { return string(e) }

// Resources are the methods on a storage implementation that are responsible
// for accessing and modifying resources.
type Resources interface {
	// ListResources returns data for the paths provided for the given user ID.
	// The results will not necessarily be 1:1 if the user has not interacted
	// with the given path(s).
	ListResources(ctx context.Context, userID uint64, paths ...string) ([]db.Resource, error)
	// GetResource returns data for a user ID/path combination. An [ErrNotFound]
	// is returned if the user does not have data for the given path.
	GetResource(ctx context.Context, userID uint64, path string) (db.Resource, error)
	// UpsertResource creates or updates the resource. This is a full PUT-style
	// update, so callers should do a GetResource first prior to calling this
	// method.
	UpsertResource(ctx context.Context, resource db.Resource) error
}

// Users are the methods on a storage implementation that are responsible for
// accessing and modifying users.
type Users interface {
	// ListUsers returns the users in a list, paginated by the given name (if
	// provided) up to the given limit of records.
	ListUsers(ctx context.Context, afterName string, limit int32) ([]db.User, error)
	// GetUser returns a single user with the specified ID. An [ErrNotFound] is
	// returned if the user ID does not exist.
	GetUser(ctx context.Context, userID uint64) (db.User, error)
	// GetUserByName returns a single user with the specified name. An
	// [ErrNotFound] is returned if the user name does not exist.
	GetUserByName(ctx context.Context, name string) (db.User, error)
	// UpsertUser creates or updates the user. This is a full PUT-style upsert.
	// An [ErrAlreadyExists] error is returned if the username is already in use.
	UpsertUser(ctx context.Context, user db.User) error
	// DeleteUser removes a user and all their associated resource data. Note
	// that this is a hard delete; data is not recoverable.
	DeleteUser(ctx context.Context, userID uint64) error
}

// Store is the combination interface for [Resources] and [Users].
type Store interface {
	Resources
	Users
	// Close releases any resources held by the store. An error is returned if
	// the store cannot be cleanly closed.
	Close() error
}
