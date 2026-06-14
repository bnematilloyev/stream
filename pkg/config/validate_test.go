package config_test

import (
	"testing"

	"github.com/sahiy/sahiy-stream/pkg/config"
)

func TestValidateProductionSecrets(t *testing.T) {
	t.Run("development skips validation", func(t *testing.T) {
		err := config.ValidateProductionSecrets("development", map[string]string{
			"JWT_ACCESS_SECRET": "",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("production rejects empty secret", func(t *testing.T) {
		err := config.ValidateProductionSecrets("production", map[string]string{
			"JWT_ACCESS_SECRET": "",
		})
		if err == nil {
			t.Fatal("expected error for empty secret")
		}
	})

	t.Run("production rejects dev default", func(t *testing.T) {
		err := config.ValidateProductionSecrets("production", map[string]string{
			"JWT_ACCESS_SECRET": "dev-access-secret-change-in-production-32",
		})
		if err == nil {
			t.Fatal("expected error for dev default secret")
		}
	})

	t.Run("production accepts strong secret", func(t *testing.T) {
		err := config.ValidateProductionSecrets("production", map[string]string{
			"JWT_ACCESS_SECRET": "super-strong-production-secret-with-enough-length",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
