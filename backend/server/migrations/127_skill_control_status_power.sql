-- 127_skill_control_status_power.sql
-- 为控制状态增加「状态强度」配置，让同一个状态可以被不同技能定义成不同强度。
--
-- 背景：《战斗系统规则》§6 规定麻痹使物理抗性、技能抗性、抗爆伤、命中、闪避降低。
-- 具体降低到百分之多少、影响哪几项，应当由施加该状态的技能自行决定，
-- 而不是在状态实现里写死。
--
-- control_status_power 的语义随 control_status_id 而定：
--   - 麻痹（11）：抗性降至百分之多少，取值 1~99，例如 50 表示各项抗性 ×50%。
--                 0 表示沿用规则默认值（50%）。
--   - 其它控制状态：暂不消费该字段，保留给后续扩展。
--
-- 该列默认 0，对既有技能配置无行为影响。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS control_status_power INTEGER NOT NULL DEFAULT 0;

ALTER TABLE skill_definition
  DROP CONSTRAINT IF EXISTS ck_skill_definition_control_status_power;

ALTER TABLE skill_definition
  ADD CONSTRAINT ck_skill_definition_control_status_power
  CHECK (control_status_power >= 0 AND control_status_power <= 100);

COMMENT ON COLUMN skill_definition.control_status_power IS
  '控制状态强度参数；麻痹表示抗性降至百分之多少（0=沿用规则默认 50%）';
