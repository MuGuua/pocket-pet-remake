export interface AdminSkillSummary {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
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
}

export interface AdminSkillStatusEffects {
  armor_break_pct: number;
  vulnerability_pct: number;
  bleed_chance_pct: number;
  bleed_rounds: number;
  bleed_damage: number;
  seal_chance_pct: number;
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
  control_rounds: number;
  control_status_id: number;
}

export interface AdminSkillPresentation {
  animation_key: string;
  cast_color: string;
  impact_color: string;
  projectile: boolean;
}

export interface AdminSkillDetail {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
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
}

export interface AdminUpsertSkillPayload {
  skill_id: number;
  skill_code: string;
  skill_name: string;
  skill_category: string;
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
  cast_color: string;
  impact_color: string;
  projectile: boolean;
  is_skill_attack: boolean;
  energy_cost: number;
  allow_crit: boolean;
  ignore_defense: boolean;
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
  control_rounds: number;
  control_status_id: number;
}
