export interface AdminSkillSummary {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
  weapon_discipline?: string;
  learn_exp_required?: number;
  learn_exp_per_use?: number;
  skill_type: string;
  target_type: string;
  energy_cost: number;
  is_basic_attack: boolean;
  is_enabled: boolean;
  status_text: string;
  created_at: string;
  updated_at: string;
}

export interface AdminSkillListResult {
  items: AdminSkillSummary[];
  total: number;
  page: number;
  page_size: number;
}

export interface AdminSkillTargetRule {
  target_type: string;
  target_count: number;
  preferred_target_hp: string;
}

export interface AdminSkillFormula {
  attack_pct: number;
  mana_pct: number;
  defense_pct: number;
  speed_pct: number;
  target_current_hp_pct: number;
  fixed_damage: number;
  heal_pct: number;
  fixed_heal: number;
  energy_cost: number;
  is_skill_attack: boolean;
  allow_crit: boolean;
  ignore_defense: boolean;
  skill_mult: number;
  skill_crit_add: number;
}

export interface AdminSkillStatusEffects {
  armor_break_pct: number;
  vulnerability_pct: number;
  bleed_chance_pct: number;
  bleed_rounds: number;
  bleed_damage: number;
  seal_chance_pct: number;
  seal_power: number;
  seal_rounds: number;
  vulnerability_chance_pct: number;
  vulnerability_rounds: number;
  vulnerability_apply_pct: number;
  armor_break_chance_pct: number;
  armor_break_rounds: number;
  slow_chance_pct: number;
  slow_rounds: number;
  slow_multiplier_pct: number;
  crit_boost_rounds: number;
  crit_boost_pct: number;
  curse_chance_pct: number;
  curse_rounds: number;
  curse_damage: number;
  curse_mana_pct: number;
  control_chance_pct: number;
  control_power: number;
  control_rounds: number;
  control_status_id: number;
}

export interface AdminSkillPresentation {
  animation_key: string;
  skill_visual_id: string;
  cast_color: string;
  impact_color: string;
  projectile: boolean;
}

export interface AdminSkillDetail {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
  weapon_discipline?: string;
  learn_exp_required?: number;
  learn_exp_per_use?: number;
  skill_type: string;
  description: string;
  acquire_method: string;
  is_basic_attack: boolean;
  is_enabled: boolean;
  status_text: string;
  sort_weight: number;
  target_rule: AdminSkillTargetRule;
  formula: AdminSkillFormula;
  status_effects: AdminSkillStatusEffects;
  presentation: AdminSkillPresentation;
  created_at: string;
  updated_at: string;
}

export interface AdminSkillListFilters {
  skill_id?: string;
  name?: string;
  category?: string;
  skill_type?: string;
  enabled?: 'true' | 'false';
  order_by?: string;
}

export interface AdminUpsertSkillPayload {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
  weapon_discipline?: string;
  learn_exp_required?: number;
  learn_exp_per_use?: number;
  skill_type: string;
  description: string;
  acquire_method: string;
  is_basic_attack: boolean;
  is_enabled: boolean;
  sort_weight: number;
  target_type: string;
  target_count: number;
  preferred_target_hp: string;
  animation_key: string;
  skill_visual_id: string;
  cast_color: string;
  impact_color: string;
  projectile: boolean;
  is_skill_attack: boolean;
  energy_cost: number;
  allow_crit: boolean;
  ignore_defense: boolean;
  skill_mult: number;
  skill_crit_add: number;
  attack_pct: number;
  mana_pct: number;
  defense_pct: number;
  speed_pct: number;
  target_current_hp_pct: number;
  fixed_damage: number;
  heal_pct: number;
  fixed_heal: number;
  armor_break_pct: number;
  vulnerability_pct: number;
  bleed_chance_pct: number;
  bleed_rounds: number;
  bleed_damage: number;
  seal_chance_pct: number;
  seal_power: number;
  seal_rounds: number;
  vulnerability_chance_pct: number;
  vulnerability_rounds: number;
  vulnerability_apply_pct: number;
  armor_break_chance_pct: number;
  armor_break_rounds: number;
  slow_chance_pct: number;
  slow_rounds: number;
  slow_multiplier_pct: number;
  crit_boost_rounds: number;
  crit_boost_pct: number;
  curse_chance_pct: number;
  curse_rounds: number;
  curse_damage: number;
  curse_mana_pct: number;
  control_chance_pct: number;
  control_power: number;
  control_rounds: number;
  control_status_id: number;
}

// defaultSkillValues 为新建技能模板提供完整默认值；skill_id / skill_code 由页面在打开弹窗时注入。
export function defaultSkillValues(skillID: number): AdminUpsertSkillPayload {
  return {
    skill_id: skillID,
    skill_code: `skill_${skillID}`,
    skill_name: '新系统技能',
    skill_category: 'pet',
    skill_type: 'attack',
    description: '',
    acquire_method: '运营配置',
    is_basic_attack: false,
    is_enabled: true,
    sort_weight: 100,
    target_type: 'enemy_single',
    target_count: 1,
    preferred_target_hp: '',
    animation_key: 'slash',
    skill_visual_id: '',
    cast_color: '#EBEBF5',
    impact_color: '#FFF2F2',
    projectile: false,
    is_skill_attack: true,
    energy_cost: 12,
    allow_crit: true,
    ignore_defense: false,
    skill_mult: 0,
    skill_crit_add: 0,
    attack_pct: 100,
    mana_pct: 30,
    defense_pct: 0,
    speed_pct: 20,
    target_current_hp_pct: 0,
    fixed_damage: 0,
    heal_pct: 0,
    fixed_heal: 0,
    armor_break_pct: 0,
    vulnerability_pct: 0,
    bleed_chance_pct: 0,
    bleed_rounds: 0,
    bleed_damage: 0,
    seal_chance_pct: 0,
    seal_power: 0,
    seal_rounds: 0,
    vulnerability_chance_pct: 0,
    vulnerability_rounds: 0,
    vulnerability_apply_pct: 0,
    armor_break_chance_pct: 0,
    armor_break_rounds: 0,
    slow_chance_pct: 0,
    slow_rounds: 0,
    slow_multiplier_pct: 0,
    crit_boost_rounds: 0,
    crit_boost_pct: 0,
    curse_chance_pct: 0,
    curse_rounds: 0,
    curse_damage: 0,
    curse_mana_pct: 0,
    control_chance_pct: 0,
    control_power: 0,
    control_rounds: 0,
    control_status_id: 0,
  };
}

// detailToPayload 将详情结构拍平成后台更新接口需要的表单载荷。
export function detailToPayload(detail: AdminSkillDetail): AdminUpsertSkillPayload {
  return {
    skill_id: detail.skill_id,
    skill_code: detail.skill_code,
    skill_name: detail.skill_name,
    skill_category: detail.skill_category,
    weapon_discipline: detail.weapon_discipline,
    learn_exp_required: detail.learn_exp_required,
    learn_exp_per_use: detail.learn_exp_per_use,
    skill_type: detail.skill_type,
    description: detail.description,
    acquire_method: detail.acquire_method,
    is_basic_attack: detail.is_basic_attack,
    is_enabled: detail.is_enabled,
    sort_weight: detail.sort_weight,
    target_type: detail.target_rule.target_type,
    target_count: detail.target_rule.target_count,
    preferred_target_hp: detail.target_rule.preferred_target_hp,
    animation_key: detail.presentation.animation_key,
    skill_visual_id: detail.presentation.skill_visual_id,
    cast_color: detail.presentation.cast_color,
    impact_color: detail.presentation.impact_color,
    projectile: detail.presentation.projectile,
    is_skill_attack: detail.formula.is_skill_attack,
    energy_cost: detail.formula.energy_cost,
    allow_crit: detail.formula.allow_crit,
    ignore_defense: detail.formula.ignore_defense,
    skill_mult: detail.formula.skill_mult,
    skill_crit_add: detail.formula.skill_crit_add,
    attack_pct: detail.formula.attack_pct,
    mana_pct: detail.formula.mana_pct,
    defense_pct: detail.formula.defense_pct,
    speed_pct: detail.formula.speed_pct,
    target_current_hp_pct: detail.formula.target_current_hp_pct,
    fixed_damage: detail.formula.fixed_damage,
    heal_pct: detail.formula.heal_pct,
    fixed_heal: detail.formula.fixed_heal,
    armor_break_pct: detail.status_effects.armor_break_pct,
    vulnerability_pct: detail.status_effects.vulnerability_pct,
    bleed_chance_pct: detail.status_effects.bleed_chance_pct,
    bleed_rounds: detail.status_effects.bleed_rounds,
    bleed_damage: detail.status_effects.bleed_damage,
    seal_chance_pct: detail.status_effects.seal_chance_pct,
    seal_power: detail.status_effects.seal_power,
    seal_rounds: detail.status_effects.seal_rounds,
    vulnerability_chance_pct: detail.status_effects.vulnerability_chance_pct,
    vulnerability_rounds: detail.status_effects.vulnerability_rounds,
    vulnerability_apply_pct: detail.status_effects.vulnerability_apply_pct,
    armor_break_chance_pct: detail.status_effects.armor_break_chance_pct,
    armor_break_rounds: detail.status_effects.armor_break_rounds,
    slow_chance_pct: detail.status_effects.slow_chance_pct,
    slow_rounds: detail.status_effects.slow_rounds,
    slow_multiplier_pct: detail.status_effects.slow_multiplier_pct,
    crit_boost_rounds: detail.status_effects.crit_boost_rounds,
    crit_boost_pct: detail.status_effects.crit_boost_pct,
    curse_chance_pct: detail.status_effects.curse_chance_pct,
    curse_rounds: detail.status_effects.curse_rounds,
    curse_damage: detail.status_effects.curse_damage,
    curse_mana_pct: detail.status_effects.curse_mana_pct,
    control_chance_pct: detail.status_effects.control_chance_pct,
    control_power: detail.status_effects.control_power,
    control_rounds: detail.status_effects.control_rounds,
    control_status_id: detail.status_effects.control_status_id,
  };
}
