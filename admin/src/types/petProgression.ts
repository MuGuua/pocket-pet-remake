export interface AdminPetLevelConfig {
  level: number;
  exp_required: number;
  attr_points: number;
  status: number;
}

export interface AdminPetAttrConvertConfig {
  attr_type: string;
  convert_rate: number;
  status: number;
}

export interface AdminPetLevelConfigListResult {
  items: AdminPetLevelConfig[];
}

export interface AdminPetAttrConvertListResult {
  items: AdminPetAttrConvertConfig[];
}

export interface AdminUpsertPetLevelConfigPayload {
  exp_required: number;
  attr_points: number;
  status: number;
}

export interface AdminUpsertPetAttrConvertPayload {
  convert_rate: number;
  status: number;
}
