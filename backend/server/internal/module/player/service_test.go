package player

import (
	"context"
	"errors"
	"testing"
)

// worldTransferProfileRepositoryStub 通过嵌入旧仓储接口复用未参与本测试的方法，单独记录轻量查询和完整档案查询次数。
type worldTransferProfileRepositoryStub struct {
	Repository
	transferProfile *WorldTransferProfile
	transferErr     error
	fullProfile     *Profile
	fullProfileErr  error
	transferCount   int
	fullCount       int
}

// FindWorldTransferProfile 返回场景切换所需的轻量档案，并记录轻量查询次数。
func (r *worldTransferProfileRepositoryStub) FindWorldTransferProfile(_ context.Context, _ uint64) (*WorldTransferProfile, error) {
	r.transferCount++
	return r.transferProfile, r.transferErr
}

// FindByPlayerID 是兼容回退入口；轻量仓储可用时不应调用该方法。
func (r *worldTransferProfileRepositoryStub) FindByPlayerID(_ context.Context, _ uint64) (*Profile, error) {
	r.fullCount++
	return r.fullProfile, r.fullProfileErr
}

// TestGetWorldTransferProfileUsesLightweightRepository 验证切图读取不会触发完整玩家档案及战斗快照链路。
func TestGetWorldTransferProfileUsesLightweightRepository(t *testing.T) {
	want := &WorldTransferProfile{
		PlayerID: 10001, Level: 8, SceneID: 3, PosX: 12, PosY: 9, PositionVersion: 27,
	}
	repo := &worldTransferProfileRepositoryStub{
		transferProfile: want,
		fullProfileErr:  errors.New("full profile must not be loaded"),
	}
	service := NewService(repo, nil, nil, nil)

	got, err := service.GetWorldTransferProfile(context.Background(), want.PlayerID)
	if err != nil {
		t.Fatalf("GetWorldTransferProfile() error = %v", err)
	}
	if got != want {
		t.Fatalf("GetWorldTransferProfile() = %+v, want %+v", got, want)
	}
	if repo.transferCount != 1 || repo.fullCount != 0 {
		t.Fatalf("repository calls lightweight=%d full=%d, want 1 and 0", repo.transferCount, repo.fullCount)
	}
}

// TestGetWorldTransferProfileReturnsNotFound 验证轻量查询没有找到玩家时保持领域层统一错误语义。
func TestGetWorldTransferProfileReturnsNotFound(t *testing.T) {
	repo := &worldTransferProfileRepositoryStub{}
	service := NewService(repo, nil, nil, nil)

	profile, err := service.GetWorldTransferProfile(context.Background(), 10001)
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("GetWorldTransferProfile() error = %v, want %v", err, ErrPlayerNotFound)
	}
	if profile != nil || repo.transferCount != 1 || repo.fullCount != 0 {
		t.Fatalf("profile=%+v lightweight=%d full=%d, want nil, 1 and 0", profile, repo.transferCount, repo.fullCount)
	}
}
