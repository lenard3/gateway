package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, confErr := pgxpool.ParseConfig(connStr)
	if confErr != nil {
		return nil, fmt.Errorf("open pool config: %w", confErr)
	}
	pool, poolErr := pgxpool.NewWithConfig(ctx, config)
	if poolErr != nil {
		return nil, fmt.Errorf("open pool: %w", poolErr)
	}

	pingErr := pool.Ping(ctx)
	if pingErr != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pool: %w", pingErr)
	}

	return pool, nil
}

func Migrate(migrationsPath string, connStr string) error {

	migrationsFull := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(migrationsFull, connStr)
	if err != nil {
		return fmt.Errorf("error during migration creation: %w", err)
	}
	defer m.Close()

	errUp := m.Up()
	if errUp != nil {
		if errors.Is(errUp, migrate.ErrNoChange) {
			slog.Info("migrations: already at latest version")
			return nil
		}
		return fmt.Errorf("error running migrate: %w", errUp)
	}

	return nil
}
