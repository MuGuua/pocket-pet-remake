-- 087_skill_passive_attribute_bonus.sql
-- 为系统技能模板增加“永久被动属性加成”显式配置字段，
-- 让后台可以直接配置属性加成，而不是继续依赖技能名前缀推断。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS passive_attr_key VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS passive_attr_mode VARCHAR(16) NOT NULL DEFAULT '';

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS passive_attr_value INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN skill_definition.passive_attr_key IS
  '永久被动属性加成字段；为空表示不使用显式属性加成。';

COMMENT ON COLUMN skill_definition.passive_attr_mode IS
  '永久被动属性加成方式：flat=固定值，percent=百分比；为空表示不使用显式属性加成。';

COMMENT ON COLUMN skill_definition.passive_attr_value IS
  '永久被动属性加成数值；解释方式由 passive_attr_key + passive_attr_mode 共同决定。';
