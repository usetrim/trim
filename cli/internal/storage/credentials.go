package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const serviceName = "TrimCLI"
const keyName = "api_token"

// ErrNotLoggedIn is returned when no API token is in the OS keychain or credentials file.
// Callers should surface chrome.ConfigNotLoggedIn / chrome.StatusNotLoggedIn from the API.
var ErrNotLoggedIn = errors.New("not_logged_in")

func SaveToken(token string) error {
	err := keyring.Set(serviceName, keyName, token)
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".trim")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "credentials.json")
	data, _ := json.Marshal(map[string]string{"token": token})
	return os.WriteFile(path, data, 0600)
}

// DeleteToken removes the API token from the OS keychain and the credentials file fallback.
// Returns ErrNotLoggedIn when neither store had a token.
func DeleteToken() error {
	keyringErr := keyring.Delete(serviceName, keyName)

	home, homeErr := os.UserHomeDir()
	fileRemoved := false
	if homeErr == nil {
		path := filepath.Join(home, ".trim", "credentials.json")
		if err := os.Remove(path); err == nil {
			fileRemoved = true
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	if keyringErr == nil || fileRemoved {
		return nil
	}
	return ErrNotLoggedIn
}

func GetToken() (string, error) {
	token, err := keyring.Get(serviceName, keyName)
	if err == nil && token != "" {
		return token, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".trim", "credentials.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", ErrNotLoggedIn
	}
	var creds map[string]string
	if err := json.Unmarshal(raw, &creds); err != nil {
		return "", err
	}
	if creds["token"] == "" {
		return "", ErrNotLoggedIn
	}
	return creds["token"], nil
}
