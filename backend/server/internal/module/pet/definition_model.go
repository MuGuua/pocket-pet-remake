package pet

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrPetDefinitionNotFound 表示后台请求的系统宠物模板不存在。
	ErrPetDefinitionNotFound = errors.New("pet definition not found")
	// ErrInvalidAdminPetDefinitionInput 表示后台提交的系统宠物模板字段非法。
	ErrInvalidAdminPetDefinitionInput = errors.New("invalid admin pet definition input")
	// ErrPetDefinitionConflict 表示 pet_id 已存在。
	ErrPetDefinitionConflict = errors.New("pet definition conflict")
	// ErrPetUnusable 表示玩家宠物对应的系统模板不存在或已停用。
	ErrPetUnusable = errors.New("pet unusable")
	// ErrInvalidAptitudeRollRange 表示野外捕捉模板的资质 roll 区间配置非法。
	ErrInvalidAptitudeRollRange = errors.New("invalid aptitude roll range")
	// ErrInvalidWildCapturePetTemplate 表示怪物关联的捕捉宠物模板不是野外捕捉类。
	ErrInvalidWildCapturePetTemplate = errors.New("invalid wild capture pet template")
	// ErrTalismanSlotAlreadyUnlocked 表示目标神符槽已经开启，继续使用道具不会产生效果。
	ErrTalismanSlotAlreadyUnlocked = errors.New("talisman slot already unlocked")
	// ErrInvalidArtifactSlot 表示法宝槽 index 非法。
	ErrInvalidArtifactSlot = errors.New("invalid artifact slot")
	// ErrArtifactSlotOccupied 表示目标法宝槽已有技能。
	ErrArtifactSlotOccupied = errors.New("artifact slot occupied")
	// ErrArtifactSlotEmpty 表示目标法宝槽为空。
	ErrArtifactSlotEmpty = errors.New("artifact slot empty")
	// ErrInvalidArtifactItem 表示背包物品不是可装备的法宝。
	ErrInvalidArtifactItem = errors.New("invalid artifact item")
)

// AdminPetDefinitionListQuery 定义系统宠物模板列表筛选参数。
type AdminPetDefinitionListQuery struct {
	PetID    uint32
	Name     string
	Enabled  *bool
	Page     uint32
	PageSize uint32
}

// Normalize 收口分页与筛选默认值。
func (q AdminPetDefinitionListQuery) Normalize() AdminPetDefinitionListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PageSize == 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	q.Name = strings.TrimSpace(q.Name)
	return q
}

// AdminPetDefinitionSummary 是列表页展示字段，基础数值与技能仅在详情页展示。
type AdminPetDefinitionSummary struct {
	PetID         uint32    `json:"pet_id"`
	PetName       string    `json:"pet_name"`
	Quality       uint32    `json:"quality"`
	Level         uint32    `json:"level"`
	AcquireMethod string    `json:"acquire_method"`
	IsEnabled     bool      `json:"is_enabled"`
	StatusText    string    `json:"status_text"`
	SkinID        string    `json:"skin_id"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// AdminPetDefinitionList 是系统宠物模板分页响应。
type AdminPetDefinitionList struct {
	Items    []AdminPetDefinitionSummary `json:"items"`
	Total    uint64                      `json:"total"`
	Page     uint32                      `json:"page"`
	PageSize uint32                      `json:"page_size"`
}

// AdminPetDefinitionBaseStats 描述模板发放时使用的基础战斗数值。
type AdminPetDefinitionBaseStats struct {
	Level   uint32 `json:"level"`
	Quality uint32 `json:"quality"`
	HP      uint32 `json:"hp"`
	HPMax   uint32 `json:"hp_max"`
	ATK     uint32 `json:"atk"`
	DEF     uint32 `json:"def"`
	SPD     uint32 `json:"spd"`
	MANA    uint32 `json:"mana"`
}

// AdminPetDefinitionGrowthAptitudes 描述宠物成长资质。
type AdminPetDefinitionGrowthAptitudes struct {
	HPApt   uint32 `json:"hp_apt"`
	ATKApt  uint32 `json:"atk_apt"`
	DEFApt  uint32 `json:"def_apt"`
	SPDApt  uint32 `json:"spd_apt"`
	MANAApt uint32 `json:"mana_apt"`
}

// AdminPetDefinitionAptitudeRollRanges 描述野外捕捉模板每项资质的 roll 区间。
type AdminPetDefinitionAptitudeRollRanges struct {
	HPAptRollMin   uint32 `json:"hp_apt_roll_min"`
	HPAptRollMax   uint32 `json:"hp_apt_roll_max"`
	ATKAptRollMin  uint32 `json:"atk_apt_roll_min"`
	ATKAptRollMax  uint32 `json:"atk_apt_roll_max"`
	DEFAptRollMin  uint32 `json:"def_apt_roll_min"`
	DEFAptRollMax  uint32 `json:"def_apt_roll_max"`
	SPDAptRollMin  uint32 `json:"spd_apt_roll_min"`
	SPDAptRollMax  uint32 `json:"spd_apt_roll_max"`
	MANAAptRollMin uint32 `json:"mana_apt_roll_min"`
	MANAAptRollMax uint32 `json:"mana_apt_roll_max"`
}

// AdminPetDefinitionDetail 是详情抽屉所需的完整模板信息。
type AdminPetDefinitionDetail struct {
	PetID           uint32                          `json:"pet_id"`
	PetName         string                          `json:"pet_name"`
	Description     string                          `json:"description"`
	AcquireMethod   string                          `json:"acquire_method"`
	IsEnabled       bool                            `json:"is_enabled"`
	StatusText      string                          `json:"status_text"`
	BaseStats            AdminPetDefinitionBaseStats          `json:"base_stats"`
	GrowthAptitudes      AdminPetDefinitionGrowthAptitudes    `json:"growth_aptitudes"`
	AptitudeRollRanges   AdminPetDefinitionAptitudeRollRanges `json:"aptitude_roll_ranges"`
	SkillIDs             []uint32                             `json:"skill_ids"`
	InnateSkillIDs       []uint32                             `json:"innate_skill_ids"`
	NormalSkillIDs       []uint32                             `json:"normal_skill_ids"`
	SkinID               string                               `json:"skin_id"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

// AdminUpsertPetDefinitionInput 描述后台新增或编辑系统宠物模板时提交的字段。
type AdminUpsertPetDefinitionInput struct {
	PetID         uint32   `json:"pet_id"`
	PetName       string   `json:"pet_name"`
	Description   string   `json:"description"`
	AcquireMethod string   `json:"acquire_method"`
	IsEnabled     bool     `json:"is_enabled"`
	Level         uint32   `json:"level"`
	Quality       uint32   `json:"quality"`
	HP            uint32   `json:"hp"`
	HPMax         uint32   `json:"hp_max"`
	ATK           uint32   `json:"atk"`
	DEF           uint32   `json:"def"`
	SPD           uint32   `json:"spd"`
	MANA          uint32   `json:"mana"`
	HPApt         uint32   `json:"hp_apt"`
	ATKApt        uint32   `json:"atk_apt"`
	DEFApt        uint32   `json:"def_apt"`
	SPDApt        uint32   `json:"spd_apt"`
	MANAApt       uint32   `json:"mana_apt"`
	HPAptRollMin  uint32   `json:"hp_apt_roll_min"`
	HPAptRollMax  uint32   `json:"hp_apt_roll_max"`
	ATKAptRollMin uint32   `json:"atk_apt_roll_min"`
	ATKAptRollMax uint32   `json:"atk_apt_roll_max"`
	DEFAptRollMin uint32   `json:"def_apt_roll_min"`
	DEFAptRollMax uint32   `json:"def_apt_roll_max"`
	SPDAptRollMin uint32   `json:"spd_apt_roll_min"`
	SPDAptRollMax uint32   `json:"spd_apt_roll_max"`
	MANAAptRollMin uint32  `json:"mana_apt_roll_min"`
	MANAAptRollMax uint32  `json:"mana_apt_roll_max"`
	SkillIDs       []uint32 `json:"skill_ids"`
	InnateSkillIDs []uint32 `json:"innate_skill_ids"`
	NormalSkillIDs []uint32 `json:"normal_skill_ids"`
	SkinID         string   `json:"skin_id"`
}

// Normalize 补齐模板字段默认值，避免运营漏填导致发放链路异常。
func (input AdminUpsertPetDefinitionInput) Normalize() AdminUpsertPetDefinitionInput {
	input.PetName = strings.TrimSpace(input.PetName)
	input.Description = strings.TrimSpace(input.Description)
	input.AcquireMethod = strings.TrimSpace(input.AcquireMethod)
	if input.Level == 0 {
		input.Level = 1
	}
	if input.Quality == 0 {
		input.Quality = 1
	}
	if input.HP == 0 {
		input.HP = 1
	}
	if input.HPMax == 0 {
		input.HPMax = input.HP
	}
	if input.ATK == 0 {
		input.ATK = 1
	}
	if input.DEF == 0 {
		input.DEF = 1
	}
	if input.SPD == 0 {
		input.SPD = 1
	}
	if input.HPApt == 0 {
		input.HPApt = 10
	}
	if input.ATKApt == 0 {
		input.ATKApt = 10
	}
	if input.DEFApt == 0 {
		input.DEFApt = 10
	}
	if input.SPDApt == 0 {
		input.SPDApt = 10
	}
	if input.MANAApt == 0 {
		input.MANAApt = 10
	}
	if input.SkillIDs == nil {
		input.SkillIDs = []uint32{}
	}
	if input.InnateSkillIDs == nil {
		input.InnateSkillIDs = []uint32{}
	}
	if input.NormalSkillIDs == nil {
		input.NormalSkillIDs = []uint32{}
	}
	if len(input.InnateSkillIDs) > MaxInnateSkillSlots {
		input.InnateSkillIDs = input.InnateSkillIDs[:MaxInnateSkillSlots]
	}
	if len(input.NormalSkillIDs) > MaxNormalSkillSlots {
		input.NormalSkillIDs = input.NormalSkillIDs[:MaxNormalSkillSlots]
	}
	input.SkinID = strings.TrimSpace(input.SkinID)
	return input
}
