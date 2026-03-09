package secret

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

// Store interface for secret management.
type Store interface {
	Get(name string) (string, error)
	Set(name, value string) error
	Delete(name string) error
	List() ([]string, error)
}

const metaKey = "mcpx:keys"

// KeyringStore implements Store using OS keychain via go-keyring.
type KeyringStore struct {
	service string
}

// NewKeyringStore creates a KeyringStore with "mcpx" as the service name.
func NewKeyringStore() *KeyringStore {
	return &KeyringStore{service: "mcpx"}
}

// Get retrieves a secret by name from the OS keychain.
func (s *KeyringStore) Get(name string) (string, error) {
	val, err := keyring.Get(s.service, name)
	if err != nil {
		return "", fmt.Errorf("secret get %q: %w", name, err)
	}
	return val, nil
}

// Set stores a secret in the OS keychain and updates the keys list.
func (s *KeyringStore) Set(name, value string) error {
	if err := keyring.Set(s.service, name, value); err != nil {
		return fmt.Errorf("secret set %q: %w", name, err)
	}
	if err := s.addKey(name); err != nil {
		return fmt.Errorf("secret set %q (update keys): %w", name, err)
	}
	return nil
}

// Delete removes a secret from the OS keychain and updates the keys list.
func (s *KeyringStore) Delete(name string) error {
	if err := keyring.Delete(s.service, name); err != nil {
		return fmt.Errorf("secret delete %q: %w", name, err)
	}
	if err := s.removeKey(name); err != nil {
		return fmt.Errorf("secret delete %q (update keys): %w", name, err)
	}
	return nil
}

// List returns all stored secret names.
func (s *KeyringStore) List() ([]string, error) {
	return s.loadKeys()
}

func (s *KeyringStore) loadKeys() ([]string, error) {
	raw, err := keyring.Get(s.service, metaKey)
	if err != nil {
		// No metadata key yet — no secrets stored.
		return nil, nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, fmt.Errorf("secret keys metadata corrupt: %w", err)
	}
	return keys, nil
}

func (s *KeyringStore) saveKeys(keys []string) error {
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	return keyring.Set(s.service, metaKey, string(data))
}

func (s *KeyringStore) addKey(name string) error {
	keys, err := s.loadKeys()
	if err != nil {
		return err
	}
	for _, k := range keys {
		if k == name {
			return nil // already tracked
		}
	}
	keys = append(keys, name)
	return s.saveKeys(keys)
}

func (s *KeyringStore) removeKey(name string) error {
	keys, err := s.loadKeys()
	if err != nil {
		return err
	}
	filtered := keys[:0]
	for _, k := range keys {
		if k != name {
			filtered = append(filtered, k)
		}
	}
	return s.saveKeys(filtered)
}
