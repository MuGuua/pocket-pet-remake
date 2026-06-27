package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pocket-pet-remake/server/internal/module/bag"
)

const runtimeBagOwnedItemQuantityQuery = `
SELECT COALESCE(SUM(quantity), 0)
FROM player_container_item
WHERE player_id = $1
  AND container_type = $2
  AND item_id = $3
`

const runtimeBagEnhanceMaterialsQuery = `
SELECT
  pci.item_id,
  COALESCE(idf.item_name, ''),
  COALESCE(SUM(pci.quantity), 0)::bigint
FROM player_container_item pci
JOIN item_definition idf ON idf.item_id = pci.item_id
WHERE pci.player_id = $1
  AND pci.container_type = $2
  AND idf.item_sub_type = $3
  AND idf.is_enabled = TRUE
  AND pci.quantity > 0
GROUP BY pci.item_id, idf.item_name, idf.sort_weight
ORDER BY idf.sort_weight DESC, pci.item_id ASC
`

type enhancePreviewStatDef struct {
	bonusKey string
	label    string
}

// runtimeEnhancePreviewStatDefs 按客户端展示顺序列出所有可参与强化预览的属性键。
var runtimeEnhancePreviewStatDefs = []enhancePreviewStatDef{
	{bonusKey: "hp_max", label: "生命"},
	{bonusKey: "mana", label: "法力"},
	{bonusKey: "atk", label: "攻击"},
	{bonusKey: "def", label: "防御"},
	{bonusKey: "spd", label: "速度"},
	{bonusKey: "spirit", label: "精力"},
	{bonusKey: "spirit_max", label: "精力上限"},
	{bonusKey: "hit_pct", label: "命中"},
	{bonusKey: "dodge_pct", label: "闪避"},
	{bonusKey: "crit_rate_pct", label: "致命"},
	{bonusKey: "crit_dmg_pct", label: "爆伤"},
	{bonusKey: "physical_resist_pct", label: "物抗"},
	{bonusKey: "reverse_physical_resist_pct", label: "逆物抗"},
	{bonusKey: "skill_resist_pct", label: "技抗"},
	{bonusKey: "reverse_skill_resist_pct", label: "逆技抗"},
	{bonusKey: "confusion_resist_pct", label: "混乱抗性"},
	{bonusKey: "sleep_resist_pct", label: "昏睡抗性"},
	{bonusKey: "paralysis_resist_pct", label: "麻痹抗性"},
	{bonusKey: "seal_resist_pct", label: "封印抗性"},
	{bonusKey: "curse_resist_pct", label: "诅咒抗性"},
	{bonusKey: "crit_dmg_resist_pct", label: "抗爆伤"},
	{bonusKey: "crit_resist_pct", label: "抗致命"},
	{bonusKey: "character_resist_pct", label: "抗人物"},
	{bonusKey: "pet_resist_pct", label: "抗宠物"},
}

// buildRuntimeEnhancePreview 为背包内可强化装备组装强化弹窗预览数据。
func buildRuntimeEnhancePreview(
	ctx context.Context,
	db DBTX,
	playerID uint64,
	itemID uint64,
	canEnhance bool,
	maxEnhanceLevel uint32,
	enhanceLevel uint32,
	equipSlot string,
	appearanceSkinID string,
	appearanceOnly bool,
	baseHP uint32,
	baseMana uint32,
	baseATK uint32,
	baseDEF uint32,
	baseSPD uint32,
	baseStatsJSON []byte,
	enhancePerLevelStatsJSON []byte,
	requiredLevel uint32,
) (*bag.RuntimeEnhancePreview, error) {
	if equipSlot == "" {
		return nil, nil
	}
	materials, err := loadRuntimeBagEnhanceMaterials(ctx, db, playerID, bag.ContainerTypeBag)
	if err != nil {
		return nil, err
	}
	preview := &bag.RuntimeEnhancePreview{
		CanEnhance:              false,
		MaxEnhanceLevel:         maxEnhanceLevel,
		EnhanceMaterialCategory: bag.ItemSubTypeEquipmentEnhance,
		Materials:               materials,
		Rows:                    []bag.RuntimeEnhancePreviewRow{},
	}
	if maxEnhanceLevel == 0 {
		return preview, nil
	}
	enhancePerLevel, err := unmarshalEnhancePerLevelJSON(enhancePerLevelStatsJSON)
	if err != nil {
		return nil, err
	}
	currentBonus, err := computeRuntimeItemBonusFromEquipmentExtra(
		equipSlot,
		appearanceSkinID,
		appearanceOnly,
		baseHP,
		baseMana,
		baseATK,
		baseDEF,
		baseSPD,
		baseStatsJSON,
		enhancePerLevelStatsJSON,
		requiredLevel,
		enhanceLevel,
	)
	if err != nil {
		return nil, err
	}
	currentSnapshot := runtimeItemBonusFromAggregate(currentBonus)
	if enhanceLevel >= maxEnhanceLevel {
		preview.Rows = buildRuntimeEnhancePreviewRowsAtMax(enhanceLevel, enhancePerLevel, currentSnapshot)
		applyDefaultEnhancePreviewCostHint(preview, materials)
		return preview, nil
	}
	targetLevel := enhanceLevel + 1
	nextBonus, err := computeRuntimeItemBonusFromEquipmentExtra(
		equipSlot,
		appearanceSkinID,
		appearanceOnly,
		baseHP,
		baseMana,
		baseATK,
		baseDEF,
		baseSPD,
		baseStatsJSON,
		enhancePerLevelStatsJSON,
		requiredLevel,
		targetLevel,
	)
	if err != nil {
		return nil, err
	}
	preview.Rows = buildRuntimeEnhancePreviewRows(
		enhanceLevel,
		targetLevel,
		enhancePerLevel,
		currentSnapshot,
		runtimeItemBonusFromAggregate(nextBonus),
	)
	if !canEnhance {
		applyDefaultEnhancePreviewCostHint(preview, materials)
		return preview, nil
	}
	cost, err := loadRuntimeEnhanceCost(ctx, db, itemID, targetLevel)
	if err != nil {
		return nil, err
	}
	if cost == nil {
		applyDefaultEnhancePreviewCostHint(preview, materials)
		return preview, nil
	}
	successRatePct, err := loadRuntimeEnhanceSuccessRate(ctx, db, targetLevel)
	if err != nil {
		return nil, err
	}
	ownedQuantity, err := loadRuntimeBagOwnedItemQuantity(ctx, db, playerID, bag.ContainerTypeBag, cost.CostItemID)
	if err != nil {
		return nil, err
	}
	costItemName, err := loadItemNameByID(ctx, db, cost.CostItemID)
	if err != nil {
		return nil, err
	}
	preview.CanEnhance = true
	preview.SuccessRatePct = successRatePct
	preview.CostItemID = cost.CostItemID
	preview.CostItemName = costItemName
	preview.CostQuantity = cost.CostQuantity
	preview.CostGoldCopper = cost.CostGoldCopper
	preview.OwnedCostQuantity = ownedQuantity
	return preview, nil
}

// applyDefaultEnhancePreviewCostHint 在不可强化时仍回填默认材料展示，避免客户端材料区空白。
func applyDefaultEnhancePreviewCostHint(preview *bag.RuntimeEnhancePreview, materials []bag.RuntimeEnhanceMaterialOption) {
	if preview == nil || preview.CostItemID > 0 || len(materials) == 0 {
		return
	}
	preview.CostItemID = materials[0].ItemID
	preview.CostItemName = materials[0].ItemName
	preview.OwnedCostQuantity = materials[0].OwnedQuantity
}

func buildRuntimeEnhancePreviewRows(
	enhanceLevel uint32,
	targetLevel uint32,
	enhancePerLevel map[string]uint32,
	currentSnapshot bag.RuntimeItemBonus,
	nextSnapshot bag.RuntimeItemBonus,
) []bag.RuntimeEnhancePreviewRow {
	rows := make([]bag.RuntimeEnhancePreviewRow, 0, len(runtimeEnhancePreviewStatDefs)+1)
	rows = append(rows, bag.RuntimeEnhancePreviewRow{
		Label:   "强化等级",
		Current: formatEnhancePreviewLevel(enhanceLevel),
		NextMin: formatEnhancePreviewLevel(targetLevel),
		NextMax: formatEnhancePreviewLevel(targetLevel),
	})
	for _, statDef := range runtimeEnhancePreviewStatDefs {
		if enhancePerLevel[statDef.bonusKey] == 0 {
			continue
		}
		isPct := isEnhancePreviewPctKey(statDef.bonusKey)
		currentValue := bonusStatValue(currentSnapshot, statDef.bonusKey)
		nextValue := bonusStatValue(nextSnapshot, statDef.bonusKey)
		nextText := formatEnhancePreviewStat(nextValue, isPct)
		rows = append(rows, bag.RuntimeEnhancePreviewRow{
			Label:   statDef.label,
			Current: formatEnhancePreviewStat(currentValue, isPct),
			NextMin: nextText,
			NextMax: nextText,
		})
	}
	return rows
}

// buildRuntimeEnhancePreviewRowsAtMax 在已达最高强化等级时，仅返回当前属性行（不含下一等级列）。
func buildRuntimeEnhancePreviewRowsAtMax(
	enhanceLevel uint32,
	enhancePerLevel map[string]uint32,
	currentSnapshot bag.RuntimeItemBonus,
) []bag.RuntimeEnhancePreviewRow {
	rows := make([]bag.RuntimeEnhancePreviewRow, 0, len(runtimeEnhancePreviewStatDefs)+1)
	rows = append(rows, bag.RuntimeEnhancePreviewRow{
		Label:   "强化等级",
		Current: formatEnhancePreviewLevel(enhanceLevel),
		NextMin: "max",
		NextMax: "max",
	})
	for _, statDef := range runtimeEnhancePreviewStatDefs {
		if enhancePerLevel[statDef.bonusKey] == 0 {
			continue
		}
		isPct := isEnhancePreviewPctKey(statDef.bonusKey)
		currentValue := bonusStatValue(currentSnapshot, statDef.bonusKey)
		rows = append(rows, bag.RuntimeEnhancePreviewRow{
			Label:   statDef.label,
			Current: formatEnhancePreviewStat(currentValue, isPct),
			NextMin: "max",
			NextMax: "max",
		})
	}
	return rows
}

func loadRuntimeBagEnhanceMaterials(
	ctx context.Context,
	db DBTX,
	playerID uint64,
	containerType string,
) ([]bag.RuntimeEnhanceMaterialOption, error) {
	if playerID == 0 {
		return []bag.RuntimeEnhanceMaterialOption{}, nil
	}
	rows, err := db.QueryContext(
		ctx,
		runtimeBagEnhanceMaterialsQuery,
		playerID,
		containerType,
		bag.ItemSubTypeEquipmentEnhance,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	materials := make([]bag.RuntimeEnhanceMaterialOption, 0, 4)
	for rows.Next() {
		var itemID int64
		var itemName string
		var ownedQuantity int64
		if scanErr := rows.Scan(&itemID, &itemName, &ownedQuantity); scanErr != nil {
			return nil, scanErr
		}
		if itemID <= 0 || ownedQuantity <= 0 {
			continue
		}
		materials = append(materials, bag.RuntimeEnhanceMaterialOption{
			ItemID:        uint64(itemID),
			ItemName:      itemName,
			OwnedQuantity: uint64(ownedQuantity),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return materials, nil
}

func loadRuntimeBagOwnedItemQuantity(ctx context.Context, db DBTX, playerID uint64, containerType string, itemID uint64) (uint64, error) {
	if playerID == 0 || itemID == 0 {
		return 0, nil
	}
	var totalQuantity int64
	if err := db.QueryRowContext(ctx, runtimeBagOwnedItemQuantityQuery, playerID, containerType, itemID).Scan(&totalQuantity); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if totalQuantity < 0 {
		return 0, nil
	}
	return uint64(totalQuantity), nil
}

func loadItemNameByID(ctx context.Context, db DBTX, itemID uint64) (string, error) {
	if itemID == 0 {
		return "", nil
	}
	var itemName string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(item_name, '') FROM item_definition WHERE item_id = $1 LIMIT 1`, itemID).Scan(&itemName); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return itemName, nil
}

func bonusStatValue(bonus bag.RuntimeItemBonus, key string) uint32 {
	switch key {
	case "hp_max":
		return bonus.HPMax
	case "mana":
		return bonus.MANA
	case "atk":
		return bonus.ATK
	case "def":
		return bonus.DEF
	case "spd":
		return bonus.SPD
	case "spirit":
		return bonus.Spirit
	case "spirit_max":
		return bonus.SpiritMax
	case "hit_pct":
		return bonus.HitPct
	case "dodge_pct":
		return bonus.DodgePct
	case "crit_rate_pct":
		return bonus.CritRatePct
	case "crit_dmg_pct":
		return bonus.CritDmgPct
	case "physical_resist_pct":
		return bonus.PhysicalResistPct
	case "reverse_physical_resist_pct":
		return bonus.ReversePhysicalResistPct
	case "skill_resist_pct":
		return bonus.SkillResistPct
	case "reverse_skill_resist_pct":
		return bonus.ReverseSkillResistPct
	case "confusion_resist_pct":
		return bonus.ConfusionResistPct
	case "sleep_resist_pct":
		return bonus.SleepResistPct
	case "paralysis_resist_pct":
		return bonus.ParalysisResistPct
	case "seal_resist_pct":
		return bonus.SealResistPct
	case "curse_resist_pct":
		return bonus.CurseResistPct
	case "crit_dmg_resist_pct":
		return bonus.CritDmgResistPct
	case "crit_resist_pct":
		return bonus.CritResistPct
	case "character_resist_pct":
		return bonus.CharacterResistPct
	case "pet_resist_pct":
		return bonus.PetResistPct
	default:
		return 0
	}
}

func formatEnhancePreviewLevel(level uint32) string {
	return fmt.Sprintf("+%d", level)
}

func formatEnhancePreviewStat(value uint32, isPct bool) string {
	if isPct {
		return fmt.Sprintf("+%d%%", value)
	}
	return fmt.Sprintf("+%d", value)
}

func isEnhancePreviewPctKey(key string) bool {
	if key == "hit_pct" {
		return false
	}
	return strings.HasSuffix(key, "_pct")
}
