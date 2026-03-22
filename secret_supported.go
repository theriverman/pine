//go:build darwin || windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/99designs/keyring"
)

type keyringSecretStore struct {
	ring      keyring.Keyring
	supported bool
}

func NewSecretStore() SecretStore {
	ring, err := keyring.Open(keyring.Config{
		ServiceName:     keyringService,
		AllowedBackends: allowedBackendsForPlatform(),
	})
	if err != nil {
		return &keyringSecretStore{supported: false}
	}

	return &keyringSecretStore{
		ring:      ring,
		supported: true,
	}
}

func allowedBackendsForPlatform() []keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return []keyring.BackendType{keyring.KeychainBackend}
	case "windows":
		return []keyring.BackendType{keyring.WinCredBackend}
	default:
		return nil
	}
}

func (s *keyringSecretStore) Supported() bool {
	return s.supported
}

func (s *keyringSecretStore) Get(alias string) (*Secret, error) {
	if !s.supported {
		return nil, errors.New("secret storage is not supported on this platform")
	}
	item, err := s.ring.Get(alias)
	if err != nil {
		return nil, err
	}
	secret := &Secret{}
	if err := json.Unmarshal(item.Data, secret); err != nil {
		return nil, fmt.Errorf("decode secret: %w", err)
	}
	return secret, nil
}

func (s *keyringSecretStore) Set(alias string, secret *Secret) error {
	if !s.supported {
		return errors.New("secret storage is not supported on this platform")
	}
	data, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("encode secret: %w", err)
	}
	return s.ring.Set(keyring.Item{
		Key:  alias,
		Data: data,
	})
}

func (s *keyringSecretStore) Delete(alias string) error {
	if !s.supported {
		return nil
	}
	err := s.ring.Remove(alias)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}
