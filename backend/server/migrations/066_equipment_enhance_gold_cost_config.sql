-- 066_equipment_enhance_gold_cost_config.sql
-- 装备强化铜币消耗全局公式：基础值 + 每级固定递增或百分比复合递增。

CREATE TABLE IF NOT EXISTS equipment_enhance_gold_cost_config (
  config_id SMALLINT PRIMARY KEY DEFAULT 1,
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  base_copper BIGINT NOT NULL DEFAULT 500,
  increment_mode VARCHAR(16) NOT NULL DEFAULT 'fixed',
  increment_fixed BIGINT NOT NULL DEFAULT 500,
  increment_percent INTEGER NOT NULL DEFAULT 0,
  description TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_enhance_gold_cost_config_singleton CHECK (config_id = 1),
  CONSTRAINT chk_enhance_gold_cost_config_base CHECK (base_copper >= 0),
  CONSTRAINT chk_enhance_gold_cost_config_mode CHECK (increment_mode IN ('fixed', 'percent')),
  CONSTRAINT chk_enhance_gold_cost_config_fixed CHECK (increment_fixed >= 0),
  CONSTRAINT chk_enhance_gold_cost_config_percent CHECK (increment_percent >= 0 AND increment_percent <= 1000)
);

DROP TRIGGER IF EXISTS trg_equipment_enhance_gold_cost_config_updated_at ON equipment_enhance_gold_cost_config;
CREATE TRIGGER trg_equipment_enhance_gold_cost_config_updated_at
BEFORE UPDATE ON equipment_enhance_gold_cost_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE equipment_enhance_gold_cost_config IS
  '人物装备强化铜币消耗全局公式；目标等级 N 的消耗由服务端按 base + 递增规则计算，覆盖 equipment_enhance_cost.cost_gold_copper 静态值';

INSERT INTO equipment_enhance_gold_cost_config (
  config_id,
  is_enabled,
  base_copper,
  increment_mode,
  increment_fixed,
  increment_percent,
  description
) VALUES (
  1,
  TRUE,
  500,
  'fixed',
  500,
  0,
  '默认：+1 消耗 500 铜，每升 1 级固定 +500 铜'
)
ON CONFLICT (config_id) DO NOTHING;
