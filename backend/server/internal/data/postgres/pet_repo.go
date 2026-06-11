package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/pet"
)

type PetRepository struct {
	db DBTX
}

func NewPetRepository(db DBTX) *PetRepository {
	return &PetRepository{db: db}
}

const listLineupByPlayerIDQuery = `
SELECT
  pp.id,
  pp.pet_id,
  pp.level,
  pp.hp,
  pp.hp_max,
  pp.atk,
  pp.def,
  pp.spd,
  pp.mana,
  pp.skill_ids
FROM player_lineup pl
JOIN player_pet pp ON pp.id = pl.pet_uid
WHERE pl.player_id = $1
ORDER BY pl.slot_index ASC
`

const listPetsByPlayerIDQuery = `
SELECT
  id,
  pet_id,
  level,
  exp,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  skill_ids
FROM player_pet
WHERE player_id = $1
ORDER BY id ASC
`

const deleteLineupByPlayerIDQuery = `
DELETE FROM player_lineup
WHERE player_id = $1
`

const insertLineupItemQuery = `
INSERT INTO player_lineup (player_id, slot_index, pet_uid)
VALUES ($1, $2, $3)
`

const updatePetHPByUIDQuery = `
UPDATE player_pet
SET hp = LEAST($3, hp_max)
WHERE player_id = $1 AND id = $2
`

const updatePetHPAndExpByUIDQuery = `
UPDATE player_pet
SET hp = LEAST($3, hp_max),
    exp = exp + $4
WHERE player_id = $1 AND id = $2
`

const adminPetListBaseQuery = `
SELECT
  pp.id,
  pp.player_id,
  p.name,
  pp.pet_id,
  pp.level,
  pp.quality,
  pp.hp,
  pp.hp_max,
  pp.atk,
  pp.def,
  pp.spd,
  pp.mana,
  EXISTS(SELECT 1 FROM player_lineup pl WHERE pl.player_id = pp.player_id AND pl.pet_uid = pp.id) AS in_lineup,
  pp.updated_at,
  pp.created_at
FROM player_pet pp
JOIN player p ON p.id = pp.player_id
`

const adminPetDetailQuery = `
SELECT
  pp.id,
  pp.player_id,
  p.name,
  pp.pet_id,
  pp.level,
  pp.exp,
  pp.quality,
  pp.hp,
  pp.hp_max,
  pp.atk,
  pp.def,
  pp.spd,
  pp.mana,
  pp.skill_ids,
  EXISTS(SELECT 1 FROM player_lineup pl WHERE pl.player_id = pp.player_id AND pl.pet_uid = pp.id) AS in_lineup,
  pp.created_at,
  pp.updated_at
FROM player_pet pp
JOIN player p ON p.id = pp.player_id
WHERE pp.id = $1
LIMIT 1
`

const insertAdminPetQuery = `
INSERT INTO player_pet (
  player_id,
  pet_id,
  level,
  exp,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  skill_ids
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING id
`

const adminPetPlayerExistsQuery = `
SELECT COUNT(1)
FROM player
WHERE id = $1 AND status = 1
`

const updateAdminPetQuery = `
UPDATE player_pet
SET pet_id = $2,
    level = $3,
    exp = $4,
    quality = $5,
    hp = $6,
    hp_max = $7,
    atk = $8,
    def = $9,
    spd = $10,
    mana = $11,
    skill_ids = $12
WHERE id = $1
`

const deleteAdminPetLineupQuery = `
DELETE FROM player_lineup
WHERE pet_uid = $1
`

const deleteAdminPetQuery = `
DELETE FROM player_pet
WHERE id = $1
`

func (r *PetRepository) ListPetsByPlayerID(ctx context.Context, playerID uint64) ([]pet.Pet, error) {
	rows, err := r.db.QueryContext(ctx, listPetsByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pets := make([]pet.Pet, 0)
	for rows.Next() {
		item, err := scanPetRow(rows)
		if err != nil {
			return nil, err
		}
		pets = append(pets, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pets, nil
}

func (r *PetRepository) ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]pet.LineupPet, error) {
	rows, err := r.db.QueryContext(ctx, listLineupByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lineup := make([]pet.LineupPet, 0)
	for rows.Next() {
		var (
			item         pet.LineupPet
			petUID       int64
			petID        int64
			level        int64
			hp           int64
			hpMax        int64
			atk          int64
			def          int64
			spd          int64
			mana         int64
			skillIDsJSON []byte
		)
		if err := rows.Scan(&petUID, &petID, &level, &hp, &hpMax, &atk, &def, &spd, &mana, &skillIDsJSON); err != nil {
			return nil, err
		}
		item.PetUID = uint64(petUID)
		item.PetID = uint32(petID)
		item.Level = uint32(level)
		item.HP = uint32(hp)
		item.HPMax = uint32(hpMax)
		item.ATK = uint32(atk)
		item.DEF = uint32(def)
		item.SPD = uint32(spd)
		item.MANA = uint32(mana)
		if len(skillIDsJSON) > 0 {
			if err := json.Unmarshal(skillIDsJSON, &item.SkillIDs); err != nil {
				return nil, fmt.Errorf("unmarshal lineup pet skill ids: %w", err)
			}
		}
		lineup = append(lineup, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lineup, nil
}

func (r *PetRepository) SetLineupByPlayerID(ctx context.Context, playerID uint64, petUIDs []uint64) error {
	if _, err := r.db.ExecContext(ctx, deleteLineupByPlayerIDQuery, playerID); err != nil {
		return err
	}
	for slotIndex, petUID := range petUIDs {
		if _, err := r.db.ExecContext(ctx, insertLineupItemQuery, playerID, slotIndex, petUID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PetRepository) UpdatePetHPByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32) (pet.Pet, error) {
	result, err := r.db.ExecContext(ctx, updatePetHPByUIDQuery, playerID, petUID, hp)
	if err != nil {
		return pet.Pet{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pet.Pet{}, err
	}
	if rowsAffected == 0 {
		return pet.Pet{}, pet.ErrPetNotFound
	}

	pets, err := r.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return pet.Pet{}, err
	}
	for _, item := range pets {
		if item.PetUID == petUID {
			return item, nil
		}
	}
	return pet.Pet{}, pet.ErrPetNotFound
}

func (r *PetRepository) UpdatePetHPAndExpByUID(ctx context.Context, playerID uint64, petUID uint64, hp uint32, expGain uint64) (pet.Pet, error) {
	result, err := r.db.ExecContext(ctx, updatePetHPAndExpByUIDQuery, playerID, petUID, hp, expGain)
	if err != nil {
		return pet.Pet{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return pet.Pet{}, err
	}
	if rowsAffected == 0 {
		return pet.Pet{}, pet.ErrPetNotFound
	}

	pets, err := r.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return pet.Pet{}, err
	}
	for _, item := range pets {
		if item.PetUID == petUID {
			return item, nil
		}
	}
	return pet.Pet{}, pet.ErrPetNotFound
}

func (r *PetRepository) ListForAdmin(ctx context.Context, query pet.AdminListQuery) (*pet.AdminPetList, error) {
	query = query.Normalize()
	conditions := []string{}
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.PetUID > 0 {
		conditions = append(conditions, "pp.id = "+nextArg(query.PetUID))
	}
	if query.PlayerID > 0 {
		conditions = append(conditions, "pp.player_id = "+nextArg(query.PlayerID))
	}
	if query.PetID > 0 {
		conditions = append(conditions, "pp.pet_id = "+nextArg(query.PetID))
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions)
	}
	countQuery := `SELECT COUNT(1) FROM player_pet pp JOIN player p ON p.id = pp.player_id ` + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}
	listQuery := adminPetListBaseQuery + whereClause + ` ORDER BY pp.id ASC LIMIT ` + nextArg(query.PageSize) + ` OFFSET ` + nextArg((query.Page-1)*query.PageSize)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]pet.AdminPetSummary, 0, query.PageSize)
	for rows.Next() {
		var item pet.AdminPetSummary
		var petUID, playerID, petID, level, quality, hp, hpMax, atk, def, spd, mana int64
		if err := rows.Scan(&petUID, &playerID, &item.PlayerName, &petID, &level, &quality, &hp, &hpMax, &atk, &def, &spd, &mana, &item.InLineup, &item.UpdatedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.PetUID = uint64(petUID)
		item.PlayerID = uint64(playerID)
		item.PetID = uint32(petID)
		item.Level = uint32(level)
		item.Quality = uint32(quality)
		item.HP = uint32(hp)
		item.HPMax = uint32(hpMax)
		item.ATK = uint32(atk)
		item.DEF = uint32(def)
		item.SPD = uint32(spd)
		item.MANA = uint32(mana)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &pet.AdminPetList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *PetRepository) FindAdminDetailByPetUID(ctx context.Context, petUID uint64) (*pet.AdminPetDetail, error) {
	var detail pet.AdminPetDetail
	var uid, playerID, petID, level, exp, quality, hp, hpMax, atk, def, spd, mana int64
	var skillIDsJSON []byte
	err := r.db.QueryRowContext(ctx, adminPetDetailQuery, petUID).Scan(&uid, &playerID, &detail.PlayerName, &petID, &level, &exp, &quality, &hp, &hpMax, &atk, &def, &spd, &mana, &skillIDsJSON, &detail.InLineup, &detail.CreatedAt, &detail.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	detail.PetUID = uint64(uid)
	detail.PlayerID = uint64(playerID)
	detail.PetID = uint32(petID)
	detail.Level = uint32(level)
	detail.Exp = uint64(exp)
	detail.Quality = uint32(quality)
	detail.HP = uint32(hp)
	detail.HPMax = uint32(hpMax)
	detail.ATK = uint32(atk)
	detail.DEF = uint32(def)
	detail.SPD = uint32(spd)
	detail.MANA = uint32(mana)
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &detail.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal admin pet skill ids: %w", err)
		}
	}
	return &detail, nil
}

func (r *PetRepository) CreateForAdmin(ctx context.Context, input pet.AdminCreatePetInput) (*pet.AdminPetDetail, error) {
	var playerCount int64
	if err := r.db.QueryRowContext(ctx, adminPetPlayerExistsQuery, input.PlayerID).Scan(&playerCount); err != nil {
		return nil, err
	}
	if playerCount == 0 {
		return nil, pet.ErrPetNotFound
	}
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet skill ids: %w", err)
	}
	var petUID int64
	if err := r.db.QueryRowContext(ctx, insertAdminPetQuery, input.PlayerID, input.PetID, input.Level, input.Exp, input.Quality, input.HP, input.HPMax, input.ATK, input.DEF, input.SPD, input.MANA, skillIDsJSON).Scan(&petUID); err != nil {
		return nil, err
	}
	return r.FindAdminDetailByPetUID(ctx, uint64(petUID))
}

func (r *PetRepository) UpdateForAdmin(ctx context.Context, petUID uint64, input pet.AdminUpdatePetInput) (*pet.AdminPetDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet skill ids: %w", err)
	}
	result, err := r.db.ExecContext(ctx, updateAdminPetQuery, petUID, input.PetID, input.Level, input.Exp, input.Quality, input.HP, input.HPMax, input.ATK, input.DEF, input.SPD, input.MANA, skillIDsJSON)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, pet.ErrPetNotFound
	}
	return r.FindAdminDetailByPetUID(ctx, petUID)
}

func (r *PetRepository) DeleteForAdmin(ctx context.Context, petUID uint64) error {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres transaction is unavailable")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)
	if _, err := tx.ExecContext(ctx, deleteAdminPetLineupQuery, petUID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, deleteAdminPetQuery, petUID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return pet.ErrPetNotFound
	}
	return tx.Commit()
}

func scanPetRow(rows *sql.Rows) (pet.Pet, error) {
	var (
		item         pet.Pet
		petUID       int64
		petID        int64
		level        int64
		exp          int64
		quality      int64
		hp           int64
		hpMax        int64
		atk          int64
		def          int64
		spd          int64
		mana         int64
		skillIDsJSON []byte
	)
	if err := rows.Scan(&petUID, &petID, &level, &exp, &quality, &hp, &hpMax, &atk, &def, &spd, &mana, &skillIDsJSON); err != nil {
		return pet.Pet{}, err
	}
	item.PetUID = uint64(petUID)
	item.PetID = uint32(petID)
	item.Level = uint32(level)
	item.Exp = uint64(exp)
	item.Quality = uint32(quality)
	item.HP = uint32(hp)
	item.HPMax = uint32(hpMax)
	item.ATK = uint32(atk)
	item.DEF = uint32(def)
	item.SPD = uint32(spd)
	item.MANA = uint32(mana)
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &item.SkillIDs); err != nil {
			return pet.Pet{}, fmt.Errorf("unmarshal pet skill ids: %w", err)
		}
	}
	return item, nil
}

func joinConditions(conditions []string) string {
	result := ""
	for index, condition := range conditions {
		if index > 0 {
			result += " AND "
		}
		result += condition
	}
	return result
}
