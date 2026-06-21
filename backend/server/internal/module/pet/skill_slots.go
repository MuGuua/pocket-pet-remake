package pet

// SkillLoadout 描述宠物实例上各分类技能槽的持久化快照。
// 战斗合并结果写入 Pet.SkillIDs；查看技能 UI 使用 BuildSkillSlotView 生成结构化视图。
type SkillLoadout struct {
	// InnateSkillIDs 天生技，最多 5 个 skill_id，0 表示空槽。
	InnateSkillIDs []uint32
	// NormalSkillIDs 普通技，固定 3 槽，默认开启。
	NormalSkillIDs []uint32
	// ActiveTalismanSkillID 主动神符技 skill_id。
	ActiveTalismanSkillID uint32
	// TalismanHeroSkillID 神符技·英雄 skill_id。
	TalismanHeroSkillID uint32
	// TalismanSlot1SkillID 神符技【1】skill_id。
	TalismanSlot1SkillID uint32
	// TalismanSlot2SkillID 神符技【2】skill_id。
	TalismanSlot2SkillID uint32
	// TalismanSlot3SkillID 神符技【3】skill_id。
	TalismanSlot3SkillID uint32
	// ActiveTalismanEnabled 主动神符技是否已由道具开启。
	ActiveTalismanEnabled bool
	// TalismanHeroEnabled 神符·英雄是否已开启。
	TalismanHeroEnabled bool
	// TalismanSlot1Enabled 神符【1】是否已开启。
	TalismanSlot1Enabled bool
	// TalismanSlot2Enabled 神符【2】是否已开启。
	TalismanSlot2Enabled bool
	// TalismanSlot3Enabled 神符【3】是否已开启。
	TalismanSlot3Enabled bool
	// ArtifactSkillIDs 法宝技三槽，下标 0~2 对应 slot_index。
	ArtifactSkillIDs [MaxArtifactSkillSlots]uint32
}

const (
	// MaxInnateSkillSlots 天生技槽位上限。
	MaxInnateSkillSlots = 5
	// MaxNormalSkillSlots 普通技槽位数。
	MaxNormalSkillSlots = 3
	// MaxArtifactSkillSlots 法宝技槽位数。
	MaxArtifactSkillSlots = 3
)

// NormalizeSkillLoadout 把各数组槽位裁剪/补齐到约定长度，便于合并与协议输出。
func NormalizeSkillLoadout(loadout SkillLoadout) SkillLoadout {
	loadout.InnateSkillIDs = normalizeSkillIDSlice(loadout.InnateSkillIDs, MaxInnateSkillSlots)
	loadout.NormalSkillIDs = normalizeSkillIDSlice(loadout.NormalSkillIDs, MaxNormalSkillSlots)
	return loadout
}

// ApplyLegacySkillIDs 在分槽字段尚未回填时，把旧 skill_ids 前 3 个迁入普通技槽。
func ApplyLegacySkillIDs(loadout *SkillLoadout, legacySkillIDs []uint32) {
	if loadout == nil || loadout.hasStructuredData() {
		return
	}
	if len(legacySkillIDs) == 0 {
		return
	}
	limit := MaxNormalSkillSlots
	if len(legacySkillIDs) < limit {
		limit = len(legacySkillIDs)
	}
	loadout.NormalSkillIDs = append([]uint32{}, legacySkillIDs[:limit]...)
}

func (loadout SkillLoadout) hasStructuredData() bool {
	if len(loadout.InnateSkillIDs) > 0 {
		return true
	}
	if len(loadout.NormalSkillIDs) > 0 {
		return true
	}
	if loadout.ActiveTalismanSkillID > 0 || loadout.TalismanHeroSkillID > 0 {
		return true
	}
	if loadout.TalismanSlot1SkillID > 0 || loadout.TalismanSlot2SkillID > 0 || loadout.TalismanSlot3SkillID > 0 {
		return true
	}
	if loadout.ActiveTalismanEnabled || loadout.TalismanHeroEnabled ||
		loadout.TalismanSlot1Enabled || loadout.TalismanSlot2Enabled || loadout.TalismanSlot3Enabled {
		return true
	}
	for _, skillID := range loadout.ArtifactSkillIDs {
		if skillID > 0 {
			return true
		}
	}
	return false
}

// BuildBattleSkillIDs 按玩法顺序去重合并战斗可用技能（含已装备法宝技）。
func BuildBattleSkillIDs(loadout SkillLoadout) []uint32 {
	loadout = NormalizeSkillLoadout(loadout)
	seen := make(map[uint32]struct{}, 16)
	result := make([]uint32, 0, 16)
	appendSkill := func(skillID uint32) {
		if skillID == 0 {
			return
		}
		if _, exists := seen[skillID]; exists {
			return
		}
		seen[skillID] = struct{}{}
		result = append(result, skillID)
	}
	for _, skillID := range loadout.InnateSkillIDs {
		appendSkill(skillID)
	}
	for _, skillID := range loadout.NormalSkillIDs {
		appendSkill(skillID)
	}
	if loadout.ActiveTalismanEnabled {
		appendSkill(loadout.ActiveTalismanSkillID)
	}
	if loadout.TalismanHeroEnabled {
		appendSkill(loadout.TalismanHeroSkillID)
	}
	if loadout.TalismanSlot1Enabled {
		appendSkill(loadout.TalismanSlot1SkillID)
	}
	if loadout.TalismanSlot2Enabled {
		appendSkill(loadout.TalismanSlot2SkillID)
	}
	if loadout.TalismanSlot3Enabled {
		appendSkill(loadout.TalismanSlot3SkillID)
	}
	for _, skillID := range loadout.ArtifactSkillIDs {
		appendSkill(skillID)
	}
	return result
}

// ResolvePetBattleSkills 根据分槽数据重算 Pet.SkillIDs；若分槽全空则保留 legacy skill_ids。
func ResolvePetBattleSkills(item *Pet) {
	if item == nil {
		return
	}
	ApplyLegacySkillIDs(&item.SkillLoadout, item.SkillIDs)
	battleSkills := BuildBattleSkillIDs(item.SkillLoadout)
	if len(battleSkills) > 0 {
		item.SkillIDs = battleSkills
		return
	}
	if len(item.SkillIDs) == 0 {
		item.SkillIDs = []uint32{}
	}
}

// SkillSlotEntry 供协议/UI 使用的单槽描述。
type SkillSlotEntry struct {
	SlotIndex uint32 `json:"slot_index"`
	SkillID   uint32 `json:"skill_id"`
	Enabled   bool   `json:"enabled,omitempty"`
}

// SkillSlotView 结构化技能槽视图。
type SkillSlotView struct {
	Innate         []SkillSlotEntry `json:"innate"`
	ActiveTalisman SkillSlotEntry   `json:"active_talisman"`
	TalismanHero   SkillSlotEntry   `json:"talisman_hero"`
	Talisman1      SkillSlotEntry   `json:"talisman_1"`
	Talisman2      SkillSlotEntry   `json:"talisman_2"`
	Talisman3      SkillSlotEntry   `json:"talisman_3"`
	Normal         []SkillSlotEntry `json:"normal"`
	Artifact       []SkillSlotEntry `json:"artifact"`
}

// BuildSkillSlotView 生成技能面板展示结构；includeArtifact 为 false 时法宝槽 skill_id 一律输出 0。
func BuildSkillSlotView(loadout SkillLoadout, includeArtifact bool) SkillSlotView {
	loadout = NormalizeSkillLoadout(loadout)
	view := SkillSlotView{
		Innate: make([]SkillSlotEntry, 0, MaxInnateSkillSlots),
		Normal: make([]SkillSlotEntry, 0, MaxNormalSkillSlots),
		Artifact: make([]SkillSlotEntry, 0, MaxArtifactSkillSlots),
		ActiveTalisman: SkillSlotEntry{SlotIndex: 0, SkillID: loadout.ActiveTalismanSkillID, Enabled: loadout.ActiveTalismanEnabled},
		TalismanHero:   SkillSlotEntry{SlotIndex: 0, SkillID: loadout.TalismanHeroSkillID, Enabled: loadout.TalismanHeroEnabled},
		Talisman1:      SkillSlotEntry{SlotIndex: 0, SkillID: loadout.TalismanSlot1SkillID, Enabled: loadout.TalismanSlot1Enabled},
		Talisman2:      SkillSlotEntry{SlotIndex: 0, SkillID: loadout.TalismanSlot2SkillID, Enabled: loadout.TalismanSlot2Enabled},
		Talisman3:      SkillSlotEntry{SlotIndex: 0, SkillID: loadout.TalismanSlot3SkillID, Enabled: loadout.TalismanSlot3Enabled},
	}
	for index, skillID := range loadout.InnateSkillIDs {
		view.Innate = append(view.Innate, SkillSlotEntry{
			SlotIndex: uint32(index),
			SkillID:   skillID,
		})
	}
	for index, skillID := range loadout.NormalSkillIDs {
		view.Normal = append(view.Normal, SkillSlotEntry{
			SlotIndex: uint32(index),
			SkillID:   skillID,
			Enabled:   true,
		})
	}
	for index := 0; index < MaxArtifactSkillSlots; index++ {
		skillID := uint32(0)
		if includeArtifact {
			skillID = loadout.ArtifactSkillIDs[index]
		}
		view.Artifact = append(view.Artifact, SkillSlotEntry{
			SlotIndex: uint32(index),
			SkillID:   skillID,
		})
	}
	return view
}

// SkillLoadoutFromDefinition 从模板天生/普通技构建新实例默认分槽（神符默认关闭）。
func SkillLoadoutFromDefinition(innateSkillIDs, normalSkillIDs []uint32) SkillLoadout {
	return NormalizeSkillLoadout(SkillLoadout{
		InnateSkillIDs: append([]uint32{}, innateSkillIDs...),
		NormalSkillIDs: append([]uint32{}, normalSkillIDs...),
	})
}

func normalizeSkillIDSlice(values []uint32, size int) []uint32 {
	result := make([]uint32, size)
	for index := 0; index < size && index < len(values); index++ {
		result[index] = values[index]
	}
	return result
}
