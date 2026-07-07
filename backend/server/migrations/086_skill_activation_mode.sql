-- 086_skill_activation_mode.sql
-- 系统技能库显式区分“主动技能 / 被动技能”。
-- skill_type 继续描述攻击/治疗/辅助等效果类型；
-- activation_mode 则描述该技能是否可以被玩家主动释放。

ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS activation_mode VARCHAR(16) NOT NULL DEFAULT 'active';

COMMENT ON COLUMN skill_definition.activation_mode IS '技能释放方式：active=主动技能，passive=被动技能；被动技能不出现在战斗可选列表中';

UPDATE skill_definition
SET activation_mode = 'passive',
    target_type = 'self',
    target_count = 0,
    preferred_target_hp = '',
    energy_cost = 0
WHERE activation_mode <> 'passive'
  AND skill_type = 'support'
  AND energy_cost = 0;
