package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrTokenNotFound = errors.New("Token not found.")

const refreshPrefix = "refresh:"
const blockedPrefix = "blocked:"

type TokenStore struct {
	client *redis.Client
}

func NewTokenStore(client *redis.Client) *TokenStore {
	return &TokenStore{client: client}
}

// OpenRedis opens connection to store and pings to ensure connection is established.
// Returns client, err
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

// StoreRefresh takes token, userID, ttl and stores all of it with "refresh:" prefix in store.
// Only returns error on failure to store
func (s *TokenStore) StoreRefresh(ctx context.Context, token string, userID string, ttl time.Duration) error {
	status := s.client.Set(ctx, refreshPrefix+token, userID, ttl)
	if status.Err() != nil {
		return fmt.Errorf("store_StoreRefresh: can't set token: %s", status.Err().Error())
	}
	return nil
}

// GetRefresh tries to get refresh token.
// Returns token, err depending on found or not
func (s *TokenStore) GetRefresh(ctx context.Context, token string) (string, error) {
	status := s.client.Get(ctx, refreshPrefix+token)

	err := status.Err()
	// redis.Nil acutally means "nothing found", not a nil error.
	// catch sentinel error first
	if err == redis.Nil {
		return "", ErrTokenNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store_GetRefresh: can't get token: %s", status.Err().Error())
	}
	return status.Val(), nil
}

// DeleteRefresh deletes given token from store.
// Returns err.
func (s *TokenStore) DeleteRefresh(ctx context.Context, token string) error {
	err := s.client.Del(ctx, refreshPrefix+token).Err()
	if err != nil {
		return fmt.Errorf("store_DeleteRefresh: can't delete token: %s", err.Error())
	}
	return nil
}

// BlocklistAccess puts a jti token into the blocklist with the value "1"
// returns error on failure
func (s *TokenStore) BlocklistAccess(ctx context.Context, jti string, ttl time.Duration) error {
	// moves jti to blocklist. "1" is a placeholder value because list needs one
	status := s.client.Set(ctx, blockedPrefix+jti, "1", ttl)

	err := status.Err()
	if err != nil {
		return fmt.Errorf("store_BlocklistAccess: can't move token to blocklist: %s", err.Error())
	}
	return nil
}

// IsBlocked checks whether a token is currently blocked
// returns [true|false], error
func (s *TokenStore) IsBlocked(ctx context.Context, jti string) (bool, error) {
	status := s.client.Exists(ctx, blockedPrefix+jti)
	err := status.Err()
	if err != nil {
		return false, fmt.Errorf("store_IsBlocked: can't check blocklist: %s", err.Error())
	}

	value := status.Val()
	if value > 0 {
		return true, nil
	}
	// basically status.Val() == 0
	return false, nil
}
