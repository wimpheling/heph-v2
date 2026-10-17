// Package server composes and runs the HTTP server.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/wimpheling/heph-v2/gen/heph/system/v1/systemv1connect"
	"github.com/wimpheling/heph-v2/internal/config"
	"github.com/wimpheling/heph-v2/internal/rpc"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	path, handler := systemv1connect.NewSystemServiceHandler(rpc.SystemService{})
	mux.Handle(path, handler)
	return mux
}

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	listener, err := net.Listen("tcp", cfg.ListenAddr.String())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return serve(ctx, listener, logger)
}

func serve(ctx context.Context, listener net.Listener, logger *slog.Logger) error {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{
		Handler:           NewHandler(),
		Protocols:         protocols,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	result := make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	logger.Info("heph server listening", "address", listener.Addr().String())

	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		if err := <-result; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
