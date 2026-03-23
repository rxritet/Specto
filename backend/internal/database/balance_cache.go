package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// BalanceCache provides Redis-backed cache operations for account balances.
type BalanceCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewBalanceCache(client *redis.Client, ttl time.Duration) *BalanceCache {
	return &BalanceCache{client: client, ttl: ttl}
}

func (c *BalanceCache) Get(ctx context.Context, accountID int64) (int64, bool, error) {
	if c == nil || c.client == nil {
		return 0, false, nil
	}

	raw, err := c.client.Get(ctx, balanceCacheKey(accountID)).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("redis get balance cache: %w", err)
	}

	balance, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse cached balance: %w", err)
	}

	return balance, true, nil
}

func (c *BalanceCache) Set(ctx context.Context, accountID, balance int64) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Set(ctx, balanceCacheKey(accountID), strconv.FormatInt(balance, 10), c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set balance cache: %w", err)
	}
	return nil
}

func (c *BalanceCache) Invalidate(ctx context.Context, accountID int64) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Del(ctx, balanceCacheKey(accountID)).Err(); err != nil {
		return fmt.Errorf("redis invalidate balance cache: %w", err)
	}
	return nil
}

func balanceCacheKey(accountID int64) string {
	return "balance:" + strconv.FormatInt(accountID, 10)
}
