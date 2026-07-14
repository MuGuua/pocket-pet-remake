-- 玩家剧情进度：用服务端持久化 flag 判断一次性剧情、NPC 解锁和后续主线入口。
CREATE TABLE IF NOT EXISTS player_story_flag (
  player_id BIGINT NOT NULL,
  flag_key VARCHAR(128) NOT NULL,
  flag_value VARCHAR(128) NOT NULL DEFAULT '1',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, flag_key)
);

-- 该迁移可能在线上批处理失败后被重新执行，因此 trigger 也要保持幂等。
DROP TRIGGER IF EXISTS set_player_story_flag_updated_at ON player_story_flag;

CREATE TRIGGER set_player_story_flag_updated_at
BEFORE UPDATE ON player_story_flag
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 场景进入触发器：进入指定 scene 后由服务端判断是否需要播放本地剧情动画。
CREATE TABLE IF NOT EXISTS scene_entry_trigger (
  id BIGSERIAL PRIMARY KEY,
  scene_id INTEGER NOT NULL,
  trigger_code VARCHAR(128) NOT NULL UNIQUE,
  priority INTEGER NOT NULL DEFAULT 0,
  once_flag_key VARCHAR(128) NOT NULL,
  required_quest_id BIGINT NOT NULL DEFAULT 0,
  required_quest_state VARCHAR(32) NOT NULL DEFAULT '',
  forbidden_quest_id BIGINT NOT NULL DEFAULT 0,
  forbidden_quest_state VARCHAR(32) NOT NULL DEFAULT '',
  client_animation_key VARCHAR(128) NOT NULL DEFAULT '',
  block_movement BOOLEAN NOT NULL DEFAULT TRUE,
  effect_accept_quest_id BIGINT NOT NULL DEFAULT 0,
  effect_set_flags JSONB NOT NULL DEFAULT '[]'::jsonb,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scene_entry_trigger_scene_status
  ON scene_entry_trigger (scene_id, status, priority DESC, id ASC);

-- 该迁移可能在线上批处理失败后被重新执行，因此 trigger 也要保持幂等。
DROP TRIGGER IF EXISTS set_scene_entry_trigger_updated_at ON scene_entry_trigger;

CREATE TRIGGER set_scene_entry_trigger_updated_at
BEFORE UPDATE ON scene_entry_trigger
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- NPC 可见性条件：按玩家剧情 flag 过滤世界快照，避免客户端自己隐藏服务端仍认为存在的 NPC。
ALTER TABLE world_entity_definition
  ADD COLUMN IF NOT EXISTS visibility_conditions_json JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 桃子在初见剧情完成前不进入该玩家的东路世界快照。
INSERT INTO world_entity_definition (
  entity_id,
  entity_code,
  display_name,
  entity_type,
  scene_id,
  status,
  visibility_conditions_json
) VALUES (
  92001,
  'taozi',
  '桃子',
  2,
  2,
  1,
  '{"required_flags":["taozi_npc_unlocked"]}'::jsonb
)
ON CONFLICT (entity_id) DO UPDATE SET
  entity_code = EXCLUDED.entity_code,
  display_name = EXCLUDED.display_name,
  entity_type = EXCLUDED.entity_type,
  scene_id = EXCLUDED.scene_id,
  status = EXCLUDED.status,
  visibility_conditions_json = EXCLUDED.visibility_conditions_json,
  updated_at = CURRENT_TIMESTAMP;

-- 第一个主线任务改为由剧情完成 Ack 后服务端接取，避免玩家进图前被 AUTO 任务提前物化。
UPDATE quest_template
SET accept_mode = 'NPC',
    start_npc_id = 92001,
    auto_track = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE quest_id = 1001;

INSERT INTO scene_entry_trigger (
  scene_id,
  trigger_code,
  priority,
  once_flag_key,
  forbidden_quest_id,
  client_animation_key,
  block_movement,
  effect_accept_quest_id,
  effect_set_flags,
  status
) VALUES (
  2,
  'first_enter_east_road_taozi',
  100,
  'intro_taozi_played',
  1001,
  '初见桃子',
  TRUE,
  1001,
  '["taozi_npc_unlocked"]'::jsonb,
  1
)
ON CONFLICT (trigger_code) DO UPDATE SET
  scene_id = EXCLUDED.scene_id,
  priority = EXCLUDED.priority,
  once_flag_key = EXCLUDED.once_flag_key,
  forbidden_quest_id = EXCLUDED.forbidden_quest_id,
  client_animation_key = EXCLUDED.client_animation_key,
  block_movement = EXCLUDED.block_movement,
  effect_accept_quest_id = EXCLUDED.effect_accept_quest_id,
  effect_set_flags = EXCLUDED.effect_set_flags,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

-- 新注册玩家从时光小屋出生由后端默认角色模型控制。
-- 线上迁移不主动改已有玩家位置，避免把已经在游玩的账号强制传送走。
-- 如果只想让本地测试账号重新体验新手链路，可在测试库手动执行：
-- UPDATE player
-- SET scene_id = 7,
--     pos_x = 4,
--     pos_y = 4,
--     updated_at = CURRENT_TIMESTAMP
-- WHERE id = 10001;
