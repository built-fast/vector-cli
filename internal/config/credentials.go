package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Credentials holds the API credentials.
type Credentials struct {
	ApiKey string `json:"api_key"`
}

// LoadCredentials reads credentials.json from the config directory.
// Returns empty credentials if the file doesn't exist.
func LoadCredentials() (*Credentials, error) {
	path := CredentialsFilePath()
	if path == "" {
		return nil, fmt.Errorf("unable to determine credentials file path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Credentials{}, nil
		}
		return nil, fmt.Errorf("unable to read credentials file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("invalid JSON in credentials file %s: %w", path, err)
	}

	return &creds, nil
}

// SaveCredentials writes credentials.json to the config directory with 0600 permissions.
// Creates the config directory if it doesn't exist.
func SaveCredentials(creds *Credentials) error {
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}

	path := CredentialsFilePath()
	if path == "" {
		return fmt.Errorf("unable to determine credentials file path")
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal credentials: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("unable to write credentials file: %w", err)
	}

	return nil
}

// ClearCredentials deletes credentials.json.
func ClearCredentials() error {
	path := CredentialsFilePath()
	if path == "" {
		return fmt.Errorf("unable to determine credentials file path")
	}

	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unable to remove credentials file: %w", err)
	}

	return nil
}
