-- 061_skill_control_power.sql
-- 技能控制双体系：概率字段（无视抗性）与威力字段（对抗控制抗性）。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS seal_power INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS control_power INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN skill_definition.seal_power IS '封印控制威力；>0 时与目标 seal 抗性对抗，忽略 seal_chance_pct 的概率语义';
COMMENT ON COLUMN skill_definition.control_power IS '通用控制威力；>0 时与目标对应 control_status 抗性对抗，忽略 control_chance_pct';
