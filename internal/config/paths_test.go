package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDir_EnvOverride(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", "/custom/config/dir")
	t.Setenv("XDG_CONFIG_HOME", "/should/be/ignored")

	dir, err := ConfigDir()
	require.NoError(t, err)
	assert.Equal(t, "/custom/config/dir", dir)
}

func TestConfigDir_XDGConfigHome(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")

	dir, err := ConfigDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/xdg/config", "vector"), dir)
}

func TestConfigDir_DefaultFallback(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	require.NoError(t, err)

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		assert.Equal(t, filepath.Join(appData, "vector"), dir)
	} else {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".config", "vector"), dir)
	}
}

func TestConfigDir_PriorityOrder(t *testing.T) {
	// VECTOR_CONFIG_DIR takes precedence over XDG_CONFIG_HOME
	t.Setenv("VECTOR_CONFIG_DIR", "/vector/dir")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/dir")

	dir, err := ConfigDir()
	require.NoError(t, err)
	assert.Equal(t, "/vector/dir", dir)

	// When VECTOR_CONFIG_DIR is unset, XDG_CONFIG_HOME is used
	t.Setenv("VECTOR_CONFIG_DIR", "")

	dir, err = ConfigDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/xdg/dir", "vector"), dir)
}

func TestEnsureConfigDir_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "config")
	t.Setenv("VECTOR_CONFIG_DIR", configPath)

	dir, err := EnsureConfigDir()
	require.NoError(t, err)
	assert.Equal(t, configPath, dir)

	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	}
}

func TestEnsureConfigDir_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("VECTOR_CONFIG_DIR", tmpDir)

	dir, err := EnsureConfigDir()
	require.NoError(t, err)
	assert.Equal(t, tmpDir, dir)
}

func TestConfigFilePath(t *testing.T) {
	t.Setenv("VECTOR_CONFIG_DIR", "/test/config")

	path := ConfigFilePath()
	assert.Equal(t, filepath.Join("/test/config", "config.json"), path)
}
