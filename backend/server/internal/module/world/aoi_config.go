package world

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrAOIConfigInvalid = errors.New("invalid AOI config")
var ErrAOIConfigUnavailable = errors.New("AOI config unavailable")

// 配置来自数据库，当前进程启动后固定使用一份，避免在线改邻域而漏发实体进入/离开。
type AOIConfig struct {
	RelayEnabled  bool      `json:"relay_enabled"`
	Enabled       bool      `json:"enabled"`
	CellSizeMilli uint32    `json:"cell_size_milli"`
	Radius        uint32    `json:"radius"`
	UpdatedAt     time.Time `json:"updated_at"`
	Reason        string    `json:"reason"`
	AdminUserID   uint64    `json:"admin_user_id"`
}

// 后台写入需携带审计原因；管理员标识由鉴权层赋值。
type UpdateAOIConfigInput struct {
	RelayEnabled  *bool  `json:"relay_enabled"`
	Enabled       bool   `json:"enabled"`
	CellSizeMilli uint32 `json:"cell_size_milli"`
	Radius        uint32 `json:"radius"`
	Reason        string `json:"reason"`
	AdminUserID   uint64 `json:"-"`
}

// 可选仓储保持旧替代仓储兼容，正式 PostgreSQL 仓储必须先执行对应迁移。
type AOIConfigRepository interface {
	GetAOIConfig(context.Context) (AOIConfig, error)
	UpdateAOIConfig(context.Context, UpdateAOIConfigInput) (AOIConfig, error)
}

// 启动加载，不在每个移动包中访问配置表；旧仓储未实现时沿用整场景广播。
func (s *Service) RefreshAOIConfig(ctx context.Context) error {
	repo, ok := s.repo.(AOIConfigRepository)
	if !ok {
		return nil
	}
	config, err := repo.GetAOIConfig(ctx)
	if err != nil {
		return err
	}
	if !validAOIConfig(config.CellSizeMilli, config.Radius) {
		return ErrAOIConfigInvalid
	}
	s.aoiMu.Lock()
	s.aoiConfig = config
	s.aoiMu.Unlock()
	return nil
}

// 返回当前进程实际使用的配置，与后台持久化的新配置分开显示。
func (s *Service) AOIConfigSnapshot() AOIConfig {
	s.aoiMu.RLock()
	defer s.aoiMu.RUnlock()
	return s.aoiConfig
}

// 后台返回数据库事实值和当前进程启动值；保存不热切换可见图。
func (s *Service) GetAdminAOIConfig(ctx context.Context) (map[string]AOIConfig, error) {
	repo, ok := s.repo.(AOIConfigRepository)
	if !ok {
		return nil, ErrAOIConfigUnavailable
	}
	config, err := repo.GetAOIConfig(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]AOIConfig{"configured": config, "active": s.AOIConfigSnapshot()}, nil
}

// 配置必须覆盖实际镜头与预加载余量，接口只校验安全数值范围；生效需要重启服务。
func (s *Service) UpdateAdminAOIConfig(ctx context.Context, input UpdateAOIConfigInput) (map[string]AOIConfig, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if !validAOIConfig(input.CellSizeMilli, input.Radius) || input.AdminUserID == 0 || input.Reason == "" || len([]rune(input.Reason)) > 500 {
		return nil, ErrAOIConfigInvalid
	}
	repo, ok := s.repo.(AOIConfigRepository)
	if !ok {
		return nil, ErrAOIConfigUnavailable
	}
	config, err := repo.UpdateAOIConfig(ctx, input)
	if err != nil {
		return nil, err
	}
	return map[string]AOIConfig{"configured": config, "active": s.AOIConfigSnapshot()}, nil
}

// 限制单包最多检查 9×9 个网格，防止错误配置产生无界邻域扫描。
func validAOIConfig(size, radius uint32) bool {
	return size >= 1000 && size <= 1000000 && radius >= 1 && radius <= 4
}

// Redis 共享场景索引能力不强迫旧内存/测试仓储实现，也不代表已经具备跨进程广播。
type SceneOnlineRepository interface {
	MarkSceneOnline(context.Context, uint64, uint32, string, string, time.Duration) error
	UnmarkSceneOnline(context.Context, uint64, string) error
	ListSceneOnlinePlayerIDs(context.Context, uint32) ([]uint64, error)
}

func (s *Service) MarkSceneOnline(ctx context.Context, id uint64, scene uint32, sessionID, owner string, lease time.Duration) error {
	if repo, ok := s.movementRepo.(SceneOnlineRepository); ok {
		return repo.MarkSceneOnline(ctx, id, scene, sessionID, owner, lease)
	}
	return nil
}
func (s *Service) UnmarkSceneOnline(ctx context.Context, id uint64, owner string) error {
	if repo, ok := s.movementRepo.(SceneOnlineRepository); ok {
		return repo.UnmarkSceneOnline(ctx, id, owner)
	}
	return nil
}
func (s *Service) ListSceneOnlinePlayerIDs(ctx context.Context, scene uint32) ([]uint64, error) {
	if repo, ok := s.movementRepo.(SceneOnlineRepository); ok {
		return repo.ListSceneOnlinePlayerIDs(ctx, scene)
	}
	return nil, ErrMovementStateNotFound
}

// 用可选能力查询共享连接代次，旧替代仓储没有跨实例事件时无需实现。
func (s *Service) SceneOnlineOwner(ctx context.Context, id uint64) (string, error) {
	if repo, ok := s.movementRepo.(interface {
		SceneOnlineOwner(context.Context, uint64) (string, error)
	}); ok {
		return repo.SceneOnlineOwner(ctx, id)
	}
	return "", ErrMovementStateNotFound
}
