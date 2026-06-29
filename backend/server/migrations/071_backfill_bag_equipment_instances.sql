-- 071_backfill_bag_equipment_instances.sql
-- 为历史数据中「背包里有装备模板但没有 equipment_instance / item_uid」的条目补建实例，
-- 否则客户端可预览 100% 成功率但强化请求会因缺少 item_uid 直接失败。

DO $$
DECLARE
  rec RECORD;
  new_uid TEXT;
BEGIN
  FOR rec IN
    SELECT
      pci.id,
      pci.player_id,
      pci.item_id,
      COALESCE(NULLIF(idf.bind_type, ''), 'none') AS bind_type
    FROM player_container_item pci
    JOIN item_definition idf ON idf.item_id = pci.item_id
    WHERE idf.item_type = 'equipment'
      AND COALESCE(pci.item_uid, '') = ''
      AND pci.quantity > 0
    ORDER BY pci.id
  LOOP
    new_uid := 'eq-' || rec.player_id || '-' || floor(extract(epoch FROM clock_timestamp()) * 1000000)::bigint || '-' || rec.id;
    INSERT INTO equipment_instance (
      item_uid, player_id, item_id, enhance_level, star_level,
      durability, max_durability, bind_type, state
    ) VALUES (
      new_uid, rec.player_id, rec.item_id, 0, 0,
      0, 0, rec.bind_type, 'bag'
    );
    UPDATE player_container_item
    SET item_uid = new_uid
    WHERE id = rec.id;
  END LOOP;
END $$;
