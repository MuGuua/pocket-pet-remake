package world

import (
	"context"
	"errors"
	"testing"
)

// 配置仓储模拟持久化事实，运行快照必须在启动加载后独立于后台保存。
type aoiConfigStoreStub struct {
	Repository
	config  AOIConfig
	writes  int
	readErr error
}

func (r *aoiConfigStoreStub) GetAOIConfig(context.Context) (AOIConfig, error) {
	return r.config, r.readErr
}
func (r *aoiConfigStoreStub) UpdateAOIConfig(_ context.Context, input UpdateAOIConfigInput) (AOIConfig, error) {
	r.writes++
	r.config = AOIConfig{Enabled: input.Enabled, CellSizeMilli: input.CellSizeMilli, Radius: input.Radius, Reason: input.Reason, AdminUserID: input.AdminUserID}
	return r.config, nil
}

// 后台保存不会热切换可见图，重新构造服务并加载后才使用新配置。
func TestAOIConfigPersistenceAndRestartSemantics(t *testing.T) {
	repo := &aoiConfigStoreStub{config: AOIConfig{CellSizeMilli: 24000, Radius: 1}}
	service := NewService(repo)
	if err := service.RefreshAOIConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	input := UpdateAOIConfigInput{Enabled: true, CellSizeMilli: 12000, Radius: 2, Reason: "  测试配置  ", AdminUserID: 1}
	state, err := service.UpdateAdminAOIConfig(context.Background(), input)
	if err != nil || !state["configured"].Enabled || state["active"].Enabled || state["configured"].Reason != "测试配置" {
		t.Fatalf("configured/active mismatch %+v %v", state, err)
	}
	for _, bad := range []UpdateAOIConfigInput{
		{CellSizeMilli: 999, Radius: 1, Reason: "test", AdminUserID: 1},
		{CellSizeMilli: 1000001, Radius: 1, Reason: "test", AdminUserID: 1},
		{CellSizeMilli: 12000, Radius: 0, Reason: "test", AdminUserID: 1},
		{CellSizeMilli: 12000, Radius: 5, Reason: "test", AdminUserID: 1},
		{CellSizeMilli: 12000, Radius: 1, Reason: " ", AdminUserID: 1},
		{CellSizeMilli: 12000, Radius: 1, Reason: "test"},
	} {
		if _, err := service.UpdateAdminAOIConfig(context.Background(), bad); !errors.Is(err, ErrAOIConfigInvalid) {
			t.Fatal("invalid config accepted")
		}
	}
	if repo.writes != 1 {
		t.Fatal("invalid input reached persistence")
	}
	restarted := NewService(repo)
	if err := restarted.RefreshAOIConfig(context.Background()); err != nil || !restarted.AOIConfigSnapshot().Enabled {
		t.Fatal("restart did not load persisted config")
	}
}
