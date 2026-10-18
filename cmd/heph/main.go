package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wimpheling/heph-v2/internal/config"
	"github.com/wimpheling/heph-v2/internal/core/postgres"
	"github.com/wimpheling/heph-v2/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("heph server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if len(os.Args) > 1 {
		command := os.Args[1]
		if command != "migrate" && command != "reset" {
			return fmt.Errorf("usage: heph [migrate|reset]")
		}
		setupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		store, err := postgres.Open(setupCtx, cfg.DatabaseURL, cfg.DatabaseSchema)
		if err != nil {
			return err
		}
		defer store.Close()
		if command == "reset" {
			if err = store.Reset(setupCtx); err != nil {
				return err
			}
		}
		return store.Migrate(setupCtx, cfg.DatabaseSchema)
	}
	return server.Run(ctx, cfg, logger)
}
