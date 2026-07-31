package postgres

import (
	"context"
	"errors"
	"fmt"

	"config-service/internal/domain"
	"config-service/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository stores configuration records in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a PostgreSQL repository and checks the database connection.
func New(ctx context.Context, databaseURL string) (*Repository, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

// Close closes all database connections.
func (r *Repository) Close() {
	r.pool.Close()
}

// Ping checks whether PostgreSQL is reachable.
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Get retrieves one config by ID.
func (r *Repository) Get(ctx context.Context, id string) (*domain.Config, error) {
	const query = `
		SELECT id, host, port, app_name, log_level, created_at, updated_at
		FROM configs
		WHERE id = $1
	`

	var cfg domain.Config

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&cfg.ID,
		&cfg.Host,
		&cfg.Port,
		&cfg.AppName,
		&cfg.LogLevel,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	return &cfg, nil
}

// Upsert creates a config or updates it when the ID already exists.
func (r *Repository) Upsert(ctx context.Context, cfg *domain.Config) error {
	const query = `
		INSERT INTO configs (
			id,
			host,
			port,
			app_name,
			log_level
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id)
		DO UPDATE SET
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			app_name = EXCLUDED.app_name,
			log_level = EXCLUDED.log_level,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		cfg.ID,
		cfg.Host,
		cfg.Port,
		cfg.AppName,
		cfg.LogLevel,
	).Scan(
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert config: %w", err)
	}

	return nil
}
