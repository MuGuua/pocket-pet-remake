import { requestJSON } from './http';
import type { UpdateWorldMovementConfigPayload, WorldMovementConfig } from '../types/worldMovement';

// 读取数据库中的权威移动配置，页面不自行缓存或拼装正式参数。
export async function fetchWorldMovementConfig(): Promise<WorldMovementConfig> {
  return requestJSON<WorldMovementConfig>({
    url: '/api/admin/world/movement-config',
    method: 'GET',
  });
}

// 更新成功时服务端已经同步刷新运行时快照，返回值就是最终生效值。
export async function updateWorldMovementConfig(payload: UpdateWorldMovementConfigPayload): Promise<WorldMovementConfig> {
  return requestJSON<WorldMovementConfig>({
    url: '/api/admin/world/movement-config',
    method: 'PUT',
    data: payload,
  });
}
