package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/pet"
)

const adminPetDefinitionListBaseQuery = `
SELECT
  pet_id,
  pet_name,
  quality,
  level,
  acquire_method,
  status,
  skin_id,
  updated_at,
  created_at
FROM pet_definition
`

const adminPetDefinitionDetailQuery = `
SELECT
  pet_id,
  pet_name,
  description,
  acquire_method,
  status,
  level,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  hp_apt,
  atk_apt,
  def_apt,
  spd_apt,
  mana_apt,
  hp_apt_roll_min,
  hp_apt_roll_max,
  atk_apt_roll_min,
  atk_apt_roll_max,
  def_apt_roll_min,
  def_apt_roll_max,
  spd_apt_roll_min,
  spd_apt_roll_max,
  mana_apt_roll_min,
  mana_apt_roll_max,
  skill_ids,
  skin_id,
  created_at,
  updated_at
FROM pet_definition
WHERE pet_id = $1
LIMIT 1
`

const insertAdminPetDefinitionQuery = `
INSERT INTO pet_definition (
  pet_id,
  pet_name,
  description,
  acquire_method,
  status,
  level,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  hp_apt,
  atk_apt,
  def_apt,
  spd_apt,
  mana_apt,
  hp_apt_roll_min,
  hp_apt_roll_max,
  atk_apt_roll_min,
  atk_apt_roll_max,
  def_apt_roll_min,
  def_apt_roll_max,
  spd_apt_roll_min,
  spd_apt_roll_max,
  mana_apt_roll_min,
  mana_apt_roll_max,
  skill_ids,
  skin_id
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29::jsonb,$30
)
`

const updateAdminPetDefinitionQuery = `
UPDATE pet_definition
SET pet_name = $2,
    description = $3,
    acquire_method = $4,
    status = $5,
    level = $6,
    quality = $7,
    hp = $8,
    hp_max = $9,
    atk = $10,
    def = $11,
    spd = $12,
    mana = $13,
    hp_apt = $14,
    atk_apt = $15,
    def_apt = $16,
    spd_apt = $17,
    mana_apt = $18,
    hp_apt_roll_min = $19,
    hp_apt_roll_max = $20,
    atk_apt_roll_min = $21,
    atk_apt_roll_max = $22,
    def_apt_roll_min = $23,
    def_apt_roll_max = $24,
    spd_apt_roll_min = $25,
    spd_apt_roll_max = $26,
    mana_apt_roll_min = $27,
    mana_apt_roll_max = $28,
    skill_ids = $29::jsonb,
    skin_id = $30
WHERE pet_id = $1
`

const deleteAdminPetDefinitionQuery = `
DELETE FROM pet_definition
WHERE pet_id = $1
`

const findPetSkinIDQuery = `
SELECT skin_id
FROM pet_definition
WHERE pet_id = $1 AND status = 1
LIMIT 1
`

// ListPetDefinitionsForAdmin 返回系统宠物模板分页列表。
func (r *PetRepository) ListPetDefinitionsForAdmin(ctx context.Context, query pet.AdminPetDefinitionListQuery) (*pet.AdminPetDefinitionList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.PetID > 0 {
		conditions = append(conditions, "pet_id = "+nextArg(query.PetID))
	}
	if query.Name != "" {
		conditions = append(conditions, "pet_name ILIKE "+nextArg("%"+query.Name+"%"))
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
	countQuery := `SELECT COUNT(1) FROM pet_definition ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminPetDefinitionListBaseQuery + whereClause + ` ORDER BY pet_id ASC LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]pet.AdminPetDefinitionSummary, 0, query.PageSize)
	for rows.Next() {
		item, err := scanAdminPetDefinitionSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &pet.AdminPetDefinitionList{
		Items:    items,
		Total:    uint64(total),
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// FindPetDefinitionForAdmin 返回单个系统宠物模板详情。
func (r *PetRepository) FindPetDefinitionForAdmin(ctx context.Context, petID uint32) (*pet.AdminPetDefinitionDetail, error) {
	row := r.db.QueryRowContext(ctx, adminPetDefinitionDetailQuery, petID)
	detail, err := scanAdminPetDefinitionDetailRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return detail, nil
}

// CreatePetDefinitionForAdmin 新增系统宠物模板。
func (r *PetRepository) CreatePetDefinitionForAdmin(ctx context.Context, input pet.AdminUpsertPetDefinitionInput) (*pet.AdminPetDefinitionDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal pet definition skill ids: %w", err)
	}
	status := int64(0)
	if input.IsEnabled {
		status = 1
	}
	if _, err := r.db.ExecContext(
		ctx,
		insertAdminPetDefinitionQuery,
		input.PetID,
		input.PetName,
		input.Description,
		input.AcquireMethod,
		status,
		input.Level,
		input.Quality,
		input.HP,
		input.HPMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		input.HPApt,
		input.ATKApt,
		input.DEFApt,
		input.SPDApt,
		input.MANAApt,
		input.HPAptRollMin,
		input.HPAptRollMax,
		input.ATKAptRollMin,
		input.ATKAptRollMax,
		input.DEFAptRollMin,
		input.DEFAptRollMax,
		input.SPDAptRollMin,
		input.SPDAptRollMax,
		input.MANAAptRollMin,
		input.MANAAptRollMax,
		skillIDsJSON,
		input.SkinID,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, pet.ErrPetDefinitionConflict
		}
		return nil, err
	}
	return r.FindPetDefinitionForAdmin(ctx, input.PetID)
}

// UpdatePetDefinitionForAdmin 更新系统宠物模板。
func (r *PetRepository) UpdatePetDefinitionForAdmin(ctx context.Context, petID uint32, input pet.AdminUpsertPetDefinitionInput) (*pet.AdminPetDefinitionDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal pet definition skill ids: %w", err)
	}
	status := int64(0)
	if input.IsEnabled {
		status = 1
	}
	result, err := r.db.ExecContext(
		ctx,
		updateAdminPetDefinitionQuery,
		petID,
		input.PetName,
		input.Description,
		input.AcquireMethod,
		status,
		input.Level,
		input.Quality,
		input.HP,
		input.HPMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		input.HPApt,
		input.ATKApt,
		input.DEFApt,
		input.SPDApt,
		input.MANAApt,
		input.HPAptRollMin,
		input.HPAptRollMax,
		input.ATKAptRollMin,
		input.ATKAptRollMax,
		input.DEFAptRollMin,
		input.DEFAptRollMax,
		input.SPDAptRollMin,
		input.SPDAptRollMax,
		input.MANAAptRollMin,
		input.MANAAptRollMax,
		skillIDsJSON,
		input.SkinID,
	)
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
	return r.FindPetDefinitionForAdmin(ctx, petID)
}

// DeletePetDefinitionForAdmin 删除系统宠物模板。
func (r *PetRepository) DeletePetDefinitionForAdmin(ctx context.Context, petID uint32) error {
	result, err := r.db.ExecContext(ctx, deleteAdminPetDefinitionQuery, petID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return pet.ErrPetDefinitionNotFound
	}
	return nil
}

// MapUsablePetDefinitionIDs 批量判断 pet_id 是否存在于启用中的系统宠物模板。
func (r *PetRepository) MapUsablePetDefinitionIDs(ctx context.Context, petIDs []uint32) (map[uint32]bool, error) {
	result := make(map[uint32]bool, len(petIDs))
	if len(petIDs) == 0 {
		return result, nil
	}
	unique := make([]uint32, 0, len(petIDs))
	seen := make(map[uint32]struct{}, len(petIDs))
	for _, petID := range petIDs {
		if petID == 0 {
			continue
		}
		if _, exists := seen[petID]; exists {
			continue
		}
		seen[petID] = struct{}{}
		unique = append(unique, petID)
	}
	if len(unique) == 0 {
		return result, nil
	}
	args := make([]any, len(unique))
	placeholders := make([]string, len(unique))
	for index, petID := range unique {
		args[index] = petID
		placeholders[index] = fmt.Sprintf("$%d", index+1)
	}
	query := fmt.Sprintf(`SELECT pet_id FROM pet_definition WHERE status = 1 AND pet_id IN (%s)`, strings.Join(placeholders, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var petID int64
		if err := rows.Scan(&petID); err != nil {
			return nil, err
		}
		result[uint32(petID)] = true
	}
	return result, rows.Err()
}

func scanAdminPetDefinitionSummaryRow(rows *sql.Rows) (pet.AdminPetDefinitionSummary, error) {
	var item pet.AdminPetDefinitionSummary
	var petID, quality, level, status int64
	if err := rows.Scan(&petID, &item.PetName, &quality, &level, &item.AcquireMethod, &status, &item.SkinID, &item.UpdatedAt, &item.CreatedAt); err != nil {
		return pet.AdminPetDefinitionSummary{}, err
	}
	item.PetID = uint32(petID)
	item.Quality = uint32(quality)
	item.Level = uint32(level)
	item.IsEnabled = status == 1
	if item.IsEnabled {
		item.StatusText = "启用"
	} else {
		item.StatusText = "停用"
	}
	return item, nil
}

// FindPetSkinID 读取启用中的宠物模板战斗外观 ID。
func (r *PetRepository) FindPetSkinID(ctx context.Context, petID uint32) (string, error) {
	var skinID string
	err := r.db.QueryRowContext(ctx, findPetSkinIDQuery, petID).Scan(&skinID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(skinID), nil
}

func scanAdminPetDefinitionDetailRow(row *sql.Row) (*pet.AdminPetDefinitionDetail, error) {
	var (
		detail       pet.AdminPetDefinitionDetail
		petID        int64
		status       int64
		level        int64
		quality      int64
		hp           int64
		hpMax        int64
		atk          int64
		def          int64
		spd          int64
		mana         int64
		hpApt        int64
		atkApt       int64
		defApt       int64
		spdApt       int64
		manaApt      int64
		hpAptRollMin int64
		hpAptRollMax int64
		atkAptRollMin int64
		atkAptRollMax int64
		defAptRollMin int64
		defAptRollMax int64
		spdAptRollMin int64
		spdAptRollMax int64
		manaAptRollMin int64
		manaAptRollMax int64
		skillIDsJSON []byte
	)
	if err := row.Scan(
		&petID,
		&detail.PetName,
		&detail.Description,
		&detail.AcquireMethod,
		&status,
		&level,
		&quality,
		&hp,
		&hpMax,
		&atk,
		&def,
		&spd,
		&mana,
		&hpApt,
		&atkApt,
		&defApt,
		&spdApt,
		&manaApt,
		&hpAptRollMin,
		&hpAptRollMax,
		&atkAptRollMin,
		&atkAptRollMax,
		&defAptRollMin,
		&defAptRollMax,
		&spdAptRollMin,
		&spdAptRollMax,
		&manaAptRollMin,
		&manaAptRollMax,
		&skillIDsJSON,
		&detail.SkinID,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	); err != nil {
		return nil, err
	}
	detail.PetID = uint32(petID)
	detail.IsEnabled = status == 1
	if detail.IsEnabled {
		detail.StatusText = "启用"
	} else {
		detail.StatusText = "停用"
	}
	detail.BaseStats = pet.AdminPetDefinitionBaseStats{
		Level:   uint32(level),
		Quality: uint32(quality),
		HP:      uint32(hp),
		HPMax:   uint32(hpMax),
		ATK:     uint32(atk),
		DEF:     uint32(def),
		SPD:     uint32(spd),
		MANA:    uint32(mana),
	}
	detail.GrowthAptitudes = pet.AdminPetDefinitionGrowthAptitudes{
		HPApt:   uint32(hpApt),
		ATKApt:  uint32(atkApt),
		DEFApt:  uint32(defApt),
		SPDApt:  uint32(spdApt),
		MANAApt: uint32(manaApt),
	}
	detail.AptitudeRollRanges = pet.AdminPetDefinitionAptitudeRollRanges{
		HPAptRollMin:   uint32(hpAptRollMin),
		HPAptRollMax:   uint32(hpAptRollMax),
		ATKAptRollMin:  uint32(atkAptRollMin),
		ATKAptRollMax:  uint32(atkAptRollMax),
		DEFAptRollMin:  uint32(defAptRollMin),
		DEFAptRollMax:  uint32(defAptRollMax),
		SPDAptRollMin:  uint32(spdAptRollMin),
		SPDAptRollMax:  uint32(spdAptRollMax),
		MANAAptRollMin: uint32(manaAptRollMin),
		MANAAptRollMax: uint32(manaAptRollMax),
	}
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &detail.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal pet definition skill ids: %w", err)
		}
	}
	if detail.SkillIDs == nil {
		detail.SkillIDs = []uint32{}
	}
	return &detail, nil
}
