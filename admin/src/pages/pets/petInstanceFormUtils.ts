import type {
  AdminCreatePetPayload,
  AdminPetDetail,
  AdminUpdatePetPayload,
} from '../../types/pet';
import {
  defaultAdminPetCombatStats,
  type AdminPetCombatStats,
} from '../../types/petCombatStats';
import { formatSkillReferenceInput, parseSkillReferenceInput, type SkillReferenceMap } from '../../utils/skillReference';

export interface PetInstanceFormValues extends AdminPetCombatStats {
  player_id?: number;
  pet_id: number;
  custom_name: string;
  level: number;
  exp: number;
  quality: number;
  hp: number;
  hp_max: number;
  atk: number;
  def: number;
  spd: number;
  mana: number;
  skill_names_text: string;
}

export function defaultPetInstanceCreateValues(playerId: number): PetInstanceFormValues {
  return {
    player_id: playerId,
    pet_id: 101,
    custom_name: '',
    level: 1,
    exp: 0,
    quality: 1,
    hp: 10,
    hp_max: 10,
    atk: 5,
    def: 5,
    spd: 5,
    mana: 0,
    skill_names_text: '普通攻击',
    ...defaultAdminPetCombatStats(),
  };
}

export function mapPetDetailToForm(detail: AdminPetDetail, skillReferenceMap: SkillReferenceMap): PetInstanceFormValues {
  return {
    pet_id: detail.pet_id,
    custom_name: detail.custom_name ?? '',
    level: detail.level,
    exp: detail.exp,
    quality: detail.quality,
    hp: detail.hp,
    hp_max: detail.hp_max,
    atk: detail.atk,
    def: detail.def,
    spd: detail.spd,
    mana: detail.mana,
    skill_names_text: formatSkillReferenceInput(detail.skill_ids, skillReferenceMap),
    spirit: detail.spirit,
    spirit_max: detail.spirit_max,
    hit_pct: detail.hit_pct,
    dodge_pct: detail.dodge_pct,
    crit_rate_pct: detail.crit_rate_pct,
    crit_dmg_pct: detail.crit_dmg_pct,
    physical_resist_pct: detail.physical_resist_pct,
    reverse_physical_resist_pct: detail.reverse_physical_resist_pct,
    skill_resist_pct: detail.skill_resist_pct,
    reverse_skill_resist_pct: detail.reverse_skill_resist_pct,
    confusion_resist_pct: detail.confusion_resist_pct,
    sleep_resist_pct: detail.sleep_resist_pct,
    paralysis_resist_pct: detail.paralysis_resist_pct,
    seal_resist_pct: detail.seal_resist_pct,
    curse_resist_pct: detail.curse_resist_pct,
    crit_dmg_resist_pct: detail.crit_dmg_resist_pct,
    crit_resist_pct: detail.crit_resist_pct,
    character_resist_pct: detail.character_resist_pct,
    pet_resist_pct: detail.pet_resist_pct,
    guard: detail.guard,
    talent_dmg_pct: detail.talent_dmg_pct,
    talent_reduce_pct: detail.talent_reduce_pct,
    element_adv_pct: detail.element_adv_pct,
    element_penalty_pct: detail.element_penalty_pct,
  };
}

function buildCombatStats(values: PetInstanceFormValues): AdminPetCombatStats {
  return {
    spirit: values.spirit,
    spirit_max: values.spirit_max,
    hit_pct: values.hit_pct,
    dodge_pct: values.dodge_pct,
    crit_rate_pct: values.crit_rate_pct,
    crit_dmg_pct: values.crit_dmg_pct,
    physical_resist_pct: values.physical_resist_pct,
    reverse_physical_resist_pct: values.reverse_physical_resist_pct,
    skill_resist_pct: values.skill_resist_pct,
    reverse_skill_resist_pct: values.reverse_skill_resist_pct,
    confusion_resist_pct: values.confusion_resist_pct,
    sleep_resist_pct: values.sleep_resist_pct,
    paralysis_resist_pct: values.paralysis_resist_pct,
    seal_resist_pct: values.seal_resist_pct,
    curse_resist_pct: values.curse_resist_pct,
    crit_dmg_resist_pct: values.crit_dmg_resist_pct,
    crit_resist_pct: values.crit_resist_pct,
    character_resist_pct: values.character_resist_pct,
    pet_resist_pct: values.pet_resist_pct,
    guard: values.guard,
    talent_dmg_pct: values.talent_dmg_pct,
    talent_reduce_pct: values.talent_reduce_pct,
    element_adv_pct: values.element_adv_pct,
    element_penalty_pct: values.element_penalty_pct,
  };
}

export function mapPetFormToCreatePayload(values: PetInstanceFormValues, skillReferenceMap: SkillReferenceMap): AdminCreatePetPayload {
  return {
    player_id: values.player_id ?? 0,
    pet_id: values.pet_id,
    level: values.level,
    exp: values.exp,
    quality: values.quality,
    hp: values.hp,
    hp_max: values.hp_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    skill_ids: parseSkillReferenceInput(values.skill_names_text, skillReferenceMap),
    ...buildCombatStats(values),
  };
}

export function mapPetFormToUpdatePayload(values: PetInstanceFormValues, skillReferenceMap: SkillReferenceMap): AdminUpdatePetPayload {
  return {
    pet_id: values.pet_id,
    custom_name: values.custom_name ?? '',
    level: values.level,
    exp: values.exp,
    quality: values.quality,
    hp: values.hp,
    hp_max: values.hp_max,
    atk: values.atk,
    def: values.def,
    spd: values.spd,
    mana: values.mana,
    skill_ids: parseSkillReferenceInput(values.skill_names_text, skillReferenceMap),
    ...buildCombatStats(values),
  };
}

export function formatPetDateTime(value: string | null | undefined): string {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}
