-- 128_seed_medicine_pouch_items.sql
-- 配置两款药囊装备，并给测试账号（player 10001）背包各发一件，供验证「战后自动恢复」。
--
-- 药囊规则（《装备系统》§14）：佩戴药囊时，战斗结束后自动回满人物与宠物的生命/精力；
-- 未佩戴则保留上一场剩余值。具体恢复哪些项由 item_medicine_pouch_extra 的开关决定。
--
--   900101 初级药囊：只回满本场出战宠物的生命。
--   900102 高级药囊：回满人物生命/精力/体力，以及整个出战编队宠物的生命/精力。
--
-- 设计说明：
--   1. 物品 ID 使用 9001xx 专属号段，并以 item_code 作为幂等判断依据，可重复执行。
--   2. 药囊只占装备槽 medicine_pouch，不提供任何战斗属性（SumEquippedBonus 会跳过药囊）。
--   3. 本迁移只把药囊发进背包，不直接写 player_equipment_slot。
--      佩戴必须走游戏内「佩戴」（WS 2072），由服务端同事务重算战斗属性；
--      直接写佩戴表会让 player 的战斗字段与装备状态不一致。
--   4. 背包格子自动选择第一个空格；背包已满时跳过发放并 RAISE NOTICE，不中断迁移。
--
-- 回滚：
--   DELETE FROM player_container_item WHERE source_type = 'migration_128';
--   DELETE FROM equipment_instance WHERE item_uid LIKE 'mig128_%';
--   DELETE FROM item_medicine_pouch_extra WHERE item_id IN (900101, 900102);
--   DELETE FROM item_equipment_extra WHERE item_id IN (900101, 900102);
--   DELETE FROM item_definition WHERE item_id IN (900101, 900102);
--   （若药囊已被佩戴，需先在游戏内卸下，或先删除 player_equipment_slot 对应行。）

BEGIN;

-- 1. 物品模板。desc 使用富文本，客户端详情面板直接渲染。
INSERT INTO item_definition (
  item_id, item_code, item_name, item_type, item_sub_type, quality, rarity, icon, "desc",
  max_stack, occupy_slots, auto_merge, usable, required_level, bind_type,
  can_sell, can_drop, can_store, can_trade, buy_price_copper, sell_price_copper, price_type, is_enabled
)
SELECT v.item_id, v.item_code, v.item_name, 'equipment', 'medicine_pouch', v.quality, v.quality, '', v.item_desc,
       1, 1, FALSE, FALSE, 1, 'none',
       TRUE, FALSE, TRUE, FALSE, 0, v.sell_price, 'base_coin', TRUE
FROM (VALUES
  (900101::BIGINT, 'medicine_pouch_basic', '初级药囊', 1,
   E'随身携带的简易药囊，装着几包止血草药。\n佩戴后效果：\n\n战斗结束后，本场出战宠物生命 [color=green]回满[/color]', 500::BIGINT),
  (900102::BIGINT, 'medicine_pouch_advanced', '高级药囊', 3,
   E'由宠物学校医务室特制的高级药囊。\n佩戴后效果：\n\n战斗结束后，人物生命、精力、体力 [color=green]回满[/color]\n出战编队全部宠物生命、精力 [color=blue]回满[/color]', 3000::BIGINT)
) AS v(item_id, item_code, item_name, quality, item_desc, sell_price)
WHERE NOT EXISTS (SELECT 1 FROM item_definition d WHERE d.item_id = v.item_id OR d.item_code = v.item_code);

-- 2. 装备扩展：部位为药囊，不可强化，无镶嵌孔，无战斗属性。
INSERT INTO item_equipment_extra (item_id, equip_slot, player_only, can_enhance, max_enhance_level, socket_count)
SELECT d.item_id, 'medicine_pouch', TRUE, FALSE, 0, 0
FROM item_definition d
WHERE d.item_id IN (900101, 900102)
ON CONFLICT (item_id) DO NOTHING;

-- 3. 药囊恢复配置。
INSERT INTO item_medicine_pouch_extra (
  item_id, restore_player_hp, restore_player_spirit, restore_player_vigor,
  restore_pet_hp, restore_pet_spirit, restore_lineup_pets
)
SELECT v.item_id, v.p_hp, v.p_spirit, v.p_vigor, v.pet_hp, v.pet_spirit, v.lineup
FROM (VALUES
  (900101::BIGINT, FALSE, FALSE, FALSE, TRUE, FALSE, FALSE),
  (900102::BIGINT, TRUE,  TRUE,  TRUE,  TRUE, TRUE,  TRUE)
) AS v(item_id, p_hp, p_spirit, p_vigor, pet_hp, pet_spirit, lineup)
WHERE EXISTS (SELECT 1 FROM item_definition d WHERE d.item_id = v.item_id)
ON CONFLICT (item_id) DO UPDATE SET
  restore_player_hp     = EXCLUDED.restore_player_hp,
  restore_player_spirit = EXCLUDED.restore_player_spirit,
  restore_player_vigor  = EXCLUDED.restore_player_vigor,
  restore_pet_hp        = EXCLUDED.restore_pet_hp,
  restore_pet_spirit    = EXCLUDED.restore_pet_spirit,
  restore_lineup_pets   = EXCLUDED.restore_lineup_pets,
  updated_at            = CURRENT_TIMESTAMP;

-- 4. 给测试账号 10001 背包各发一件（装备需要 equipment_instance 实例）。
DO $$
DECLARE
  v_player_id BIGINT := 10001;
  v_item_id   BIGINT;
  v_item_uid  VARCHAR(64);
  v_slot      INTEGER;
  v_capacity  INTEGER;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM player WHERE id = v_player_id) THEN
    RAISE NOTICE 'migration 128: player % not found, skip granting pouches', v_player_id;
    RETURN;
  END IF;

  SELECT COALESCE(MAX(capacity), 30) INTO v_capacity
  FROM player_container WHERE player_id = v_player_id AND container_type = 'bag';

  FOREACH v_item_id IN ARRAY ARRAY[900101, 900102]::BIGINT[] LOOP
    v_item_uid := 'mig128_' || v_player_id || '_' || v_item_id;
    CONTINUE WHEN NOT EXISTS (SELECT 1 FROM item_definition WHERE item_id = v_item_id);
    -- 已发过（在背包或已佩戴）则跳过，保证可重复执行。
    CONTINUE WHEN EXISTS (SELECT 1 FROM equipment_instance WHERE item_uid = v_item_uid);

    SELECT s INTO v_slot
    FROM generate_series(1, v_capacity) AS s
    WHERE NOT EXISTS (
      SELECT 1 FROM player_container_item pci
      WHERE pci.player_id = v_player_id AND pci.container_type = 'bag' AND pci.slot_index = s
    )
    ORDER BY s
    LIMIT 1;

    IF v_slot IS NULL THEN
      RAISE NOTICE 'migration 128: bag of player % is full, skip item %', v_player_id, v_item_id;
      CONTINUE;
    END IF;

    INSERT INTO equipment_instance (item_uid, player_id, item_id, bind_type, state)
    VALUES (v_item_uid, v_player_id, v_item_id, 'none', 'bag');

    INSERT INTO player_container_item (
      player_id, container_type, slot_index, item_id, item_uid, quantity, is_bound, source_type, source_ref_id
    ) VALUES (v_player_id, 'bag', v_slot, v_item_id, v_item_uid, 1, FALSE, 'migration_128', 128);

    RAISE NOTICE 'migration 128: granted item % to player % bag slot %', v_item_id, v_player_id, v_slot;
  END LOOP;
END $$;

COMMIT;
