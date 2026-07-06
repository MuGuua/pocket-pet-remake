-- 为地图暗雷增加多编队配置。旧的 spawn_monster_ids 继续保留作兼容字段，formations 承载权重与每个编队的怪物槽位。
ALTER TABLE scene_wild_encounter
  ADD COLUMN IF NOT EXISTS formations JSONB NOT NULL DEFAULT '[]'::jsonb;

-- 将旧配置回填为一个默认编队，保证迁移后现有地图暗雷仍能正常开战。
UPDATE scene_wild_encounter
SET formations = jsonb_build_array(jsonb_build_object(
  'formation_name', '默认编队',
  'weight', 10000,
  'spawn_monster_ids', spawn_monster_ids
))
WHERE jsonb_array_length(formations) = 0
  AND jsonb_typeof(spawn_monster_ids) = 'array'
  AND jsonb_array_length(spawn_monster_ids) > 0;
