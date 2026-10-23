-- check_migrations_127_130.sql
-- 只读检查：确认 127 / 128 / 129 / 130 四个迁移是否已在目标库执行。
-- 全部是 SELECT，不写库、不加锁、不建表，生产库可以直接跑。
--
-- 用法：
--   psql "postgres://<user>:<pass>@<host>:5432/pocket_pet?sslmode=disable" \
--        -f backend/docs/check_migrations_127_130.sql
-- 或直接在 DataGrip / Navicat 里整段执行。
--
-- 判读：ok / has_* 为 true 表示该项就绪。

\echo '=== 127 控制状态强度 ==='
SELECT
  '127' AS migration,
  EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'skill_definition' AND column_name = 'control_status_power'
  ) AS ok,
  'skill_definition.control_status_power 是否存在' AS check_item;

SELECT
  '127' AS migration,
  COUNT(*) FILTER (WHERE control_status_power > 0) AS configured_skills,
  COUNT(*) AS total_skills
FROM skill_definition;

\echo '=== 128 药囊物品 ==='
SELECT
  '128' AS migration,
  item_id,
  item_code,
  item_name,
  is_enabled
FROM item_definition
WHERE item_id IN (900101, 900102)
ORDER BY item_id;

SELECT
  '128' AS migration,
  d.item_id,
  e.equip_slot,
  m.restore_player_hp,
  m.restore_player_spirit,
  m.restore_pet_hp,
  m.restore_lineup_pets
FROM item_definition d
LEFT JOIN item_equipment_extra e ON e.item_id = d.item_id
LEFT JOIN item_medicine_pouch_extra m ON m.item_id = d.item_id
WHERE d.item_id IN (900101, 900102)
ORDER BY d.item_id;

-- 测试账号 10001 是否拿到了药囊（在背包或已佩戴）
SELECT
  '128' AS migration,
  item_uid,
  item_id,
  state
FROM equipment_instance
WHERE item_uid LIKE 'mig128_%'
ORDER BY item_uid;

\echo '=== 129 任务刷新周期 ==='
SELECT
  '129' AS migration,
  EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'quest_template' AND column_name = 'refresh_cycle'
  ) AS has_refresh_cycle,
  EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'quest_template' AND column_name = 'refresh_hour'
  ) AS has_refresh_hour;

SELECT
  '129' AS migration,
  refresh_cycle,
  is_repeatable,
  COUNT(*) AS quest_count
FROM quest_template
GROUP BY refresh_cycle, is_repeatable
ORDER BY refresh_cycle, is_repeatable;

\echo '=== 130 容器物品进入时间 ==='
SELECT
  '130' AS migration,
  EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'player_container_item' AND column_name = 'entered_at'
  ) AS has_entered_at,
  EXISTS (
    SELECT 1 FROM pg_indexes
    WHERE tablename = 'player_container_item'
      AND indexname = 'idx_player_container_item_entered'
  ) AS has_entered_index;

-- 若 entered_at 已加上：确认存量数据是否已按 created_at 回填（期望 0 行）
SELECT
  '130' AS migration,
  COUNT(*) AS backfill_remaining_rows
FROM player_container_item
WHERE entered_at IS DISTINCT FROM created_at;
