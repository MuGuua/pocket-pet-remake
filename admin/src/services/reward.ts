import { requestJSON } from './http';
import type { AdminRewardGrantPayload, AdminRewardGrantResult } from '../types/reward';

/** 调用服务端统一发奖入口；物品、货币、经验、宠物全部走 reward.Service，不在前端拼装结果。 */
export async function grantAdminRewards(payload: AdminRewardGrantPayload): Promise<AdminRewardGrantResult> {
  return requestJSON<AdminRewardGrantResult>({ url: '/api/admin/rewards', method: 'POST', data: payload });
}
