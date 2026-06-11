package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"pocket-pet-remake/server/internal/module/auth"
	"pocket-pet-remake/server/internal/module/player"
)

type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

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
  pos_y,
  hp,
  hp_max,
  energy,
  energy_max,
  atk,
  def,
  spd,
  mana,
  hit_pct,
  dodge_pct,
  crit_rate_pct,
  crit_dmg_pct,
  physical_resist_pct,
  skill_resist_pct,
  confusion_resist_pct,
  sleep_resist_pct,
  paralysis_resist_pct,
  seal_resist_pct,
  curse_resist_pct,
  crit_resist_pct,
  crit_dmg_resist_pct,
  character_resist_pct,
  pet_resist_pct,
  mercenary_resist_pct,
  generic_shield_pct,
  skill_ids
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

const findAdminPlayerDetailByIDQuery = `
SELECT
  p.id,
  p.account_id,
  a.account_name,
  p.name,
  p.level,
  p.exp,
  p.gold,
  p.status,
  p.scene_id,
  p.pos_x,
  p.pos_y,
  p.hp,
  p.hp_max,
  p.energy,
  p.energy_max,
  p.atk,
  p.def,
  p.spd,
  p.mana,
  p.hit_pct,
  p.dodge_pct,
  p.crit_rate_pct,
  p.crit_dmg_pct,
  p.physical_resist_pct,
  p.skill_resist_pct,
  p.confusion_resist_pct,
  p.sleep_resist_pct,
  p.paralysis_resist_pct,
  p.seal_resist_pct,
  p.curse_resist_pct,
  p.crit_resist_pct,
  p.crit_dmg_resist_pct,
  p.character_resist_pct,
  p.pet_resist_pct,
  p.mercenary_resist_pct,
  p.generic_shield_pct,
  p.skill_ids,
  a.last_login_at,
  p.created_at,
  p.updated_at
FROM player p
LEFT JOIN account a ON a.id = p.account_id
WHERE p.id = $1
LIMIT 1
`

const insertAdminAccountQuery = `
INSERT INTO account (
  account_name,
  password_hash,
  status
) VALUES ($1, $2, 1)
RETURNING id
`

const insertAdminPlayerWithSkillsQuery = `
INSERT INTO player (
  account_id,
  name,
  level,
  exp,
  gold,
  scene_id,
  pos_x,
  pos_y,
  hp,
  hp_max,
  status,
  energy,
  energy_max,
  atk,
  def,
  spd,
  mana,
  skill_ids
) VALUES (
  $1, $2, $3, 0, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING id
`

const insertAdminPlayerQuery = `
INSERT INTO player (
  account_id,
  name,
  level,
  exp,
  gold,
  scene_id,
  pos_x,
  pos_y,
  hp,
  hp_max,
  status,
  energy,
  energy_max,
  atk,
  def,
  spd,
  mana
) VALUES (
  $1, $2, $3, 0, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING id
`

const updateAdminPlayerQuery = `
UPDATE player
SET name = $2,
    level = $3,
    exp = $4,
    gold = $5,
    scene_id = $6,
    pos_x = $7,
    pos_y = $8,
    hp = $9,
    hp_max = $10,
    energy = $11,
    energy_max = $12,
    atk = $13,
    def = $14,
    spd = $15,
    mana = $16,
    status = $17,
    skill_ids = $18
WHERE id = $1
`

const softDeleteAdminPlayerQuery = `
UPDATE player p
SET status = 0
WHERE p.id = $1
`

const softDeleteAdminAccountByPlayerIDQuery = `
UPDATE account a
SET status = 0,
    updated_at = CURRENT_TIMESTAMP
FROM player p
WHERE p.account_id = a.id AND p.id = $1
`

func (r *PlayerRepository) FindByPlayerID(ctx context.Context, playerID uint64) (*player.Profile, error) {
	var (
		profile                                                                player.Profile
		profileID                                                              int64
		level                                                                  int64
		exp                                                                    int64
		gold                                                                   int64
		sceneID                                                                int64
		posX, posY                                                             int64
		hp, hpMax                                                              int64
		energy, energyMax                                                      int64
		atk, def, spd, mana                                                    int64
		hitPct, dodgePct                                                       int64
		critRatePct, critDmgPct                                                int64
		physicalResistPct, skillResistPct                                      int64
		confusionResistPct, sleepResistPct                                     int64
		paralysisResistPct, sealResistPct                                      int64
		curseResistPct, critResistPct, critDmgResistPct                        int64
		characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct int64
		skillIDsJSON                                                           []byte
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
		&hp,
		&hpMax,
		&energy,
		&energyMax,
		&atk,
		&def,
		&spd,
		&mana,
		&hitPct,
		&dodgePct,
		&critRatePct,
		&critDmgPct,
		&physicalResistPct,
		&skillResistPct,
		&confusionResistPct,
		&sleepResistPct,
		&paralysisResistPct,
		&sealResistPct,
		&curseResistPct,
		&critResistPct,
		&critDmgResistPct,
		&characterResistPct,
		&petResistPct,
		&mercenaryResistPct,
		&genericShieldPct,
		&skillIDsJSON,
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
	profile.HP = uint32(hp)
	profile.HPMax = uint32(hpMax)
	profile.Energy = uint32(energy)
	profile.EnergyMax = uint32(energyMax)
	profile.ATK = uint32(atk)
	profile.DEF = uint32(def)
	profile.SPD = uint32(spd)
	profile.MANA = uint32(mana)
	profile.HitPct = uint32(hitPct)
	profile.DodgePct = uint32(dodgePct)
	profile.CritRatePct = uint32(critRatePct)
	profile.CritDmgPct = uint32(critDmgPct)
	profile.PhysicalResistPct = uint32(physicalResistPct)
	profile.SkillResistPct = uint32(skillResistPct)
	profile.ConfusionResistPct = uint32(confusionResistPct)
	profile.SleepResistPct = uint32(sleepResistPct)
	profile.ParalysisResistPct = uint32(paralysisResistPct)
	profile.SealResistPct = uint32(sealResistPct)
	profile.CurseResistPct = uint32(curseResistPct)
	profile.CritResistPct = uint32(critResistPct)
	profile.CritDmgResistPct = uint32(critDmgResistPct)
	profile.CharacterResistPct = uint32(characterResistPct)
	profile.PetResistPct = uint32(petResistPct)
	profile.MercenaryResistPct = uint32(mercenaryResistPct)
	profile.GenericShieldPct = uint32(genericShieldPct)
	if len(skillIDsJSON) > 0 {
		// 人物技能配置和宠物技能一样从数据库权威读取，避免人物参战时再回退到硬编码列表。
		if err := json.Unmarshal(skillIDsJSON, &profile.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal player skill ids: %w", err)
		}
	}
	return &profile, nil
}

func (r *PlayerRepository) ListForAdmin(ctx context.Context, query player.AdminListQuery) (*player.AdminPlayerList, error) {
	query = query.Normalize()

	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if query.PlayerID > 0 {
		conditions = append(conditions, "p.id = "+nextArg(query.PlayerID))
	}
	if query.Name != "" {
		conditions = append(conditions, "p.name ILIKE "+nextArg("%"+query.Name+"%"))
	}
	if query.Status != nil {
		conditions = append(conditions, "p.status = "+nextArg(*query.Status))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := `
SELECT COUNT(1)
FROM player p
LEFT JOIN account a ON a.id = p.account_id
` + whereClause

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	listQuery := `
SELECT
  p.id,
  a.account_name,
  p.name,
  p.level,
  p.gold,
  p.status,
  p.scene_id,
  p.hp,
  p.hp_max,
  p.energy,
  p.energy_max,
  a.last_login_at,
  p.updated_at,
  p.created_at
FROM player p
LEFT JOIN account a ON a.id = p.account_id
` + whereClause + `
ORDER BY p.id ASC
LIMIT ` + nextArg(query.PageSize) + `
OFFSET ` + nextArg((query.Page-1)*query.PageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]player.AdminPlayerSummary, 0, query.PageSize)
	for rows.Next() {
		var (
			item                                   player.AdminPlayerSummary
			playerID, level, gold, status, sceneID int64
			hp, hpMax, energy, energyMax           int64
			lastLoginAt                            sql.NullTime
		)
		if err := rows.Scan(
			&playerID,
			&item.AccountName,
			&item.Name,
			&level,
			&gold,
			&status,
			&sceneID,
			&hp,
			&hpMax,
			&energy,
			&energyMax,
			&lastLoginAt,
			&item.UpdatedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.PlayerID = uint64(playerID)
		item.Level = uint32(level)
		item.Gold = uint64(gold)
		item.Status = uint32(status)
		item.StatusText = player.AdminPlayerStatusText(item.Status)
		item.SceneID = uint32(sceneID)
		item.HP = uint32(hp)
		item.HPMax = uint32(hpMax)
		item.Energy = uint32(energy)
		item.EnergyMax = uint32(energyMax)
		if lastLoginAt.Valid {
			value := lastLoginAt.Time
			item.LastLoginAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &player.AdminPlayerList{Items: items, Total: uint64(total), Page: query.Page, PageSize: query.PageSize}, nil
}

func (r *PlayerRepository) FindAdminDetailByPlayerID(ctx context.Context, playerID uint64) (*player.AdminPlayerDetail, error) {
	var (
		detail                                                                 player.AdminPlayerDetail
		accountID, detailPlayerID                                              int64
		level, exp, gold, status, sceneID                                      int64
		posX, posY                                                             int64
		hp, hpMax                                                              int64
		energy, energyMax                                                      int64
		atk, def, spd, mana                                                    int64
		hitPct, dodgePct                                                       int64
		critRatePct, critDmgPct                                                int64
		physicalResistPct, skillResistPct                                      int64
		confusionResistPct, sleepResistPct                                     int64
		paralysisResistPct, sealResistPct                                      int64
		curseResistPct, critResistPct, critDmgResistPct                        int64
		characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct int64
		skillIDsJSON                                                           []byte
		lastLoginAt                                                            sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, findAdminPlayerDetailByIDQuery, playerID).Scan(
		&detailPlayerID,
		&accountID,
		&detail.AccountName,
		&detail.Name,
		&level,
		&exp,
		&gold,
		&status,
		&sceneID,
		&posX,
		&posY,
		&hp,
		&hpMax,
		&energy,
		&energyMax,
		&atk,
		&def,
		&spd,
		&mana,
		&hitPct,
		&dodgePct,
		&critRatePct,
		&critDmgPct,
		&physicalResistPct,
		&skillResistPct,
		&confusionResistPct,
		&sleepResistPct,
		&paralysisResistPct,
		&sealResistPct,
		&curseResistPct,
		&critResistPct,
		&critDmgResistPct,
		&characterResistPct,
		&petResistPct,
		&mercenaryResistPct,
		&genericShieldPct,
		&skillIDsJSON,
		&lastLoginAt,
		&detail.CreatedAt,
		&detail.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return buildAdminPlayerDetail(&detail, accountID, detailPlayerID, level, exp, gold, status, sceneID, posX, posY, hp, hpMax, energy, energyMax, atk, def, spd, mana, hitPct, dodgePct, critRatePct, critDmgPct, physicalResistPct, skillResistPct, confusionResistPct, sleepResistPct, paralysisResistPct, sealResistPct, curseResistPct, critResistPct, critDmgResistPct, characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct, skillIDsJSON, lastLoginAt)
}

func (r *PlayerRepository) CreateForAdmin(ctx context.Context, input player.AdminCreatePlayerInput) (*player.AdminPlayerDetail, error) {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	var accountID int64
	if err := tx.QueryRowContext(ctx, insertAdminAccountQuery, input.AccountName, auth.HashPassword(input.Password)).Scan(&accountID); err != nil {
		return nil, mapPlayerPersistenceError(err)
	}

	var playerID int64
	if len(input.SkillIDs) > 0 {
		skillIDsJSON, err := json.Marshal(input.SkillIDs)
		if err != nil {
			return nil, fmt.Errorf("marshal admin player skill ids: %w", err)
		}
		if err := tx.QueryRowContext(ctx, insertAdminPlayerWithSkillsQuery,
			accountID,
			input.Name,
			input.Level,
			input.Gold,
			input.SceneID,
			input.PosX,
			input.PosY,
			input.HP,
			input.HPMax,
			input.Status,
			input.Energy,
			input.EnergyMax,
			input.ATK,
			input.DEF,
			input.SPD,
			input.MANA,
			skillIDsJSON,
		).Scan(&playerID); err != nil {
			return nil, mapPlayerPersistenceError(err)
		}
	} else {
		if err := tx.QueryRowContext(ctx, insertAdminPlayerQuery,
			accountID,
			input.Name,
			input.Level,
			input.Gold,
			input.SceneID,
			input.PosX,
			input.PosY,
			input.HP,
			input.HPMax,
			input.Status,
			input.Energy,
			input.EnergyMax,
			input.ATK,
			input.DEF,
			input.SPD,
			input.MANA,
		).Scan(&playerID); err != nil {
			return nil, mapPlayerPersistenceError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindAdminDetailByPlayerID(ctx, uint64(playerID))
}

func (r *PlayerRepository) UpdateForAdmin(ctx context.Context, playerID uint64, input player.AdminUpdatePlayerInput) (*player.AdminPlayerDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin player skill ids: %w", err)
	}
	result, err := r.db.ExecContext(ctx, updateAdminPlayerQuery,
		playerID,
		input.Name,
		input.Level,
		input.Exp,
		input.Gold,
		input.SceneID,
		input.PosX,
		input.PosY,
		input.HP,
		input.HPMax,
		input.Energy,
		input.EnergyMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		input.Status,
		skillIDsJSON,
	)
	if err != nil {
		return nil, mapPlayerPersistenceError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, player.ErrPlayerNotFound
	}
	return r.FindAdminDetailByPlayerID(ctx, playerID)
}

func (r *PlayerRepository) DeleteForAdmin(ctx context.Context, playerID uint64) error {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	result, err := tx.ExecContext(ctx, softDeleteAdminPlayerQuery, playerID)
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
	if _, err := tx.ExecContext(ctx, softDeleteAdminAccountByPlayerIDQuery, playerID); err != nil {
		return err
	}
	return tx.Commit()
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

func (r *PlayerRepository) beginTx(ctx context.Context) (*sql.Tx, error) {
	beginner, ok := r.db.(txBeginner)
	if !ok {
		return nil, fmt.Errorf("postgres transaction is unavailable")
	}
	return beginner.BeginTx(ctx, nil)
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func buildAdminPlayerDetail(
	detail *player.AdminPlayerDetail,
	accountID int64,
	detailPlayerID int64,
	level, exp, gold, status, sceneID int64,
	posX, posY int64,
	hp, hpMax int64,
	energy, energyMax int64,
	atk, def, spd, mana int64,
	hitPct, dodgePct int64,
	critRatePct, critDmgPct int64,
	physicalResistPct, skillResistPct int64,
	confusionResistPct, sleepResistPct int64,
	paralysisResistPct, sealResistPct int64,
	curseResistPct, critResistPct, critDmgResistPct int64,
	characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct int64,
	skillIDsJSON []byte,
	lastLoginAt sql.NullTime,
) (*player.AdminPlayerDetail, error) {
	detail.PlayerID = uint64(detailPlayerID)
	detail.AccountID = uint64(accountID)
	detail.Level = uint32(level)
	detail.Exp = uint64(exp)
	detail.Gold = uint64(gold)
	detail.Status = uint32(status)
	detail.StatusText = player.AdminPlayerStatusText(detail.Status)
	detail.SceneID = uint32(sceneID)
	detail.PosX = int32(posX)
	detail.PosY = int32(posY)
	detail.HP = uint32(hp)
	detail.HPMax = uint32(hpMax)
	detail.Energy = uint32(energy)
	detail.EnergyMax = uint32(energyMax)
	detail.ATK = uint32(atk)
	detail.DEF = uint32(def)
	detail.SPD = uint32(spd)
	detail.MANA = uint32(mana)
	detail.HitPct = uint32(hitPct)
	detail.DodgePct = uint32(dodgePct)
	detail.CritRatePct = uint32(critRatePct)
	detail.CritDmgPct = uint32(critDmgPct)
	detail.PhysicalResistPct = uint32(physicalResistPct)
	detail.SkillResistPct = uint32(skillResistPct)
	detail.ConfusionResistPct = uint32(confusionResistPct)
	detail.SleepResistPct = uint32(sleepResistPct)
	detail.ParalysisResistPct = uint32(paralysisResistPct)
	detail.SealResistPct = uint32(sealResistPct)
	detail.CurseResistPct = uint32(curseResistPct)
	detail.CritResistPct = uint32(critResistPct)
	detail.CritDmgResistPct = uint32(critDmgResistPct)
	detail.CharacterResistPct = uint32(characterResistPct)
	detail.PetResistPct = uint32(petResistPct)
	detail.MercenaryResistPct = uint32(mercenaryResistPct)
	detail.GenericShieldPct = uint32(genericShieldPct)
	if lastLoginAt.Valid {
		value := lastLoginAt.Time
		detail.LastLoginAt = &value
	}
	if len(skillIDsJSON) > 0 {
		// 后台详情直接展示数据库里的角色技能快照，方便运营确认人物出战配置是否正确。
		if err := json.Unmarshal(skillIDsJSON, &detail.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal admin player skill ids: %w", err)
		}
	}
	return detail, nil
}

func mapPlayerPersistenceError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "uk_player_name":
			return player.ErrPlayerNameDuplicated
		case "uk_account_name":
			return player.ErrAccountNameDuplicated
		}
	}
	return err
}
