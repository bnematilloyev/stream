package viewers_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sahiy/sahiy-stream/pkg/viewers"
)

func TestCounterHeartbeatAndCount(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	counter := viewers.NewCounter(client, 45*time.Second)

	stats, err := counter.Heartbeat(context.Background(), "stream-1", "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Concurrent != 1 || stats.Unique != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	stats, err = counter.Heartbeat(context.Background(), "stream-1", "session-b")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Concurrent != 2 || stats.Unique != 2 {
		t.Fatalf("expected 2 viewers, got %+v", stats)
	}

	stats, err = counter.Heartbeat(context.Background(), "stream-1", "session-a")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Concurrent != 2 || stats.Unique != 2 {
		t.Fatalf("duplicate heartbeat should not increase unique: %+v", stats)
	}
}
