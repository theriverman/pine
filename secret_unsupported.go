//go:build !darwin && !windows && !linux

package main

import "errors"

type unsupportedSecretStore struct{}

func NewSecretStore() SecretStore {
	return unsupportedSecretStore{}
}

func (unsupportedSecretStore) Get(alias string) (*Secret, error) {
	return nil, errors.New("secret storage is not supported on this platform")
}

func (unsupportedSecretStore) Set(alias string, secret *Secret) error {
	return errors.New("secret storage is not supported on this platform")
}

func (unsupportedSecretStore) Delete(alias string) error { return nil }

func (unsupportedSecretStore) Supported() bool { return false }
