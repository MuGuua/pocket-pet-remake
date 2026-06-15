-- 玩家当前形象 ID：世界场景与战斗场景共用，对应客户端 unit_skins/{skin_id}.tres。

ALTER TABLE player
  ADD COLUMN IF NOT EXISTS skin_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN player.skin_id IS '玩家当前形象资源 ID，对应 client/resources/battle/unit_skins/{skin_id}.tres';

UPDATE player
SET skin_id = '初始形象男_001'
WHERE skin_id = '' OR skin_id IS NULL;
