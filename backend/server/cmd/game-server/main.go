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

	configPath, err := config.LoadDefaultYAMLFile()
	if err != nil {
		logger.Fatalf("resolve yaml config: %v", err)
	}
	logger.Printf("loaded yaml config file: %s", configPath)

	// 运行时配置现在统一从 YAML 文件读取，避免随着字段增多继续把
	// 玩家、网络和存储配置散落到一长串环境变量中。
	cfg, err := config.LoadFromYAMLFile(configPath)
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

// GOCACHE=/Users/wangzhiwei/study/pocket-pet-remake/.tmp/go-build-cache go run ./server/cmd/game-server
