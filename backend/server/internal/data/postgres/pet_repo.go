package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"pocket-pet-remake/server/internal/module/pet"
	"pocket-pet-remake/server/internal/module/petprogression"
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
  pp.spirit,
  pp.spirit_max,
  pp.hit_pct,
  pp.dodge_pct,
  pp.crit_rate_pct,
  pp.crit_dmg_pct,
  pp.physical_resist_pct,
  pp.reverse_physical_resist_pct,
  pp.skill_resist_pct,
  pp.reverse_skill_resist_pct,
  pp.confusion_resist_pct,
  pp.sleep_resist_pct,
  pp.paralysis_resist_pct,
  pp.seal_resist_pct,
  pp.curse_resist_pct,
  pp.crit_dmg_resist_pct,
  pp.crit_resist_pct,
  pp.character_resist_pct,
  pp.pet_resist_pct,
  pp.guard,
  pp.talent_dmg_pct,
  pp.talent_reduce_pct,
  pp.element_adv_pct,
  pp.element_penalty_pct,
  pp.skill_ids,
  pp.innate_skill_ids,
  pp.normal_skill_ids,
  pp.active_talisman_skill_id,
  pp.talisman_hero_skill_id,
  pp.talisman_slot_1_skill_id,
  pp.talisman_slot_2_skill_id,
  pp.talisman_slot_3_skill_id,
  pp.active_talisman_enabled,
  pp.talisman_hero_enabled,
  pp.talisman_slot_1_enabled,
  pp.talisman_slot_2_enabled,
  pp.talisman_slot_3_enabled
FROM player_lineup pl
JOIN player_pet pp ON pp.id = pl.pet_uid
WHERE pl.player_id = $1
ORDER BY pl.slot_index ASC
`

const listPetsByPlayerIDQuery = `
SELECT
  pp.id,
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
  pp.spirit,
  pp.spirit_max,
  pp.hit_pct,
  pp.dodge_pct,
  pp.crit_rate_pct,
  pp.crit_dmg_pct,
  pp.physical_resist_pct,
  pp.reverse_physical_resist_pct,
  pp.skill_resist_pct,
  pp.reverse_skill_resist_pct,
  pp.confusion_resist_pct,
  pp.sleep_resist_pct,
  pp.paralysis_resist_pct,
  pp.seal_resist_pct,
  pp.curse_resist_pct,
  pp.crit_dmg_resist_pct,
  pp.crit_resist_pct,
  pp.character_resist_pct,
  pp.pet_resist_pct,
  pp.guard,
  pp.talent_dmg_pct,
  pp.talent_reduce_pct,
  pp.element_adv_pct,
  pp.element_penalty_pct,
  pp.skill_ids,
  pp.innate_skill_ids,
  pp.normal_skill_ids,
  pp.active_talisman_skill_id,
  pp.talisman_hero_skill_id,
  pp.talisman_slot_1_skill_id,
  pp.talisman_slot_2_skill_id,
  pp.talisman_slot_3_skill_id,
  pp.active_talisman_enabled,
  pp.talisman_hero_enabled,
  pp.talisman_slot_1_enabled,
  pp.talisman_slot_2_enabled,
  pp.talisman_slot_3_enabled,
  pp.hp_apt,
  pp.atk_apt,
  pp.def_apt,
  pp.spd_apt,
  pp.mana_apt,
  pp.grant_source,
  pp.capture_monster_id,
  pp.free_attr_points,
  pp.alloc_hp_points,
  pp.alloc_atk_points,
  pp.alloc_spd_points,
  pp.alloc_mana_points,
  pp.alloc_def_points,
  pp.base_hp_apt,
  pp.base_atk_apt,
  pp.base_def_apt,
  pp.base_spd_apt,
  pp.base_mana_apt,
  pp.extra_hp_apt,
  pp.extra_atk_apt,
  pp.extra_def_apt,
  pp.extra_spd_apt,
  pp.extra_mana_apt,
  pp.evolution_level,
  pp.rebirth_level,
  COALESCE(pd.aptitude_profile, 'normal') AS aptitude_profile
FROM player_pet pp
LEFT JOIN pet_definition pd ON pd.pet_id = pp.pet_id
WHERE pp.player_id = $1
ORDER BY pp.id ASC
`

const listArtifactEquipmentByPlayerIDQuery = `
SELECT pet_uid, slot_index, skill_id
FROM pet_artifact_equipment
WHERE player_id = $1
ORDER BY pet_uid ASC, slot_index ASC
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

const runtimePetDefinitionQuery = `
SELECT
  pet_id,
  level,
  quality,
  hp,
  hp_max,
  atk,
  def,
  spd,
  mana,
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
  acquire_method,
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
  COALESCE(aptitude_profile, 'normal') AS aptitude_profile
FROM pet_definition
WHERE pet_id = $1
  AND status = 1
LIMIT 1
`

const insertRuntimePetQuery = `
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
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
  hp_apt,
  atk_apt,
  def_apt,
  spd_apt,
  mana_apt,
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
  grant_source,
  capture_monster_id
) VALUES (
  $1, $2, $3, 0, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
  $14, $15, $16, $17, $18,
  $19, $20, $21, $22, $23,
  $24, $25, $26, $27, $28,
  $29, $30
)
RETURNING id
`

const adminPetListBaseQuery = `
SELECT
  pp.id,
  pp.player_id,
  p.name,
  pp.pet_id,
  COALESCE(pd.pet_name, ''),
  COALESCE(pp.custom_name, ''),
  pp.level,
  pp.quality,
  pp.hp,
  pp.hp_max,
  pp.atk,
  pp.def,
  pp.spd,
  pp.mana,
  pp.skill_ids,
  EXISTS(SELECT 1 FROM player_lineup pl WHERE pl.player_id = pp.player_id AND pl.pet_uid = pp.id) AS in_lineup,
  pp.updated_at,
  pp.created_at
FROM player_pet pp
JOIN player p ON p.id = pp.player_id
LEFT JOIN pet_definition pd ON pd.pet_id = pp.pet_id
`

const adminPetDetailQuery = `
SELECT
  pp.id,
  pp.player_id,
  p.name,
  pp.pet_id,
  COALESCE(pd.pet_name, ''),
  COALESCE(pp.custom_name, ''),
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
  pp.innate_skill_ids,
  pp.normal_skill_ids,
  pp.spirit,
  pp.spirit_max,
  pp.hit_pct,
  pp.dodge_pct,
  pp.crit_rate_pct,
  pp.crit_dmg_pct,
  pp.physical_resist_pct,
  pp.reverse_physical_resist_pct,
  pp.skill_resist_pct,
  pp.reverse_skill_resist_pct,
  pp.confusion_resist_pct,
  pp.sleep_resist_pct,
  pp.paralysis_resist_pct,
  pp.seal_resist_pct,
  pp.curse_resist_pct,
  pp.crit_dmg_resist_pct,
  pp.crit_resist_pct,
  pp.character_resist_pct,
  pp.pet_resist_pct,
  pp.guard,
  pp.talent_dmg_pct,
  pp.talent_reduce_pct,
  pp.element_adv_pct,
  pp.element_penalty_pct,
  EXISTS(SELECT 1 FROM player_lineup pl WHERE pl.player_id = pp.player_id AND pl.pet_uid = pp.id) AS in_lineup,
  pp.created_at,
  pp.updated_at
FROM player_pet pp
JOIN player p ON p.id = pp.player_id
LEFT JOIN pet_definition pd ON pd.pet_id = pp.pet_id
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
  skill_ids,
  innate_skill_ids,
  normal_skill_ids,
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
  element_penalty_pct
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
  $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38
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
    skill_ids = $12,
    innate_skill_ids = $13,
    normal_skill_ids = $14,
    spirit = $15,
    spirit_max = $16,
    hit_pct = $17,
    dodge_pct = $18,
    crit_rate_pct = $19,
    crit_dmg_pct = $20,
    physical_resist_pct = $21,
    reverse_physical_resist_pct = $22,
    skill_resist_pct = $23,
    reverse_skill_resist_pct = $24,
    confusion_resist_pct = $25,
    sleep_resist_pct = $26,
    paralysis_resist_pct = $27,
    seal_resist_pct = $28,
    curse_resist_pct = $29,
    crit_dmg_resist_pct = $30,
    crit_resist_pct = $31,
    character_resist_pct = $32,
    pet_resist_pct = $33,
    guard = $34,
    talent_dmg_pct = $35,
    talent_reduce_pct = $36,
    element_adv_pct = $37,
    element_penalty_pct = $38,
    custom_name = $39
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
	caps, err := r.LoadCombatStatCaps(ctx)
	if err != nil {
		return nil, err
	}
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
		pet.ClampPetCombatStats(&item, caps)
		pets = append(pets, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	artifactMap, err := r.loadArtifactEquipmentByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	for index := range pets {
		applyArtifactEquipment(&pets[index], artifactMap[pets[index].PetUID])
		pet.ResolvePetBattleSkills(&pets[index])
	}
	return pets, nil
}

func (r *PetRepository) ListLineupByPlayerID(ctx context.Context, playerID uint64) ([]pet.LineupPet, error) {
	caps, err := r.LoadCombatStatCaps(ctx)
	if err != nil {
		return nil, err
	}
	artifactMap, err := r.loadArtifactEquipmentByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, listLineupByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lineup := make([]pet.LineupPet, 0)
	for rows.Next() {
		var (
			item                     pet.LineupPet
			loadout                  pet.SkillLoadout
			petUID                   int64
			petID                    int64
			level                    int64
			hp                       int64
			hpMax                    int64
			atk                      int64
			def                      int64
			spd                      int64
			mana                     int64
			spirit                   int64
			spiritMax                int64
			hitPct                   int64
			dodgePct                 int64
			critRatePct              int64
			critDmgPct               int64
			physicalResistPct        int64
			reversePhysicalResistPct int64
			skillResistPct           int64
			reverseSkillResistPct    int64
			confusionResistPct       int64
			sleepResistPct           int64
			paralysisResistPct       int64
			sealResistPct            int64
			curseResistPct           int64
			critDmgResistPct         int64
			critResistPct            int64
			characterResistPct       int64
			petResistPct             int64
			guard                    int64
			talentDmgPct             int64
			talentReducePct          int64
			elementAdvPct            int64
			elementPenaltyPct        int64
			skillIDsJSON             []byte
			innateSkillIDsJSON       []byte
			normalSkillIDsJSON       []byte
			activeTalismanSkillID    int64
			talismanHeroSkillID      int64
			talismanSlot1SkillID     int64
			talismanSlot2SkillID     int64
			talismanSlot3SkillID     int64
			activeTalismanEnabled    bool
			talismanHeroEnabled      bool
			talismanSlot1Enabled     bool
			talismanSlot2Enabled     bool
			talismanSlot3Enabled     bool
		)
		if err := rows.Scan(
			&petUID, &petID, &level, &hp, &hpMax, &atk, &def, &spd, &mana,
			&spirit, &spiritMax, &hitPct, &dodgePct, &critRatePct, &critDmgPct,
			&physicalResistPct, &reversePhysicalResistPct, &skillResistPct, &reverseSkillResistPct,
			&confusionResistPct, &sleepResistPct, &paralysisResistPct, &sealResistPct, &curseResistPct,
			&critDmgResistPct, &critResistPct, &characterResistPct, &petResistPct,
			&guard, &talentDmgPct, &talentReducePct, &elementAdvPct, &elementPenaltyPct,
			&skillIDsJSON,
			&innateSkillIDsJSON, &normalSkillIDsJSON,
			&activeTalismanSkillID, &talismanHeroSkillID,
			&talismanSlot1SkillID, &talismanSlot2SkillID, &talismanSlot3SkillID,
			&activeTalismanEnabled, &talismanHeroEnabled,
			&talismanSlot1Enabled, &talismanSlot2Enabled, &talismanSlot3Enabled,
		); err != nil {
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
		loadout = decodeSkillLoadout(
			innateSkillIDsJSON,
			normalSkillIDsJSON,
			uint32(activeTalismanSkillID),
			uint32(talismanHeroSkillID),
			uint32(talismanSlot1SkillID),
			uint32(talismanSlot2SkillID),
			uint32(talismanSlot3SkillID),
			activeTalismanEnabled,
			talismanHeroEnabled,
			talismanSlot1Enabled,
			talismanSlot2Enabled,
			talismanSlot3Enabled,
		)
		if len(skillIDsJSON) > 0 {
			item.SkillIDs, err = decodeFlexibleSkillIDJSONArray(skillIDsJSON)
			if err != nil {
				return nil, fmt.Errorf("unmarshal lineup pet skill ids: %w", err)
			}
		}
		pet.ApplyLegacySkillIDs(&loadout, item.SkillIDs)
		applyArtifactSkillsToLoadout(&loadout, artifactMap[item.PetUID])
		item.SkillIDs = pet.MergeBattleSkillIDs(loadout, item.SkillIDs)
		if len(item.SkillIDs) == 0 && len(skillIDsJSON) > 0 {
			item.SkillIDs, err = decodeFlexibleSkillIDJSONArray(skillIDsJSON)
			if err != nil {
				return nil, fmt.Errorf("unmarshal lineup pet legacy skill ids: %w", err)
			}
		}
		pet.ClampLineupPetCombatStats(&item, caps)
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

func (r *PetRepository) FindPetByUID(ctx context.Context, playerID uint64, petUID uint64) (pet.Pet, error) {
	item, err := r.loadPetByUID(ctx, playerID, petUID)
	if err != nil {
		return pet.Pet{}, err
	}
	if item == nil {
		return pet.Pet{}, pet.ErrPetNotFound
	}
	return *item, nil
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

func (r *PetRepository) GrantRuntimePet(ctx context.Context, playerID uint64, petID uint32, reasonType string, reasonRefID uint64, operatorType string, operatorID uint64) (*pet.RuntimeGrantResult, error) {
	definition, err := r.loadRuntimePetDefinition(ctx, petID)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, pet.ErrPetNotFound
	}
	aptitudes := pet.GrowthAptitudes{
		HPApt:   definition.HPApt,
		ATKApt:  definition.ATKApt,
		DEFApt:  definition.DEFApt,
		SPDApt:  definition.SPDApt,
		MANAApt: definition.MANAApt,
	}
	splitAptitudes := buildBaseExtraAptitudes(definition, aptitudes)
	return r.insertRuntimePet(ctx, playerID, definition, aptitudes, splitAptitudes, pet.GrantSourceTemplate, 0, reasonType, reasonRefID, operatorType, operatorID)
}

func (r *PetRepository) GrantWildCapturePet(ctx context.Context, playerID uint64, petID uint32, captureMonsterID uint32, reasonType string, reasonRefID uint64) (*pet.RuntimeGrantResult, error) {
	definition, err := r.loadRuntimePetDefinition(ctx, petID)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, pet.ErrPetNotFound
	}
	if !pet.IsWildCaptureAcquireMethod(definition.AcquireMethod) {
		return nil, pet.ErrInvalidWildCapturePetTemplate
	}
	rollRanges := pet.AptitudeRollRanges{
		HPAptMin: definition.HPAptRollMin, HPAptMax: definition.HPAptRollMax,
		ATKAptMin: definition.ATKAptRollMin, ATKAptMax: definition.ATKAptRollMax,
		DEFAptMin: definition.DEFAptRollMin, DEFAptMax: definition.DEFAptRollMax,
		SPDAptMin: definition.SPDAptRollMin, SPDAptMax: definition.SPDAptRollMax,
		MANAAptMin: definition.MANAAptRollMin, MANAAptMax: definition.MANAAptRollMax,
	}
	if err := pet.ValidateAptitudeRollRanges(rollRanges); err != nil {
		return nil, err
	}
	aptitudes := pet.RollWildCaptureAptitudes(rollRanges, nil)
	splitAptitudes := buildBaseExtraAptitudes(definition, aptitudes)
	return r.insertRuntimePet(ctx, playerID, definition, aptitudes, splitAptitudes, pet.GrantSourceWildCapture, captureMonsterID, reasonType, reasonRefID, "", 0)
}

func (r *PetRepository) insertRuntimePet(
	ctx context.Context,
	playerID uint64,
	definition *runtimePetDefinitionRow,
	aptitudes pet.GrowthAptitudes,
	splitAptitudes petprogression.GrowthAptitudes,
	grantSource string,
	captureMonsterID uint32,
	reasonType string,
	reasonRefID uint64,
	operatorType string,
	operatorID uint64,
) (*pet.RuntimeGrantResult, error) {
	level := definition.Level
	if level == 0 {
		level = 1
	}
	combat := petprogression.RecalculateCombatStats(petprogression.ProgressionState{
		Level:           level,
		AptitudeProfile: definition.AptitudeProfile,
		Aptitudes:       splitAptitudes,
	}, petprogression.DefaultConvertRates())
	combat = resolveGrantedPetCombatStats(combat, definition)
	hp := combat.HPMax
	if definition.HP > 0 && definition.HP <= combat.HPMax {
		hp = definition.HP
	} else if hp == 0 && combat.HPMax > 0 {
		hp = combat.HPMax
	}
	innateSkillIDs := decodeSkillIDJSONArray(definition.InnateSkillIDsJSON)
	normalSkillIDs := decodeSkillIDJSONArray(definition.NormalSkillIDsJSON)
	if len(normalSkillIDs) == 0 {
		normalSkillIDs = decodeSkillIDJSONArray(definition.SkillIDsJSON)
		if len(normalSkillIDs) > pet.MaxNormalSkillSlots {
			normalSkillIDs = normalSkillIDs[:pet.MaxNormalSkillSlots]
		}
	}
	loadout := pet.SkillLoadoutFromDefinition(innateSkillIDs, normalSkillIDs)
	battleSkillIDsJSON, err := json.Marshal(pet.BuildBattleSkillIDs(loadout))
	if err != nil {
		return nil, fmt.Errorf("marshal granted pet battle skill ids: %w", err)
	}
	innateSkillIDsJSON, err := json.Marshal(loadout.InnateSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal granted pet innate skill ids: %w", err)
	}
	normalSkillIDsJSON, err := json.Marshal(loadout.NormalSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal granted pet normal skill ids: %w", err)
	}
	var petUID uint64
	if err := r.db.QueryRowContext(
		ctx,
		insertRuntimePetQuery,
		playerID,
		definition.PetID,
		level,
		definition.Quality,
		hp,
		combat.HPMax,
		combat.ATK,
		combat.DEF,
		combat.SPD,
		combat.MANA,
		battleSkillIDsJSON,
		innateSkillIDsJSON,
		normalSkillIDsJSON,
		aptitudes.HPApt,
		aptitudes.ATKApt,
		aptitudes.DEFApt,
		aptitudes.SPDApt,
		aptitudes.MANAApt,
		splitAptitudes.BaseHPApt,
		splitAptitudes.BaseATKApt,
		splitAptitudes.BaseDEFApt,
		splitAptitudes.BaseSPDApt,
		splitAptitudes.BaseMANAApt,
		splitAptitudes.ExtraHPApt,
		splitAptitudes.ExtraATKApt,
		splitAptitudes.ExtraDEFApt,
		splitAptitudes.ExtraSPDApt,
		splitAptitudes.ExtraMANAApt,
		grantSource,
		captureMonsterID,
	).Scan(&petUID); err != nil {
		return nil, err
	}
	grantedPet, err := r.loadPetByUID(ctx, playerID, petUID)
	if err != nil {
		return nil, err
	}
	if grantedPet == nil {
		return nil, pet.ErrPetNotFound
	}
	_ = reasonType
	_ = reasonRefID
	_ = operatorType
	_ = operatorID
	return &pet.RuntimeGrantResult{Pet: *grantedPet}, nil
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
		var skillIDsJSON []byte
		if err := rows.Scan(&petUID, &playerID, &item.PlayerName, &petID, &item.PetName, &item.CustomName, &level, &quality, &hp, &hpMax, &atk, &def, &spd, &mana, &skillIDsJSON, &item.InLineup, &item.UpdatedAt, &item.CreatedAt); err != nil {
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
		if len(skillIDsJSON) > 0 {
			item.SkillIDs, err = decodeFlexibleSkillIDJSONArray(skillIDsJSON)
			if err != nil {
				return nil, fmt.Errorf("unmarshal admin pet list skill ids: %w", err)
			}
		}
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
	var spirit, spiritMax, hitPct, dodgePct, critRatePct, critDmgPct int64
	var physicalResistPct, reversePhysicalResistPct, skillResistPct, reverseSkillResistPct int64
	var confusionResistPct, sleepResistPct, paralysisResistPct, sealResistPct, curseResistPct int64
	var critDmgResistPct, critResistPct, characterResistPct, petResistPct int64
	var guard, talentDmgPct, talentReducePct, elementAdvPct, elementPenaltyPct int64
	var skillIDsJSON, innateSkillIDsJSON, normalSkillIDsJSON []byte
	err := r.db.QueryRowContext(ctx, adminPetDetailQuery, petUID).Scan(
		&uid, &playerID, &detail.PlayerName, &petID, &detail.PetName, &detail.CustomName, &level, &exp, &quality,
		&hp, &hpMax, &atk, &def, &spd, &mana, &skillIDsJSON, &innateSkillIDsJSON, &normalSkillIDsJSON,
		&spirit, &spiritMax, &hitPct, &dodgePct, &critRatePct, &critDmgPct,
		&physicalResistPct, &reversePhysicalResistPct, &skillResistPct, &reverseSkillResistPct,
		&confusionResistPct, &sleepResistPct, &paralysisResistPct, &sealResistPct, &curseResistPct,
		&critDmgResistPct, &critResistPct, &characterResistPct, &petResistPct,
		&guard, &talentDmgPct, &talentReducePct, &elementAdvPct, &elementPenaltyPct,
		&detail.InLineup, &detail.CreatedAt, &detail.UpdatedAt,
	)
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
	detail.AdminPetCombatStats = pet.AdminPetCombatStats{
		Spirit:                   uint32(spirit),
		SpiritMax:                uint32(spiritMax),
		HitPct:                   uint32(hitPct),
		DodgePct:                 uint32(dodgePct),
		CritRatePct:              uint32(critRatePct),
		CritDmgPct:               uint32(critDmgPct),
		PhysicalResistPct:        uint32(physicalResistPct),
		ReversePhysicalResistPct: uint32(reversePhysicalResistPct),
		SkillResistPct:           uint32(skillResistPct),
		ReverseSkillResistPct:    uint32(reverseSkillResistPct),
		ConfusionResistPct:       uint32(confusionResistPct),
		SleepResistPct:           uint32(sleepResistPct),
		ParalysisResistPct:       uint32(paralysisResistPct),
		SealResistPct:            uint32(sealResistPct),
		CurseResistPct:           uint32(curseResistPct),
		CritDmgResistPct:         uint32(critDmgResistPct),
		CritResistPct:            uint32(critResistPct),
		CharacterResistPct:       uint32(characterResistPct),
		PetResistPct:             uint32(petResistPct),
		Guard:                    uint32(guard),
		TalentDmgPct:             uint32(talentDmgPct),
		TalentReducePct:          uint32(talentReducePct),
		ElementAdvPct:            uint32(elementAdvPct),
		ElementPenaltyPct:        uint32(elementPenaltyPct),
	}
	if len(skillIDsJSON) > 0 {
		detail.SkillIDs, err = decodeFlexibleSkillIDJSONArray(skillIDsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal admin pet skill ids: %w", err)
		}
	}
	if len(innateSkillIDsJSON) > 0 {
		detail.InnateSkillIDs, err = decodeFlexibleSkillIDJSONArray(innateSkillIDsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal admin pet innate skill ids: %w", err)
		}
	}
	if len(normalSkillIDsJSON) > 0 {
		detail.NormalSkillIDs, err = decodeFlexibleSkillIDJSONArray(normalSkillIDsJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal admin pet normal skill ids: %w", err)
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
	innateSkillIDsJSON, err := json.Marshal(input.InnateSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet innate skill ids: %w", err)
	}
	normalSkillIDsJSON, err := json.Marshal(input.NormalSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet normal skill ids: %w", err)
	}
	var petUID int64
	if err := r.db.QueryRowContext(
		ctx,
		insertAdminPetQuery,
		input.PlayerID,
		input.PetID,
		input.Level,
		input.Exp,
		input.Quality,
		input.HP,
		input.HPMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		skillIDsJSON,
		innateSkillIDsJSON,
		normalSkillIDsJSON,
		input.Spirit,
		input.SpiritMax,
		input.HitPct,
		input.DodgePct,
		input.CritRatePct,
		input.CritDmgPct,
		input.PhysicalResistPct,
		input.ReversePhysicalResistPct,
		input.SkillResistPct,
		input.ReverseSkillResistPct,
		input.ConfusionResistPct,
		input.SleepResistPct,
		input.ParalysisResistPct,
		input.SealResistPct,
		input.CurseResistPct,
		input.CritDmgResistPct,
		input.CritResistPct,
		input.CharacterResistPct,
		input.PetResistPct,
		input.Guard,
		input.TalentDmgPct,
		input.TalentReducePct,
		input.ElementAdvPct,
		input.ElementPenaltyPct,
	).Scan(&petUID); err != nil {
		return nil, err
	}
	return r.FindAdminDetailByPetUID(ctx, uint64(petUID))
}

func (r *PetRepository) UpdateForAdmin(ctx context.Context, petUID uint64, input pet.AdminUpdatePetInput) (*pet.AdminPetDetail, error) {
	skillIDsJSON, err := json.Marshal(input.SkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet skill ids: %w", err)
	}
	innateSkillIDsJSON, err := json.Marshal(input.InnateSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet innate skill ids: %w", err)
	}
	normalSkillIDsJSON, err := json.Marshal(input.NormalSkillIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal admin pet normal skill ids: %w", err)
	}
	result, err := r.db.ExecContext(
		ctx,
		updateAdminPetQuery,
		petUID,
		input.PetID,
		input.Level,
		input.Exp,
		input.Quality,
		input.HP,
		input.HPMax,
		input.ATK,
		input.DEF,
		input.SPD,
		input.MANA,
		skillIDsJSON,
		innateSkillIDsJSON,
		normalSkillIDsJSON,
		input.Spirit,
		input.SpiritMax,
		input.HitPct,
		input.DodgePct,
		input.CritRatePct,
		input.CritDmgPct,
		input.PhysicalResistPct,
		input.ReversePhysicalResistPct,
		input.SkillResistPct,
		input.ReverseSkillResistPct,
		input.ConfusionResistPct,
		input.SleepResistPct,
		input.ParalysisResistPct,
		input.SealResistPct,
		input.CurseResistPct,
		input.CritDmgResistPct,
		input.CritResistPct,
		input.CharacterResistPct,
		input.PetResistPct,
		input.Guard,
		input.TalentDmgPct,
		input.TalentReducePct,
		input.ElementAdvPct,
		input.ElementPenaltyPct,
		input.CustomName,
	)
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

type runtimePetDefinitionRow struct {
	PetID              uint32
	Level              uint32
	Quality            uint32
	HP                 uint32
	HPMax              uint32
	ATK                uint32
	DEF                uint32
	SPD                uint32
	MANA               uint32
	SkillIDsJSON       []byte
	InnateSkillIDsJSON []byte
	NormalSkillIDsJSON []byte
	AcquireMethod      string
	HPApt              uint32
	ATKApt             uint32
	DEFApt             uint32
	SPDApt             uint32
	MANAApt            uint32
	HPAptRollMin       uint32
	HPAptRollMax       uint32
	ATKAptRollMin      uint32
	ATKAptRollMax      uint32
	DEFAptRollMin      uint32
	DEFAptRollMax      uint32
	SPDAptRollMin      uint32
	SPDAptRollMax      uint32
	MANAAptRollMin     uint32
	MANAAptRollMax     uint32
	AptitudeProfile    string
}

func (r *PetRepository) loadRuntimePetDefinition(ctx context.Context, petID uint32) (*runtimePetDefinitionRow, error) {
	var value runtimePetDefinitionRow
	if err := r.db.QueryRowContext(ctx, runtimePetDefinitionQuery, petID).Scan(
		&value.PetID,
		&value.Level,
		&value.Quality,
		&value.HP,
		&value.HPMax,
		&value.ATK,
		&value.DEF,
		&value.SPD,
		&value.MANA,
		&value.SkillIDsJSON,
		&value.InnateSkillIDsJSON,
		&value.NormalSkillIDsJSON,
		&value.AcquireMethod,
		&value.HPApt,
		&value.ATKApt,
		&value.DEFApt,
		&value.SPDApt,
		&value.MANAApt,
		&value.HPAptRollMin,
		&value.HPAptRollMax,
		&value.ATKAptRollMin,
		&value.ATKAptRollMax,
		&value.DEFAptRollMin,
		&value.DEFAptRollMax,
		&value.SPDAptRollMin,
		&value.SPDAptRollMax,
		&value.MANAAptRollMin,
		&value.MANAAptRollMax,
		&value.AptitudeProfile,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &value, nil
}

func (r *PetRepository) loadPetByUID(ctx context.Context, playerID uint64, petUID uint64) (*pet.Pet, error) {
	pets, err := r.ListPetsByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	lineup, err := r.ListLineupByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	lineupSet := make(map[uint64]struct{}, len(lineup))
	for _, current := range lineup {
		lineupSet[current.PetUID] = struct{}{}
	}
	for _, current := range pets {
		if current.PetUID != petUID {
			continue
		}
		_, current.InLineup = lineupSet[current.PetUID]
		copyValue := current
		return &copyValue, nil
	}
	return nil, nil
}

func scanPetRow(rows *sql.Rows) (pet.Pet, error) {
	var (
		item                     pet.Pet
		petUID                   int64
		petID                    int64
		level                    int64
		exp                      int64
		quality                  int64
		hp                       int64
		hpMax                    int64
		atk                      int64
		def                      int64
		spd                      int64
		mana                     int64
		spirit                   int64
		spiritMax                int64
		hitPct                   int64
		dodgePct                 int64
		critRatePct              int64
		critDmgPct               int64
		physicalResistPct        int64
		reversePhysicalResistPct int64
		skillResistPct           int64
		reverseSkillResistPct    int64
		confusionResistPct       int64
		sleepResistPct           int64
		paralysisResistPct       int64
		sealResistPct            int64
		curseResistPct           int64
		critDmgResistPct         int64
		critResistPct            int64
		characterResistPct       int64
		petResistPct             int64
		guard                    int64
		talentDmgPct             int64
		talentReducePct          int64
		elementAdvPct            int64
		elementPenaltyPct        int64
		skillIDsJSON             []byte
		innateSkillIDsJSON       []byte
		normalSkillIDsJSON       []byte
		activeTalismanSkillID    int64
		talismanHeroSkillID      int64
		talismanSlot1SkillID     int64
		talismanSlot2SkillID     int64
		talismanSlot3SkillID     int64
		activeTalismanEnabled    bool
		talismanHeroEnabled      bool
		talismanSlot1Enabled     bool
		talismanSlot2Enabled     bool
		talismanSlot3Enabled     bool
		hpApt                    int64
		atkApt                   int64
		defApt                   int64
		spdApt                   int64
		manaApt                  int64
		grantSource              string
		captureMonsterID         int64
		freeAttrPoints           int64
		allocHP                  int64
		allocATK                 int64
		allocSPD                 int64
		allocMANA                int64
		allocDEF                 int64
		baseHPApt                int64
		baseATKApt               int64
		baseDEFApt               int64
		baseSPDApt               int64
		baseMANAApt              int64
		extraHPApt               int64
		extraATKApt              int64
		extraDEFApt              int64
		extraSPDApt              int64
		extraMANAApt             int64
		evolutionLevel           int64
		rebirthLevel             int64
		aptitudeProfile          string
	)
	if err := rows.Scan(
		&petUID, &petID, &level, &exp, &quality, &hp, &hpMax, &atk, &def, &spd, &mana,
		&spirit, &spiritMax, &hitPct, &dodgePct, &critRatePct, &critDmgPct,
		&physicalResistPct, &reversePhysicalResistPct, &skillResistPct, &reverseSkillResistPct,
		&confusionResistPct, &sleepResistPct, &paralysisResistPct, &sealResistPct, &curseResistPct,
		&critDmgResistPct, &critResistPct, &characterResistPct, &petResistPct,
		&guard, &talentDmgPct, &talentReducePct, &elementAdvPct, &elementPenaltyPct,
		&skillIDsJSON,
		&innateSkillIDsJSON, &normalSkillIDsJSON,
		&activeTalismanSkillID, &talismanHeroSkillID,
		&talismanSlot1SkillID, &talismanSlot2SkillID, &talismanSlot3SkillID,
		&activeTalismanEnabled, &talismanHeroEnabled,
		&talismanSlot1Enabled, &talismanSlot2Enabled, &talismanSlot3Enabled,
		&hpApt, &atkApt, &defApt, &spdApt, &manaApt, &grantSource, &captureMonsterID,
		&freeAttrPoints, &allocHP, &allocATK, &allocSPD, &allocMANA, &allocDEF,
		&baseHPApt, &baseATKApt, &baseDEFApt, &baseSPDApt, &baseMANAApt,
		&extraHPApt, &extraATKApt, &extraDEFApt, &extraSPDApt, &extraMANAApt,
		&evolutionLevel, &rebirthLevel, &aptitudeProfile,
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
	item.GrowthAptitudes = pet.GrowthAptitudes{
		HPApt:   uint32(hpApt),
		ATKApt:  uint32(atkApt),
		DEFApt:  uint32(defApt),
		SPDApt:  uint32(spdApt),
		MANAApt: uint32(manaApt),
	}
	item.GrantSource = grantSource
	item.CaptureMonsterID = uint32(captureMonsterID)
	item.FreeAttrPoints = uint32(freeAttrPoints)
	item.AllocHPPoints = uint32(allocHP)
	item.AllocATKPoints = uint32(allocATK)
	item.AllocSPDPoints = uint32(allocSPD)
	item.AllocMANAPoints = uint32(allocMANA)
	item.AllocDEFPoints = uint32(allocDEF)
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
	item.AptitudeProfile = aptitudeProfile
	item.SkillLoadout = decodeSkillLoadout(
		innateSkillIDsJSON,
		normalSkillIDsJSON,
		uint32(activeTalismanSkillID),
		uint32(talismanHeroSkillID),
		uint32(talismanSlot1SkillID),
		uint32(talismanSlot2SkillID),
		uint32(talismanSlot3SkillID),
		activeTalismanEnabled,
		talismanHeroEnabled,
		talismanSlot1Enabled,
		talismanSlot2Enabled,
		talismanSlot3Enabled,
	)
	if len(skillIDsJSON) > 0 {
		parsedSkillIDs, parseErr := decodeFlexibleSkillIDJSONArray(skillIDsJSON)
		if parseErr != nil {
			return pet.Pet{}, fmt.Errorf("unmarshal pet skill ids: %w", parseErr)
		}
		item.SkillIDs = parsedSkillIDs
	}
	pet.ApplyLegacySkillIDs(&item.SkillLoadout, item.SkillIDs)
	item.SkillIDs = pet.MergeBattleSkillIDs(item.SkillLoadout, item.SkillIDs)
	return item, nil
}

func buildBaseExtraAptitudes(definition *runtimePetDefinitionRow, total pet.GrowthAptitudes) petprogression.GrowthAptitudes {
	baseTemplate := petprogression.BaseTemplateFromTotals(
		definition.HPApt,
		definition.ATKApt,
		definition.DEFApt,
		definition.SPDApt,
		definition.MANAApt,
	)
	return petprogression.SplitTotalAptitudes(baseTemplate, petprogression.GrowthAptitudes{
		BaseHPApt:   total.HPApt,
		BaseATKApt:  total.ATKApt,
		BaseDEFApt:  total.DEFApt,
		BaseSPDApt:  total.SPDApt,
		BaseMANAApt: total.MANAApt,
	})
}

// resolveGrantedPetCombatStats 在成长公式尚未产生有效属性时（典型为 1 级、分配点为 0），
// 回退到宠物模板基础战斗值，避免新发放实例 hp=0 导致战斗内被判定死亡。
func resolveGrantedPetCombatStats(calculated petprogression.CombatStats, definition *runtimePetDefinitionRow) petprogression.CombatStats {
	if calculated.HPMax > 0 {
		return calculated
	}
	if definition == nil {
		return calculated
	}
	fallback := petprogression.CombatStats{
		HPMax: definition.HPMax,
		ATK:   definition.ATK,
		DEF:   definition.DEF,
		SPD:   definition.SPD,
		MANA:  definition.MANA,
	}
	if fallback.HPMax == 0 && definition.HP > 0 {
		fallback.HPMax = definition.HP
	}
	if fallback.HPMax == 0 {
		fallback.HPMax = 1
	}
	return fallback
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

func decodeFlexibleSkillIDJSONArray(raw []byte) ([]uint32, error) {
	if len(raw) == 0 {
		return []uint32{}, nil
	}
	return unmarshalFlexibleUint32Array(raw)
}

func decodeSkillIDJSONArray(raw []byte) []uint32 {
	values, err := decodeFlexibleSkillIDJSONArray(raw)
	if err != nil {
		return []uint32{}
	}
	return values
}

func decodeSkillLoadout(
	innateSkillIDsJSON []byte,
	normalSkillIDsJSON []byte,
	activeTalismanSkillID uint32,
	talismanHeroSkillID uint32,
	talismanSlot1SkillID uint32,
	talismanSlot2SkillID uint32,
	talismanSlot3SkillID uint32,
	activeTalismanEnabled bool,
	talismanHeroEnabled bool,
	talismanSlot1Enabled bool,
	talismanSlot2Enabled bool,
	talismanSlot3Enabled bool,
) pet.SkillLoadout {
	return pet.NormalizeSkillLoadout(pet.SkillLoadout{
		InnateSkillIDs:        decodeSkillIDJSONArray(innateSkillIDsJSON),
		NormalSkillIDs:        decodeSkillIDJSONArray(normalSkillIDsJSON),
		ActiveTalismanSkillID: activeTalismanSkillID,
		TalismanHeroSkillID:   talismanHeroSkillID,
		TalismanSlot1SkillID:  talismanSlot1SkillID,
		TalismanSlot2SkillID:  talismanSlot2SkillID,
		TalismanSlot3SkillID:  talismanSlot3SkillID,
		ActiveTalismanEnabled: activeTalismanEnabled,
		TalismanHeroEnabled:   talismanHeroEnabled,
		TalismanSlot1Enabled:  talismanSlot1Enabled,
		TalismanSlot2Enabled:  talismanSlot2Enabled,
		TalismanSlot3Enabled:  talismanSlot3Enabled,
	})
}

func (r *PetRepository) loadArtifactEquipmentByPlayerID(ctx context.Context, playerID uint64) (map[uint64][pet.MaxArtifactSkillSlots]uint32, error) {
	rows, err := r.db.QueryContext(ctx, listArtifactEquipmentByPlayerIDQuery, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[uint64][pet.MaxArtifactSkillSlots]uint32)
	for rows.Next() {
		var petUID int64
		var slotIndex int64
		var skillID int64
		if err := rows.Scan(&petUID, &slotIndex, &skillID); err != nil {
			return nil, err
		}
		if slotIndex < 0 || slotIndex >= pet.MaxArtifactSkillSlots {
			continue
		}
		slots := result[uint64(petUID)]
		slots[slotIndex] = uint32(skillID)
		result[uint64(petUID)] = slots
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func applyArtifactEquipment(item *pet.Pet, artifactSlots [pet.MaxArtifactSkillSlots]uint32) {
	if item == nil {
		return
	}
	item.SkillLoadout.ArtifactSkillIDs = artifactSlots
}

func applyArtifactSkillsToLoadout(loadout *pet.SkillLoadout, artifactSlots [pet.MaxArtifactSkillSlots]uint32) {
	if loadout == nil {
		return
	}
	loadout.ArtifactSkillIDs = artifactSlots
}
