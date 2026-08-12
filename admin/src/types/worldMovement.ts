export interface WorldMovementConfig {
  speed_milli_cells_per_second: number;
  max_elapsed_ms: number;
  axis_tolerance_milli: number;
  updated_at: string;
  last_update_reason: string;
  updated_by_admin_user_id: number;
}

export interface UpdateWorldMovementConfigPayload {
  speed_milli_cells_per_second: number;
  max_elapsed_ms: number;
  axis_tolerance_milli: number;
  reason: string;
}

// SceneBoundary 是数据库中的场景权威矩形边界，坐标单位统一为千分之一场景格。
export interface SceneBoundary {
  scene_id: number;
  scene_code: string;
  scene_name: string;
  min_x_milli: number;
  min_y_milli: number;
  max_x_milli: number;
  max_y_milli: number;
  updated_at: string;
  last_update_reason: string;
  updated_by_admin_user_id: number;
}

// UpdateSceneBoundaryPayload 要求管理员同时提交完整矩形和操作原因，避免局部更新产生无效边界。
export interface UpdateSceneBoundaryPayload {
  min_x_milli: number;
  min_y_milli: number;
  max_x_milli: number;
  max_y_milli: number;
  reason: string;
}
