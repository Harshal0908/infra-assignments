package service_test

import (
	"context"
	"errors"
	"testing"

	"config-service/internal/domain"
	"config-service/internal/repository"
	"config-service/internal/service"
)

func TestUpsertConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *domain.Config
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name: "missing id",
			config: &domain.Config{
				Host:     "localhost",
				Port:     8080,
				AppName:  "config-service",
				LogLevel: "INFO",
			},
		},
		{
			name: "missing host",
			config: &domain.Config{
				ID:       "cfg_1",
				Port:     8080,
				AppName:  "config-service",
				LogLevel: "INFO",
			},
		},
		{
			name: "invalid port",
			config: &domain.Config{
				ID:       "cfg_1",
				Host:     "localhost",
				Port:     70000,
				AppName:  "config-service",
				LogLevel: "INFO",
			},
		},
		{
			name: "missing app name",
			config: &domain.Config{
				ID:       "cfg_1",
				Host:     "localhost",
				Port:     8080,
				LogLevel: "INFO",
			},
		},
		{
			name: "missing log level",
			config: &domain.Config{
				ID:      "cfg_1",
				Host:    "localhost",
				Port:    8080,
				AppName: "config-service",
			},
		},
	}

	repo := repository.NewInMemory()
	svc := service.New(repo)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := svc.UpsertConfig(context.Background(), test.config)

			if !errors.Is(err, service.ErrInvalidConfig) {
				t.Fatalf("expected ErrInvalidConfig, got %v", err)
			}
		})
	}
}

func TestUpsertConfigValid(t *testing.T) {
	repo := repository.NewInMemory()
	svc := service.New(repo)

	config := &domain.Config{
		ID:       "cfg_1",
		Host:     "localhost",
		Port:     8080,
		AppName:  "config-service",
		LogLevel: "INFO",
	}

	err := svc.UpsertConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
