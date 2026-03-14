package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCredentials_FileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	creds, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "", creds.ApiKey)
}

func TestLoadCredentials_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	data := []byte(`{"api_key": "test-token-123"}`)
	err := os.WriteFile(filepath.Join(tmpDir, "credentials.json"), data, 0o600)
	require.NoError(t, err)

	creds, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, "test-token-123", creds.ApiKey)
}

func TestLoadCredentials_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	data := []byte(`{not valid json}`)
	err := os.WriteFile(filepath.Join(tmpDir, "credentials.json"), data, 0o600)
	require.NoError(t, err)

	creds, err := LoadCredentials()
	assert.Nil(t, creds)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON in credentials file")
}

func TestSaveCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "vector")
	t.Setenv("VECTOR_CONFIG_DIR", configDir)

	creds := &Credentials{ApiKey: "my-secret-key"}
	err := SaveCredentials(creds)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file contents
	credPath := filepath.Join(configDir, "credentials.json")
	data, err := os.ReadFile(credPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"api_key": "my-secret-key"`)

	// Verify file permissions are 0600
	fileInfo, err := os.Stat(credPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestSaveCredentials_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	original := &Credentials{ApiKey: "roundtrip-key"}
	err := SaveCredentials(original)
	require.NoError(t, err)

	loaded, err := LoadCredentials()
	require.NoError(t, err)
	assert.Equal(t, original.ApiKey, loaded.ApiKey)
}

func TestClearCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Save credentials first
	creds := &Credentials{ApiKey: "to-be-cleared"}
	err := SaveCredentials(creds)
	require.NoError(t, err)

	// Verify file exists
	credPath := filepath.Join(tmpDir, "credentials.json")
	_, err = os.Stat(credPath)
	require.NoError(t, err)

	// Clear credentials
	err = ClearCredentials()
	require.NoError(t, err)

	// Verify file is gone
	_, err = os.Stat(credPath)
	assert.True(t, os.IsNotExist(err))
}

func TestClearCredentials_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	// Should not error when file doesn't exist
	err := ClearCredentials()
	assert.NoError(t, err)
}
