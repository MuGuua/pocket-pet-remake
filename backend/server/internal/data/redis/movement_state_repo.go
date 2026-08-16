package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"pocket-pet-remake/server/internal/module/world"
)

const movementStateTTL = 24 * time.Hour

const compareAndSetMovementScript = `
local raw = redis.call('GET', KEYS[1])
if not raw then return 'not_found' end
local current = cjson.decode(raw)
if current.session_id ~= ARGV[1] then return 'session_mismatch' end
if tonumber(current.scene_id) ~= tonumber(ARGV[2]) or tonumber(current.scene_version) ~= tonumber(ARGV[3]) then return 'scene_mismatch' end
if tonumber(current.last_move_seq) ~= tonumber(ARGV[4]) or tonumber(ARGV[5]) <= tonumber(current.last_move_seq) then return 'stale_sequence' end
redis.call('SET', KEYS[1], ARGV[6], 'EX', ARGV[7])
redis.call('SADD', KEYS[2], ARGV[8])
return 'ok'
`

const claimDirtyMovementPlayersScript = `
local limit = tonumber(ARGV[1])
if not limit or limit <= 0 then return {} end
return redis.call('SPOP', KEYS[1], limit)
`

const requeueDirtyMovementPlayersScript = `
for index = 1, #ARGV do
    redis.call('SADD', KEYS[1], ARGV[index])
end
return #ARGV
`

type movementStatePayload struct {
	PlayerID         uint64 `json:"player_id"`
	SessionID        string `json:"session_id"`
	SceneID          uint32 `json:"scene_id"`
	SceneVersion     uint32 `json:"scene_version"`
	PreciseX         int32  `json:"precise_x"`
	PreciseY         int32  `json:"precise_y"`
	PersistedX       int32  `json:"persisted_x"`
	PersistedY       int32  `json:"persisted_y"`
	FacingX          int32  `json:"facing_x"`
	FacingY          int32  `json:"facing_y"`
	Moving           bool   `json:"moving"`
	Speed            uint32 `json:"speed"`
	LastMoveSeq      uint32 `json:"last_move_seq"`
	LastServerTickMS int64  `json:"last_server_tick_ms"`
	PositionVersion  uint64 `json:"position_version"`
}

// MovementStateRepository 使用 Redis 保存在线玩家状态，并通过 Lua 保证移动序号比较与更新不可分割。
type MovementStateRepository struct {
	client    Client
	keyPrefix string
}

// NewMovementStateRepository 创建 Redis 权威移动状态仓储。
func NewMovementStateRepository(client Client, keyPrefix string) *MovementStateRepository {
	return &MovementStateRepository{client: client, keyPrefix: keyPrefix}
}

// Load 读取玩家最新的在线权威移动状态。
func (r *MovementStateRepository) Load(ctx context.Context, playerID uint64) (*world.MovementState, error) {
	raw, err := r.client.Get(ctx, r.playerKey(playerID))
	if errors.Is(err, ErrCacheMiss) {
		return nil, world.ErrMovementStateNotFound
	}
	if err != nil {
		return nil, err
	}
	var payload movementStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode movement state: %w", err)
	}
	state := payload.toDomain()
	return &state, nil
}

// Initialize 用 PostgreSQL权威位置或切图结果覆盖初始化当前会话的 Redis状态。
func (r *MovementStateRepository) Initialize(ctx context.Context, state world.MovementState) error {
	payload, err := encodeMovementState(state)
	if err != nil {
		return err
	}
	return r.client.SetEX(ctx, r.playerKey(state.PlayerID), payload, movementStateTTL)
}

// CompareAndSet 仅允许同会话、同场景代次且序号连续向前的状态覆盖当前值。
func (r *MovementStateRepository) CompareAndSet(ctx context.Context, expectedMoveSeq uint32, state world.MovementState) error {
	payload, err := encodeMovementState(state)
	if err != nil {
		return err
	}
	result, err := r.client.Eval(
		ctx,
		compareAndSetMovementScript,
		[]string{r.playerKey(state.PlayerID), r.dirtyKey()},
		state.SessionID,
		state.SceneID,
		state.SceneVersion,
		expectedMoveSeq,
		state.LastMoveSeq,
		payload,
		int64(movementStateTTL/time.Second),
		state.PlayerID,
	)
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
	case "stale_sequence":
		return world.ErrMovementSequenceStale
	default:
		return fmt.Errorf("unexpected movement state result: %v", result)
	}
}

// ClaimDirtyPlayerIDs 使用 Redis SPOP 原子领取一批待持久化玩家。
// 成员在领取时即从当前集合移除；若领取后玩家再次移动，CAS 会重新 SADD，因此新脏状态不会被旧批次清理。
func (r *MovementStateRepository) ClaimDirtyPlayerIDs(ctx context.Context, limit uint32) ([]uint64, error) {
	if limit == 0 {
		return []uint64{}, nil
	}
	result, err := r.client.Eval(ctx, claimDirtyMovementPlayersScript, []string{r.dirtyKey()}, limit)
	if err != nil {
		return nil, err
	}
	return decodeDirtyPlayerIDs(result)
}

// RequeueDirtyPlayerIDs 把 PostgreSQL 写回失败的玩家重新加入 dirty 集合，保证后续批次可以重试。
func (r *MovementStateRepository) RequeueDirtyPlayerIDs(ctx context.Context, playerIDs []uint64) error {
	if len(playerIDs) == 0 {
		return nil
	}
	args := make([]any, len(playerIDs))
	for index, playerID := range playerIDs {
		if playerID == 0 {
			return fmt.Errorf("requeue dirty movement player: player id must be greater than zero")
		}
		args[index] = playerID
	}
	_, err := r.client.Eval(ctx, requeueDirtyMovementPlayersScript, []string{r.dirtyKey()}, args...)
	return err
}

// Delete 删除玩家退出世界后的短时状态；调用方必须先完成最终持久化。
func (r *MovementStateRepository) Delete(ctx context.Context, playerID uint64) error {
	return r.client.Del(ctx, r.playerKey(playerID))
}

func (r *MovementStateRepository) playerKey(playerID uint64) string {
	return fmt.Sprintf("%s:world:movement:player:%d", r.keyPrefix, playerID)
}

func (r *MovementStateRepository) dirtyKey() string {
	return r.keyPrefix + ":world:movement:dirty"
}

// decodeDirtyPlayerIDs 将 Redis Lua 返回的字符串集合成员转换为领域层使用的玩家编号。
func decodeDirtyPlayerIDs(result any) ([]uint64, error) {
	if result == nil {
		return []uint64{}, nil
	}
	values, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("decode dirty movement players: unexpected result %T", result)
	}
	playerIDs := make([]uint64, 0, len(values))
	for _, value := range values {
		var rawPlayerID string
		switch typedValue := value.(type) {
		case string:
			rawPlayerID = typedValue
		case []byte:
			rawPlayerID = string(typedValue)
		default:
			return nil, fmt.Errorf("decode dirty movement player: unexpected member %T", value)
		}
		playerID, err := strconv.ParseUint(rawPlayerID, 10, 64)
		if err != nil || playerID == 0 {
			return nil, fmt.Errorf("decode dirty movement player %q: invalid player id", rawPlayerID)
		}
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs, nil
}

func encodeMovementState(state world.MovementState) ([]byte, error) {
	return json.Marshal(movementStatePayloadFromDomain(state))
}

func movementStatePayloadFromDomain(state world.MovementState) movementStatePayload {
	return movementStatePayload{
		PlayerID: state.PlayerID, SessionID: state.SessionID, SceneID: state.SceneID, SceneVersion: state.SceneVersion,
		PreciseX: state.PrecisePos.X, PreciseY: state.PrecisePos.Y, PersistedX: state.PersistedPos.X, PersistedY: state.PersistedPos.Y,
		FacingX: state.Facing.X, FacingY: state.Facing.Y, Moving: state.Moving, Speed: state.Speed,
		LastMoveSeq: state.LastMoveSeq, LastServerTickMS: state.LastServerTickMS, PositionVersion: state.PositionVersion,
	}
}

func (p movementStatePayload) toDomain() world.MovementState {
	return world.MovementState{
		PlayerID: p.PlayerID, SessionID: p.SessionID, SceneID: p.SceneID, SceneVersion: p.SceneVersion,
		PrecisePos: world.Vec2i{X: p.PreciseX, Y: p.PreciseY}, PersistedPos: world.Vec2i{X: p.PersistedX, Y: p.PersistedY},
		Facing: world.Vec2i{X: p.FacingX, Y: p.FacingY}, Moving: p.Moving, Speed: p.Speed,
		LastMoveSeq: p.LastMoveSeq, LastServerTickMS: p.LastServerTickMS, PositionVersion: p.PositionVersion,
	}
}
