-- 125_monster_boss_drop_pool.sql
-- Boss 专属掉落池：按遭遇战配置一组候选物品，胜利时按权重抽取，与固定奖励叠加。
-- 与 monster_battle_reward（固定奖励）分离，避免把概率掉落塞进固定奖励列表后无法表达“多选一”。

CREATE TABLE IF NOT EXISTS monster_boss_drop_pool (
  id BIGSERIAL PRIMARY KEY,
  scene_id INTEGER NOT NULL,
  item_id BIGINT NOT NULL,
  quantity BIGINT NOT NULL DEFAULT 1,
  -- 权重：同一遭遇战内所有启用条目的权重之和为抽取基数，权重越大越容易抽中。
  weight INTEGER NOT NULL DEFAULT 100,
  -- 抽取组：同组内按权重抽 1 个；不同组互不影响，可同时命中。默认 1。
  draw_group INTEGER NOT NULL DEFAULT 1,
  -- 每名玩家是否只能获得一次；配合 bag.RecordUniqueItemObtained 去重。
  grant_once SMALLINT NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_monster_boss_drop_pool_quantity CHECK (quantity > 0),
  CONSTRAINT ck_monster_boss_drop_pool_weight CHECK (weight > 0),
  CONSTRAINT ck_monster_boss_drop_pool_draw_group CHECK (draw_group > 0),
  CONSTRAINT ck_monster_boss_drop_pool_grant_once CHECK (grant_once IN (0, 1)),
  CONSTRAINT ck_monster_boss_drop_pool_status CHECK (status IN (0, 1))
);

COMMENT ON TABLE monster_boss_drop_pool IS
  'Boss 专属权重掉落池；按 scene_id 关联 scene_wild_encounter，每个场景下按 draw_group 分组，每组按 weight 抽取一个物品。';

CREATE INDEX IF NOT EXISTS idx_monster_boss_drop_pool_scene
  ON monster_boss_drop_pool (scene_id, draw_group ASC, sort_order ASC, id ASC);

CREATE TRIGGER trg_monster_boss_drop_pool_updated_at
BEFORE UPDATE ON monster_boss_drop_pool
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 无需新增后台权限：Boss 掉落接口复用 031 迁移已建立的
-- scene_wild_encounters:view / scene_wild_encounters:edit。
