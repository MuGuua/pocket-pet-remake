-- 077_player_pet_custom_name.sql
-- 玩家宠物实例支持自定义展示名，供客户端与运营后台展示。

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS custom_name VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN player_pet.custom_name IS '玩家自定义宠物名；为空时客户端回退为系统模板名';
