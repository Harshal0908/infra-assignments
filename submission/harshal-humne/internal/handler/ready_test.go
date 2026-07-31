package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"config-service/internal/domain"
	"config-service/internal/handler"
	"config-service/internal/repository"
	"config-service/internal/service"
)

type unavailableRepository struct{}

func (unavailableRepository) Get(
	_ context.Context,
	_ string,
) (*domain.Config, error) {
	return nil, repository.ErrNotFound
}

func (unavailableRepository) Upsert(
	_ context.Context,
	_ *domain.Config,
) error {
	return nil
}

func (unavailableRepository) Ping(_ context.Context) error {
	return errors.New("database unavailable")
}

func TestReady(t *testing.T) {
	repo := repository.NewInMemory()
	svc := service.New(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNotReady(t *testing.T) {
	svc := service.New(unavailableRepository{})
	h := handler.New(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
