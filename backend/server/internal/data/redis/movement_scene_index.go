package redis

import (
	"context"
	"fmt"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

// 初始化/切图与旧成员清理在同一个脚本内完成；断开保留移动状态供重连，不靠删除状态判在线。
const initializeIndexedMovementScript = `
local raw = redis.call('GET', KEYS[1])
if raw then
    local previous = cjson.decode(raw)
    redis.call('ZREM', ARGV[4]..tostring(previous.scene_id), ARGV[3])
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
return 'ok'
`
const markSceneOnlineScript = `
local raw = redis.call('GET', KEYS[1])
if not raw then return 'not_found' end
local state = cjson.decode(raw)
if state.session_id ~= ARGV[3] then return 'session_mismatch' end
if tonumber(state.scene_id) ~= tonumber(ARGV[2]) then return 'scene_mismatch' end
local ttl = redis.call('PTTL', KEYS[1])
if ttl <= 0 then return 'not_found' end
local lease = math.min(ttl, tonumber(ARGV[5]))
local now = redis.call('TIME')
local expires = tonumber(now[1])*1000 + math.floor(tonumber(now[2])/1000) + lease
redis.call('SET', KEYS[3], ARGV[4], 'PX', lease)
redis.call('ZADD', KEYS[2], expires, ARGV[1])
if redis.call('PTTL', KEYS[2]) < tonumber(ARGV[5]) then redis.call('PEXPIRE', KEYS[2], ARGV[5]) end
return 'ok'
`
const unmarkSceneOnlineScript = `
if redis.call('GET', KEYS[2]) ~= ARGV[1] then return 0 end
local raw = redis.call('GET', KEYS[1])
if raw then
    local state = cjson.decode(raw)
    redis.call('ZREM', ARGV[2]..tostring(state.scene_id), ARGV[3])
end
redis.call('DEL', KEYS[2])
return 1
`
const deleteIndexedMovementScript = `
local raw = redis.call('GET', KEYS[1])
if raw then
    local state = cjson.decode(raw)
    redis.call('ZREM', ARGV[1]..tostring(state.scene_id), ARGV[2])
end
redis.call('DEL', KEYS[1], KEYS[2])
return 1
`
const listSceneOnlineScript = `
local now = redis.call('TIME')
local at = tonumber(now[1])*1000 + math.floor(tonumber(now[2])/1000)
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', at)
local members = redis.call('ZRANGE', KEYS[1], 0, -1)
local result = {}
for _,id in ipairs(members) do
    local raw = redis.call('GET', ARGV[1]..id)
    local owner = redis.call('EXISTS', ARGV[2]..id)
    if raw and owner == 1 and tonumber(cjson.decode(raw).scene_id) == tonumber(ARGV[3]) then
        result[#result+1] = id
    else
        redis.call('ZREM', KEYS[1], id)
    end
end
return result
`

// 连接代次使用不具备鉴权能力的 ConnID，旧连接迟到清理不能移除新连接重新登记的成员。
func (r *MovementStateRepository) MarkSceneOnline(ctx context.Context, id uint64, scene uint32, sessionID, owner string, lease time.Duration) error {
	result, err := r.client.Eval(ctx, markSceneOnlineScript, []string{r.playerKey(id), r.sceneKey(scene), r.ownerKey(id)}, id, scene, sessionID, owner, max(int64(1), lease.Milliseconds()))
	if err != nil {
		return err
	}
	switch fmt.Sprint(result) {
	case "ok":
		return nil
	case "not_found":
		return world.ErrMovementStateNotFound
	case "session_mismatch":
		return world.ErrMovementSessionMismatch
	case "scene_mismatch":
		return world.ErrMovementSceneMismatch
	}
	return fmt.Errorf("unexpected scene online result %v", result)
}

// 移除在线成员但保留权威坐标和 dirty 集合；最终写回与重连沿用原流程。
func (r *MovementStateRepository) UnmarkSceneOnline(ctx context.Context, id uint64, owner string) error {
	_, err := r.client.Eval(ctx, unmarkSceneOnlineScript, []string{r.playerKey(id), r.ownerKey(id)}, owner, r.sceneKeyPrefix(), id)
	return err
}

// 跨进程共享索引按 Redis 服务器时间清理过期/断开/已切图成员，不把残留集合当成在线事实。
func (r *MovementStateRepository) ListSceneOnlinePlayerIDs(ctx context.Context, scene uint32) ([]uint64, error) {
	result, err := r.client.Eval(ctx, listSceneOnlineScript, []string{r.sceneKey(scene)}, r.keyPrefix+":world:movement:player:", r.keyPrefix+":world:movement:owner:", scene)
	if err != nil {
		return nil, err
	}
	return decodeDirtyPlayerIDs(result)
}
func (r *MovementStateRepository) sceneKeyPrefix() string {
	return r.keyPrefix + ":world:movement:scene:"
}
func (r *MovementStateRepository) sceneKey(id uint32) string {
	return fmt.Sprintf("%s%d", r.sceneKeyPrefix(), id)
}
func (r *MovementStateRepository) ownerKey(id uint64) string {
	return fmt.Sprintf("%s:world:movement:owner:%d", r.keyPrefix, id)
}

// 校验事件发布者仍持有在线连接代次，旧实例/旧连接迟到消息不能覆盖重连后的实体。
func (r *MovementStateRepository) SceneOnlineOwner(ctx context.Context, id uint64) (string, error) {
	raw, err := r.client.Get(ctx, r.ownerKey(id))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
