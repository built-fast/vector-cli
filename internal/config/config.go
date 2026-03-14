// Package config handles configuration and credentials loading.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const defaultAPIURL = "https://api.builtfast.com"

// Config holds the CLI configuration.
type Config struct {
	ApiURL string `json:"api_url"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		ApiURL: defaultAPIURL,
	}
}

// LoadConfig reads config.json from the config directory.
// Returns default config if the file doesn't exist.
func LoadConfig() (*Config, error) {
	path := ConfigFilePath()
	if path == "" {
		return nil, fmt.Errorf("unable to determine config file path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in config file %s: %w", path, err)
	}

	return cfg, nil
}

// SaveConfig writes config.json to the config directory.
// Creates the config directory if it doesn't exist.
func SaveConfig(cfg *Config) error {
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}

	path := ConfigFilePath()
	if path == "" {
		return fmt.Errorf("unable to determine config file path")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal config: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("unable to write config file: %w", err)
	}

	return nil
}
