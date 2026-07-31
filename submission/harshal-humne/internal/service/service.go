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
	id = strings.TrimSpace(id)

	if id == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidID)
	}

	if len(id) > 64 {
		return nil, fmt.Errorf("%w: id must not exceed 64 characters", ErrInvalidID)
	}

	return s.repo.Get(ctx, id)
}

// UpsertConfig creates or updates a Config record.
func (s *Service) UpsertConfig(ctx context.Context, cfg *domain.Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is required", ErrInvalidConfig)
	}

	cfg.ID = strings.TrimSpace(cfg.ID)
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	cfg.LogLevel = strings.ToUpper(strings.TrimSpace(cfg.LogLevel))

	switch {
	case cfg.ID == "":
		return fmt.Errorf("%w: id is required", ErrInvalidConfig)
	case len(cfg.ID) > 64:
		return fmt.Errorf("%w: id must not exceed 64 characters", ErrInvalidConfig)
	case cfg.Host == "":
		return fmt.Errorf("%w: host is required", ErrInvalidConfig)
	case len(cfg.Host) > 255:
		return fmt.Errorf("%w: host must not exceed 255 characters", ErrInvalidConfig)
	case cfg.Port < 1 || cfg.Port > 65535:
		return fmt.Errorf("%w: port must be between 1 and 65535", ErrInvalidConfig)
	case cfg.AppName == "":
		return fmt.Errorf("%w: app_name is required", ErrInvalidConfig)
	case len(cfg.AppName) > 128:
		return fmt.Errorf("%w: app_name must not exceed 128 characters", ErrInvalidConfig)
	case !validLogLevel(cfg.LogLevel):
		return fmt.Errorf(
			"%w: log_level must be DEBUG, INFO, WARN, or ERROR",
			ErrInvalidConfig,
		)
	}

	return s.repo.Upsert(ctx, cfg)
}

// Ready checks whether the storage system is available.
func (s *Service) Ready(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func validLogLevel(level string) bool {
	switch level {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return true
	default:
		return false
	}
}