package postgres

import (
	"context"

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
