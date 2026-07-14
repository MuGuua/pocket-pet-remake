-- 地图 NPC 的实体 ID 由数据库序列生成，编码固定使用 npc_{entity_id}。
-- 序列从当前最大实体 ID 之后继续，避免与既有 NPC、玩家实体配置冲突。

CREATE SEQUENCE IF NOT EXISTS world_entity_definition_entity_id_seq;

SELECT setval(
  'world_entity_definition_entity_id_seq',
  GREATEST(COALESCE((SELECT MAX(entity_id) FROM world_entity_definition), 0), 90000),
  TRUE
);

COMMENT ON SEQUENCE world_entity_definition_entity_id_seq IS '后台创建地图 NPC 时使用的实体 ID 序列';
