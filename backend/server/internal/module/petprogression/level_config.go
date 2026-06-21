package petprogression

// LevelConfig 描述宠物某一等级升到下一级所需经验与升级奖励。
type LevelConfig struct {
	Level       uint32 `json:"level"`
	ExpRequired uint64 `json:"exp_required"`
	AttrPoints  uint32 `json:"attr_points"`
	Status      uint32 `json:"status"`
}

// ConvertConfig 描述单项战斗属性的资质转化率常数。
type ConvertConfig struct {
	AttrType    string  `json:"attr_type"`
	ConvertRate float64 `json:"convert_rate"`
	Status      uint32  `json:"status"`
}

// ProgressionState 是单只宠物成长计算使用的权威快照。
type ProgressionState struct {
	PlayerID         uint64
	PetUID           uint64
	PetID            uint32
	Level            uint32
	Exp              uint64
	FreeAttrPoints   uint32
	ManualPoints     ManualAllocatedPoints
	EvolutionLevel   uint32
	RebirthLevel     uint32
	AptitudeProfile  string
	Aptitudes        GrowthAptitudes
	HP               uint32
	Combat           CombatStats
}

// ExpApplyResult 描述一次宠物经验结算的结果。
type ExpApplyResult struct {
	Level            uint32
	Exp              uint64
	LevelUpCount     uint32
	AttrPointsGained uint32
	FreeAttrPoints   uint32
	ExpToNext        uint64
	Combat           CombatStats
	HP               uint32
}

// AdminUpsertLevelConfigInput 是后台编辑宠物等级经验配置的入参。
type AdminUpsertLevelConfigInput struct {
	ExpRequired uint64 `json:"exp_required"`
	AttrPoints  uint32 `json:"attr_points"`
	Status      uint32 `json:"status"`
}

// AdminUpsertConvertConfigInput 是后台编辑单项资质转化率的入参。
type AdminUpsertConvertConfigInput struct {
	ConvertRate float64 `json:"convert_rate"`
	Status      uint32  `json:"status"`
}

// PetProgressionTarget 定位单只待重算战斗属性的宠物实例。
type PetProgressionTarget struct {
	PlayerID uint64 `json:"player_id"`
	PetUID   uint64 `json:"pet_uid"`
}

// AdminRecalculateCombatStatsInput 是后台批量重算宠物战斗属性的筛选条件；0 表示不限制。
type AdminRecalculateCombatStatsInput struct {
	PlayerID uint64 `json:"player_id"`
	PetUID   uint64 `json:"pet_uid"`
}

// AdminRecalculateCombatStatsResult 描述批量重算影响范围。
type AdminRecalculateCombatStatsResult struct {
	UpdatedCount uint32 `json:"updated_count"`
}
