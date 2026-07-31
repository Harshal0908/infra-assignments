package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"config-service/internal/domain"
	"config-service/internal/repository"
)

// ErrInvalidConfig represents invalid configuration data.
var ErrInvalidConfig = errors.New("invalid config")

// ErrInvalidID represents an invalid configuration ID.
var ErrInvalidID = errors.New("invalid config id")

// Service implements business logic for Config operations.
type Service struct {
	repo repository.Repository
}

// New creates a Service backed by the given repository.
func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// GetConfig retrieves a Config by ID.
func (s *Service) GetConfig(ctx context.Context, id string) (*domain.Config, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidID)
	}

	return s.repo.Get(ctx, id)
}

// UpsertConfig creates or updates a Config record.
func (s *Service) UpsertConfig(ctx context.Context, cfg *domain.Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}

	switch {
	case strings.TrimSpace(cfg.ID) == "":
		return fmt.Errorf("%w: id is required", ErrInvalidConfig)
	case strings.TrimSpace(cfg.Host) == "":
		return fmt.Errorf("%w: host is required", ErrInvalidConfig)
	case cfg.Port < 1 || cfg.Port > 65535:
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidConfig)
	case strings.TrimSpace(cfg.AppName) == "":
		return fmt.Errorf("%w: app_name is required", ErrInvalidConfig)
	case strings.TrimSpace(cfg.LogLevel) == "":
		return fmt.Errorf("%w: log_level is required", ErrInvalidConfig)
	}

	return s.repo.Upsert(ctx, cfg)
}
