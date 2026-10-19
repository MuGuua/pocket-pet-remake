package monster

import (
	"math/rand"
	"strings"
)

// BossDropEntry 描述 Boss 掉落池中的一条候选掉落配置。
// 同一 draw_group 内的条目按 weight 加权抽取一个；不同 draw_group 相互独立，可以同时命中。
type BossDropEntry struct {
	ID       uint64 `json:"id"`
	SceneID  uint32 `json:"scene_id"`
	ItemID   uint64 `json:"item_id"`
	Quantity uint64 `json:"quantity"`
	// Weight 是加权抽取的权重，必须大于 0；同组内权重之和为抽取基数。
	Weight int `json:"weight"`
	// DrawGroup 是抽取组编号，从 1 开始；同组抽 1 个，不同组互不影响。
	DrawGroup int `json:"draw_group"`
	// GrantOnce 为 1 时表示每名玩家仅首次可获得该物品。
	GrantOnce int `json:"grant_once"`
	SortOrder int `json:"sort_order"`
	Status    int `json:"status"`
	// ItemName 仅后台列表展示用，来自 item_definition 关联查询，不入库。
	ItemName string `json:"item_name,omitempty"`
}

// NormalizeBossDropEntry 统一把后台输入收敛到合法的入库形态。
// 非法权重、组号与数量都在这里兜底，避免脏数据进入抽取流程后概率计算出现除零。
func NormalizeBossDropEntry(entry BossDropEntry) BossDropEntry {
	if entry.Quantity == 0 {
		entry.Quantity = 1
	}
	if entry.Weight <= 0 {
		entry.Weight = 100
	}
	if entry.DrawGroup <= 0 {
		entry.DrawGroup = 1
	}
	if entry.GrantOnce > 1 {
		entry.GrantOnce = 1
	}
	if entry.Status == 0 {
		entry.Status = 1
	}
	return entry
}

// AdminBossDropInput 是后台保存 Boss 掉落池时的输入项，字段与展示结构一致但不含主键。
type AdminBossDropInput struct {
	ItemID    uint64 `json:"item_id"`
	Quantity  uint64 `json:"quantity"`
	Weight    int    `json:"weight"`
	DrawGroup int    `json:"draw_group"`
	GrantOnce int    `json:"grant_once"`
	SortOrder int    `json:"sort_order"`
	Status    int    `json:"status"`
}

// AdminReplaceBossDropsInput 描述覆盖保存某遭遇战全部 Boss 掉落。
type AdminReplaceBossDropsInput struct {
	Drops []AdminBossDropInput `json:"drops"`
}

// SanitizeBossDropInputs 把后台输入转换为合法的领域条目，并补齐默认排序号。
func SanitizeBossDropInputs(inputs []AdminBossDropInput) []BossDropEntry {
	result := make([]BossDropEntry, 0, len(inputs))
	for index, input := range inputs {
		entry := NormalizeBossDropEntry(BossDropEntry{
			ItemID:    input.ItemID,
			Quantity:  input.Quantity,
			Weight:    input.Weight,
			DrawGroup: input.DrawGroup,
			GrantOnce: input.GrantOnce,
			SortOrder: input.SortOrder,
			Status:    input.Status,
		})
		if entry.ItemID == 0 {
			continue
		}
		if entry.SortOrder == 0 {
			entry.SortOrder = index + 1
		}
		result = append(result, entry)
	}
	return result
}

// DrawBossDrops 按抽取组从 Boss 掉落池中抽取本次战斗实际获得的物品。
// 每个 draw_group 独立抽一次；组内按 weight 加权，权重越大越容易命中。
// entries 必须已经按 status 过滤；本函数只处理抽取，不再重复判断启用状态。
func DrawBossDrops(entries []BossDropEntry) []PVEItemReward {
	if len(entries) == 0 {
		return []PVEItemReward{}
	}
	grouped := make(map[int][]BossDropEntry)
	for _, entry := range entries {
		if entry.Status != 1 || entry.ItemID == 0 || entry.Weight <= 0 {
			continue
		}
		group := entry.DrawGroup
		if group <= 0 {
			group = 1
		}
		grouped[group] = append(grouped[group], entry)
	}
	if len(grouped) == 0 {
		return []PVEItemReward{}
	}
	result := make([]PVEItemReward, 0, len(grouped))
	// 固定按组号升序抽取，保证同一份配置在不同进程里产出顺序稳定，便于测试与日志比对。
	for group := 1; group <= len(grouped)+1; group++ {
		candidates, exists := grouped[group]
		if !exists || len(candidates) == 0 {
			continue
		}
		selected := pickWeightedBossDrop(candidates)
		if selected == nil {
			continue
		}
		result = append(result, PVEItemReward{
			ItemID:    selected.ItemID,
			Quantity:  selected.Quantity,
			GrantOnce: selected.GrantOnce == 1,
		})
	}
	return result
}

// pickWeightedBossDrop 在候选列表内按权重随机抽取一条；总权重非法时回退到第一条。
func pickWeightedBossDrop(candidates []BossDropEntry) *BossDropEntry {
	totalWeight := 0
	for index := range candidates {
		weight := candidates[index].Weight
		if weight <= 0 {
			continue
		}
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return &candidates[0]
	}
	roll := rand.Intn(totalWeight)
	accumulated := 0
	for index := range candidates {
		weight := candidates[index].Weight
		if weight <= 0 {
			continue
		}
		accumulated += weight
		if roll < accumulated {
			return &candidates[index]
		}
	}
	return &candidates[len(candidates)-1]
}

// BossDropsForScene 过滤出指定场景启用中的掉落条目，供抽取与后台展示复用。
func BossDropsForScene(entries []BossDropEntry, sceneID uint32) []BossDropEntry {
	if sceneID == 0 {
		return nil
	}
	result := make([]BossDropEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.SceneID != sceneID || entry.Status != 1 {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// trimBossDropItemName 仅用于后台展示的空值兜底，避免前端出现 null。
func trimBossDropItemName(name string) string {
	return strings.TrimSpace(name)
}
