package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "https://api.builtfast.com", cfg.ApiURL)
}

func TestLoadConfig_FileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://api.builtfast.com", cfg.ApiURL)
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	data := []byte(`{"api_url": "https://custom.example.com"}`)
	err := os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://custom.example.com", cfg.ApiURL)
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	data := []byte(`{not valid json}`)
	err := os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0o644)
	require.NoError(t, err)

	cfg, err := LoadConfig()
	assert.Nil(t, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON in config file")
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "vector")
	t.Setenv("VECTOR_CONFIG_DIR", configDir)

	cfg := &Config{ApiURL: "https://custom.example.com"}
	err := SaveConfig(cfg)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file contents
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"api_url": "https://custom.example.com"`)
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	original := &Config{ApiURL: "https://roundtrip.example.com"}
	err := SaveConfig(original)
	require.NoError(t, err)

	loaded, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, original.ApiURL, loaded.ApiURL)
}
