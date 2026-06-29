package monster

import (
	"context"
	"errors"
	"strings"
	"sync"
)

const (
	RewardTypeExp  = "exp"
	RewardTypeItem = "item"
	RewardTypeGold = "gold"

	ExpTargetPlayer = "player"
	ExpTargetPet    = "pet"
)

var (
	ErrInvalidBattleRewardInput = errors.New("invalid monster battle reward input")
)

// BattleRewardEntry 描述单条怪物战斗奖励配置。
type BattleRewardEntry struct {
	ID         uint64 `json:"id"`
	MonsterID  uint32 `json:"monster_id"`
	RewardType string `json:"reward_type"`
	ExpTarget  string `json:"exp_target"`
	ItemID     uint64 `json:"item_id"`
	Quantity   uint64 `json:"quantity"`
	ExpValue   uint64 `json:"exp_value"`
	SortOrder  uint32 `json:"sort_order"`
	Status     uint32 `json:"status"`
	// GrantOnce 为 1 时表示该物品奖励每名玩家仅首次获得，之后战斗不再重复发放。
	GrantOnce uint32 `json:"grant_once"`
	// ItemName 仅后台列表展示用，来自 item_definition 关联查询，不入库。
	ItemName string `json:"item_name,omitempty"`
}

// AdminBattleRewardInput 是后台批量保存怪物战斗奖励时使用的输入项。
type AdminBattleRewardInput struct {
	RewardType string `json:"reward_type"`
	ExpTarget  string `json:"exp_target"`
	ItemID     uint64 `json:"item_id"`
	Quantity   uint64 `json:"quantity"`
	ExpValue   uint64 `json:"exp_value"`
	SortOrder  uint32 `json:"sort_order"`
	Status     uint32 `json:"status"`
	GrantOnce  uint32 `json:"grant_once"`
}

func (input AdminBattleRewardInput) Normalize() AdminBattleRewardInput {
	input.RewardType = strings.ToLower(strings.TrimSpace(input.RewardType))
	input.ExpTarget = strings.ToLower(strings.TrimSpace(input.ExpTarget))
	if input.ExpTarget == "" {
		input.ExpTarget = ExpTargetPlayer
	}
	if input.Status == 0 {
		input.Status = 1
	}
	if input.GrantOnce > 1 {
		input.GrantOnce = 1
	}
	return input
}

// AdminReplaceBattleRewardsInput 描述后台覆盖保存某怪物全部战斗奖励。
type AdminReplaceBattleRewardsInput struct {
	Rewards []AdminBattleRewardInput `json:"rewards"`
}

// PVERewardBundle 是服务端按怪物配置汇总后的 PVE 奖励结果。
type PVERewardBundle struct {
	PlayerExp uint64
	PetExp    uint64
	Gold      uint64
	Items     []PVEItemReward
}

// PVEItemReward 描述战斗掉落的物品奖励。
type PVEItemReward struct {
	ItemID    uint64
	Quantity  uint64
	GrantOnce bool
}

type battleRewardCache struct {
	mu      sync.RWMutex
	entries map[uint32][]BattleRewardEntry
}

func newBattleRewardCache() *battleRewardCache {
	return &battleRewardCache{entries: map[uint32][]BattleRewardEntry{}}
}

func (c *battleRewardCache) replace(entries []BattleRewardEntry) {
	grouped := make(map[uint32][]BattleRewardEntry)
	for _, entry := range entries {
		if entry.Status != 1 {
			continue
		}
		grouped[entry.MonsterID] = append(grouped[entry.MonsterID], entry)
	}
	c.mu.Lock()
	c.entries = grouped
	c.mu.Unlock()
}

func (c *battleRewardCache) bundleForMonster(monsterID uint32) PVERewardBundle {
	c.mu.RLock()
	entries := append([]BattleRewardEntry(nil), c.entries[monsterID]...)
	c.mu.RUnlock()

	bundle := PVERewardBundle{Items: []PVEItemReward{}}
	for _, entry := range entries {
		switch entry.RewardType {
		case RewardTypeExp:
			switch entry.ExpTarget {
			case ExpTargetPet:
				bundle.PetExp += entry.ExpValue
			default:
				bundle.PlayerExp += entry.ExpValue
			}
		case RewardTypeItem:
			if entry.ItemID == 0 || entry.Quantity == 0 {
				continue
			}
			bundle.Items = append(bundle.Items, PVEItemReward{
				ItemID:    entry.ItemID,
				Quantity:  entry.Quantity,
				GrantOnce: entry.GrantOnce == 1,
			})
		case RewardTypeGold:
			bundle.Gold += entry.ExpValue
		}
	}
	return bundle
}

// RefreshBattleRewardCache 从数据库加载全部启用中的怪物战斗奖励配置。
func (s *Service) RefreshBattleRewardCache(ctx context.Context) error {
	if s.repo == nil {
		return nil
	}
	entries, err := s.repo.ListBattleRewards(ctx)
	if err != nil {
		return err
	}
	if s.battleRewardCache == nil {
		s.battleRewardCache = newBattleRewardCache()
	}
	s.battleRewardCache.replace(entries)
	return nil
}

// ResolvePVERewardBundle 按怪物模板汇总战斗奖励，供 battle 模块消费。
func (s *Service) ResolvePVERewardBundle(monsterID uint32) PVERewardBundle {
	if s == nil || s.battleRewardCache == nil || monsterID == 0 {
		return PVERewardBundle{Items: []PVEItemReward{}}
	}
	return s.battleRewardCache.bundleForMonster(monsterID)
}

// ListAdminBattleRewards 返回指定怪物的战斗奖励配置。
func (s *Service) ListAdminBattleRewards(ctx context.Context, monsterID uint32) ([]BattleRewardEntry, error) {
	if s.repo == nil || monsterID == 0 {
		return nil, ErrInvalidBattleRewardInput
	}
	return s.repo.ListBattleRewardsByMonsterID(ctx, monsterID)
}

// ReplaceAdminBattleRewards 覆盖保存怪物战斗奖励并刷新运行时缓存。
func (s *Service) ReplaceAdminBattleRewards(ctx context.Context, monsterID uint32, input AdminReplaceBattleRewardsInput) ([]BattleRewardEntry, error) {
	if s.repo == nil || monsterID == 0 {
		return nil, ErrInvalidBattleRewardInput
	}
	normalized := make([]AdminBattleRewardInput, 0, len(input.Rewards))
	for index, reward := range input.Rewards {
		item := reward.Normalize()
		if err := validateAdminBattleRewardInput(item); err != nil {
			return nil, err
		}
		if item.SortOrder == 0 {
			item.SortOrder = uint32(index + 1)
		}
		normalized = append(normalized, item)
	}
	updated, err := s.repo.ReplaceBattleRewardsForMonster(ctx, monsterID, normalized)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshBattleRewardCache(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func validateAdminBattleRewardInput(input AdminBattleRewardInput) error {
	switch input.RewardType {
	case RewardTypeExp:
		if input.ExpValue == 0 {
			return ErrInvalidBattleRewardInput
		}
		if input.ExpTarget != ExpTargetPlayer && input.ExpTarget != ExpTargetPet {
			return ErrInvalidBattleRewardInput
		}
	case RewardTypeItem:
		if input.ItemID == 0 || input.Quantity == 0 {
			return ErrInvalidBattleRewardInput
		}
		if input.GrantOnce > 1 {
			return ErrInvalidBattleRewardInput
		}
	case RewardTypeGold:
		if input.ExpValue == 0 {
			return ErrInvalidBattleRewardInput
		}
	default:
		return ErrInvalidBattleRewardInput
	}
	return nil
}
