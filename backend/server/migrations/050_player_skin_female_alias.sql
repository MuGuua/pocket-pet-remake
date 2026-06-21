-- 女性初始形象 skin_id 与客户端 unit_skins 资源对齐；历史测试数据可能仍使用旧 ID「决斗者_001」。
UPDATE player
SET skin_id = '初始形象女_002'
WHERE skin_id = '决斗者_001';

COMMENT ON COLUMN player.skin_id IS '玩家当前形象资源 ID，对应 client/resources/battle/unit_skins/{skin_id}.tres';
