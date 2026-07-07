package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/combatcalc"
	"pocket-pet-remake/server/internal/module/equipment"
	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/player"
	"pocket-pet-remake/server/internal/module/progression"
)

const upsertPlayerCombatSnapshotQuery = `
INSERT INTO player_combat_snapshot (
  player_id,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  vigor,
  vigor_max,
  spirit,
  spirit_max,
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
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
  $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
  $31, $32, $33, $34, $35, $36, CURRENT_TIMESTAMP
)
ON CONFLICT (player_id) DO UPDATE SET
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  mana = EXCLUDED.mana,
  vigor = EXCLUDED.vigor,
  vigor_max = EXCLUDED.vigor_max,
  spirit = EXCLUDED.spirit,
  spirit_max = EXCLUDED.spirit_max,
  hit_pct = EXCLUDED.hit_pct,
  dodge_pct = EXCLUDED.dodge_pct,
  crit_rate_pct = EXCLUDED.crit_rate_pct,
  crit_dmg_pct = EXCLUDED.crit_dmg_pct,
  physical_resist_pct = EXCLUDED.physical_resist_pct,
  skill_resist_pct = EXCLUDED.skill_resist_pct,
  confusion_resist_pct = EXCLUDED.confusion_resist_pct,
  sleep_resist_pct = EXCLUDED.sleep_resist_pct,
  paralysis_resist_pct = EXCLUDED.paralysis_resist_pct,
  seal_resist_pct = EXCLUDED.seal_resist_pct,
  curse_resist_pct = EXCLUDED.curse_resist_pct,
  crit_resist_pct = EXCLUDED.crit_resist_pct,
  crit_dmg_resist_pct = EXCLUDED.crit_dmg_resist_pct,
  character_resist_pct = EXCLUDED.character_resist_pct,
  pet_resist_pct = EXCLUDED.pet_resist_pct,
  mercenary_resist_pct = EXCLUDED.mercenary_resist_pct,
  generic_shield_pct = EXCLUDED.generic_shield_pct,
  guard = EXCLUDED.guard,
  talent_dmg_pct = EXCLUDED.talent_dmg_pct,
  talent_reduce_pct = EXCLUDED.talent_reduce_pct,
  element_adv_pct = EXCLUDED.element_adv_pct,
  element_penalty_pct = EXCLUDED.element_penalty_pct,
  skill_ids = EXCLUDED.skill_ids,
  skin_id = EXCLUDED.skin_id,
  updated_at = CURRENT_TIMESTAMP
`

const findPlayerCombatSnapshotQuery = `
SELECT
  p.id,
  p.name,
  p.level,
  p.exp,
  p.free_attr_points,
  p.strength,
  p.vitality,
  p.agility,
  p.mind,
  p.gold,
  p.scene_id,
  p.pos_x,
  p.pos_y,
  s.hp,
  s.hp_max,
  s.vigor,
  s.vigor_max,
  s.spirit,
  s.spirit_max,
  s.atk,
  s.def,
  s.spd,
  s.mana,
  s.hit_pct,
  s.dodge_pct,
  s.crit_rate_pct,
  s.crit_dmg_pct,
  s.physical_resist_pct,
  s.skill_resist_pct,
  s.confusion_resist_pct,
  s.sleep_resist_pct,
  s.paralysis_resist_pct,
  s.seal_resist_pct,
  s.curse_resist_pct,
  s.crit_resist_pct,
  s.crit_dmg_resist_pct,
  s.character_resist_pct,
  s.pet_resist_pct,
  s.mercenary_resist_pct,
  s.generic_shield_pct,
  s.guard,
  s.talent_dmg_pct,
  s.talent_reduce_pct,
  s.element_adv_pct,
  s.element_penalty_pct,
  s.skill_ids,
  s.skin_id,
  p.base_hp_max,
  p.base_atk,
  p.base_def,
  p.base_spd,
  p.base_mana,
  p.base_hit_pct,
  p.base_dodge_pct
FROM player_combat_snapshot s
JOIN player p ON p.id = s.player_id
WHERE s.player_id = $1 AND p.status = 1
LIMIT 1
`

const listPlayerCombatSnapshotAttrRulesQuery = `
SELECT source_attr, target_attr, convert_rate
FROM player_attr_convert_config
WHERE status = 1
ORDER BY id ASC
`

const findPlayerCombatSnapshotHPQuery = `
SELECT hp
FROM player
WHERE id = $1 AND status = 1
LIMIT 1
`

const deletePlayerPetCombatSnapshotsByPlayerIDQuery = `
DELETE FROM player_pet_combat_snapshot
WHERE player_id = $1
`

const upsertPlayerPetCombatSnapshotQuery = `
INSERT INTO player_pet_combat_snapshot (
  pet_uid,
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
  spirit,
  spirit_max,
  hit_pct,
  dodge_pct,
  crit_rate_pct,
  crit_dmg_pct,
  physical_resist_pct,
  reverse_physical_resist_pct,
  skill_resist_pct,
  reverse_skill_resist_pct,
  confusion_resist_pct,
  sleep_resist_pct,
  paralysis_resist_pct,
  seal_resist_pct,
  curse_resist_pct,
  crit_dmg_resist_pct,
  crit_resist_pct,
  character_resist_pct,
  pet_resist_pct,
  guard,
  talent_dmg_pct,
  talent_reduce_pct,
  element_adv_pct,
  element_penalty_pct,
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
  grant_source,
  capture_monster_id,
  free_attr_points,
  alloc_hp_points,
  alloc_atk_points,
  alloc_spd_points,
  alloc_mana_points,
  alloc_def_points,
  base_hp_apt,
  base_atk_apt,
  base_def_apt,
  base_spd_apt,
  base_mana_apt,
  extra_hp_apt,
  extra_atk_apt,
  extra_def_apt,
  extra_spd_apt,
  extra_mana_apt,
  evolution_level,
  rebirth_level,
  aptitude_profile,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
  $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
  $31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
  $41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
  $51, $52, $53, $54, $55, $56, $57, $58, $59, $60,
  CURRENT_TIMESTAMP
)
ON CONFLICT (pet_uid) DO UPDATE SET
  player_id = EXCLUDED.player_id,
  pet_id = EXCLUDED.pet_id,
  level = EXCLUDED.level,
  exp = EXCLUDED.exp,
  quality = EXCLUDED.quality,
  hp = EXCLUDED.hp,
  hp_max = EXCLUDED.hp_max,
  atk = EXCLUDED.atk,
  def = EXCLUDED.def,
  spd = EXCLUDED.spd,
  mana = EXCLUDED.mana,
  spirit = EXCLUDED.spirit,
  spirit_max = EXCLUDED.spirit_max,
  hit_pct = EXCLUDED.hit_pct,
  dodge_pct = EXCLUDED.dodge_pct,
  crit_rate_pct = EXCLUDED.crit_rate_pct,
  crit_dmg_pct = EXCLUDED.crit_dmg_pct,
  physical_resist_pct = EXCLUDED.physical_resist_pct,
  reverse_physical_resist_pct = EXCLUDED.reverse_physical_resist_pct,
  skill_resist_pct = EXCLUDED.skill_resist_pct,
  reverse_skill_resist_pct = EXCLUDED.reverse_skill_resist_pct,
  confusion_resist_pct = EXCLUDED.confusion_resist_pct,
  sleep_resist_pct = EXCLUDED.sleep_resist_pct,
  paralysis_resist_pct = EXCLUDED.paralysis_resist_pct,
  seal_resist_pct = EXCLUDED.seal_resist_pct,
  curse_resist_pct = EXCLUDED.curse_resist_pct,
  crit_dmg_resist_pct = EXCLUDED.crit_dmg_resist_pct,
  crit_resist_pct = EXCLUDED.crit_resist_pct,
  character_resist_pct = EXCLUDED.character_resist_pct,
  pet_resist_pct = EXCLUDED.pet_resist_pct,
  guard = EXCLUDED.guard,
  talent_dmg_pct = EXCLUDED.talent_dmg_pct,
  talent_reduce_pct = EXCLUDED.talent_reduce_pct,
  element_adv_pct = EXCLUDED.element_adv_pct,
  element_penalty_pct = EXCLUDED.element_penalty_pct,
  skill_ids = EXCLUDED.skill_ids,
  innate_skill_ids = EXCLUDED.innate_skill_ids,
  normal_skill_ids = EXCLUDED.normal_skill_ids,
  grant_source = EXCLUDED.grant_source,
  capture_monster_id = EXCLUDED.capture_monster_id,
  free_attr_points = EXCLUDED.free_attr_points,
  alloc_hp_points = EXCLUDED.alloc_hp_points,
  alloc_atk_points = EXCLUDED.alloc_atk_points,
  alloc_spd_points = EXCLUDED.alloc_spd_points,
  alloc_mana_points = EXCLUDED.alloc_mana_points,
  alloc_def_points = EXCLUDED.alloc_def_points,
  base_hp_apt = EXCLUDED.base_hp_apt,
  base_atk_apt = EXCLUDED.base_atk_apt,
  base_def_apt = EXCLUDED.base_def_apt,
  base_spd_apt = EXCLUDED.base_spd_apt,
  base_mana_apt = EXCLUDED.base_mana_apt,
  extra_hp_apt = EXCLUDED.extra_hp_apt,
  extra_atk_apt = EXCLUDED.extra_atk_apt,
  extra_def_apt = EXCLUDED.extra_def_apt,
  extra_spd_apt = EXCLUDED.extra_spd_apt,
  extra_mana_apt = EXCLUDED.extra_mana_apt,
  evolution_level = EXCLUDED.evolution_level,
  rebirth_level = EXCLUDED.rebirth_level,
  aptitude_profile = EXCLUDED.aptitude_profile,
  updated_at = CURRENT_TIMESTAMP
`

const listPlayerPetCombatSnapshotsByPlayerIDQuery = `
SELECT
  pet_uid,
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
  spirit,
  spirit_max,
  hit_pct,
  dodge_pct,
  crit_rate_pct,
  crit_dmg_pct,
  physical_resist_pct,
  reverse_physical_resist_pct,
  skill_resist_pct,
  reverse_skill_resist_pct,
  confusion_resist_pct,
  sleep_resist_pct,
  paralysis_resist_pct,
  seal_resist_pct,
  curse_resist_pct,
  crit_dmg_resist_pct,
  crit_resist_pct,
  character_resist_pct,
  pet_resist_pct,
  guard,
  talent_dmg_pct,
  talent_reduce_pct,
  element_adv_pct,
  element_penalty_pct,
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
  grant_source,
  capture_monster_id,
  free_attr_points,
  alloc_hp_points,
  alloc_atk_points,
  alloc_spd_points,
  alloc_mana_points,
  alloc_def_points,
  base_hp_apt,
  base_atk_apt,
  base_def_apt,
  base_spd_apt,
  base_mana_apt,
  extra_hp_apt,
  extra_atk_apt,
  extra_def_apt,
  extra_spd_apt,
  extra_mana_apt,
  evolution_level,
  rebirth_level,
  aptitude_profile
FROM player_pet_combat_snapshot
WHERE player_id = $1
ORDER BY pet_uid ASC
`

const listPlayerLineupCombatSnapshotsByPlayerIDQuery = `
SELECT
  s.pet_uid,
  s.pet_id,
  s.level,
  s.hp,
  s.hp_max,
  s.atk,
  s.def,
  s.spd,
  s.mana,
  s.spirit,
  s.spirit_max,
  s.hit_pct,
  s.dodge_pct,
  s.crit_rate_pct,
  s.crit_dmg_pct,
  s.physical_resist_pct,
  s.reverse_physical_resist_pct,
  s.skill_resist_pct,
  s.reverse_skill_resist_pct,
  s.confusion_resist_pct,
  s.sleep_resist_pct,
  s.paralysis_resist_pct,
  s.seal_resist_pct,
  s.curse_resist_pct,
  s.crit_dmg_resist_pct,
  s.crit_resist_pct,
  s.character_resist_pct,
  s.pet_resist_pct,
  s.guard,
  s.talent_dmg_pct,
  s.talent_reduce_pct,
  s.element_adv_pct,
  s.element_penalty_pct,
  s.skill_ids
FROM player_lineup pl
JOIN player_pet_combat_snapshot s ON s.pet_uid = pl.pet_uid
WHERE pl.player_id = $1
ORDER BY pl.slot_index ASC
`

const findPlayerPetCombatSnapshotByUIDQuery = `
SELECT
  pet_uid,
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
  spirit,
  spirit_max,
  hit_pct,
  dodge_pct,
  crit_rate_pct,
  crit_dmg_pct,
  physical_resist_pct,
  reverse_physical_resist_pct,
  skill_resist_pct,
  reverse_skill_resist_pct,
  confusion_resist_pct,
  sleep_resist_pct,
  paralysis_resist_pct,
  seal_resist_pct,
  curse_resist_pct,
  crit_dmg_resist_pct,
  crit_resist_pct,
  character_resist_pct,
  pet_resist_pct,
  guard,
  talent_dmg_pct,
  talent_reduce_pct,
  element_adv_pct,
  element_penalty_pct,
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
  grant_source,
  capture_monster_id,
  free_attr_points,
  alloc_hp_points,
  alloc_atk_points,
  alloc_spd_points,
  alloc_mana_points,
  alloc_def_points,
  base_hp_apt,
  base_atk_apt,
  base_def_apt,
  base_spd_apt,
  base_mana_apt,
  extra_hp_apt,
  extra_atk_apt,
  extra_def_apt,
  extra_spd_apt,
  extra_mana_apt,
  evolution_level,
  rebirth_level,
  aptitude_profile
FROM player_pet_combat_snapshot
WHERE player_id = $1 AND pet_uid = $2
LIMIT 1
`

// RefreshPlayerCombatSnapshot 重新计算并写入单个玩家的最终战斗属性快照。
func (r *PlayerRepository) RefreshPlayerCombatSnapshot(ctx context.Context, playerID uint64) error {
	rawProfile, err := r.findPlayerByIDRaw(ctx, playerID)
	if err != nil {
		return err
	}
	if rawProfile == nil {
		return player.ErrPlayerNotFound
	}

	rules, err := r.loadPlayerAttrConvertRules(ctx)
	if err != nil {
		return err
	}
	allocated := progression.AllocatedAttrs{
		Strength: rawProfile.Strength,
		Vitality: rawProfile.Vitality,
		Agility:  rawProfile.Agility,
		Mind:     rawProfile.Mind,
	}
	combatBonus := buildPlayerCombatBonusFromRules(allocated, rules)

	pieceTemplates, err := loadEquippedPieceTemplates(ctx, r.db, playerID)
	if err != nil {
		return err
	}
	equipmentBonus := equipment.SumEquippedBonus(pieceTemplates)
	skinID := equipment.ResolveEquippedSkinIDForSnapshot(pieceTemplates, rawProfile)

	hpMax := combatcalc.FinalMajorStat(combatcalc.MajorStatModifiers{
		Base:      rawProfile.BaseHPMax,
		FlatBonus: int32(combatBonus.HPMax + equipmentBonus.HPMax),
	})
	atk := combatcalc.FinalMajorStat(combatcalc.MajorStatModifiers{
		Base:      rawProfile.BaseATK,
		FlatBonus: int32(combatBonus.ATK + equipmentBonus.ATK),
	})
	def := combatcalc.FinalMajorStat(combatcalc.MajorStatModifiers{
		Base:      rawProfile.BaseDEF,
		FlatBonus: int32(combatBonus.DEF + equipmentBonus.DEF),
	})
	spd := combatcalc.FinalMajorStat(combatcalc.MajorStatModifiers{
		Base:      rawProfile.BaseSPD,
		FlatBonus: int32(combatBonus.SPD + equipmentBonus.SPD),
	})
	mana := combatcalc.FinalMajorStat(combatcalc.MajorStatModifiers{
		Base:      rawProfile.BaseMANA,
		FlatBonus: int32(combatBonus.MANA + equipmentBonus.MANA),
	})

	hitPct := rawProfile.BaseHitPct + combatBonus.HitPct + equipmentBonus.HitPct
	dodgePct := rawProfile.BaseDodgePct + combatBonus.DodgePct + equipmentBonus.DodgePct

	starter := player.DefaultStarterProfile()
	spiritMax := starter.SpiritMax + equipmentBonus.SpiritMax
	spirit := starter.Spirit + equipmentBonus.Spirit
	if spiritMax > 0 && spirit > spiritMax {
		spirit = spiritMax
	}

	currentHP := rawProfile.HP
	if currentHP > hpMax {
		currentHP = hpMax
	}

	skillIDsJSON, err := json.Marshal(rawProfile.SkillIDs)
	if err != nil {
		return fmt.Errorf("marshal player combat snapshot skill ids: %w", err)
	}

	result := equipment.RecalcResult{
		HPMax:              hpMax,
		ATK:                atk,
		DEF:                def,
		SPD:                spd,
		MANA:               mana,
		HitPct:             hitPct,
		DodgePct:           dodgePct,
		Spirit:             spirit,
		SpiritMax:          spiritMax,
		CritRatePct:        starter.CritRatePct + equipmentBonus.CritRatePct,
		CritDmgPct:         starter.CritDmgPct + equipmentBonus.CritDmgPct,
		PhysicalResistPct:  equipmentBonus.PhysicalResistPct,
		SkillResistPct:     equipmentBonus.SkillResistPct,
		ConfusionResistPct: equipmentBonus.ConfusionResistPct,
		SleepResistPct:     equipmentBonus.SleepResistPct,
		ParalysisResistPct: equipmentBonus.ParalysisResistPct,
		SealResistPct:      equipmentBonus.SealResistPct,
		CurseResistPct:     equipmentBonus.CurseResistPct,
		CritResistPct:      equipmentBonus.CritResistPct,
		CritDmgResistPct:   equipmentBonus.CritDmgResistPct,
		CharacterResistPct: equipmentBonus.CharacterResistPct,
		PetResistPct:       equipmentBonus.PetResistPct,
		SkinID:             skinID,
	}
	return savePlayerCombatSnapshot(ctx, r.db, rawProfile, result, currentHP, skillIDsJSON)
}

// FindPlayerCombatSnapshot 读取单个玩家的最终战斗属性快照。
func (r *PlayerRepository) FindPlayerCombatSnapshot(ctx context.Context, playerID uint64) (*player.Profile, error) {
	profile, err := scanPlayerCombatSnapshot(ctx, r.db, playerID)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// RefreshPlayerPetCombatSnapshots 按玩家全量刷新宠物最终属性快照。
func (r *PetRepository) RefreshPlayerPetCombatSnapshots(ctx context.Context, playerID uint64) error {
	rawPets, err := r.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, deletePlayerPetCombatSnapshotsByPlayerIDQuery, playerID); err != nil {
		return err
	}
	for index := range rawPets {
		current := rawPets[index]
		skillIDsJSON, err := json.Marshal(current.SkillIDs)
		if err != nil {
			return fmt.Errorf("marshal pet combat snapshot skill ids: %w", err)
		}
		innateJSON, err := json.Marshal(current.SkillLoadout.InnateSkillIDs)
		if err != nil {
			return fmt.Errorf("marshal pet combat snapshot innate skill ids: %w", err)
		}
		normalJSON, err := json.Marshal(current.SkillLoadout.NormalSkillIDs)
		if err != nil {
			return fmt.Errorf("marshal pet combat snapshot normal skill ids: %w", err)
		}
		_, err = r.db.ExecContext(
			ctx,
			upsertPlayerPetCombatSnapshotQuery,
			current.PetUID,
			playerID,
			current.PetID,
			current.Level,
			current.Exp,
			current.Quality,
			current.HP,
			current.HPMax,
			current.ATK,
			current.DEF,
			current.SPD,
			current.MANA,
			current.Spirit,
			current.SpiritMax,
			current.HitPct,
			current.DodgePct,
			current.CritRatePct,
			current.CritDmgPct,
			current.PhysicalResistPct,
			current.ReversePhysicalResistPct,
			current.SkillResistPct,
			current.ReverseSkillResistPct,
			current.ConfusionResistPct,
			current.SleepResistPct,
			current.ParalysisResistPct,
			current.SealResistPct,
			current.CurseResistPct,
			current.CritDmgResistPct,
			current.CritResistPct,
			current.CharacterResistPct,
			current.PetResistPct,
			current.Guard,
			current.TalentDmgPct,
			current.TalentReducePct,
			current.ElementAdvPct,
			current.ElementPenaltyPct,
			skillIDsJSON,
			innateJSON,
			normalJSON,
			current.GrantSource,
			current.CaptureMonsterID,
			current.FreeAttrPoints,
			current.AllocHPPoints,
			current.AllocATKPoints,
			current.AllocSPDPoints,
			current.AllocMANAPoints,
			current.AllocDEFPoints,
			current.BaseHPApt,
			current.BaseATKApt,
			current.BaseDEFApt,
			current.BaseSPDApt,
			current.BaseMANAApt,
			current.ExtraHPApt,
			current.ExtraATKApt,
			current.ExtraDEFApt,
			current.ExtraSPDApt,
			current.ExtraMANAApt,
			current.EvolutionLevel,
			current.RebirthLevel,
			current.AptitudeProfile,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListPlayerPetCombatSnapshotsByPlayerID 返回玩家宠物最终属性快照列表。
func (r *PetRepository) ListPlayerPetCombatSnapshotsByPlayerID(ctx context.Context, playerID uint64) ([]pet.Pet, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerPetCombatSnapshotsByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]pet.Pet, 0, 8)
	for rows.Next() {
		item, err := scanPetCombatSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListPlayerLineupCombatSnapshotsByPlayerID 返回玩家编队最终战斗属性快照。
func (r *PetRepository) ListPlayerLineupCombatSnapshotsByPlayerID(ctx context.Context, playerID uint64) ([]pet.LineupPet, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerLineupCombatSnapshotsByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]pet.LineupPet, 0, 4)
	for rows.Next() {
		item, err := scanLineupCombatSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// FindPlayerPetCombatSnapshotByUID 返回单只宠物最终属性快照。
func (r *PetRepository) FindPlayerPetCombatSnapshotByUID(ctx context.Context, playerID uint64, petUID uint64) (*pet.Pet, error) {
	row := r.db.QueryRowContext(ctx, findPlayerPetCombatSnapshotByUIDQuery, playerID, petUID)
	item, err := scanPetCombatSnapshotRowFromScanner(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func scanPlayerCombatSnapshot(ctx context.Context, db DBTX, playerID uint64) (*player.Profile, error) {
	var (
		profile                                                                  player.Profile
		profileID                                                                int64
		level                                                                    int64
		exp                                                                      int64
		freeAttrPoints                                                           int64
		strength, vitality, agility, mind                                        int64
		gold                                                                     int64
		sceneID                                                                  int64
		posX, posY                                                               int64
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

	err := db.QueryRowContext(ctx, findPlayerCombatSnapshotQuery, playerID).Scan(
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
	profile.SkinID = skinID
	profile.BaseHPMax = uint32(baseHPMax)
	profile.BaseATK = uint32(baseATK)
	profile.BaseDEF = uint32(baseDEF)
	profile.BaseSPD = uint32(baseSPD)
	profile.BaseMANA = uint32(baseMANA)
	profile.BaseHitPct = uint32(baseHitPct)
	profile.BaseDodgePct = uint32(baseDodgePct)
	if len(skillIDsJSON) > 0 {
		if err := json.Unmarshal(skillIDsJSON, &profile.SkillIDs); err != nil {
			return nil, fmt.Errorf("unmarshal player combat snapshot skill ids: %w", err)
		}
	}
	return &profile, nil
}

func savePlayerCombatSnapshot(ctx context.Context, db DBTX, rawProfile *player.Profile, result equipment.RecalcResult, currentHP uint32, skillIDsJSON []byte) error {
	if rawProfile == nil {
		return player.ErrPlayerNotFound
	}
	if currentHP > result.HPMax {
		currentHP = result.HPMax
	}
	_, err := db.ExecContext(
		ctx,
		upsertPlayerCombatSnapshotQuery,
		rawProfile.PlayerID,
		currentHP,
		result.HPMax,
		result.ATK,
		result.DEF,
		result.SPD,
		result.MANA,
		rawProfile.Vigor,
		rawProfile.VigorMax,
		result.Spirit,
		result.SpiritMax,
		result.HitPct,
		result.DodgePct,
		result.CritRatePct,
		result.CritDmgPct,
		result.PhysicalResistPct,
		result.SkillResistPct,
		result.ConfusionResistPct,
		result.SleepResistPct,
		result.ParalysisResistPct,
		result.SealResistPct,
		result.CurseResistPct,
		result.CritResistPct,
		result.CritDmgResistPct,
		result.CharacterResistPct,
		result.PetResistPct,
		rawProfile.MercenaryResistPct,
		rawProfile.GenericShieldPct,
		rawProfile.Guard,
		rawProfile.TalentDmgPct,
		rawProfile.TalentReducePct,
		rawProfile.ElementAdvPct,
		rawProfile.ElementPenaltyPct,
		skillIDsJSON,
		result.SkinID,
	)
	return err
}

func savePlayerCombatSnapshotInTx(ctx context.Context, tx *sql.Tx, playerID uint64, result equipment.RecalcResult, currentHP uint32) error {
	rawProfile, err := scanRawPlayerByIDTx(ctx, tx, playerID)
	if err != nil {
		return err
	}
	if rawProfile == nil {
		return player.ErrPlayerNotFound
	}
	skillIDsJSON, err := json.Marshal(rawProfile.SkillIDs)
	if err != nil {
		return fmt.Errorf("marshal player combat snapshot skill ids: %w", err)
	}
	return savePlayerCombatSnapshot(ctx, tx, rawProfile, result, currentHP, skillIDsJSON)
}

func (r *PlayerRepository) loadPlayerAttrConvertRules(ctx context.Context) ([]progression.AttrConvertConfig, error) {
	rows, err := r.db.QueryContext(ctx, listPlayerCombatSnapshotAttrRulesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]progression.AttrConvertConfig, 0, 8)
	for rows.Next() {
		var item progression.AttrConvertConfig
		var convertRate int64
		if err := rows.Scan(&item.SourceAttr, &item.TargetAttr, &convertRate); err != nil {
			return nil, err
		}
		item.ConvertRate = uint32(convertRate)
		item.Status = 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildPlayerCombatBonusFromRules(allocated progression.AllocatedAttrs, rules []progression.AttrConvertConfig) progression.CombatBonus {
	bonus := progression.CombatBonus{}
	for _, rule := range rules {
		if rule.Status == 0 || rule.ConvertRate == 0 {
			continue
		}
		var source uint32
		switch rule.SourceAttr {
		case progression.SourceAttrStrength:
			source = allocated.Strength
		case progression.SourceAttrVitality:
			source = allocated.Vitality
		case progression.SourceAttrAgility:
			source = allocated.Agility
		case progression.SourceAttrMind:
			source = allocated.Mind
		default:
			continue
		}
		if source == 0 {
			continue
		}
		delta := source * rule.ConvertRate
		switch rule.TargetAttr {
		case "hp_max":
			bonus.HPMax += delta
		case "atk":
			bonus.ATK += delta
		case "def":
			bonus.DEF += delta
		case "spd":
			bonus.SPD += delta
		case "mana":
			bonus.MANA += delta
		case "hit_pct":
			bonus.HitPct += delta
		case "dodge_pct":
			bonus.DodgePct += delta
		}
	}
	return bonus
}

func scanPetCombatSnapshotRow(rows *sql.Rows) (pet.Pet, error) {
	return scanPetCombatSnapshotRowFromScanner(rows)
}

func scanPetCombatSnapshotRowFromScanner(scanner interface{ Scan(dest ...any) error }) (pet.Pet, error) {
	var (
		item                                                                                           pet.Pet
		petUID, petID, level, exp, quality                                                             int64
		hp, hpMax, atk, def, spd, mana                                                                 int64
		spirit, spiritMax                                                                              int64
		hitPct, dodgePct, critRatePct, critDmgPct                                                      int64
		physicalResistPct, reversePhysicalResistPct                                                    int64
		skillResistPct, reverseSkillResistPct                                                          int64
		confusionResistPct, sleepResistPct, paralysisResistPct, sealResistPct, curseResistPct          int64
		critDmgResistPct, critResistPct, characterResistPct, petResistPct                              int64
		guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct                         int64
		skillIDsJSON, innateSkillIDsJSON, normalSkillIDsJSON                                           []byte
		captureMonsterID                                                                               int64
		freeAttrPoints, allocHPPoints, allocATKPoints, allocSPDPoints, allocMANAPoints, allocDEFPoints int64
		baseHPApt, baseATKApt, baseDEFApt, baseSPDApt, baseMANAApt                                     int64
		extraHPApt, extraATKApt, extraDEFApt, extraSPDApt, extraMANAApt                                int64
		evolutionLevel, rebirthLevel                                                                   int64
	)

	if err := scanner.Scan(
		&petUID,
		&petID,
		&level,
		&exp,
		&quality,
		&hp,
		&hpMax,
		&atk,
		&def,
		&spd,
		&mana,
		&spirit,
		&spiritMax,
		&hitPct,
		&dodgePct,
		&critRatePct,
		&critDmgPct,
		&physicalResistPct,
		&reversePhysicalResistPct,
		&skillResistPct,
		&reverseSkillResistPct,
		&confusionResistPct,
		&sleepResistPct,
		&paralysisResistPct,
		&sealResistPct,
		&curseResistPct,
		&critDmgResistPct,
		&critResistPct,
		&characterResistPct,
		&petResistPct,
		&guard,
		&talentDmgPct,
		&talentReducePct,
		&elementAdvPct,
		&elementPenaltyPct,
		&skillIDsJSON,
		&innateSkillIDsJSON,
		&normalSkillIDsJSON,
		&item.GrantSource,
		&captureMonsterID,
		&freeAttrPoints,
		&allocHPPoints,
		&allocATKPoints,
		&allocSPDPoints,
		&allocMANAPoints,
		&allocDEFPoints,
		&baseHPApt,
		&baseATKApt,
		&baseDEFApt,
		&baseSPDApt,
		&baseMANAApt,
		&extraHPApt,
		&extraATKApt,
		&extraDEFApt,
		&extraSPDApt,
		&extraMANAApt,
		&evolutionLevel,
		&rebirthLevel,
		&item.AptitudeProfile,
	); err != nil {
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
	item.Spirit = uint32(spirit)
	item.SpiritMax = uint32(spiritMax)
	item.HitPct = uint32(hitPct)
	item.DodgePct = uint32(dodgePct)
	item.CritRatePct = uint32(critRatePct)
	item.CritDmgPct = uint32(critDmgPct)
	item.PhysicalResistPct = uint32(physicalResistPct)
	item.ReversePhysicalResistPct = uint32(reversePhysicalResistPct)
	item.SkillResistPct = uint32(skillResistPct)
	item.ReverseSkillResistPct = uint32(reverseSkillResistPct)
	item.ConfusionResistPct = uint32(confusionResistPct)
	item.SleepResistPct = uint32(sleepResistPct)
	item.ParalysisResistPct = uint32(paralysisResistPct)
	item.SealResistPct = uint32(sealResistPct)
	item.CurseResistPct = uint32(curseResistPct)
	item.CritDmgResistPct = uint32(critDmgResistPct)
	item.CritResistPct = uint32(critResistPct)
	item.CharacterResistPct = uint32(characterResistPct)
	item.PetResistPct = uint32(petResistPct)
	item.Guard = uint32(guard)
	item.TalentDmgPct = uint32(talentDmgPct)
	item.TalentReducePct = uint32(talentReducePct)
	item.ElementAdvPct = uint32(elementAdvPct)
	item.ElementPenaltyPct = uint32(elementPenaltyPct)
	item.CaptureMonsterID = uint32(captureMonsterID)
	item.FreeAttrPoints = uint32(freeAttrPoints)
	item.AllocHPPoints = uint32(allocHPPoints)
	item.AllocATKPoints = uint32(allocATKPoints)
	item.AllocSPDPoints = uint32(allocSPDPoints)
	item.AllocMANAPoints = uint32(allocMANAPoints)
	item.AllocDEFPoints = uint32(allocDEFPoints)
	item.BaseHPApt = uint32(baseHPApt)
	item.BaseATKApt = uint32(baseATKApt)
	item.BaseDEFApt = uint32(baseDEFApt)
	item.BaseSPDApt = uint32(baseSPDApt)
	item.BaseMANAApt = uint32(baseMANAApt)
	item.ExtraHPApt = uint32(extraHPApt)
	item.ExtraATKApt = uint32(extraATKApt)
	item.ExtraDEFApt = uint32(extraDEFApt)
	item.ExtraSPDApt = uint32(extraSPDApt)
	item.ExtraMANAApt = uint32(extraMANAApt)
	item.EvolutionLevel = uint32(evolutionLevel)
	item.RebirthLevel = uint32(rebirthLevel)
	item.GrowthAptitudes = pet.GrowthAptitudes{
		HPApt:   item.BaseHPApt + item.ExtraHPApt,
		ATKApt:  item.BaseATKApt + item.ExtraATKApt,
		DEFApt:  item.BaseDEFApt + item.ExtraDEFApt,
		SPDApt:  item.BaseSPDApt + item.ExtraSPDApt,
		MANAApt: item.BaseMANAApt + item.ExtraMANAApt,
	}

	if len(skillIDsJSON) > 0 {
		ids, err := decodeFlexibleSkillIDJSONArray(skillIDsJSON)
		if err != nil {
			return pet.Pet{}, fmt.Errorf("unmarshal pet combat snapshot skill ids: %w", err)
		}
		item.SkillIDs = ids
	}
	if len(innateSkillIDsJSON) > 0 {
		ids, err := decodeFlexibleSkillIDJSONArray(innateSkillIDsJSON)
		if err != nil {
			return pet.Pet{}, fmt.Errorf("unmarshal pet combat snapshot innate skill ids: %w", err)
		}
		item.SkillLoadout.InnateSkillIDs = ids
	}
	if len(normalSkillIDsJSON) > 0 {
		ids, err := decodeFlexibleSkillIDJSONArray(normalSkillIDsJSON)
		if err != nil {
			return pet.Pet{}, fmt.Errorf("unmarshal pet combat snapshot normal skill ids: %w", err)
		}
		item.SkillLoadout.NormalSkillIDs = ids
	}
	return item, nil
}

func scanLineupCombatSnapshotRow(rows *sql.Rows) (pet.LineupPet, error) {
	var (
		item                                                                                  pet.LineupPet
		petUID, petID, level                                                                  int64
		hp, hpMax, atk, def, spd, mana                                                        int64
		spirit, spiritMax                                                                     int64
		hitPct, dodgePct, critRatePct, critDmgPct                                             int64
		physicalResistPct, reversePhysicalResistPct                                           int64
		skillResistPct, reverseSkillResistPct                                                 int64
		confusionResistPct, sleepResistPct, paralysisResistPct, sealResistPct, curseResistPct int64
		critDmgResistPct, critResistPct, characterResistPct, petResistPct                     int64
		guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct                int64
		skillIDsJSON                                                                          []byte
	)
	if err := rows.Scan(
		&petUID,
		&petID,
		&level,
		&hp,
		&hpMax,
		&atk,
		&def,
		&spd,
		&mana,
		&spirit,
		&spiritMax,
		&hitPct,
		&dodgePct,
		&critRatePct,
		&critDmgPct,
		&physicalResistPct,
		&reversePhysicalResistPct,
		&skillResistPct,
		&reverseSkillResistPct,
		&confusionResistPct,
		&sleepResistPct,
		&paralysisResistPct,
		&sealResistPct,
		&curseResistPct,
		&critDmgResistPct,
		&critResistPct,
		&characterResistPct,
		&petResistPct,
		&guard,
		&talentDmgPct,
		&talentReducePct,
		&elementAdvPct,
		&elementPenaltyPct,
		&skillIDsJSON,
	); err != nil {
		return pet.LineupPet{}, err
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
	item.Spirit = uint32(spirit)
	item.SpiritMax = uint32(spiritMax)
	item.HitPct = uint32(hitPct)
	item.DodgePct = uint32(dodgePct)
	item.CritRatePct = uint32(critRatePct)
	item.CritDmgPct = uint32(critDmgPct)
	item.PhysicalResistPct = uint32(physicalResistPct)
	item.ReversePhysicalResistPct = uint32(reversePhysicalResistPct)
	item.SkillResistPct = uint32(skillResistPct)
	item.ReverseSkillResistPct = uint32(reverseSkillResistPct)
	item.ConfusionResistPct = uint32(confusionResistPct)
	item.SleepResistPct = uint32(sleepResistPct)
	item.ParalysisResistPct = uint32(paralysisResistPct)
	item.SealResistPct = uint32(sealResistPct)
	item.CurseResistPct = uint32(curseResistPct)
	item.CritDmgResistPct = uint32(critDmgResistPct)
	item.CritResistPct = uint32(critResistPct)
	item.CharacterResistPct = uint32(characterResistPct)
	item.PetResistPct = uint32(petResistPct)
	item.Guard = uint32(guard)
	item.TalentDmgPct = uint32(talentDmgPct)
	item.TalentReducePct = uint32(talentReducePct)
	item.ElementAdvPct = uint32(elementAdvPct)
	item.ElementPenaltyPct = uint32(elementPenaltyPct)
	if len(skillIDsJSON) > 0 {
		ids, err := decodeFlexibleSkillIDJSONArray(skillIDsJSON)
		if err != nil {
			return pet.LineupPet{}, fmt.Errorf("unmarshal lineup combat snapshot skill ids: %w", err)
		}
		item.SkillIDs = ids
	}
	return item, nil
}
