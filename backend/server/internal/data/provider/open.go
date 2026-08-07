package provider

import (
	"fmt"
	"io"

	"pocket-pet-remake/server/internal/config"
	pgrepo "pocket-pet-remake/server/internal/data/postgres"
	redisrepo "pocket-pet-remake/server/internal/data/redis"
)

// OpenDependencies 按启动顺序连接 PostgreSQL 与 Redis，并返回统一的运行时依赖。
// 返回的关闭器按 Redis、PostgreSQL 排列，供应用退出时按当前生命周期管理逻辑释放连接。
func OpenDependencies(cfg config.Config) (Dependencies, []io.Closer, error) {
	postgresDB, err := pgrepo.Open(cfg.Postgres)
	if err != nil {
		// 包装依赖名称，避免连接超时时启动日志只显示缺少上下文的 deadline exceeded。
		return Dependencies{}, nil, fmt.Errorf("open postgres dependency: %w", err)
	}

	redisClient, err := redisrepo.Open(cfg.Redis)
	if err != nil {
		// Redis 初始化失败时立即释放已经建立的 PostgreSQL 连接，避免启动失败后泄漏资源。
		_ = postgresDB.Close()
		return Dependencies{}, nil, fmt.Errorf("open redis dependency: %w", err)
	}

	closers := []io.Closer{redisClient, postgresDB}
	deps := Dependencies{
		Postgres: postgresDB,
		Redis:    redisClient,
	}
	return deps, closers, nil
}
