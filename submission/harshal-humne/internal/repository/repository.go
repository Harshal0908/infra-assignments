package repository

import (
	"context"
	"errors"
	"sync"

	"config-service/internal/domain"
)

// ErrNotFound is returned when a config record does not exist.
var ErrNotFound = errors.New("config not found")

// Repository defines storage operations for Config records.
type Repository interface {
	Get(ctx context.Context, id string) (*domain.Config, error)
	Upsert(ctx context.Context, cfg *domain.Config) error
	Ping(ctx context.Context) error
}

// InMemory is a thread-safe, in-memory Repository implementation.
type InMemory struct {
	mu   sync.RWMutex
	data map[string]*domain.Config
}

// NewInMemory returns an initialized InMemory repository.
func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]*domain.Config),
	}
}

// Ping reports that the in-memory repository is available.
func (r *InMemory) Ping(_ context.Context) error {
	return nil
}

// Get retrieves a Config by its ID.
func (r *InMemory) Get(_ context.Context, id string) (*domain.Config, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cfg, ok := r.data[id]
	if !ok {
		return nil, ErrNotFound
	}

	return cfg, nil
}

// Upsert creates or replaces a Config record.
func (r *InMemory) Upsert(_ context.Context, cfg *domain.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[cfg.ID] = cfg

	return nil
}
