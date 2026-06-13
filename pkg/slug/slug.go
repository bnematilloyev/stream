package slug

import (
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,48}[a-z0-9]$`)

func Validate(value string) bool {
	return slugRegex.MatchString(value)
}

func Normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
