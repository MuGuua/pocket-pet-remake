-- 技能品质只控制客户端按钮边框，不参与战斗公式与释放规则。
ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS skill_quality VARCHAR(16) NOT NULL DEFAULT 'normal';

UPDATE skill_definition
SET skill_quality = CASE
  WHEN skill_name LIKE '%绝世%' THEN 'peerless'
  WHEN skill_name LIKE '%圣技%' THEN 'sacred'
  WHEN skill_name LIKE '%魂技%' THEN 'soul'
  WHEN skill_name LIKE '%神技%' THEN 'divine'
  ELSE 'normal'
END;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'skill_definition_quality_check'
  ) THEN
    ALTER TABLE skill_definition
      ADD CONSTRAINT skill_definition_quality_check
      CHECK (skill_quality IN ('normal', 'divine', 'soul', 'sacred', 'peerless'));
  END IF;
END $$;

COMMENT ON COLUMN skill_definition.skill_quality IS '技能展示品质：normal普通、divine神技、soul魂技、sacred圣技、peerless绝世';
