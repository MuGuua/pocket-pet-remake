package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/skill"
)

// SkillRepository 把系统技能模板映射到 skill_definition 主表。
type SkillRepository struct {
	db DBTX
}

// NewSkillRepository 构造技能模板仓储。
func NewSkillRepository(db DBTX) *SkillRepository {
	return &SkillRepository{db: db}
}

const skillDefinitionSelectColumns = `
  skill_id,
  skill_code,
  skill_name,
  skill_category,
  weapon_discipline,
  learn_exp_required,
  learn_exp_per_use,
  skill_type,
  description,
  acquire_method,
  target_type,
  target_count,
  preferred_target_hp,
  animation_key,
  skill_visual_id,
  cast_color,
  impact_color,
  projectile,
  is_skill_attack,
  is_basic_attack,
  energy_cost,
  allow_crit,
  ignore_defense,
  skill_mult,
  skill_crit_add,
  attack_pct,
  mana_pct,
  defense_pct,
  speed_pct,
  target_current_hp_pct,
  fixed_damage,
  heal_pct,
  fixed_heal,
  armor_break_pct,
  vulnerability_pct,
  bleed_chance_pct,
  bleed_rounds,
  bleed_damage,
  seal_chance_pct,
  seal_power,
  seal_rounds,
  vulnerability_chance_pct,
  vulnerability_rounds,
  vulnerability_apply_pct,
  armor_break_chance_pct,
  armor_break_rounds,
  slow_chance_pct,
  slow_rounds,
  slow_multiplier_pct,
  crit_boost_rounds,
  crit_boost_pct,
  curse_chance_pct,
  curse_rounds,
  curse_damage,
  curse_mana_pct,
  control_chance_pct,
  control_power,
  control_rounds,
  control_status_id,
  sort_weight,
  status,
  created_at,
  updated_at
`

const adminSkillListBaseQuery = `
SELECT
  skill_id,
  skill_code,
  skill_name,
  skill_category,
  skill_type,
  target_type,
  energy_cost,
  is_basic_attack,
  status,
  updated_at,
  created_at
FROM skill_definition
`

const skillDefinitionInsertColumns = `
  skill_id, skill_code, skill_name, skill_category, weapon_discipline, learn_exp_required, learn_exp_per_use, skill_type, description, acquire_method,
  target_type, target_count, preferred_target_hp,
  animation_key, skill_visual_id, cast_color, impact_color, projectile,
  is_skill_attack, is_basic_attack, energy_cost, allow_crit, ignore_defense, skill_mult, skill_crit_add,
  attack_pct, mana_pct, defense_pct, speed_pct, target_current_hp_pct, fixed_damage, heal_pct, fixed_heal,
  armor_break_pct, vulnerability_pct,
  bleed_chance_pct, bleed_rounds, bleed_damage,
  seal_chance_pct, seal_power, seal_rounds,
  vulnerability_chance_pct, vulnerability_rounds, vulnerability_apply_pct,
  armor_break_chance_pct, armor_break_rounds,
  slow_chance_pct, slow_rounds, slow_multiplier_pct,
  crit_boost_rounds, crit_boost_pct,
  curse_chance_pct, curse_rounds, curse_damage, curse_mana_pct,
  control_chance_pct, control_power, control_rounds, control_status_id,
  sort_weight, status
`

const skillDefinitionInsertPlaceholders = `
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56,$57,$58,$59,$60,$61
`

const insertSkillDefinitionQuery = `
INSERT INTO skill_definition (` + skillDefinitionInsertColumns + `)
VALUES (` + skillDefinitionInsertPlaceholders + `)
`

const updateSkillDefinitionQuery = `
UPDATE skill_definition
SET skill_code = $2,
    skill_name = $3,
    skill_category = $4,
    weapon_discipline = $5,
    learn_exp_required = $6,
    learn_exp_per_use = $7,
    skill_type = $8,
    description = $9,
    acquire_method = $10,
    target_type = $11,
    target_count = $12,
    preferred_target_hp = $13,
    animation_key = $14,
    skill_visual_id = $15,
    cast_color = $16,
    impact_color = $17,
    projectile = $18,
    is_skill_attack = $19,
    is_basic_attack = $20,
    energy_cost = $21,
    allow_crit = $22,
    ignore_defense = $23,
    skill_mult = $24,
    skill_crit_add = $25,
    attack_pct = $26,
    mana_pct = $27,
    defense_pct = $28,
    speed_pct = $29,
    target_current_hp_pct = $30,
    fixed_damage = $31,
    heal_pct = $32,
    fixed_heal = $33,
    armor_break_pct = $34,
    vulnerability_pct = $35,
    bleed_chance_pct = $36,
    bleed_rounds = $37,
    bleed_damage = $38,
    seal_chance_pct = $39,
    seal_power = $40,
    seal_rounds = $41,
    vulnerability_chance_pct = $42,
    vulnerability_rounds = $43,
    vulnerability_apply_pct = $44,
    armor_break_chance_pct = $45,
    armor_break_rounds = $46,
    slow_chance_pct = $47,
    slow_rounds = $48,
    slow_multiplier_pct = $49,
    crit_boost_rounds = $50,
    crit_boost_pct = $51,
    curse_chance_pct = $52,
    curse_rounds = $53,
    curse_damage = $54,
    curse_mana_pct = $55,
    control_chance_pct = $56,
    control_power = $57,
    control_rounds = $58,
    control_status_id = $59,
    sort_weight = $60,
    status = $61
WHERE skill_id = $1
`

const deleteSkillDefinitionQuery = `DELETE FROM skill_definition WHERE skill_id = $1`

func adminSkillListOrderClause(orderBy string) string {
	if orderBy == "skill_id_desc" {
		return ` ORDER BY skill_id DESC`
	}
	return ` ORDER BY sort_weight ASC, skill_id ASC`
}

func (r *SkillRepository) ListForAdmin(ctx context.Context, query skill.AdminListQuery) (*skill.AdminList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 6)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.SkillID > 0 {
		conditions = append(conditions, "skill_id = "+nextArg(query.SkillID))
	}
	if query.Name != "" {
		conditions = append(conditions, "(skill_name ILIKE "+nextArg("%"+query.Name+"%")+" OR skill_code ILIKE "+nextArg("%"+query.Name+"%")+")")
	}
	if query.Category != "" {
		conditions = append(conditions, "skill_category = "+nextArg(query.Category))
	}
	if query.Type != "" {
		conditions = append(conditions, "skill_type = "+nextArg(query.Type))
	}
	if query.Enabled != nil {
		status := int64(0)
		if *query.Enabled {
			status = 1
		}
		conditions = append(conditions, "status = "+nextArg(status))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM skill_definition `+whereClause, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminSkillListBaseQuery + whereClause + adminSkillListOrderClause(query.OrderBy) + ` LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]skill.AdminSummary, 0, query.PageSize)
	for rows.Next() {
		item, err := scanAdminSkillSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &skill.AdminList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *SkillRepository) FindForAdmin(ctx context.Context, skillID uint32) (*skill.AdminDetail, error) {
	query := `SELECT ` + skillDefinitionSelectColumns + ` FROM skill_definition WHERE skill_id = $1 LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, skillID)
	detail, err := scanAdminSkillDetailRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return detail, nil
}

func (r *SkillRepository) CreateForAdmin(ctx context.Context, input skill.AdminUpsertInput) (*skill.AdminDetail, error) {
	status := skillStatusFromEnabled(input.IsEnabled)
	if _, err := r.db.ExecContext(ctx, insertSkillDefinitionQuery, skillUpsertArgs(input.SkillID, input, status)...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, skill.ErrSkillDefinitionConflict
		}
		return nil, err
	}
	return r.FindForAdmin(ctx, input.SkillID)
}

func (r *SkillRepository) UpdateForAdmin(ctx context.Context, skillID uint32, input skill.AdminUpsertInput) (*skill.AdminDetail, error) {
	status := skillStatusFromEnabled(input.IsEnabled)
	result, err := r.db.ExecContext(ctx, updateSkillDefinitionQuery, skillUpsertArgs(skillID, input, status)...)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	return r.FindForAdmin(ctx, skillID)
}

func (r *SkillRepository) DeleteForAdmin(ctx context.Context, skillID uint32) error {
	result, err := r.db.ExecContext(ctx, deleteSkillDefinitionQuery, skillID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return skill.ErrSkillDefinitionNotFound
	}
	return nil
}

func (r *SkillRepository) ListEnabledRuntimeDefinitions(ctx context.Context) ([]skill.RuntimeDefinition, error) {
	query := `SELECT ` + skillDefinitionSelectColumns + ` FROM skill_definition WHERE status = 1 ORDER BY skill_id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]skill.RuntimeDefinition, 0, 16)
	for rows.Next() {
		item, err := scanRuntimeSkillDefinitionRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SkillRepository) MapUsableSkillIDs(ctx context.Context, skillIDs []uint32) (map[uint32]bool, error) {
	result := make(map[uint32]bool, len(skillIDs))
	if len(skillIDs) == 0 {
		return result, nil
	}
	unique := make([]uint32, 0, len(skillIDs))
	seen := make(map[uint32]struct{}, len(skillIDs))
	for _, skillID := range skillIDs {
		if skillID == 0 {
			continue
		}
		if _, exists := seen[skillID]; exists {
			continue
		}
		seen[skillID] = struct{}{}
		unique = append(unique, skillID)
	}
	if len(unique) == 0 {
		return result, nil
	}
	args := make([]any, len(unique))
	placeholders := make([]string, len(unique))
	for index, skillID := range unique {
		args[index] = skillID
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}
	query := fmt.Sprintf(`SELECT skill_id FROM skill_definition WHERE status = 1 AND skill_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var skillID int64
		if err := rows.Scan(&skillID); err != nil {
			return nil, err
		}
		result[uint32(skillID)] = true
	}
	return result, rows.Err()
}

// MapSkillCategoriesByIDs 批量读取 skill_id 对应的 skill_category，供装备武器技能引用校验。
func (r *SkillRepository) MapSkillCategoriesByIDs(ctx context.Context, skillIDs []uint32) (map[uint32]string, error) {
	result := make(map[uint32]string, len(skillIDs))
	if len(skillIDs) == 0 {
		return result, nil
	}
	unique := make([]uint32, 0, len(skillIDs))
	seen := make(map[uint32]struct{}, len(skillIDs))
	for _, skillID := range skillIDs {
		if skillID == 0 {
			continue
		}
		if _, exists := seen[skillID]; exists {
			continue
		}
		seen[skillID] = struct{}{}
		unique = append(unique, skillID)
	}
	if len(unique) == 0 {
		return result, nil
	}
	args := make([]any, len(unique))
	placeholders := make([]string, len(unique))
	for index, skillID := range unique {
		args[index] = skillID
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}
	query := fmt.Sprintf(`SELECT skill_id, skill_category FROM skill_definition WHERE skill_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var skillID int64
		var category string
		if err := rows.Scan(&skillID, &category); err != nil {
			return nil, err
		}
		result[uint32(skillID)] = category
	}
	return result, rows.Err()
}

func (r *SkillRepository) MapSkillWeaponDisciplinesByIDs(ctx context.Context, skillIDs []uint32) (map[uint32]string, error) {
	result := make(map[uint32]string, len(skillIDs))
	if len(skillIDs) == 0 {
		return result, nil
	}
	unique := make([]uint32, 0, len(skillIDs))
	seen := make(map[uint32]struct{}, len(skillIDs))
	for _, skillID := range skillIDs {
		if skillID == 0 {
			continue
		}
		if _, exists := seen[skillID]; exists {
			continue
		}
		seen[skillID] = struct{}{}
		unique = append(unique, skillID)
	}
	if len(unique) == 0 {
		return result, nil
	}
	args := make([]any, len(unique))
	placeholders := make([]string, len(unique))
	for index, skillID := range unique {
		args[index] = skillID
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}
	query := fmt.Sprintf(`SELECT skill_id, weapon_discipline FROM skill_definition WHERE skill_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var skillID int64
		var discipline string
		if err := rows.Scan(&skillID, &discipline); err != nil {
			return nil, err
		}
		result[uint32(skillID)] = discipline
	}
	return result, rows.Err()
}

func skillStatusFromEnabled(enabled bool) int64 {
	if enabled {
		return 1
	}
	return 0
}

func skillUpsertArgs(skillID uint32, input skill.AdminUpsertInput, status int64) []any {
	return []any{
		skillID, input.SkillCode, input.SkillName, input.SkillCategory, input.WeaponDiscipline, input.LearnExpRequired, input.LearnExpPerUse, input.SkillType, input.Description, input.AcquireMethod,
		input.TargetType, input.TargetCount, input.PreferredTargetHP,
		input.AnimationKey, input.SkillVisualID, input.CastColor, input.ImpactColor, input.Projectile,
		input.IsSkillAttack, input.IsBasicAttack, input.EnergyCost, input.AllowCrit, input.IgnoreDefense,
		input.SkillMult, input.SkillCritAdd,
		input.AttackPct, input.ManaPct, input.DefensePct, input.SpeedPct, input.TargetCurrentHPPct, input.FixedDamage, input.HealPct, input.FixedHeal,
		input.ArmorBreakPct, input.VulnerabilityPct,
		input.BleedChancePct, input.BleedRounds, input.BleedDamage,
		input.SealChancePct, input.SealPower, input.SealRounds,
		input.VulnerabilityChancePct, input.VulnerabilityRounds, input.VulnerabilityApplyPct,
		input.ArmorBreakChancePct, input.ArmorBreakRounds,
		input.SlowChancePct, input.SlowRounds, input.SlowMultiplierPct,
		input.CritBoostRounds, input.CritBoostPct,
		input.CurseChancePct, input.CurseRounds, input.CurseDamage, input.CurseManaPct,
		input.ControlChancePct, input.ControlPower, input.ControlRounds, input.ControlStatusID,
		input.SortWeight, status,
	}
}

func scanAdminSkillSummaryRow(rows *sql.Rows) (skill.AdminSummary, error) {
	var item skill.AdminSummary
	var skillID, energyCost, status int64
	var isBasicAttack bool
	if err := rows.Scan(&skillID, &item.SkillCode, &item.SkillName, &item.SkillCategory, &item.SkillType, &item.TargetType, &energyCost, &isBasicAttack, &status, &item.UpdatedAt, &item.CreatedAt); err != nil {
		return skill.AdminSummary{}, err
	}
	item.SkillID = uint32(skillID)
	item.EnergyCost = uint32(energyCost)
	item.IsBasicAttack = isBasicAttack
	item.IsEnabled = status == 1
	if item.IsEnabled {
		item.StatusText = "启用"
	} else {
		item.StatusText = "停用"
	}
	return item, nil
}

type skillDefinitionRow struct {
	skillID       int64
	skillCode     string
	skillName     string
	skillCategory string
	weaponDiscipline string
	learnExpRequired int64
	learnExpPerUse int64
	skillType     string
	description   string
	acquireMethod string
	targetType    string
	targetCount   int64
	preferredHP   string
	animationKey  string
	skillVisualID string
	castColor     string
	impactColor   string
	projectile    bool
	isSkillAttack bool
	isBasicAttack bool
	energyCost    int64
	allowCrit     bool
	ignoreDefense bool
	skillMult     int64
	skillCritAdd  int64
	attackPct     int64
	manaPct       int64
	defensePct    int64
	speedPct      int64
	targetHPPct   int64
	fixedDamage   int64
	healPct       int64
	fixedHeal     int64
	armorBreakPct int64
	vulnPct       int64
	bleedChance   int64
	bleedRounds   int64
	bleedDamage   int64
	sealChance    int64
	sealPower     int64
	sealRounds    int64
	vulnChance    int64
	vulnRounds    int64
	vulnApply     int64
	abChance      int64
	abRounds      int64
	slowChance    int64
	slowRounds    int64
	slowMult      int64
	critRounds    int64
	critPct       int64
	curseChance   int64
	curseRounds   int64
	curseDamage   int64
	curseManaPct  int64
	controlChance int64
	controlPower  int64
	controlRounds int64
	controlStatus int64
	sortWeight    int64
	status        int64
	createdAt     time.Time
	updatedAt     time.Time
}

func scanSkillDefinitionRow(scanner interface {
	Scan(dest ...any) error
}) (skillDefinitionRow, error) {
	var row skillDefinitionRow
	err := scanner.Scan(
		&row.skillID, &row.skillCode, &row.skillName, &row.skillCategory, &row.weaponDiscipline, &row.learnExpRequired, &row.learnExpPerUse, &row.skillType, &row.description, &row.acquireMethod,
		&row.targetType, &row.targetCount, &row.preferredHP,
		&row.animationKey, &row.skillVisualID, &row.castColor, &row.impactColor, &row.projectile,
		&row.isSkillAttack, &row.isBasicAttack, &row.energyCost, &row.allowCrit, &row.ignoreDefense,
		&row.skillMult, &row.skillCritAdd,
		&row.attackPct, &row.manaPct, &row.defensePct, &row.speedPct, &row.targetHPPct, &row.fixedDamage, &row.healPct, &row.fixedHeal,
		&row.armorBreakPct, &row.vulnPct,
		&row.bleedChance, &row.bleedRounds, &row.bleedDamage,
		&row.sealChance, &row.sealPower, &row.sealRounds,
		&row.vulnChance, &row.vulnRounds, &row.vulnApply,
		&row.abChance, &row.abRounds,
		&row.slowChance, &row.slowRounds, &row.slowMult,
		&row.critRounds, &row.critPct,
		&row.curseChance, &row.curseRounds, &row.curseDamage, &row.curseManaPct,
		&row.controlChance, &row.controlPower, &row.controlRounds, &row.controlStatus,
		&row.sortWeight, &row.status, &row.createdAt, &row.updatedAt,
	)
	return row, err
}

func scanAdminSkillDetailRow(row *sql.Row) (*skill.AdminDetail, error) {
	raw, err := scanSkillDefinitionRow(row)
	if err != nil {
		return nil, err
	}
	return adminDetailFromRow(raw), nil
}

func scanRuntimeSkillDefinitionRow(rows *sql.Rows) (skill.RuntimeDefinition, error) {
	raw, err := scanSkillDefinitionRow(rows)
	if err != nil {
		return skill.RuntimeDefinition{}, err
	}
	return runtimeFromRow(raw), nil
}

func adminDetailFromRow(raw skillDefinitionRow) *skill.AdminDetail {
	detail := &skill.AdminDetail{
		SkillID:       uint32(raw.skillID),
		SkillCode:     raw.skillCode,
		SkillName:     raw.skillName,
		SkillCategory:    raw.skillCategory,
		WeaponDiscipline: raw.weaponDiscipline,
		LearnExpRequired: uint32(raw.learnExpRequired),
		LearnExpPerUse:   uint32(raw.learnExpPerUse),
		SkillType:        raw.skillType,
		Description:   raw.description,
		AcquireMethod: raw.acquireMethod,
		IsBasicAttack: raw.isBasicAttack,
		IsEnabled:     raw.status == 1,
		SortWeight:    uint32(raw.sortWeight),
		TargetRule: skill.AdminTargetRule{
			TargetType:        raw.targetType,
			TargetCount:       uint32(raw.targetCount),
			PreferredTargetHP: raw.preferredHP,
		},
		Formula: skill.AdminFormula{
			AttackPct:          int32(raw.attackPct),
			ManaPct:            int32(raw.manaPct),
			DefensePct:         int32(raw.defensePct),
			SpeedPct:           int32(raw.speedPct),
			TargetCurrentHPPct: int32(raw.targetHPPct),
			FixedDamage:        int32(raw.fixedDamage),
			HealPct:            int32(raw.healPct),
			FixedHeal:          int32(raw.fixedHeal),
			EnergyCost:         uint32(raw.energyCost),
			IsSkillAttack:      raw.isSkillAttack,
			AllowCrit:          raw.allowCrit,
			IgnoreDefense:      raw.ignoreDefense,
			SkillMult:          uint32(raw.skillMult),
			SkillCritAdd:       uint32(raw.skillCritAdd),
		},
		StatusEffects: skill.AdminStatusEffects{
			ArmorBreakPct:          uint32(raw.armorBreakPct),
			VulnerabilityPct:       uint32(raw.vulnPct),
			BleedChancePct:         uint32(raw.bleedChance),
			BleedRounds:            uint32(raw.bleedRounds),
			BleedDamage:            int32(raw.bleedDamage),
			SealChancePct:          uint32(raw.sealChance),
			SealPower:              uint32(raw.sealPower),
			SealRounds:             uint32(raw.sealRounds),
			VulnerabilityChancePct: uint32(raw.vulnChance),
			VulnerabilityRounds:    uint32(raw.vulnRounds),
			VulnerabilityApplyPct:  uint32(raw.vulnApply),
			ArmorBreakChancePct:    uint32(raw.abChance),
			ArmorBreakRounds:       uint32(raw.abRounds),
			SlowChancePct:          uint32(raw.slowChance),
			SlowRounds:             uint32(raw.slowRounds),
			SlowMultiplierPct:      uint32(raw.slowMult),
			CritBoostRounds:        uint32(raw.critRounds),
			CritBoostPct:           uint32(raw.critPct),
			CurseChancePct:         uint32(raw.curseChance),
			CurseRounds:            uint32(raw.curseRounds),
			CurseDamage:            int32(raw.curseDamage),
			CurseManaPct:           int32(raw.curseManaPct),
			ControlChancePct:       uint32(raw.controlChance),
			ControlPower:           uint32(raw.controlPower),
			ControlRounds:          uint32(raw.controlRounds),
			ControlStatusID:        uint32(raw.controlStatus),
		},
		Presentation: skill.AdminPresentation{
			AnimationKey:  raw.animationKey,
			SkillVisualID: raw.skillVisualID,
			CastColor:     raw.castColor,
			ImpactColor:  raw.impactColor,
			Projectile:   raw.projectile,
		},
	}
	if detail.IsEnabled {
		detail.StatusText = "启用"
	} else {
		detail.StatusText = "停用"
	}
	detail.CreatedAt = raw.createdAt
	detail.UpdatedAt = raw.updatedAt
	return detail
}

func runtimeFromRow(raw skillDefinitionRow) skill.RuntimeDefinition {
	return skill.RuntimeDefinition{
		SkillID:                uint32(raw.skillID),
		SkillType:              raw.skillType,
		SkillCategory:          raw.skillCategory,
		WeaponDiscipline:       raw.weaponDiscipline,
		LearnExpRequired:       uint32(raw.learnExpRequired),
		LearnExpPerUse:         uint32(raw.learnExpPerUse),
		SkillName:              raw.skillName,
		TargetType:             raw.targetType,
		TargetCount:            uint32(raw.targetCount),
		PreferredTargetHP:      raw.preferredHP,
		AnimationKey:           raw.animationKey,
		SkillVisualID:          raw.skillVisualID,
		CastColor:              raw.castColor,
		ImpactColor:            raw.impactColor,
		Projectile:             raw.projectile,
		IsSkillAttack:          raw.isSkillAttack,
		EnergyCost:             uint32(raw.energyCost),
		SkillMult:              uint32(raw.skillMult),
		SkillCritAdd:           uint32(raw.skillCritAdd),
		AttackPct:              int32(raw.attackPct),
		ManaPct:                int32(raw.manaPct),
		DefensePct:             int32(raw.defensePct),
		SpeedPct:               int32(raw.speedPct),
		TargetCurrentHPPct:     int32(raw.targetHPPct),
		FixedDamage:            int32(raw.fixedDamage),
		HealPct:                int32(raw.healPct),
		FixedHeal:              int32(raw.fixedHeal),
		AllowCrit:              raw.allowCrit,
		IgnoreDefense:          raw.ignoreDefense,
		ArmorBreakPct:          uint32(raw.armorBreakPct),
		VulnerabilityPct:       uint32(raw.vulnPct),
		BleedChancePct:         uint32(raw.bleedChance),
		BleedRounds:            uint32(raw.bleedRounds),
		BleedDamage:            int32(raw.bleedDamage),
		SealChancePct:          uint32(raw.sealChance),
		SealPower:              uint32(raw.sealPower),
		SealRounds:             uint32(raw.sealRounds),
		VulnerabilityChancePct: uint32(raw.vulnChance),
		VulnerabilityRounds:    uint32(raw.vulnRounds),
		VulnerabilityApplyPct:  uint32(raw.vulnApply),
		ArmorBreakChancePct:    uint32(raw.abChance),
		ArmorBreakRounds:       uint32(raw.abRounds),
		SlowChancePct:          uint32(raw.slowChance),
		SlowRounds:             uint32(raw.slowRounds),
		SlowMultiplierPct:      uint32(raw.slowMult),
		CritBoostRounds:        uint32(raw.critRounds),
		CritBoostPct:           uint32(raw.critPct),
		CurseChancePct:         uint32(raw.curseChance),
		CurseRounds:            uint32(raw.curseRounds),
		CurseDamage:            int32(raw.curseDamage),
		CurseManaPct:           int32(raw.curseManaPct),
		ControlChancePct:       uint32(raw.controlChance),
		ControlPower:           uint32(raw.controlPower),
		ControlRounds:          uint32(raw.controlRounds),
		ControlStatusID:        uint32(raw.controlStatus),
		IsBasicAttack:          raw.isBasicAttack,
	}
}
