package database

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestBalanceCacheSetGetInvalidate(t *testing.T) {
	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mini.Close()

	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	defer client.Close()

	cache := NewBalanceCache(client, time.Minute)
	ctx := context.Background()

	if err := cache.Set(ctx, 42, 12345); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	balance, ok, err := cache.Get(ctx, 42)
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if balance != 12345 {
		t.Fatalf("expected balance 12345, got %d", balance)
	}

	if err := cache.Invalidate(ctx, 42); err != nil {
		t.Fatalf("invalidate cache: %v", err)
	}

	_, ok, err = cache.Get(ctx, 42)
	if err != nil {
		t.Fatalf("get cache after invalidate: %v", err)
	}
	if ok {
		t.Fatal("expected cache miss after invalidate")
	}
}
