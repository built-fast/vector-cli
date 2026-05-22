package config

import (
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "vector-cli"
	keyringAccount = "api-token"
)

// ErrKeyringDisabled is returned when the VECTOR_NO_KEYRING environment variable is set.
var ErrKeyringDisabled = errors.New("keyring is disabled (VECTOR_NO_KEYRING is set)")

// Save stores the API token in the OS keyring.
func Save(token string) error {
	if os.Getenv("VECTOR_NO_KEYRING") != "" {
		return ErrKeyringDisabled
	}
	return keyring.Set(keyringService, keyringAccount, token)
}

// Load retrieves the API token from the OS keyring.
func Load() (string, error) {
	if os.Getenv("VECTOR_NO_KEYRING") != "" {
		return "", ErrKeyringDisabled
	}
	return keyring.Get(keyringService, keyringAccount)
}

// Delete removes the API token from the OS keyring.
func Delete() error {
	if os.Getenv("VECTOR_NO_KEYRING") != "" {
		return ErrKeyringDisabled
	}
	return keyring.Delete(keyringService, keyringAccount)
}
