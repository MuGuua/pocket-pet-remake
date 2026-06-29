import type { AdminUpsertSkillPayload } from './skillDefinition';
import { formatControlStatusLabel } from '../utils/displayLabels';

/** 技能效果配置条目类型：每条对应一类公式/状态/表现配置。 */
export type SkillEffectConfigType =
  | 'damage_skill_mult'
  | 'damage_attack_pct'
  | 'heal'
  | 'fixed_damage'
  | 'formula_flags'
  | 'advanced_coefficients'
  | 'seal'
  | 'control'
  | 'bleed'
  | 'curse'
  | 'crit_boost'
  | 'slow'
  | 'armor_break'
  | 'vulnerability'
  | 'presentation';

/** 列表编辑器使用的技能效果配置条目。 */
export interface SkillEffectConfigEntry {
  entry_type: SkillEffectConfigType;
  sort_order: number;
  skill_mult?: number;
  attack_pct?: number;
  heal_pct?: number;
  fixed_heal?: number;
  fixed_damage?: number;
  is_skill_attack?: boolean;
  allow_crit?: boolean;
  ignore_defense?: boolean;
  mana_pct?: number;
  defense_pct?: number;
  speed_pct?: number;
  skill_crit_add?: number;
  target_current_hp_pct?: number;
  seal_chance_pct?: number;
  seal_power?: number;
  seal_rounds?: number;
  control_chance_pct?: number;
  control_power?: number;
  control_rounds?: number;
  control_status_id?: number;
  bleed_chance_pct?: number;
  bleed_rounds?: number;
  bleed_damage?: number;
  curse_chance_pct?: number;
  curse_rounds?: number;
  curse_damage?: number;
  curse_mana_pct?: number;
  crit_boost_pct?: number;
  crit_boost_rounds?: number;
  slow_chance_pct?: number;
  slow_rounds?: number;
  slow_multiplier_pct?: number;
  armor_break_pct?: number;
  armor_break_chance_pct?: number;
  armor_break_rounds?: number;
  vulnerability_pct?: number;
  vulnerability_chance_pct?: number;
  vulnerability_rounds?: number;
  vulnerability_apply_pct?: number;
  animation_key?: string;
  skill_visual_id?: string;
  cast_color?: string;
  impact_color?: string;
  projectile?: boolean;
}

export const SKILL_EFFECT_CONFIG_TYPE_LABELS: Record<SkillEffectConfigType, string> = {
  damage_skill_mult: '伤害 · 技能倍数',
  damage_attack_pct: '伤害 · 攻击系数',
  heal: '治疗',
  fixed_damage: '固定伤害',
  formula_flags: '战斗开关',
  advanced_coefficients: '高级系数',
  seal: '封印',
  control: '通用控制',
  bleed: '流血',
  curse: '诅咒',
  crit_boost: '暴击增益',
  slow: '减速',
  armor_break: '破甲',
  vulnerability: '易伤',
  presentation: '战斗表现',
};

export const SKILL_EFFECT_CONFIG_TYPE_OPTIONS = Object.entries(SKILL_EFFECT_CONFIG_TYPE_LABELS).map(
  ([value, label]) => ({ value: value as SkillEffectConfigType, label }),
);

/** 每种效果类型最多允许一条，避免运营重复配置互相覆盖。 */
export const UNIQUE_SKILL_EFFECT_CONFIG_TYPES = new Set<SkillEffectConfigType>([
  'damage_skill_mult',
  'damage_attack_pct',
  'heal',
  'fixed_damage',
  'formula_flags',
  'advanced_coefficients',
  'seal',
  'control',
  'bleed',
  'curse',
  'crit_boost',
  'slow',
  'armor_break',
  'vulnerability',
  'presentation',
]);

/** 新建攻击技能时的默认效果条目。 */
export function createDefaultSkillEffectEntries(): SkillEffectConfigEntry[] {
  return [
    normalizeSkillEffectConfigEntry({ entry_type: 'damage_attack_pct', attack_pct: 100 }, 0),
    normalizeSkillEffectConfigEntry({
      entry_type: 'formula_flags',
      is_skill_attack: true,
      allow_crit: true,
      ignore_defense: false,
    }, 1),
    normalizeSkillEffectConfigEntry({
      entry_type: 'presentation',
      animation_key: 'slash',
      cast_color: '#EBEBF5',
      impact_color: '#FFF2F2',
      projectile: false,
    }, 2),
  ];
}

/** 将表单条目整理为规范结构。 */
export function normalizeSkillEffectConfigEntry(
  formValues: Partial<SkillEffectConfigEntry>,
  fallbackSortOrder: number,
): SkillEffectConfigEntry {
  const entryType = formValues.entry_type ?? 'damage_attack_pct';
  const sortOrder = Number(formValues.sort_order ?? 0) > 0 ? Number(formValues.sort_order) : fallbackSortOrder + 1;
  return {
    entry_type: entryType,
    sort_order: sortOrder,
    skill_mult: Number(formValues.skill_mult ?? 0),
    attack_pct: Number(formValues.attack_pct ?? 0),
    heal_pct: Number(formValues.heal_pct ?? 0),
    fixed_heal: Number(formValues.fixed_heal ?? 0),
    fixed_damage: Number(formValues.fixed_damage ?? 0),
    is_skill_attack: Boolean(formValues.is_skill_attack),
    allow_crit: Boolean(formValues.allow_crit),
    ignore_defense: Boolean(formValues.ignore_defense),
    mana_pct: Number(formValues.mana_pct ?? 0),
    defense_pct: Number(formValues.defense_pct ?? 0),
    speed_pct: Number(formValues.speed_pct ?? 0),
    skill_crit_add: Number(formValues.skill_crit_add ?? 0),
    target_current_hp_pct: Number(formValues.target_current_hp_pct ?? 0),
    seal_chance_pct: Number(formValues.seal_chance_pct ?? 0),
    seal_power: Number(formValues.seal_power ?? 0),
    seal_rounds: Number(formValues.seal_rounds ?? 0),
    control_chance_pct: Number(formValues.control_chance_pct ?? 0),
    control_power: Number(formValues.control_power ?? 0),
    control_rounds: Number(formValues.control_rounds ?? 0),
    control_status_id: Number(formValues.control_status_id ?? 0),
    bleed_chance_pct: Number(formValues.bleed_chance_pct ?? 0),
    bleed_rounds: Number(formValues.bleed_rounds ?? 0),
    bleed_damage: Number(formValues.bleed_damage ?? 0),
    curse_chance_pct: Number(formValues.curse_chance_pct ?? 0),
    curse_rounds: Number(formValues.curse_rounds ?? 0),
    curse_damage: Number(formValues.curse_damage ?? 0),
    curse_mana_pct: Number(formValues.curse_mana_pct ?? 0),
    crit_boost_pct: Number(formValues.crit_boost_pct ?? 0),
    crit_boost_rounds: Number(formValues.crit_boost_rounds ?? 0),
    slow_chance_pct: Number(formValues.slow_chance_pct ?? 0),
    slow_rounds: Number(formValues.slow_rounds ?? 0),
    slow_multiplier_pct: Number(formValues.slow_multiplier_pct ?? 0),
    armor_break_pct: Number(formValues.armor_break_pct ?? 0),
    armor_break_chance_pct: Number(formValues.armor_break_chance_pct ?? 0),
    armor_break_rounds: Number(formValues.armor_break_rounds ?? 0),
    vulnerability_pct: Number(formValues.vulnerability_pct ?? 0),
    vulnerability_chance_pct: Number(formValues.vulnerability_chance_pct ?? 0),
    vulnerability_rounds: Number(formValues.vulnerability_rounds ?? 0),
    vulnerability_apply_pct: Number(formValues.vulnerability_apply_pct ?? 0),
    animation_key: String(formValues.animation_key ?? '').trim(),
    skill_visual_id: String(formValues.skill_visual_id ?? '').trim(),
    cast_color: String(formValues.cast_color ?? '').trim(),
    impact_color: String(formValues.impact_color ?? '').trim(),
    projectile: Boolean(formValues.projectile),
  };
}

/** 从服务端扁平载荷还原为列表条目。 */
export function skillEffectEntriesFromPayload(payload: AdminUpsertSkillPayload): SkillEffectConfigEntry[] {
  const entries: SkillEffectConfigEntry[] = [];
  let sortOrder = 0;
  const push = (partial: Partial<SkillEffectConfigEntry>) => {
    entries.push(normalizeSkillEffectConfigEntry(partial, sortOrder));
    sortOrder += 1;
  };

  if (payload.skill_mult > 0) {
    push({ entry_type: 'damage_skill_mult', skill_mult: payload.skill_mult });
  }
  if (payload.attack_pct !== 0) {
    push({ entry_type: 'damage_attack_pct', attack_pct: payload.attack_pct });
  }
  if (payload.heal_pct > 0 || payload.fixed_heal > 0) {
    push({ entry_type: 'heal', heal_pct: payload.heal_pct, fixed_heal: payload.fixed_heal });
  }
  if (payload.fixed_damage > 0) {
    push({ entry_type: 'fixed_damage', fixed_damage: payload.fixed_damage });
  }
  if (payload.is_skill_attack || payload.allow_crit || payload.ignore_defense) {
    push({
      entry_type: 'formula_flags',
      is_skill_attack: payload.is_skill_attack,
      allow_crit: payload.allow_crit,
      ignore_defense: payload.ignore_defense,
    });
  }
  if (
    payload.mana_pct !== 0
    || payload.defense_pct !== 0
    || payload.speed_pct !== 0
    || payload.skill_crit_add > 0
    || payload.target_current_hp_pct > 0
  ) {
    push({
      entry_type: 'advanced_coefficients',
      mana_pct: payload.mana_pct,
      defense_pct: payload.defense_pct,
      speed_pct: payload.speed_pct,
      skill_crit_add: payload.skill_crit_add,
      target_current_hp_pct: payload.target_current_hp_pct,
    });
  }
  if (payload.seal_chance_pct > 0 || payload.seal_power > 0 || payload.seal_rounds > 0) {
    push({
      entry_type: 'seal',
      seal_chance_pct: payload.seal_chance_pct,
      seal_power: payload.seal_power,
      seal_rounds: payload.seal_rounds,
    });
  }
  if (payload.control_chance_pct > 0 || payload.control_power > 0 || payload.control_rounds > 0 || payload.control_status_id > 0) {
    push({
      entry_type: 'control',
      control_chance_pct: payload.control_chance_pct,
      control_power: payload.control_power,
      control_rounds: payload.control_rounds,
      control_status_id: payload.control_status_id,
    });
  }
  if (payload.bleed_chance_pct > 0 || payload.bleed_rounds > 0 || payload.bleed_damage !== 0) {
    push({
      entry_type: 'bleed',
      bleed_chance_pct: payload.bleed_chance_pct,
      bleed_rounds: payload.bleed_rounds,
      bleed_damage: payload.bleed_damage,
    });
  }
  if (payload.curse_chance_pct > 0 || payload.curse_rounds > 0 || payload.curse_damage !== 0 || payload.curse_mana_pct > 0) {
    push({
      entry_type: 'curse',
      curse_chance_pct: payload.curse_chance_pct,
      curse_rounds: payload.curse_rounds,
      curse_damage: payload.curse_damage,
      curse_mana_pct: payload.curse_mana_pct,
    });
  }
  if (payload.crit_boost_pct > 0 || payload.crit_boost_rounds > 0) {
    push({
      entry_type: 'crit_boost',
      crit_boost_pct: payload.crit_boost_pct,
      crit_boost_rounds: payload.crit_boost_rounds,
    });
  }
  if (payload.slow_chance_pct > 0 || payload.slow_rounds > 0 || payload.slow_multiplier_pct > 0) {
    push({
      entry_type: 'slow',
      slow_chance_pct: payload.slow_chance_pct,
      slow_rounds: payload.slow_rounds,
      slow_multiplier_pct: payload.slow_multiplier_pct,
    });
  }
  if (payload.armor_break_pct > 0 || payload.armor_break_chance_pct > 0 || payload.armor_break_rounds > 0) {
    push({
      entry_type: 'armor_break',
      armor_break_pct: payload.armor_break_pct,
      armor_break_chance_pct: payload.armor_break_chance_pct,
      armor_break_rounds: payload.armor_break_rounds,
    });
  }
  if (
    payload.vulnerability_pct > 0
    || payload.vulnerability_chance_pct > 0
    || payload.vulnerability_rounds > 0
    || payload.vulnerability_apply_pct > 0
  ) {
    push({
      entry_type: 'vulnerability',
      vulnerability_pct: payload.vulnerability_pct,
      vulnerability_chance_pct: payload.vulnerability_chance_pct,
      vulnerability_rounds: payload.vulnerability_rounds,
      vulnerability_apply_pct: payload.vulnerability_apply_pct,
    });
  }
  if (payload.animation_key || payload.cast_color || payload.impact_color || payload.projectile || payload.skill_visual_id) {
    push({
      entry_type: 'presentation',
      animation_key: payload.animation_key,
      skill_visual_id: payload.skill_visual_id,
      cast_color: payload.cast_color,
      impact_color: payload.impact_color,
      projectile: payload.projectile,
    });
  }
  return entries.sort((left, right) => left.sort_order - right.sort_order);
}

/** 将列表条目合并回服务端扁平载荷。 */
export function mergePayloadFromSkillEffectEntries(
  basePayload: AdminUpsertSkillPayload,
  entries: SkillEffectConfigEntry[] | undefined,
): AdminUpsertSkillPayload {
  const nextPayload: AdminUpsertSkillPayload = {
    ...basePayload,
    skill_mult: 0,
    attack_pct: 0,
    heal_pct: 0,
    fixed_heal: 0,
    fixed_damage: 0,
    is_skill_attack: false,
    allow_crit: false,
    ignore_defense: false,
    mana_pct: 0,
    defense_pct: 0,
    speed_pct: 0,
    skill_crit_add: 0,
    target_current_hp_pct: 0,
    seal_chance_pct: 0,
    seal_power: 0,
    seal_rounds: 0,
    control_chance_pct: 0,
    control_power: 0,
    control_rounds: 0,
    control_status_id: 0,
    bleed_chance_pct: 0,
    bleed_rounds: 0,
    bleed_damage: 0,
    curse_chance_pct: 0,
    curse_rounds: 0,
    curse_damage: 0,
    curse_mana_pct: 0,
    crit_boost_rounds: 0,
    crit_boost_pct: 0,
    slow_chance_pct: 0,
    slow_rounds: 0,
    slow_multiplier_pct: 0,
    armor_break_pct: 0,
    armor_break_chance_pct: 0,
    armor_break_rounds: 0,
    vulnerability_pct: 0,
    vulnerability_chance_pct: 0,
    vulnerability_rounds: 0,
    vulnerability_apply_pct: 0,
    animation_key: '',
    skill_visual_id: '',
    cast_color: '',
    impact_color: '',
    projectile: false,
  };

  const sortedEntries = [...(entries ?? [])].sort((left, right) => left.sort_order - right.sort_order);
  sortedEntries.forEach((entry) => {
    switch (entry.entry_type) {
      case 'damage_skill_mult':
        nextPayload.skill_mult = Number(entry.skill_mult ?? 0);
        break;
      case 'damage_attack_pct':
        nextPayload.attack_pct = Number(entry.attack_pct ?? 0);
        break;
      case 'heal':
        nextPayload.heal_pct = Number(entry.heal_pct ?? 0);
        nextPayload.fixed_heal = Number(entry.fixed_heal ?? 0);
        break;
      case 'fixed_damage':
        nextPayload.fixed_damage = Number(entry.fixed_damage ?? 0);
        break;
      case 'formula_flags':
        nextPayload.is_skill_attack = Boolean(entry.is_skill_attack);
        nextPayload.allow_crit = Boolean(entry.allow_crit);
        nextPayload.ignore_defense = Boolean(entry.ignore_defense);
        break;
      case 'advanced_coefficients':
        nextPayload.mana_pct = Number(entry.mana_pct ?? 0);
        nextPayload.defense_pct = Number(entry.defense_pct ?? 0);
        nextPayload.speed_pct = Number(entry.speed_pct ?? 0);
        nextPayload.skill_crit_add = Number(entry.skill_crit_add ?? 0);
        nextPayload.target_current_hp_pct = Number(entry.target_current_hp_pct ?? 0);
        break;
      case 'seal':
        nextPayload.seal_chance_pct = Number(entry.seal_chance_pct ?? 0);
        nextPayload.seal_power = Number(entry.seal_power ?? 0);
        nextPayload.seal_rounds = Number(entry.seal_rounds ?? 0);
        break;
      case 'control':
        nextPayload.control_chance_pct = Number(entry.control_chance_pct ?? 0);
        nextPayload.control_power = Number(entry.control_power ?? 0);
        nextPayload.control_rounds = Number(entry.control_rounds ?? 0);
        nextPayload.control_status_id = Number(entry.control_status_id ?? 0);
        break;
      case 'bleed':
        nextPayload.bleed_chance_pct = Number(entry.bleed_chance_pct ?? 0);
        nextPayload.bleed_rounds = Number(entry.bleed_rounds ?? 0);
        nextPayload.bleed_damage = Number(entry.bleed_damage ?? 0);
        break;
      case 'curse':
        nextPayload.curse_chance_pct = Number(entry.curse_chance_pct ?? 0);
        nextPayload.curse_rounds = Number(entry.curse_rounds ?? 0);
        nextPayload.curse_damage = Number(entry.curse_damage ?? 0);
        nextPayload.curse_mana_pct = Number(entry.curse_mana_pct ?? 0);
        break;
      case 'crit_boost':
        nextPayload.crit_boost_pct = Number(entry.crit_boost_pct ?? 0);
        nextPayload.crit_boost_rounds = Number(entry.crit_boost_rounds ?? 0);
        break;
      case 'slow':
        nextPayload.slow_chance_pct = Number(entry.slow_chance_pct ?? 0);
        nextPayload.slow_rounds = Number(entry.slow_rounds ?? 0);
        nextPayload.slow_multiplier_pct = Number(entry.slow_multiplier_pct ?? 0);
        break;
      case 'armor_break':
        nextPayload.armor_break_pct = Number(entry.armor_break_pct ?? 0);
        nextPayload.armor_break_chance_pct = Number(entry.armor_break_chance_pct ?? 0);
        nextPayload.armor_break_rounds = Number(entry.armor_break_rounds ?? 0);
        break;
      case 'vulnerability':
        nextPayload.vulnerability_pct = Number(entry.vulnerability_pct ?? 0);
        nextPayload.vulnerability_chance_pct = Number(entry.vulnerability_chance_pct ?? 0);
        nextPayload.vulnerability_rounds = Number(entry.vulnerability_rounds ?? 0);
        nextPayload.vulnerability_apply_pct = Number(entry.vulnerability_apply_pct ?? 0);
        break;
      case 'presentation':
        nextPayload.animation_key = String(entry.animation_key ?? '').trim();
        nextPayload.skill_visual_id = String(entry.skill_visual_id ?? '').trim();
        nextPayload.cast_color = String(entry.cast_color ?? '').trim();
        nextPayload.impact_color = String(entry.impact_color ?? '').trim();
        nextPayload.projectile = Boolean(entry.projectile);
        break;
      default:
        break;
    }
  });
  return nextPayload;
}

/** 列表摘要文案，供表格展示。 */
export function formatSkillEffectConfigSummary(entry: SkillEffectConfigEntry): string {
  switch (entry.entry_type) {
    case 'damage_skill_mult':
      return `技能倍数 ${Number(entry.skill_mult ?? 0)}`;
    case 'damage_attack_pct':
      return `攻击系数 ${Number(entry.attack_pct ?? 0)}%`;
    case 'heal':
      return `治疗 ${Number(entry.heal_pct ?? 0)}% + 固定 ${Number(entry.fixed_heal ?? 0)}`;
    case 'fixed_damage':
      return `固定伤害 ${Number(entry.fixed_damage ?? 0)}`;
    case 'formula_flags': {
      const flags: string[] = [];
      if (entry.is_skill_attack) flags.push('技能攻击');
      if (entry.allow_crit) flags.push('允许暴击');
      if (entry.ignore_defense) flags.push('忽略防御');
      return flags.length > 0 ? flags.join(' / ') : '未启用开关';
    }
    case 'advanced_coefficients':
      return `法${Number(entry.mana_pct ?? 0)} / 防${Number(entry.defense_pct ?? 0)} / 速${Number(entry.speed_pct ?? 0)} / 爆伤+${Number(entry.skill_crit_add ?? 0)}`;
    case 'seal':
      return `概率 ${Number(entry.seal_chance_pct ?? 0)}% / 威力 ${Number(entry.seal_power ?? 0)} / ${Number(entry.seal_rounds ?? 0)} 回合`;
    case 'control':
      return `概率 ${Number(entry.control_chance_pct ?? 0)}% / 威力 ${Number(entry.control_power ?? 0)} / ${Number(entry.control_rounds ?? 0)} 回合 / ${formatControlStatusLabel(Number(entry.control_status_id ?? 0))}`;
    case 'bleed':
      return `概率 ${Number(entry.bleed_chance_pct ?? 0)}% / ${Number(entry.bleed_rounds ?? 0)} 回合 / 伤害 ${Number(entry.bleed_damage ?? 0)}`;
    case 'curse':
      return `概率 ${Number(entry.curse_chance_pct ?? 0)}% / ${Number(entry.curse_rounds ?? 0)} 回合 / 伤害 ${Number(entry.curse_damage ?? 0)}`;
    case 'crit_boost':
      return `暴击 +${Number(entry.crit_boost_pct ?? 0)}% / ${Number(entry.crit_boost_rounds ?? 0)} 回合`;
    case 'slow':
      return `概率 ${Number(entry.slow_chance_pct ?? 0)}% / ${Number(entry.slow_rounds ?? 0)} 回合 / 倍率 ${Number(entry.slow_multiplier_pct ?? 0)}%`;
    case 'armor_break':
      return `破甲 ${Number(entry.armor_break_pct ?? 0)}% / 概率 ${Number(entry.armor_break_chance_pct ?? 0)}%`;
    case 'vulnerability':
      return `易伤 ${Number(entry.vulnerability_pct ?? 0)}% / 概率 ${Number(entry.vulnerability_chance_pct ?? 0)}%`;
    case 'presentation':
      return `${entry.animation_key || '默认动画'} / ${entry.projectile ? '投射' : '非投射'}`;
    default:
      return '-';
  }
}

/** 新建条目时的默认表单值。 */
export function createDefaultSkillEffectConfigRow(
  entryType: SkillEffectConfigType,
  nextSortOrder: number,
): SkillEffectConfigEntry {
  switch (entryType) {
    case 'damage_skill_mult':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, skill_mult: 30 }, nextSortOrder);
    case 'damage_attack_pct':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, attack_pct: 100 }, nextSortOrder);
    case 'heal':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, heal_pct: 20, fixed_heal: 0 }, nextSortOrder);
    case 'fixed_damage':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, fixed_damage: 10 }, nextSortOrder);
    case 'formula_flags':
      return normalizeSkillEffectConfigEntry({
        entry_type: entryType,
        is_skill_attack: true,
        allow_crit: true,
        ignore_defense: false,
      }, nextSortOrder);
    case 'advanced_coefficients':
      return normalizeSkillEffectConfigEntry({
        entry_type: entryType,
        mana_pct: 30,
        defense_pct: 0,
        speed_pct: 20,
        skill_crit_add: 0,
        target_current_hp_pct: 0,
      }, nextSortOrder);
    case 'seal':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, seal_chance_pct: 0, seal_power: 0, seal_rounds: 1 }, nextSortOrder);
    case 'control':
      return normalizeSkillEffectConfigEntry({
        entry_type: entryType,
        control_chance_pct: 0,
        control_power: 0,
        control_rounds: 1,
        control_status_id: 0,
      }, nextSortOrder);
    case 'bleed':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, bleed_chance_pct: 30, bleed_rounds: 2, bleed_damage: 3 }, nextSortOrder);
    case 'curse':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, curse_chance_pct: 30, curse_rounds: 2, curse_damage: 3, curse_mana_pct: 0 }, nextSortOrder);
    case 'crit_boost':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, crit_boost_pct: 20, crit_boost_rounds: 2 }, nextSortOrder);
    case 'slow':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, slow_chance_pct: 100, slow_rounds: 2, slow_multiplier_pct: 70 }, nextSortOrder);
    case 'armor_break':
      return normalizeSkillEffectConfigEntry({ entry_type: entryType, armor_break_pct: 0, armor_break_chance_pct: 100, armor_break_rounds: 2 }, nextSortOrder);
    case 'vulnerability':
      return normalizeSkillEffectConfigEntry({
        entry_type: entryType,
        vulnerability_pct: 0,
        vulnerability_chance_pct: 100,
        vulnerability_rounds: 2,
        vulnerability_apply_pct: 12,
      }, nextSortOrder);
    case 'presentation':
      return normalizeSkillEffectConfigEntry({
        entry_type: entryType,
        animation_key: 'slash',
        cast_color: '#EBEBF5',
        impact_color: '#FFF2F2',
        projectile: false,
      }, nextSortOrder);
    default:
      return normalizeSkillEffectConfigEntry({ entry_type: 'damage_attack_pct', attack_pct: 100 }, nextSortOrder);
  }
}
