package config

import (
	"fmt"
	"strings"
)

const minSecretLength = 32

var devSecretDefaults = []string{
	"dev-access-secret-change-in-production-32",
	"dev-refresh-secret-change-in-production-32",
	"change-me-access-secret-min-32-chars!!",
	"change-me-refresh-secret-min-32-chars!",
	"dev-media-hook-secret-min-32-chars!!",
	"dev-playback-signing-secret-min-32!!",
}

// ValidateProductionSecrets ensures critical secrets are set and not dev defaults in production.
func ValidateProductionSecrets(appEnv string, secrets map[string]string) error {
	if !IsProduction(appEnv) {
		return nil
	}
	for name, value := range secrets {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("production requires %s", name)
		}
		if len(value) < minSecretLength {
			return fmt.Errorf("%s must be at least %d characters in production", name, minSecretLength)
		}
		if isDevSecretDefault(value) {
			return fmt.Errorf("%s must not use development default in production", name)
		}
	}
	return nil
}

func IsProduction(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), "production")
}

func isDevSecretDefault(value string) bool {
	for _, d := range devSecretDefaults {
		if value == d {
			return true
		}
	}
	return false
}
