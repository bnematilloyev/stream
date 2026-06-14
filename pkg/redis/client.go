package redis

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewClient(url string) (*redis.Client, error) {
	client, err := NewClientFromConfig(Config{URL: url})
	if err != nil {
		return nil, err
	}
	standalone, ok := client.(*redis.Client)
	if !ok {
		return nil, fmt.Errorf("expected standalone redis client")
	}
	return standalone, nil
}
