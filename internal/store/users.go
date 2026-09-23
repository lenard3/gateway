package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailExists = errors.New("Email already exists.")
var ErrUserNotFound = errors.New("User not found.")

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserStore struct {
	pool *pgxpool.Pool
}

// NewUserStore returns a new pool for postgres
func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// Create creates a new entry for a user based on email and pwhash.
// Inserts new user and returns every field via User struct.
// Throws sentinal error on unique violation (email)
func (s *UserStore) Create(ctx context.Context, email string, passwordHash string) (*User, error) {
	var u User
	var pgErr *pgconn.PgError
	row := s.pool.QueryRow(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash, created_at, modified_at", email, passwordHash)

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		// Throws unique violation error back to caller to return to user later
		if ok := errors.As(err, &pgErr); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("store_Create: cant assign db return to user: %w", err)
	}
	return &u, nil
}

// GetByEmail tries to get user data with email.
// Returns User struct or error.
// Throws sentinal error on email not found or normal error on failure.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	row := s.pool.QueryRow(ctx, "SELECT id, email, password_hash, created_at, modified_at FROM users WHERE email = $1", email)

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("store_GetByEmail: User retrieval failed: %w", err)
	}
	return &u, nil
}
