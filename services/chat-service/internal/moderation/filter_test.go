package moderation_test

import (
	"strings"
	"testing"

	"github.com/sahiy/sahiy-stream/services/chat-service/internal/moderation"
)

func TestFilter(t *testing.T) {
	content, ok := moderation.Filter("  hello world  ")
	if !ok || content != "hello world" {
		t.Fatalf("expected trimmed content, got %q ok=%v", content, ok)
	}

	_, ok = moderation.Filter("")
	if ok {
		t.Fatal("empty should fail")
	}

	long := strings.Repeat("a", moderation.MaxMessageLength+1)
	_, ok = moderation.Filter(long)
	if ok {
		t.Fatal("too long should fail")
	}

	_, ok = moderation.Filter("this is spam content")
	if ok {
		t.Fatal("banned word should fail")
	}
}
