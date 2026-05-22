package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir resolves the configuration directory path without creating it.
// Resolution order: VECTOR_CONFIG_DIR env → XDG_CONFIG_HOME/vector → platform default.
// Platform defaults: ~/.config/vector on Linux/macOS, %APPDATA%/vector on Windows.
func ConfigDir() (string, error) {
	if dir := os.Getenv("VECTOR_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "vector"), nil
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("%%APPDATA%% is not set")
		}
		return filepath.Join(appData, "vector"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "vector"), nil
}

// EnsureConfigDir creates the config directory with 0700 permissions if it doesn't exist.
func EnsureConfigDir() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("unable to create config directory: %w", err)
	}

	return dir, nil
}

// ConfigFilePath returns the path to config.json within the config directory.
func ConfigFilePath() string {
	dir, err := ConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "config.json")
}
