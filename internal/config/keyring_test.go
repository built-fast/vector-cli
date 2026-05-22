package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestSave(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	err := Save("test-token")
	require.NoError(t, err)

	// Verify it was stored
	token, err := keyring.Get(keyringService, keyringAccount)
	require.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestLoad(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Store a token first
	require.NoError(t, keyring.Set(keyringService, keyringAccount, "my-token"))

	token, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "my-token", token)
}

func TestLoad_NotFound(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	_, err := Load()
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestDelete(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Store a token first
	require.NoError(t, keyring.Set(keyringService, keyringAccount, "to-delete"))

	err := Delete()
	require.NoError(t, err)

	// Verify it was removed
	_, err = keyring.Get(keyringService, keyringAccount)
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestDelete_NotFound(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	err := Delete()
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestSave_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "1")

	err := Save("test-token")
	assert.ErrorIs(t, err, ErrKeyringDisabled)
}

func TestLoad_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "1")

	_, err := Load()
	assert.ErrorIs(t, err, ErrKeyringDisabled)
}

func TestDelete_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "1")

	err := Delete()
	assert.ErrorIs(t, err, ErrKeyringDisabled)
}

func TestSaveLoadDelete_RoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Save
	require.NoError(t, Save("roundtrip-token"))

	// Load
	token, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "roundtrip-token", token)

	// Delete
	require.NoError(t, Delete())

	// Load again — should fail
	_, err = Load()
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}
