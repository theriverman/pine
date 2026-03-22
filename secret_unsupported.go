//go:build !darwin && !windows

package main

func NewSecretStore() SecretStore {
	return unsupportedSecretStore{}
}
