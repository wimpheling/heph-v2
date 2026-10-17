package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/wimpheling/heph-v2/internal/config"
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
	return server.Run(ctx, cfg, logger)
}
