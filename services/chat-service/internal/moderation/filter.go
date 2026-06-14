package moderation

import (
	"strings"
	"unicode/utf8"
)

const MaxMessageLength = 500

var bannedWords = []string{
	"spam", "scam", "hack",
}

// Filter validates and sanitizes chat content.
func Filter(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if content == "" || utf8.RuneCountInString(content) > MaxMessageLength {
		return "", false
	}
	lower := strings.ToLower(content)
	for _, word := range bannedWords {
		if strings.Contains(lower, word) {
			return "", false
		}
	}
	return content, true
}
