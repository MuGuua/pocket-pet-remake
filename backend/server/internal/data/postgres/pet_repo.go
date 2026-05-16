package postgres

import (
	"context"
	"encoding/json"
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
  pp.hp_max
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

func (r *PetRepository) ListPetsByPlayerID(ctx context.Context, playerID uint64) ([]pet.Pet, error) {
	rows, err := r.db.QueryContext(ctx, listPetsByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pets := make([]pet.Pet, 0)
	for rows.Next() {
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
			skillIDsJSON []byte
		)
		if err := rows.Scan(&petUID, &petID, &level, &exp, &quality, &hp, &hpMax, &atk, &def, &spd, &skillIDsJSON); err != nil {
			return nil, err
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
		if len(skillIDsJSON) > 0 {
			if err := json.Unmarshal(skillIDsJSON, &item.SkillIDs); err != nil {
				return nil, fmt.Errorf("unmarshal pet skill ids: %w", err)
			}
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
			item   pet.LineupPet
			petUID int64
			petID  int64
			level  int64
			hp     int64
			hpMax  int64
		)
		if err := rows.Scan(&petUID, &petID, &level, &hp, &hpMax); err != nil {
			return nil, err
		}
		item.PetUID = uint64(petUID)
		item.PetID = uint32(petID)
		item.Level = uint32(level)
		item.HP = uint32(hp)
		item.HPMax = uint32(hpMax)
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
