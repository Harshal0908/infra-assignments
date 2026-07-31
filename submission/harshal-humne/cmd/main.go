package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"config-service/internal/handler"
	postgresrepo "config-service/internal/repository/postgres"
	"config-service/internal/service"
)

func main() {
	portStr := os.Getenv("APP_PORT")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		log.Fatal("APP_PORT must be an integer between 1 and 65535")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	startupCtx, cancelStartup := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelStartup()

	repo, err := postgresrepo.New(startupCtx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer repo.Close()

	svc := service.New(repo)
	h := handler.New(svc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("starting config-service on :%d", port)
		serverErrors <- srv.ListenAndServe()
	}()

	signalCtx, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}

	case <-signalCtx.Done():
		log.Print("shutdown signal received")

		shutdownCtx, cancelShutdown := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancelShutdown()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)

			if closeErr := srv.Close(); closeErr != nil {
				log.Printf("forced server close failed: %v", closeErr)
			}
		}

		err := <-serverErrors
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server stopped with error: %v", err)
		}
	}

	log.Print("config-service stopped")
}
