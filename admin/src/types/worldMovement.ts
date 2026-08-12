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
