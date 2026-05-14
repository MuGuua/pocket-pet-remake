package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/data/memory"
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
}

func New(cfg config.Config, logger *log.Logger) (*App, error) {
	accountRepo := memory.NewAccountRepository(cfg)
	playerRepo := memory.NewPlayerRepository(cfg)
	petRepo := memory.NewPetRepository(cfg)
	worldRepo := memory.NewWorldRepository()
	wsTokenRepo := memory.NewWSTokenRepository()
	signer := auth.NewHMACSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(accountRepo, wsTokenRepo, signer, cfg.WSTokenTTL)
	playerService := player.NewService(playerRepo)
	petService := pet.NewService(petRepo)
	worldService := world.NewService(worldRepo)
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
	}, nil
}

func (a *App) Run(ctx context.Context) error {
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
