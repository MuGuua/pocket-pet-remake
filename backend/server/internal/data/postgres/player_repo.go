package postgres

import (
	"context"
	"database/sql"
	"errors"

	"pocket-pet-remake/server/internal/module/player"
)

type PlayerRepository struct {
	db DBTX
}

func NewPlayerRepository(db DBTX) *PlayerRepository {
	return &PlayerRepository{db: db}
}

const findPlayerByIDQuery = `
SELECT
  id,
  name,
  level,
  gold,
  scene_id,
  pos_x,
  pos_y
FROM player
WHERE id = $1 AND status = 1
LIMIT 1
`

const updatePlayerPositionQuery = `
UPDATE player
SET scene_id = $2,
    pos_x = $3,
    pos_y = $4
WHERE id = $1
`

func (r *PlayerRepository) FindByPlayerID(ctx context.Context, playerID uint64) (*player.Profile, error) {
	var (
		profile    player.Profile
		profileID  int64
		level      int64
		gold       int64
		sceneID    int64
		posX, posY int64
	)

	err := r.db.QueryRowContext(ctx, findPlayerByIDQuery, playerID).Scan(
		&profileID,
		&profile.Name,
		&level,
		&gold,
		&sceneID,
		&posX,
		&posY,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	profile.PlayerID = uint64(profileID)
	profile.Level = uint32(level)
	profile.Gold = uint32(gold)
	profile.SceneID = uint32(sceneID)
	profile.PosX = int32(posX)
	profile.PosY = int32(posY)
	return &profile, nil
}

func (r *PlayerRepository) UpdatePosition(ctx context.Context, playerID uint64, sceneID uint32, posX, posY int32) error {
	result, err := r.db.ExecContext(ctx, updatePlayerPositionQuery, playerID, sceneID, posX, posY)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return player.ErrPlayerNotFound
	}
	return nil
}
