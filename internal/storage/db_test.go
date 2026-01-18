package storage

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/storage/db"
)

func TestDB(t *testing.T) {
	t.Parallel()

	cfg := eratov1.Config_builder{
		DbFilepath: filepath.Join(t.TempDir(), "db.sqlite"),
	}.Build()
	store, err := NewDB(t.Context(), cfg, slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	const userID = 123
	const userName = "test"
	err = store.UpsertUser(t.Context(), db.User{
		ID:           userID,
		Name:         userName,
		PasswordHash: []byte{},
	})
	require.NoError(t, err)

	_, err = store.ListResources(t.Context(), userID)
	require.NoError(t, err)

	t.Run("UpsertResources", func(t *testing.T) {
		t.Parallel()

		path := t.Name()
		res := db.Resource{
			User: userID,
			Path: path,
		}
		err := store.UpsertResource(t.Context(), res)
		require.NoError(t, err)

		actual, err := store.GetResource(t.Context(), userID, path)
		require.NoError(t, err)
		assert.Equal(t, res, actual)

		res.Starred = true
		err = store.UpsertResource(t.Context(), res)
		require.NoError(t, err)

		actual, err = store.GetResource(t.Context(), userID, path)
		require.NoError(t, err)
		assert.Equal(t, res, actual)
	})

	t.Run("GetResource", func(t *testing.T) {
		t.Parallel()

		path := t.Name()
		_, err := store.GetResource(t.Context(), userID, path)
		require.ErrorIs(t, err, ErrNotFound)

		res := db.Resource{
			User:    userID,
			Path:    path,
			Starred: true,
			ViewTime: sql.NullTime{
				Valid: true,
				Time:  time.Now().Round(-1), // since the monotonic part won't be equal
			},
		}
		err = store.UpsertResource(t.Context(), res)
		require.NoError(t, err)

		actual, err := store.GetResource(t.Context(), userID, path)
		require.NoError(t, err)
		assert.Equal(t, res, actual)
	})

	t.Run("ListResources", func(t *testing.T) {
		t.Parallel()

		res, err := store.ListResources(t.Context(), userID)
		require.NoError(t, err)
		assert.Empty(t, res)

		path := t.Name()
		res, err = store.ListResources(t.Context(), userID, path)
		require.NoError(t, err)
		assert.Empty(t, res)

		res1 := db.Resource{
			User: userID,
			Path: path + "/1",
		}
		res2 := db.Resource{
			User: userID,
			Path: path + "/2",
		}
		err = store.UpsertResource(t.Context(), res1)
		require.NoError(t, err)
		err = store.UpsertResource(t.Context(), res2)
		require.NoError(t, err)

		res, err = store.ListResources(t.Context(), userID, res1.Path, res2.Path, "unknown/path")
		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Contains(t, res, res1)
		assert.Contains(t, res, res2)
	})

	// These operations are tested together since it needs to atomically handle
	// modifying the users in the system.
	t.Run("UserCRUD", func(t *testing.T) {
		t.Parallel()

		res, err := store.ListUsers(t.Context(), "", 100)
		require.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, userName, res[0].Name)

		res, err = store.ListUsers(t.Context(), userName, 100)
		require.NoError(t, err)
		assert.Empty(t, res)

		user := db.User{
			ID:   userID,
			Name: userName,
		}

		actual, err := store.GetUser(t.Context(), userID)
		require.NoError(t, err)
		assert.Equal(t, user, actual)

		_, err = store.GetUser(t.Context(), 0)
		require.ErrorIs(t, err, ErrNotFound)

		actual, err = store.GetUserByName(t.Context(), userName)
		require.NoError(t, err)
		assert.Equal(t, user, actual)

		_, err = store.GetUserByName(t.Context(), "not a real user")
		require.ErrorIs(t, err, ErrNotFound)

		err = store.UpsertUser(t.Context(), db.User{
			Name:         userName,
			PasswordHash: []byte{},
		})
		require.ErrorIs(t, err, ErrAlreadyExists)

		err = store.UpsertUser(t.Context(), db.User{Name: "ab", PasswordHash: []byte{}})
		require.ErrorIs(t, err, ErrInvalidUsername)

		err = store.UpsertUser(t.Context(), db.User{Name: "invalid/name", PasswordHash: []byte{}})
		require.ErrorIs(t, err, ErrInvalidUsername)

		user = db.User{
			Name:         "user_crud_test",
			PasswordHash: []byte("foobar"),
		}
		err = store.UpsertUser(t.Context(), user)
		require.NoError(t, err)

		user, err = store.GetUserByName(t.Context(), user.Name)
		require.NoError(t, err)

		err = store.DeleteUser(t.Context(), user.ID)
		require.NoError(t, err)
		_, err = store.GetUserByName(t.Context(), user.Name)
		require.ErrorIs(t, err, ErrNotFound)

		err = store.DeleteUser(t.Context(), user.ID)
		require.NoError(t, err)
	})
}
