export interface AdminPlayerLevelConfig {
  level: number;
  exp_required: number;
  attr_points: number;
  bonus_atk: number;
  bonus_hp_max: number;
  bonus_spd: number;
  bonus_mana: number;
  status: number;
}

export interface AdminPlayerAttrConvertConfig {
  id: number;
  source_attr: string;
  target_attr: string;
  convert_rate: number;
  status: number;
}

export interface AdminPlayerLevelConfigListResult {
  items: AdminPlayerLevelConfig[];
}

export interface AdminPlayerAttrConvertListResult {
  items: AdminPlayerAttrConvertConfig[];
}

export interface AdminUpsertPlayerLevelConfigPayload {
  exp_required: number;
  attr_points: number;
  bonus_atk: number;
  bonus_hp_max: number;
  bonus_spd: number;
  bonus_mana: number;
  status: number;
}

export interface AdminUpsertPlayerAttrConvertPayload {
  source_attr: string;
  target_attr: string;
  convert_rate: number;
  status: number;
}
