-- 053_pet_combat_stat_caps.sql
-- 宠物扩展战斗属性字段与各项数值封顶配置。

ALTER TABLE player_pet
  ADD COLUMN IF NOT EXISTS spirit INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS spirit_max INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS hit_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS dodge_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS crit_rate_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS crit_dmg_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS physical_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reverse_physical_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS skill_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reverse_skill_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS confusion_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS sleep_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS paralysis_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS seal_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS curse_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS crit_dmg_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS crit_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS character_resist_pct INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS pet_resist_pct INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN player_pet.spirit IS '当前精力';
COMMENT ON COLUMN player_pet.spirit_max IS '精力上限';
COMMENT ON COLUMN player_pet.reverse_physical_resist_pct IS '逆物理攻击抗性（逆物）';
COMMENT ON COLUMN player_pet.reverse_skill_resist_pct IS '逆技能攻击抗性（逆技）';

CREATE TABLE IF NOT EXISTS pet_combat_stat_cap (
  stat_key VARCHAR(64) PRIMARY KEY,
  cap_value INTEGER NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_pet_combat_stat_cap_value CHECK (cap_value >= 0)
);

CREATE TRIGGER trg_pet_combat_stat_cap_updated_at
BEFORE UPDATE ON pet_combat_stat_cap
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO pet_combat_stat_cap (stat_key, cap_value, description, status)
VALUES
  ('hp_max', 1500000, '生命值上限', 1),
  ('spirit', 1000, '精力', 1),
  ('spirit_max', 1000, '精力上限', 1),
  ('atk', 250000, '攻击', 1),
  ('def', 250000, '防御', 1),
  ('spd', 30000, '速度', 1),
  ('mana', 50000, '法力', 1),
  ('hit_pct', 250, '命中', 1),
  ('dodge_pct', 200, '闪避', 1),
  ('crit_rate_pct', 150, '致命', 1),
  ('crit_dmg_pct', 2000, '爆伤（百分比）', 1),
  ('physical_resist_pct', 150, '物理攻击抗性', 1),
  ('reverse_physical_resist_pct', 100, '逆物理攻击抗性', 1),
  ('skill_resist_pct', 150, '技能攻击抗性', 1),
  ('reverse_skill_resist_pct', 100, '逆技能攻击抗性', 1),
  ('confusion_resist_pct', 700, '混乱抗性', 1),
  ('sleep_resist_pct', 700, '昏睡抗性', 1),
  ('paralysis_resist_pct', 700, '麻痹抗性', 1),
  ('seal_resist_pct', 700, '封印抗性', 1),
  ('curse_resist_pct', 700, '诅咒抗性', 1),
  ('crit_dmg_resist_pct', 1000, '抗爆伤', 1),
  ('crit_resist_pct', 100, '抗致命', 1),
  ('character_resist_pct', 100, '抗人物', 1),
  ('pet_resist_pct', 100, '抗宠物', 1)
ON CONFLICT (stat_key) DO UPDATE SET
  cap_value = EXCLUDED.cap_value,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
