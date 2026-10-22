-- 130_container_item_entered_at.sql
-- 背包与仓库改为按「进入容器的时间」排序：最早进入的排在第一页第一个，最新进入的排在最后。
--
-- entered_at 表示物品进入当前容器的时间，与记录创建时间 created_at 区分：
--   - 新获得、移动到新格子、卸下装备回背包：插入新行，取默认值（当前时间），自然排在最后；
--   - 在背包 / 仓库之间移动且合并到已有堆叠：服务端把该堆叠的 entered_at 刷新为当前时间，
--     保证「移动过来的物品」一定出现在最后一页最后一个；
--   - 奖励、购买等新获得但合并进已有堆叠：不刷新，原堆叠保持原位置。
-- 排序规则：entered_at ASC，同一时间（同一事务内插入）再按 slot_index ASC。
--
-- 存量数据以 created_at 回填，保持现有物品的相对先后。

ALTER TABLE player_container_item
  ADD COLUMN IF NOT EXISTS entered_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE player_container_item
SET entered_at = created_at
WHERE entered_at <> created_at;

CREATE INDEX IF NOT EXISTS idx_player_container_item_entered
  ON player_container_item (player_id, container_type, entered_at, slot_index);

COMMENT ON COLUMN player_container_item.entered_at IS
  '物品进入当前容器的时间；背包与仓库按 entered_at, slot_index 升序展示';
