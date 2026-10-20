/** 后台补发支持的奖励类型，与服务端 validateAdminRewardEntries 白名单保持一致。 */
export type AdminRewardEntryType = 'item' | 'gold' | 'exp' | 'pet';

/** 单条补发奖励；不同类型只读取各自字段（item 用 item_id/count，gold/exp 用 value，pet 用 pet_id）。 */
export interface AdminRewardEntry {
  type: AdminRewardEntryType;
  value?: number;
  item_id?: number;
  item_name?: string;
  count?: number;
  pet_id?: number;
}

/** POST /api/admin/rewards 请求体；reason 为必填操作原因，会写入 admin_operation_log 审计。 */
export interface AdminRewardGrantPayload {
  player_id: number;
  reason: string;
  rewards: AdminRewardEntry[];
}

/** 补发接口的服务端返回值；granted 为实际到账条目，可能少于请求条目。 */
export interface AdminRewardGrantResult {
  player_id: number;
  reason: string;
  granted: AdminRewardEntry[];
  wallet?: { total_copper: number; gold: number; silver: number; copper: number };
  granted_pets: Array<{ pet_uid: number; pet_id: number; pet_name?: string }>;
}
