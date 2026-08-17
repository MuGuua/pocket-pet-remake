package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
  free_attr_points,
  strength,
  vitality,
  agility,
  mind,
  gold,
  scene_id,
  pos_x,
  pos_y,
  position_version,
  hp,
  hp_max,
  vigor,
  vigor_max,
  spirit,
  spirit_max,
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
  guard,
  talent_dmg_pct,
  talent_reduce_pct,
  element_adv_pct,
  element_penalty_pct,
  skill_ids,
  skin_id,
  base_hp_max,
  base_atk,
  base_def,
  base_spd,
  base_mana,
  base_hit_pct,
  base_dodge_pct
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

const updatePlayerPositionIfNewerQuery = `
UPDATE player
SET scene_id = $2,
    pos_x = $3,
    pos_y = $4,
    position_version = $5
WHERE id = $1
  AND position_version < $5
`

const listWorldPlayerSummariesQueryPrefix = `
SELECT
  id,
  name,
  level,
  exp,
  scene_id,
  pos_x,
  pos_y,
  hp,
  hp_max,
  vigor,
  vigor_max,
  spirit,
  spirit_max,
  skin_id
FROM player
WHERE status = 1 AND id IN (`

const findAdminPlayerDetailByIDQuery = `
SELECT
  p.id,
  p.account_id,
  a.account_name,
  p.name,
  p.level,
  p.exp,
  p.free_attr_points,
  p.strength,
  p.vitality,
  p.agility,
  p.mind,
  p.gold,
  p.status,
  p.scene_id,
  p.pos_x,
  p.pos_y,
  p.hp,
  p.hp_max,
  p.vigor,
  p.vigor_max,
  p.spirit,
  p.spirit_max,
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
  p.guard,
  p.talent_dmg_pct,
  p.talent_reduce_pct,
  p.element_adv_pct,
  p.element_penalty_pct,
  p.skill_ids,
  p.skin_id,
  a.last_login_at,
  p.created_at,
  p.updated_at
FROM player p
LEFT JOIN account a ON a.id = p.account_id
WHERE p.id = $1
LIMIT 1
`

const listAdminPlayerEquippedItemsQuery = `
SELECT
  pes.equip_slot,
  COALESCE(ei.item_uid, ''),
  COALESCE(ei.item_id, 0),
  COALESCE(idf.item_name, ''),
  COALESCE(ei.enhance_level, 0)
FROM player_equipment_slot pes
LEFT JOIN equipment_instance ei ON ei.item_uid = pes.item_uid
LEFT JOIN item_definition idf ON idf.item_id = ei.item_id
WHERE pes.player_id = $1
ORDER BY pes.equip_slot ASC
`

const insertAdminAccountQuery = `
INSERT INTO account (
  account_name,
  password_hash,
  status
) VALUES ($1, $2, 1)
RETURNING id
`

const insertStarterPlayerQuery = `
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
  vigor,
  vigor_max,
  spirit,
  spirit_max,
  atk,
  def,
  spd,
  mana,
  hit_pct,
  dodge_pct,
  crit_rate_pct,
  crit_dmg_pct,
  base_hp_max,
  base_atk,
  base_def,
  base_spd,
  base_mana,
  base_hit_pct,
  base_dodge_pct,
  skill_ids,
  skin_id
) VALUES (
  $1, $2, $3, 0, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
  $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
)
RETURNING id
`

const insertStarterPlayerBagContainerQuery = `
INSERT INTO player_container (player_id, container_type, capacity, max_capacity)
VALUES ($1, 'bag', $2, $3)
ON CONFLICT (player_id, container_type) DO NOTHING
`

const insertStarterPlayerWarehouseContainerQuery = `
INSERT INTO player_container (player_id, container_type, capacity, max_capacity)
VALUES ($1, 'warehouse', 30, 300)
ON CONFLICT (player_id, container_type) DO NOTHING
`

const insertStarterPlayerWalletQuery = `
INSERT INTO player_wallet (player_id, currency_copper_total)
VALUES ($1, 0)
ON CONFLICT (player_id) DO NOTHING
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
    vigor = $11,
    vigor_max = $12,
    spirit = $13,
    spirit_max = $14,
    atk = $15,
    def = $16,
    spd = $17,
    mana = $18,
    status = $19,
    skin_id = $20,
    skill_ids = $21,
    base_hp_max = $10,
    base_atk = $15,
    base_def = $16,
    base_spd = $17,
    base_mana = $18
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

const findAdminAccountIDByPlayerIDQuery = `
SELECT account_id
FROM player
WHERE id = $1
`

const lockAdminAccountStatusQuery = `
SELECT status
FROM account
WHERE id = $1
FOR UPDATE
`

const lockAdminAccountPlayerStatusesQuery = `
SELECT status
FROM player
WHERE account_id = $1
FOR UPDATE
`

func (r *PlayerRepository) FindByPlayerID(ctx context.Context, playerID uint64) (*player.Profile, error) {
	if playerID == 0 {
		return nil, nil
	}
	if err := r.RefreshPlayerCombatSnapshot(ctx, playerID); err != nil {
		return nil, err
	}
	profile, err := r.FindPlayerCombatSnapshot(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}
	return r.findPlayerByIDRaw(ctx, playerID)
}

// ListWorldSummariesByPlayerIDs 使用一次查询返回同屏展示需要的玩家轻量字段。
func (r *PlayerRepository) ListWorldSummariesByPlayerIDs(ctx context.Context, playerIDs []uint64) ([]player.WorldSummary, error) {
	if len(playerIDs) == 0 {
		return []player.WorldSummary{}, nil
	}
	placeholders := make([]string, 0, len(playerIDs))
	args := make([]any, 0, len(playerIDs))
	for index, playerID := range playerIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", index+1))
		args = append(args, playerID)
	}
	query := listWorldPlayerSummariesQueryPrefix + strings.Join(placeholders, ",") + ") ORDER BY id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]player.WorldSummary, 0, len(playerIDs))
	for rows.Next() {
		var value player.WorldSummary
		if err := rows.Scan(
			&value.PlayerID,
			&value.Name,
			&value.Level,
			&value.Exp,
			&value.SceneID,
			&value.PosX,
			&value.PosY,
			&value.HP,
			&value.HPMax,
			&value.Vigor,
			&value.VigorMax,
			&value.Spirit,
			&value.SpiritMax,
			&value.SkinID,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PlayerRepository) findPlayerByIDRaw(ctx context.Context, playerID uint64) (*player.Profile, error) {
	return scanRawPlayerByID(ctx, r.db, playerID)
}

func scanRawPlayerByIDTx(ctx context.Context, tx *sql.Tx, playerID uint64) (*player.Profile, error) {
	return scanRawPlayerByID(ctx, tx, playerID)
}

func scanRawPlayerByID(ctx context.Context, db DBTX, playerID uint64) (*player.Profile, error) {
	var (
		profile                                                                  player.Profile
		profileID                                                                int64
		level                                                                    int64
		exp                                                                      int64
		freeAttrPoints                                                           int64
		strength, vitality, agility, mind                                        int64
		gold                                                                     int64
		sceneID                                                                  int64
		posX, posY, positionVersion                                              int64
		hp, hpMax                                                                int64
		vigor, vigorMax                                                          int64
		spirit, spiritMax                                                        int64
		atk, def, spd, mana                                                      int64
		hitPct, dodgePct                                                         int64
		critRatePct, critDmgPct                                                  int64
		physicalResistPct, skillResistPct                                        int64
		confusionResistPct, sleepResistPct                                       int64
		paralysisResistPct, sealResistPct                                        int64
		curseResistPct, critResistPct, critDmgResistPct                          int64
		characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct   int64
		guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct   int64
		skillIDsJSON                                                             []byte
		skinID                                                                   string
		baseHPMax, baseATK, baseDEF, baseSPD, baseMANA, baseHitPct, baseDodgePct int64
	)

	err := db.QueryRowContext(ctx, findPlayerByIDQuery, playerID).Scan(
		&profileID,
		&profile.Name,
		&level,
		&exp,
		&freeAttrPoints,
		&strength,
		&vitality,
		&agility,
		&mind,
		&gold,
		&sceneID,
		&posX,
		&posY,
		&positionVersion,
		&hp,
		&hpMax,
		&vigor,
		&vigorMax,
		&spirit,
		&spiritMax,
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
		&guard,
		&talentDmgPct,
		&talentReducePct,
		&elementAdvPct,
		&elementPenaltyPct,
		&skillIDsJSON,
		&skinID,
		&baseHPMax,
		&baseATK,
		&baseDEF,
		&baseSPD,
		&baseMANA,
		&baseHitPct,
		&baseDodgePct,
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
	profile.FreeAttrPoints = uint32(freeAttrPoints)
	profile.Strength = uint32(strength)
	profile.Vitality = uint32(vitality)
	profile.Agility = uint32(agility)
	profile.Mind = uint32(mind)
	profile.Gold = uint32(gold)
	profile.SceneID = uint32(sceneID)
	profile.PosX = int32(posX)
	profile.PosY = int32(posY)
	profile.PositionVersion = uint64(positionVersion)
	profile.HP = uint32(hp)
	profile.HPMax = uint32(hpMax)
	profile.Vigor = uint32(vigor)
	profile.VigorMax = uint32(vigorMax)
	profile.Spirit = uint32(spirit)
	profile.SpiritMax = uint32(spiritMax)
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
	profile.Guard = uint32(guard)
	profile.TalentDmgPct = uint32(talentDmgPct)
	profile.TalentReducePct = uint32(talentReducePct)
	profile.ElementAdvPct = uint32(elementAdvPct)
	profile.ElementPenaltyPct = uint32(elementPenaltyPct)
	profile.SkinID = strings.TrimSpace(skinID)
	profile.BaseHPMax = uint32(baseHPMax)
	profile.BaseATK = uint32(baseATK)
	profile.BaseDEF = uint32(baseDEF)
	profile.BaseSPD = uint32(baseSPD)
	profile.BaseMANA = uint32(baseMANA)
	profile.BaseHitPct = uint32(baseHitPct)
	profile.BaseDodgePct = uint32(baseDodgePct)
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
  p.vigor,
  p.vigor_max,
  p.spirit,
  p.spirit_max,
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
			item                                          player.AdminPlayerSummary
			playerID, level, gold, status, sceneID        int64
			hp, hpMax, vigor, vigorMax, spirit, spiritMax int64
			lastLoginAt                                   sql.NullTime
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
			&vigor,
			&vigorMax,
			&spirit,
			&spiritMax,
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
		item.Vigor = uint32(vigor)
		item.VigorMax = uint32(vigorMax)
		item.Spirit = uint32(spirit)
		item.SpiritMax = uint32(spiritMax)
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
		level, exp, freeAttrPoints                                             int64
		strength, vitality, agility, mind                                      int64
		gold, status, sceneID                                                  int64
		posX, posY                                                             int64
		hp, hpMax                                                              int64
		vigor, vigorMax                                                        int64
		spirit, spiritMax                                                      int64
		atk, def, spd, mana                                                    int64
		hitPct, dodgePct                                                       int64
		critRatePct, critDmgPct                                                int64
		physicalResistPct, skillResistPct                                      int64
		confusionResistPct, sleepResistPct                                     int64
		paralysisResistPct, sealResistPct                                      int64
		curseResistPct, critResistPct, critDmgResistPct                        int64
		characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct int64
		guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct int64
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
		&freeAttrPoints,
		&strength,
		&vitality,
		&agility,
		&mind,
		&gold,
		&status,
		&sceneID,
		&posX,
		&posY,
		&hp,
		&hpMax,
		&vigor,
		&vigorMax,
		&spirit,
		&spiritMax,
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
		&guard,
		&talentDmgPct,
		&talentReducePct,
		&elementAdvPct,
		&elementPenaltyPct,
		&skillIDsJSON,
		&detail.SkinID,
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

	equippedItems, err := r.loadAdminPlayerEquippedItems(ctx, playerID)
	if err != nil {
		return nil, err
	}
	return buildAdminPlayerDetail(&detail, accountID, detailPlayerID, level, exp, freeAttrPoints, strength, vitality, agility, mind, gold, status, sceneID, posX, posY, hp, hpMax, vigor, vigorMax, spirit, spiritMax, atk, def, spd, mana, hitPct, dodgePct, critRatePct, critDmgPct, physicalResistPct, skillResistPct, confusionResistPct, sleepResistPct, paralysisResistPct, sealResistPct, curseResistPct, critResistPct, critDmgResistPct, characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct, guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct, skillIDsJSON, lastLoginAt, equippedItems)
}

func (r *PlayerRepository) CreateForAdmin(ctx context.Context, input player.AdminCreatePlayerInput) (*player.AdminPlayerDetail, error) {
	input = input.Normalize()
	stats := input.ResolveCreateStats()
	starter := player.DefaultStarterProfile()
	status := input.Status
	if status == 0 {
		status = 1
	}
	tx, err := r.beginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	var accountID int64
	if err := tx.QueryRowContext(ctx, insertAdminAccountQuery, input.AccountName, auth.HashPassword(input.Password)).Scan(&accountID); err != nil {
		return nil, mapPlayerPersistenceError(err)
	}

	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin player skill ids: %w", err)
	}

	var playerID int64
	if err := tx.QueryRowContext(ctx, insertStarterPlayerQuery,
		accountID,
		input.Name,
		input.Level,
		input.Gold,
		input.SceneID,
		input.PosX,
		input.PosY,
		stats.HP,
		stats.HPMax,
		status,
		stats.Vigor,
		stats.VigorMax,
		stats.Spirit,
		stats.SpiritMax,
		stats.ATK,
		stats.DEF,
		stats.SPD,
		stats.MANA,
		stats.HitPct,
		stats.DodgePct,
		stats.CritRatePct,
		stats.CritDmgPct,
		stats.BaseHPMax,
		stats.BaseATK,
		stats.BaseDEF,
		stats.BaseSPD,
		stats.BaseMANA,
		stats.BaseHitPct,
		stats.BaseDodgePct,
		skillIDsJSON,
		resolveAdminPlayerSkinID(input.SkinID),
	).Scan(&playerID); err != nil {
		return nil, mapPlayerPersistenceError(err)
	}

	if _, err := tx.ExecContext(ctx, insertStarterPlayerBagContainerQuery, playerID, starter.BagCapacity, starter.BagMaxCapacity); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, insertStarterPlayerWarehouseContainerQuery, playerID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, insertStarterPlayerWalletQuery, playerID); err != nil {
		return nil, err
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
		input.Vigor,
		input.VigorMax,
		input.Spirit,
		input.SpiritMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		input.Status,
		resolveAdminPlayerSkinID(input.SkinID),
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

// PurgeDisabledAccountForAdmin 在同一事务内删除账号下全部玩家数据和账号本身。
// 只有 account 与其全部 player 均已处于 status=0 时才允许执行，避免误删仍可登录或仍在使用的账号。
func (r *PlayerRepository) PurgeDisabledAccountForAdmin(ctx context.Context, playerID uint64) error {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	var accountID uint64
	if err := tx.QueryRowContext(ctx, findAdminAccountIDByPlayerIDQuery, playerID).Scan(&accountID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return player.ErrPlayerNotFound
		}
		return err
	}
	var accountStatus uint32
	if err := tx.QueryRowContext(ctx, lockAdminAccountStatusQuery, accountID).Scan(&accountStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return player.ErrPlayerNotFound
		}
		return err
	}
	playerStatusRows, err := tx.QueryContext(ctx, lockAdminAccountPlayerStatusesQuery, accountID)
	if err != nil {
		return err
	}
	allPlayersDisabled := true
	playerCount := 0
	for playerStatusRows.Next() {
		var status uint32
		if err := playerStatusRows.Scan(&status); err != nil {
			playerStatusRows.Close()
			return err
		}
		playerCount++
		if status != 0 {
			allPlayersDisabled = false
		}
	}
	if err := playerStatusRows.Err(); err != nil {
		playerStatusRows.Close()
		return err
	}
	playerStatusRows.Close()
	if accountStatus != 0 || playerCount == 0 || !allPlayersDisabled {
		return player.ErrPlayerMustBeDisabled
	}

	// 删除顺序先处理引用装备、宠物的子表，再处理所有 player_id 数据，最后删除 player/account 主记录。
	deleteQueries := []string{
		`DELETE FROM player_equipment_slot WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_equipment_snapshot WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_container_item WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM equipment_instance_socket WHERE item_uid IN (SELECT ei.item_uid FROM equipment_instance ei JOIN player p ON p.id = ei.player_id WHERE p.account_id = $1)`,
		`DELETE FROM equipment_instance WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM pet_artifact_equipment WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_pet_combat_snapshot WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM pet_attr_allocate_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_lineup WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_pet WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_combat_snapshot WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_skill_progress_snapshot WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_skill_progress WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_attr_allocate_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_feature_unlock WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_unique_item_obtained WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_npc_dialogue_session WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_story_flag WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_quest_event_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_quest_objective WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_quest WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM battle_record WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM item_change_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM container_expand_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM currency_change_log WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_wallet WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_container WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
		`DELETE FROM player_item WHERE player_id IN (SELECT id FROM player WHERE account_id = $1)`,
	}
	for _, query := range deleteQueries {
		if _, err := tx.ExecContext(ctx, query, accountID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM player WHERE account_id = $1`, accountID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM account WHERE id = $1 AND status = 0`, accountID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return player.ErrPlayerMustBeDisabled
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

// UpdatePositionIfNewer 通过 PostgreSQL 条件更新保证旧 Redis 批次不能覆盖已经落库的新位置。
// 返回 false 且 error 为 nil 同时覆盖“版本不够新”和“玩家不存在”两种无需重试的情况。
func (r *PlayerRepository) UpdatePositionIfNewer(
	ctx context.Context,
	playerID uint64,
	sceneID uint32,
	posX int32,
	posY int32,
	positionVersion uint64,
) (bool, error) {
	if positionVersion > math.MaxInt64 {
		return false, fmt.Errorf("position version %d exceeds PostgreSQL BIGINT", positionVersion)
	}
	result, err := r.db.ExecContext(
		ctx,
		updatePlayerPositionIfNewerQuery,
		playerID,
		sceneID,
		posX,
		posY,
		int64(positionVersion),
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
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
	level, exp, freeAttrPoints int64,
	strength, vitality, agility, mind int64,
	gold, status, sceneID int64,
	posX, posY int64,
	hp, hpMax int64,
	vigor, vigorMax, spirit, spiritMax int64,
	atk, def, spd, mana int64,
	hitPct, dodgePct int64,
	critRatePct, critDmgPct int64,
	physicalResistPct, skillResistPct int64,
	confusionResistPct, sleepResistPct int64,
	paralysisResistPct, sealResistPct int64,
	curseResistPct, critResistPct, critDmgResistPct int64,
	characterResistPct, petResistPct, mercenaryResistPct, genericShieldPct int64,
	guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct int64,
	skillIDsJSON []byte,
	lastLoginAt sql.NullTime,
	equippedItems []player.AdminPlayerEquippedItem,
) (*player.AdminPlayerDetail, error) {
	detail.PlayerID = uint64(detailPlayerID)
	detail.AccountID = uint64(accountID)
	detail.Level = uint32(level)
	detail.Exp = uint64(exp)
	detail.FreeAttrPoints = uint32(freeAttrPoints)
	detail.Strength = uint32(strength)
	detail.Vitality = uint32(vitality)
	detail.Agility = uint32(agility)
	detail.Mind = uint32(mind)
	detail.Gold = uint64(gold)
	detail.Status = uint32(status)
	detail.StatusText = player.AdminPlayerStatusText(detail.Status)
	detail.SceneID = uint32(sceneID)
	detail.PosX = int32(posX)
	detail.PosY = int32(posY)
	detail.HP = uint32(hp)
	detail.HPMax = uint32(hpMax)
	detail.Vigor = uint32(vigor)
	detail.VigorMax = uint32(vigorMax)
	detail.Spirit = uint32(spirit)
	detail.SpiritMax = uint32(spiritMax)
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
	detail.Guard = uint32(guard)
	detail.TalentDmgPct = uint32(talentDmgPct)
	detail.TalentReducePct = uint32(talentReducePct)
	detail.ElementAdvPct = uint32(elementAdvPct)
	detail.ElementPenaltyPct = uint32(elementPenaltyPct)
	detail.SkinID = strings.TrimSpace(detail.SkinID)
	detail.EquippedItems = equippedItems
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

func (r *PlayerRepository) loadAdminPlayerEquippedItems(ctx context.Context, playerID uint64) ([]player.AdminPlayerEquippedItem, error) {
	result := player.DefaultAdminPlayerEquippedItems()
	indexBySlot := make(map[string]int, len(result))
	for index, item := range result {
		indexBySlot[item.EquipSlot] = index
	}

	rows, err := r.db.QueryContext(ctx, listAdminPlayerEquippedItemsQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			equipSlot    string
			itemUID      string
			itemID       int64
			itemName     string
			enhanceLevel int64
		)
		if err := rows.Scan(&equipSlot, &itemUID, &itemID, &itemName, &enhanceLevel); err != nil {
			return nil, err
		}
		slotIndex, ok := indexBySlot[equipSlot]
		if !ok {
			continue
		}
		result[slotIndex] = player.AdminPlayerEquippedItem{
			EquipSlot:      equipSlot,
			EquipSlotLabel: resolveAdminPlayerEquipSlotLabel(equipSlot),
			ItemUID:        strings.TrimSpace(itemUID),
			ItemID:         uint64(itemID),
			ItemName:       strings.TrimSpace(itemName),
			EnhanceLevel:   uint32(enhanceLevel),
			IsEmpty:        strings.TrimSpace(itemUID) == "",
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func resolveAdminPlayerEquipSlotLabel(slot string) string {
	for _, item := range player.DefaultAdminPlayerEquippedItems() {
		if item.EquipSlot == slot {
			return item.EquipSlotLabel
		}
	}
	return slot
}

func resolveAdminPlayerSkinID(skinID string) string {
	skinID = strings.TrimSpace(skinID)
	if skinID == "" {
		return player.DefaultPlayerSkinID
	}
	return skinID
}

const countActivePlayersQuery = `
SELECT COUNT(1)
FROM player
WHERE status = 1
`

func (r *PlayerRepository) CountActivePlayers(ctx context.Context) (uint64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, countActivePlayersQuery).Scan(&total); err != nil {
		return 0, err
	}
	return uint64(total), nil
}

// AddRewardAttribute 只允许白名单字段累加，避免奖励配置把任意 SQL 列名拼进更新语句。
func (r *PlayerRepository) AddRewardAttribute(ctx context.Context, playerID uint64, attrKey string, value uint32) error {
	columnName := ""
	baseColumnName := ""
	alsoIncreaseHP := false
	switch strings.ToLower(strings.TrimSpace(attrKey)) {
	case "free_attr_points":
		columnName = "free_attr_points"
	case "strength":
		columnName = "strength"
	case "vitality":
		columnName = "vitality"
	case "agility":
		columnName = "agility"
	case "mind":
		columnName = "mind"
	case "hp_max":
		columnName = "base_hp_max"
		baseColumnName = "base_hp_max"
		alsoIncreaseHP = true
	case "atk":
		columnName = "base_atk"
		baseColumnName = "base_atk"
	case "def":
		columnName = "base_def"
		baseColumnName = "base_def"
	case "spd":
		columnName = "base_spd"
		baseColumnName = "base_spd"
	case "mana":
		columnName = "base_mana"
		baseColumnName = "base_mana"
	default:
		return player.ErrInvalidRewardAttrKey
	}
	if playerID == 0 || value == 0 {
		return nil
	}
	query := fmt.Sprintf("UPDATE player SET %s = %s + $2 WHERE id = $1 AND status = 1", columnName, columnName)
	if alsoIncreaseHP {
		query = fmt.Sprintf("UPDATE player SET %s = %s + $2, hp = hp + $2 WHERE id = $1 AND status = 1", columnName, columnName)
	}
	if baseColumnName != "" {
		switch baseColumnName {
		case "base_hp_max":
			query = "UPDATE player SET base_hp_max = base_hp_max + $2, hp = hp + $2 WHERE id = $1 AND status = 1"
		case "base_atk":
			query = "UPDATE player SET base_atk = base_atk + $2 WHERE id = $1 AND status = 1"
		case "base_def":
			query = "UPDATE player SET base_def = base_def + $2 WHERE id = $1 AND status = 1"
		case "base_spd":
			query = "UPDATE player SET base_spd = base_spd + $2 WHERE id = $1 AND status = 1"
		case "base_mana":
			query = "UPDATE player SET base_mana = base_mana + $2 WHERE id = $1 AND status = 1"
		}
	}
	result, err := r.db.ExecContext(ctx, query, playerID, value)
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
