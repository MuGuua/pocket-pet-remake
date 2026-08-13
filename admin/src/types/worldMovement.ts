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


export type SceneNavigationStatus = 0 | 1 | 2;

// SceneNavigation 是后台展示的静态通行位图版本摘要，原始位图不会回传到列表页。
export interface SceneNavigation {
  navigation_id: number; scene_id: number; scene_code: string; scene_name: string; version: number;
  origin_x_milli: number; origin_y_milli: number; grid_width: number; grid_height: number; cell_size_milli: number;
  data_hash: string; walkable_cell_count: number; source_scene_path: string; status: SceneNavigationStatus;
  change_reason: string; publish_reason: string; created_by_admin_user_id: number; published_by_admin_user_id: number;
  created_at: string; published_at: string; updated_at: string;
}

// SceneNavigationExportData 对应 Godot 导出工具生成的 JSON。
export interface SceneNavigationExportData {
  scene_id: number; origin_x_milli: number; origin_y_milli: number; grid_width: number; grid_height: number;
  cell_size_milli: number; navigation_data: string; source_scene_path: string;
}
export interface CreateSceneNavigationDraftPayload extends SceneNavigationExportData { reason: string; }
export interface PublishSceneNavigationPayload { reason: string; }
export interface RollbackSceneNavigationPayload { source_version: number; reason: string; }
