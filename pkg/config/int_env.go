package config

import "strconv"

// IntEnv reads a positive integer from env or returns fallback.
func IntEnv(key string, fallback int) int {
	raw := Get(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
