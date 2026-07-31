package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	log.Printf("starting config-service on :%d", port)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
