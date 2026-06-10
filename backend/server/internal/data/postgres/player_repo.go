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
  exp,
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

const addPlayerGoldAndExpQuery = `
UPDATE player
SET gold = gold + $2,
    exp = exp + $3
WHERE id = $1
`

func (r *PlayerRepository) FindByPlayerID(ctx context.Context, playerID uint64) (*player.Profile, error) {
	var (
		profile    player.Profile
		profileID  int64
		level      int64
		exp        int64
		gold       int64
		sceneID    int64
		posX, posY int64
	)

	err := r.db.QueryRowContext(ctx, findPlayerByIDQuery, playerID).Scan(
		&profileID,
		&profile.Name,
		&level,
		&exp,
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
	profile.Exp = uint64(exp)
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

func (r *PlayerRepository) AddGoldAndExp(ctx context.Context, playerID uint64, gold uint32, exp uint64) (*player.Profile, error) {
	result, err := r.db.ExecContext(ctx, addPlayerGoldAndExpQuery, playerID, gold, exp)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, player.ErrPlayerNotFound
	}
	return r.FindByPlayerID(ctx, playerID)
}
