package sec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	t.Run("string password", func(t *testing.T) {
		t.Parallel()
		hash, err := HashPassword("mypassword")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("byte slice password", func(t *testing.T) {
		t.Parallel()
		hash, err := HashPassword([]byte("mypassword"))
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestComparePassword(t *testing.T) {
	t.Parallel()

	// Pre-generate a hash for testing
	password := "correctpassword"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	t.Run("correct password string", func(t *testing.T) {
		t.Parallel()
		err := ComparePassword(password, hash)
		assert.NoError(t, err)
	})

	t.Run("correct password bytes", func(t *testing.T) {
		t.Parallel()
		err := ComparePassword([]byte(password), hash)
		assert.NoError(t, err)
	})

	t.Run("incorrect password", func(t *testing.T) {
		t.Parallel()
		err := ComparePassword("wrongpassword", hash)
		assert.Error(t, err)
	})
}
