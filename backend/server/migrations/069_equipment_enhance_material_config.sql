-- 069_equipment_enhance_material_config.sql
-- 强化/锻造材料专属效果：成功率修正与失败惩罚，按 item_id 配置。

CREATE TABLE IF NOT EXISTS equipment_enhance_material_config (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id) ON DELETE CASCADE,
  success_rate_mode VARCHAR(16) NOT NULL DEFAULT 'base',
  success_rate_bonus_pct INTEGER NOT NULL DEFAULT 0,
  success_rate_override_pct INTEGER NOT NULL DEFAULT 0,
  guaranteed_success BOOLEAN NOT NULL DEFAULT FALSE,
  failure_penalty VARCHAR(16) NOT NULL DEFAULT 'damage',
  failure_level_delta INTEGER NOT NULL DEFAULT 0,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_enhance_material_success_mode CHECK (success_rate_mode IN ('base', 'bonus', 'override')),
  CONSTRAINT chk_enhance_material_bonus CHECK (success_rate_bonus_pct >= 0 AND success_rate_bonus_pct <= 100),
  CONSTRAINT chk_enhance_material_override CHECK (success_rate_override_pct >= 0 AND success_rate_override_pct <= 100),
  CONSTRAINT chk_enhance_material_failure_penalty CHECK (failure_penalty IN ('damage', 'none', 'level_down')),
  CONSTRAINT chk_enhance_material_level_delta CHECK (failure_level_delta >= 0 AND failure_level_delta <= 15)
);

DROP TRIGGER IF EXISTS trg_equipment_enhance_material_config_updated_at ON equipment_enhance_material_config;
CREATE TRIGGER trg_equipment_enhance_material_config_updated_at
BEFORE UPDATE ON equipment_enhance_material_config
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE equipment_enhance_material_config IS
  '强化材料（item_sub_type=equipment_enhance）专属锻造效果：成功率模式/失败惩罚；未配置时服务端按 base+damage 兜底';

-- 3201 强化石默认：沿用全局成功率，失败损坏（与当前运行时一致）
INSERT INTO equipment_enhance_material_config (
  item_id, success_rate_mode, success_rate_bonus_pct, success_rate_override_pct,
  guaranteed_success, failure_penalty, failure_level_delta, description, status
)
VALUES (
  3201, 'base', 0, 0, FALSE, 'damage', 0,
  '标准锻造石：使用全局成功率，失败时装备损坏。', 1
)
ON CONFLICT (item_id) DO UPDATE SET
  success_rate_mode = EXCLUDED.success_rate_mode,
  failure_penalty = EXCLUDED.failure_penalty,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;
