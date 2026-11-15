package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"utils-server/internal/config"
	"utils-server/internal/database"
	"utils-server/internal/handlers"
	"utils-server/internal/repository"
	"utils-server/internal/services/password"
	"utils-server/internal/services/presence"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()

	if err := db.EnsureSchema(); err != nil {
		return fmt.Errorf("schema: %w", err)
	}

	repo := repository.New(db)
	passwordSvc := password.New()
	activeUsers, err := presence.New(cfg.ActiveUsersLimit)
	if err != nil {
		return fmt.Errorf("presence: %w", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handlers.New(repo, passwordSvc, activeUsers).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Print("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}
