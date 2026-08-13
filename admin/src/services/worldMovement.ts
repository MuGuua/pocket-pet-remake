import { requestJSON } from './http';
import type {
  CreateSceneNavigationDraftPayload,
  PublishSceneNavigationPayload,
  RollbackSceneNavigationPayload,
  SceneBoundary,
  SceneNavigation,
  UpdateSceneBoundaryPayload,
  UpdateWorldMovementConfigPayload,
  WorldMovementConfig,
} from '../types/worldMovement';

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

// 读取全部启用场景的数据库边界，列表顺序由服务端按 scene_id 固定。
export async function fetchSceneBoundaries(): Promise<SceneBoundary[]> {
  return requestJSON<SceneBoundary[]>({
    url: '/api/admin/world/scene-boundaries',
    method: 'GET',
  });
}

// 更新单场景边界；服务端持久化成功后会立即原子替换对应运行时缓存。
export async function updateSceneBoundary(sceneID: number, payload: UpdateSceneBoundaryPayload): Promise<SceneBoundary> {
  return requestJSON<SceneBoundary>({
    url: `/api/admin/world/scene-boundaries/${sceneID}`,
    method: 'PUT',
    data: payload,
  });
}


export async function fetchSceneNavigations(sceneID: number): Promise<SceneNavigation[]> {
  return requestJSON<SceneNavigation[]>({ url: `/api/admin/world/scene-navigations?scene_id=${sceneID}`, method: 'GET' });
}
export async function createSceneNavigationDraft(payload: CreateSceneNavigationDraftPayload): Promise<SceneNavigation> {
  return requestJSON<SceneNavigation>({ url: '/api/admin/world/scene-navigations', method: 'POST', data: payload });
}
export async function publishSceneNavigation(navigationID: number, payload: PublishSceneNavigationPayload): Promise<SceneNavigation> {
  return requestJSON<SceneNavigation>({ url: `/api/admin/world/scene-navigations/${navigationID}/publish`, method: 'POST', data: payload });
}
export async function rollbackSceneNavigation(sceneID: number, payload: RollbackSceneNavigationPayload): Promise<SceneNavigation> {
  return requestJSON<SceneNavigation>({ url: `/api/admin/world/scene-navigations/scenes/${sceneID}/rollback`, method: 'POST', data: payload });
}
