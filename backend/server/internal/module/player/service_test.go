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

// 只读仓储单独统计展示和旧档案入口，避免错误时隐式回退到写快照路径。
type displayProfileRepositoryStub struct {
	*worldTransferProfileRepositoryStub
	profile *Profile
	err     error
	reads   int
}

func (r *displayProfileRepositoryStub) FindDisplayProfile(context.Context, uint64) (*Profile, error) {
	r.reads++
	return r.profile, r.err
}

// 固定战斗状态用于确认非战斗优化不会改变战斗期读取分支。
type displayBattleState bool

func (s displayBattleState) IsPlayerInActiveBattle(uint64) bool { return bool(s) }

// 展示优先走只读能力，缺失和失败不重查旧视图；战斗期及旧仓储保留原行为。
func TestDisplayProfileRepositoryPaths(t *testing.T) {
	for _, mode := range []string{"fresh", "missing", "error", "battle"} {
		t.Run(mode, func(t *testing.T) {
			repo := &displayProfileRepositoryStub{worldTransferProfileRepositoryStub: &worldTransferProfileRepositoryStub{fullProfile: &Profile{PlayerID: 1, ATK: 10}}, profile: &Profile{PlayerID: 1, ATK: 99}}
			if mode == "missing" {
				repo.profile = nil
			}
			if mode == "error" {
				repo.err = errors.New("source unavailable")
			}
			svc := NewService(repo, nil, nil, nil)
			svc.SetBattleChecker(displayBattleState(mode == "battle"))
			actual, err := svc.GetDisplayProfile(context.Background(), 1)
			switch mode {
			case "fresh":
				if err != nil || actual.ATK != 99 || repo.reads != 1 || repo.fullCount != 0 {
					t.Fatal("readonly route not selected")
				}
			case "missing":
				if !errors.Is(err, ErrPlayerNotFound) || repo.fullCount != 0 {
					t.Fatal("missing source must fail")
				}
			case "error":
				if !errors.Is(err, repo.err) || repo.fullCount != 0 {
					t.Fatal("source error must not fall back")
				}
			case "battle":
				if err != nil || actual.ATK != 10 || repo.reads != 0 || repo.fullCount != 1 {
					t.Fatal("battle read path changed")
				}
			}
		})
	}
	legacy := &worldTransferProfileRepositoryStub{fullProfile: &Profile{PlayerID: 1}}
	if _, err := NewService(legacy, nil, nil, nil).GetDisplayProfile(context.Background(), 1); err != nil || legacy.fullCount != 1 {
		t.Fatal("legacy repository fallback failed")
	}
}

// 可独立控制刷新/读取结果，验证非战斗开战档案不再重复进入完整读刷新链路。
type readySnapshotRepositoryStub struct {
	Repository
	snapshot   *Profile
	fallback   *Profile
	refreshErr error
	readErr    error
	refreshes  int
	reads      int
	fullReads  int
}

// 模拟一次成功或失败的持久化刷新，并记录次数。
func (r *readySnapshotRepositoryStub) RefreshPlayerCombatSnapshot(context.Context, uint64) error {
	r.refreshes++
	return r.refreshErr
}

// 返回刚刷新完成的快照，不触发再次刷新。
func (r *readySnapshotRepositoryStub) FindPlayerCombatSnapshot(context.Context, uint64) (*Profile, error) {
	r.reads++
	return r.snapshot, r.readErr
}

// 记录旧入口回退，战斗期及缺失快照的兼容语义必须继续保留。
func (r *readySnapshotRepositoryStub) FindByPlayerID(context.Context, uint64) (*Profile, error) {
	r.fullReads++
	return r.fallback, nil
}

// 校验成功、错误、缺失与战斗分支；错误不得以完整旧档案静默替代。
func TestBattleReadyProfileReadsRefreshedSnapshotOnce(t *testing.T) {
	for _, mode := range []string{"success", "refresh_error", "read_error", "missing", "battle"} {
		t.Run(mode, func(t *testing.T) {
			want := &Profile{PlayerID: 1, ATK: 99, Exp: 10}
			repo := &readySnapshotRepositoryStub{snapshot: want, fallback: &Profile{PlayerID: 1, ATK: 7}}
			if mode == "refresh_error" {
				repo.refreshErr = errors.New("refresh failed")
			}
			if mode == "read_error" {
				repo.readErr = errors.New("read failed")
			}
			if mode == "missing" {
				repo.snapshot = nil
			}
			svc := NewService(repo, nil, nil, nil)
			svc.SetBattleChecker(displayBattleState(mode == "battle"))
			got, err := svc.GetBattleReadyProfile(context.Background(), 1)
			switch mode {
			case "refresh_error":
				if !errors.Is(err, repo.refreshErr) || repo.reads != 0 || repo.fullReads != 0 {
					t.Fatal("refresh error behavior changed")
				}
			case "read_error":
				if !errors.Is(err, repo.readErr) || repo.reads != 1 || repo.fullReads != 0 {
					t.Fatal("read error hidden by fallback")
				}
			case "missing":
				if err != nil || got != repo.fallback || repo.fullReads != 1 {
					t.Fatal("missing snapshot fallback changed")
				}
			case "battle":
				if err != nil || got != repo.fallback || repo.refreshes != 0 || repo.reads != 0 || repo.fullReads != 1 {
					t.Fatal("active battle path changed")
				}
			default:
				if err != nil || got != want || repo.refreshes != 1 || repo.reads != 1 || repo.fullReads != 0 {
					t.Fatal("successful refreshed snapshot reentered full profile read")
				}
			}
		})
	}
}

// 同一用例的外层已刷新时直接读结果，不再写库；旧仓储仍走既有兼容入口。
func TestPreparedBattleProfileDoesNotRefresh(t *testing.T) {
	repo := &readySnapshotRepositoryStub{snapshot: &Profile{PlayerID: 1, ATK: 123}}
	result, err := NewService(repo, nil, nil, nil).GetPreparedBattleProfile(context.Background(), 1)
	if err != nil || result.ATK != 123 || repo.refreshes != 0 || repo.reads != 1 || repo.fullReads != 0 {
		t.Fatal("prepared profile triggered duplicate refresh")
	}
}
