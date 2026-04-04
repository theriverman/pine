//go:build !darwin && !windows && !linux

package main

func NewSecretStore() SecretStore {
	return unsupportedSecretStore{}
}
