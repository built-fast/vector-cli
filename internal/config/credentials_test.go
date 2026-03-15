package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestLoadCredentials_NoToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	creds, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "", creds.ApiKey)
}

func TestLoadCredentials_WithToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	require.NoError(t, keyring.Set(keyringService, keyringAccount, "test-token-123"))

	creds, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "test-token-123", creds.ApiKey)
}

func TestLoadCredentials_KeyringDisabled(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "1")

	creds, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "", creds.ApiKey)
}

func TestSaveCredentials(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	creds := &Credentials{ApiKey: "my-secret-key"}
	err := SaveCredentials(creds)
	require.NoError(t, err)

	// Verify it was stored in keyring
	token, err := keyring.Get(keyringService, keyringAccount)
	require.NoError(t, err)
	assert.Equal(t, "my-secret-key", token)
}

func TestSaveCredentials_RoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	original := &Credentials{ApiKey: "roundtrip-key"}
	err := SaveCredentials(original)
	require.NoError(t, err)

	loaded, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, original.ApiKey, loaded.ApiKey)
}

func TestClearCredentials(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Save credentials first
	creds := &Credentials{ApiKey: "to-be-cleared"}
	require.NoError(t, SaveCredentials(creds))

	// Clear credentials
	err := ClearCredentials()
	require.NoError(t, err)

	// Verify token is gone
	loaded, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "", loaded.ApiKey)
}

func TestClearCredentials_NoToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv("VECTOR_NO_KEYRING", "")

	// Should not error when no token exists
	err := ClearCredentials()
	assert.NoError(t, err)
}
