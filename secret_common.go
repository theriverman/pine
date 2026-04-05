package main

import (
	"fmt"
	"os"
)

type SecretStore interface {
	Get(alias string) (*Secret, error)
	Set(alias string, secret *Secret) error
	Delete(alias string) error
	Supported() bool
}

func resolveCredentials(instance *Instance, store SecretStore) (*Credentials, error) {
	creds := &Credentials{
		AuthType: instance.AuthType,
		Username: stringsOrEnv(instance.Username, os.Getenv(envUsername)),
	}

	if authType := os.Getenv(envAuthType); authType != "" {
		creds.AuthType = authType
	}

	switch creds.AuthType {
	case "token":
		if envTokenValue := os.Getenv(envToken); envTokenValue != "" {
			creds.Token = envTokenValue
			return creds, nil
		}
		if store.Supported() {
			secret, err := store.Get(instance.Alias)
			if err == nil && secret.Token != "" {
				creds.Token = secret.Token
				return creds, nil
			}
		}
		return nil, fmt.Errorf("token is required; set %s or store a token for instance %q", envToken, instance.Alias)
	case "normal", "ldap":
		if envPasswordValue := os.Getenv(envPassword); envPasswordValue != "" {
			creds.Password = envPasswordValue
			return creds, nil
		}
		if store.Supported() {
			secret, err := store.Get(instance.Alias)
			if err == nil && secret.Password != "" {
				creds.Password = secret.Password
				return creds, nil
			}
		}
		return nil, fmt.Errorf("password is required; set %s or store a password for instance %q", envPassword, instance.Alias)
	default:
		return nil, fmt.Errorf("unsupported auth type %q", creds.AuthType)
	}
}

func stringsOrEnv(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
