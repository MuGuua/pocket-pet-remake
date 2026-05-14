package app

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/provider"
	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/session"
	"pocket-pet-remake/server/internal/module/world"
	httptransport "pocket-pet-remake/server/internal/transport/http"
	wstransport "pocket-pet-remake/server/internal/transport/ws"
)

type App struct {
	server         *http.Server
	sessionService *session.Service
	logger         *log.Logger
	cleanupClosers []io.Closer
}

func New(cfg config.Config, logger *log.Logger) (*App, error) {
	deps, closers, err := provider.OpenDependencies(cfg)
	if err != nil {
		return nil, err
	}
	return newApp(cfg, logger, deps, closers)
}

func NewWithDependencies(cfg config.Config, logger *log.Logger, deps provider.Dependencies) (*App, error) {
	return newApp(cfg, logger, deps, nil)
}

func newApp(cfg config.Config, logger *log.Logger, deps provider.Dependencies, closers []io.Closer) (*App, error) {
	repos, err := provider.NewConfiguredBundle(cfg, deps)
	if err != nil {
		return nil, err
	}

	signer := auth.NewHMACSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(repos.Accounts, repos.WSTokens, signer, cfg.WSTokenTTL)
	playerService := player.NewService(repos.Players)
	petService := pet.NewService(repos.Pets)
	worldService := world.NewService(repos.World)
	sessionService := session.NewService(logger, cfg.HeartbeatInterval, cfg.HeartbeatTimeout)

	authHandler := wstransport.NewAuthHandler(authService, sessionService)
	worldHandler := wstransport.NewWorldHandler(sessionService, playerService, petService, worldService)
	wsRouter := wstransport.NewRouter(authHandler, worldHandler, sessionService)
	wsHub := wstransport.NewHub(logger, wsRouter, sessionService)
	loginHandler := httptransport.NewLoginHandler(authService)
	httpHandler := buildHTTPHandler(loginHandler, wsHub)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		server:         server,
		sessionService: sessionService,
		logger:         logger,
		cleanupClosers: closers,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.closeResources()
	go a.sessionService.StartSweeper(ctx)

	errCh := make(chan error, 1)
	go func() {
		a.logger.Printf("game server listening on %s", a.server.Addr)
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) closeResources() {
	for _, closer := range a.cleanupClosers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil {
			a.logger.Printf("close resource: %v", err)
		}
	}
}
