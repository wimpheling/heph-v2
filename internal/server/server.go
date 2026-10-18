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

	authv1 "github.com/wimpheling/heph-v2/gen/heph/authorization/v1"
	"github.com/wimpheling/heph-v2/gen/heph/authorization/v1/authorizationv1connect"
	systemv1 "github.com/wimpheling/heph-v2/gen/heph/system/v1"
	"github.com/wimpheling/heph-v2/gen/heph/system/v1/systemv1connect"
	"github.com/wimpheling/heph-v2/internal/config"
	"github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/core/postgres"
	bootstrap "github.com/wimpheling/heph-v2/internal/drivers/bootstrap/identity"
	"github.com/wimpheling/heph-v2/internal/rpc"
	"github.com/wimpheling/heph-v2/internal/rpc/platform"
)

func NewHandler(service *authorization.Service, auth identity.Authenticator) (http.Handler, error) {
	mux := http.NewServeMux()
	registry := platform.NewRegistry(auth)
	system := rpc.NewSystemService(registry)
	grants := rpc.NewAuthorizationService(registry, service)
	if err := platform.Register[systemv1connect.SystemServiceHandler](registry, mux, systemv1.File_heph_system_v1_system_proto.Services().ByName("SystemService"), systemv1connect.NewSystemServiceHandler, system); err != nil {
		return nil, err
	}
	if err := platform.Register[authorizationv1connect.AuthorizationServiceHandler](registry, mux, authv1.File_heph_authorization_v1_authorization_proto.Services().ByName("AuthorizationService"), authorizationv1connect.NewAuthorizationServiceHandler, grants); err != nil {
		return nil, err
	}
	return mux, nil
}

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	auth, err := bootstrap.New(cfg.Identities)
	if err != nil {
		return err
	}
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	store, err := postgres.Open(startup, cfg.DatabaseURL, cfg.DatabaseSchema)
	if err != nil {
		return err
	}
	defer store.Close()
	if err = store.Migrate(startup, cfg.DatabaseSchema); err != nil {
		return err
	}
	actors := make([]identity.ActorRef, 0, len(cfg.Identities))
	for _, credential := range cfg.Identities {
		actors = append(actors, identity.ActorRef{Source: credential.Source, Subject: credential.Subject})
	}
	if err = store.Bootstrap(startup, actors); err != nil {
		return err
	}
	handler, err := NewHandler(authorization.New(store), auth)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", cfg.ListenAddr.String())
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return serve(ctx, listener, logger, handler)
}

func serve(ctx context.Context, listener net.Listener, logger *slog.Logger, handler http.Handler) error {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	server := &http.Server{
		Handler:           handler,
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
