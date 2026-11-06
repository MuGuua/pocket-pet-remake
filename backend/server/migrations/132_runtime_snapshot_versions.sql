BEGIN;

-- 由维护者执行。版本是持久化派生视图的失效依据，0 号行代表公共配置，不代表玩家。
CREATE TABLE IF NOT EXISTS runtime_snapshot_source (
 player_id bigint PRIMARY KEY CHECK(player_id >= 0),
 version bigint NOT NULL DEFAULT 1 CHECK(version > 0)
);
INSERT INTO runtime_snapshot_source(player_id) VALUES(0) ON CONFLICT DO NOTHING;
INSERT INTO runtime_snapshot_source(player_id) SELECT id FROM player ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS runtime_snapshot_stamp (
 player_id bigint NOT NULL CHECK(player_id > 0),
 view_name text NOT NULL CHECK(view_name IN ('player','pets','equipment','skills')),
 player_version bigint NOT NULL,
 config_version bigint NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY(player_id,view_name)
);

-- 触发器与正式写入共用事务，覆盖游戏、后台、奖励和维护 SQL，回滚也回滚失效标记。
CREATE OR REPLACE FUNCTION invalidate_runtime_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE before_row jsonb; after_row jsonb; old_id bigint; new_id bigint;
BEGIN
 IF TG_OP <> 'INSERT' THEN before_row := to_jsonb(OLD); END IF;
 IF TG_OP <> 'DELETE' THEN after_row := to_jsonb(NEW); END IF;
 IF TG_TABLE_NAME='player' AND TG_OP='UPDATE' THEN
  -- 普通位置写回不影响战斗/装备/技能快照，避免移动使所有视图持续失效。
  IF (before_row - ARRAY['scene_id','pos_x','pos_y','position_version','updated_at']) =
     (after_row - ARRAY['scene_id','pos_x','pos_y','position_version','updated_at']) THEN RETURN NEW; END IF;
 END IF;
 IF TG_TABLE_NAME='equipment_instance' THEN
  INSERT INTO runtime_snapshot_source(player_id,version)
   SELECT DISTINCT player_id,2 FROM player_equipment_slot
   WHERE item_uid IN (before_row->>'item_uid',after_row->>'item_uid') ORDER BY player_id
   ON CONFLICT(player_id) DO UPDATE SET version=runtime_snapshot_source.version+1;
 ELSIF TG_ARGV[0]='global' THEN
  UPDATE runtime_snapshot_source SET version=version+1 WHERE player_id=0;
 ELSE
  old_id := (before_row ->> TG_ARGV[0])::bigint;
  new_id := (after_row ->> TG_ARGV[0])::bigint;
  -- 固定编号顺序处理归属转移，旧主人和新主人都需要失效。
  INSERT INTO runtime_snapshot_source(player_id,version)
   SELECT DISTINCT id,2 FROM unnest(ARRAY[old_id,new_id]) AS ids(id) WHERE id > 0 ORDER BY id
   ON CONFLICT(player_id) DO UPDATE SET version=runtime_snapshot_source.version+1;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;

-- 这些是四类视图实际读取的源表；添加新的快照依赖时必须同步补充触发器。
DO $$
DECLARE source text; ownership text;
BEGIN
 FOREACH source IN ARRAY ARRAY['player','player_pet','player_lineup','pet_artifact_equipment','player_equipment_slot','player_skill_progress',
 'equipment_instance','item_definition','item_equipment_extra','player_attr_convert_config','pet_definition','pet_combat_stat_cap'] LOOP
  ownership := CASE WHEN source='player' THEN 'id'
   WHEN source IN ('player_pet','player_lineup','pet_artifact_equipment','player_equipment_slot','player_skill_progress') THEN 'player_id'
   ELSE 'global' END;
  EXECUTE format('DROP TRIGGER IF EXISTS runtime_snapshot_invalidate ON %I',source);
  EXECUTE format('CREATE TRIGGER runtime_snapshot_invalidate AFTER INSERT OR UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION invalidate_runtime_snapshot(%L)',source,ownership);
 END LOOP;
END $$;
COMMIT;

-- 回滚先部署旧后端，再移除上述源表的 runtime_snapshot_invalidate 触发器，随后删除函数和两张版本表。
