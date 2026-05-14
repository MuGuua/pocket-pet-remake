package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"pocket-pet-remake/server/internal/app"
	"pocket-pet-remake/server/internal/config"
	"pocket-pet-remake/server/internal/platform/logx"
)

func main() {
	logger := logx.New()

	if loadedPath, err := config.LoadDefaultEnvFiles(); err != nil {
		logger.Fatalf("load env file: %v", err)
	} else if loadedPath != "" {
		logger.Printf("loaded config env file: %s", loadedPath)
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatalf("bootstrap app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		logger.Fatalf("run app: %v", err)
	}
}
