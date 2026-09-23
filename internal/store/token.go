package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTokenNotFound = errors.New("Token not found.")

type TokenStore struct {
	client *redis.Client
}

func NewTokenStore(client *redis.Client) *TokenStore {
	return &TokenStore{client: client}
}

func OpenRedis(ctx context.Context, connStr string) (*redis.Client, error) {
	options, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, fmt.Errorf("store_OpenRedis: failed to parse connection string: %w", err)
	}

	rdb := redis.NewClient(options)
	err = rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, fmt.Errorf("store_OpenRedis: failed to ping redis store: %w", err)
	}
	return rdb, nil
}

func (s *TokenStore) StoreRefresh(ctx context.Context, token string, userID string, ttl time.Duration) error {
	status := s.client.Set(ctx, "redis:"+token, userID, ttl)
	if status.Err() != nil {
		return fmt.Errorf("store_StoreRefresh: can't set token: %s", status.Err().Error())
	}
	return nil
}

func (s *TokenStore) GetRefresh(ctx context.Context, token string) (string, error) {
	status := s.client.Get(ctx, token)
	err := status.Err()
	if err != redis.Nil {
		return "", fmt.Errorf("store_GetRefresh: can't get token: %s", status.Err().Error())
	}
	return status.Val(), nil
}

func (s *TokenStore) DeleteRefresh(ctx context.Context, token string) error {
	err := s.client.Del(ctx, "refresh"+token).Err()
	if err != redis.Nil {
		return fmt.Errorf("store_DeleteRefresh: can't delete token: %s", err.Error())
	}
	return nil
}
