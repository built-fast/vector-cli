package config

import (
	"github.com/zalando/go-keyring"
)

// Credentials holds the API credentials.
type Credentials struct {
	ApiKey string `json:"api_key"`
}

// LoadCredentials retrieves the API token from the keyring and returns it as a Credentials struct.
// Returns empty credentials if no token is stored or if the keyring is disabled.
func LoadCredentials() (*Credentials, error) {
	token, err := Load()
	if err != nil {
		if err == ErrKeyringDisabled || err == keyring.ErrNotFound {
			return &Credentials{}, nil
		}
		return nil, err
	}
	return &Credentials{ApiKey: token}, nil
}

// SaveCredentials stores the API token from the Credentials struct into the keyring.
func SaveCredentials(creds *Credentials) error {
	return Save(creds.ApiKey)
}

// ClearCredentials removes the API token from the keyring.
func ClearCredentials() error {
	err := Delete()
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
