-- 013_bag_wallet_foundation.sql
--
-- 本迁移为背包、仓库、装备实例、钱包与日志体系补充新的持久化底座。
-- 当前仓库里已经存在旧版 `player_item` 与 `player.gold` 字段，因此这里采用“新增结构 + 回填初始化”的保守方案：
-- 1. 不删除旧表，也不直接改旧字段语义，避免影响当前已在线的链路。
-- 2. 新代码切换完成前，旧结构仍可继续被当前业务使用。
-- 3. 用户可先执行本迁移，再分阶段切服务端读写逻辑。

-- 统一物品模板主表。
-- 所有正式物品都应先注册到该表，再由背包、仓库、掉落、任务奖励、商店等系统引用。
CREATE TABLE IF NOT EXISTS item_definition (
  item_id BIGINT PRIMARY KEY,
  item_code VARCHAR(64) NOT NULL UNIQUE,
  item_name VARCHAR(128) NOT NULL,
  item_type VARCHAR(32) NOT NULL,
  item_sub_type VARCHAR(32) NOT NULL DEFAULT '',
  quality INTEGER NOT NULL DEFAULT 1,
  rarity INTEGER NOT NULL DEFAULT 1,
  icon VARCHAR(255) NOT NULL DEFAULT '',
  "desc" TEXT NOT NULL DEFAULT '',
  max_stack INTEGER NOT NULL DEFAULT 1,
  occupy_slots INTEGER NOT NULL DEFAULT 1,
  auto_merge BOOLEAN NOT NULL DEFAULT TRUE,
  sort_weight INTEGER NOT NULL DEFAULT 0,
  usable BOOLEAN NOT NULL DEFAULT FALSE,
  use_scope VARCHAR(32) NOT NULL DEFAULT '',
  target_type VARCHAR(32) NOT NULL DEFAULT '',
  required_level INTEGER NOT NULL DEFAULT 0,
  required_scene_id BIGINT NOT NULL DEFAULT 0,
  bind_type VARCHAR(32) NOT NULL DEFAULT 'none',
  can_sell BOOLEAN NOT NULL DEFAULT FALSE,
  can_drop BOOLEAN NOT NULL DEFAULT FALSE,
  can_store BOOLEAN NOT NULL DEFAULT TRUE,
  can_trade BOOLEAN NOT NULL DEFAULT FALSE,
  expire_at_rule VARCHAR(64) NOT NULL DEFAULT '',
  effect_type VARCHAR(32) NOT NULL DEFAULT '',
  effect_value BIGINT NOT NULL DEFAULT 0,
  effect_params_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  buy_price_copper BIGINT NOT NULL DEFAULT 0,
  sell_price_copper BIGINT NOT NULL DEFAULT 0,
  recycle_price_copper BIGINT NOT NULL DEFAULT 0,
  price_type VARCHAR(32) NOT NULL DEFAULT 'base_coin',
  is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_item_definition_max_stack CHECK (max_stack >= 1),
  CONSTRAINT chk_item_definition_occupy_slots CHECK (occupy_slots >= 1),
  CONSTRAINT chk_item_definition_buy_price CHECK (buy_price_copper >= 0),
  CONSTRAINT chk_item_definition_sell_price CHECK (sell_price_copper >= 0),
  CONSTRAINT chk_item_definition_recycle_price CHECK (recycle_price_copper >= 0)
);

CREATE INDEX IF NOT EXISTS idx_item_definition_type ON item_definition (item_type);
CREATE INDEX IF NOT EXISTS idx_item_definition_sub_type ON item_definition (item_sub_type);
CREATE INDEX IF NOT EXISTS idx_item_definition_enabled ON item_definition (is_enabled);

-- 装备模板扩展表。
-- 只存装备模板的静态特有字段，不重复保存通用描述、价格或堆叠规则。
CREATE TABLE IF NOT EXISTS item_equipment_extra (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id),
  equip_slot VARCHAR(32) NOT NULL,
  base_hp BIGINT NOT NULL DEFAULT 0,
  base_mana BIGINT NOT NULL DEFAULT 0,
  base_atk BIGINT NOT NULL DEFAULT 0,
  base_def BIGINT NOT NULL DEFAULT 0,
  base_spd BIGINT NOT NULL DEFAULT 0,
  career_limit VARCHAR(32) NOT NULL DEFAULT '',
  pet_only BOOLEAN NOT NULL DEFAULT FALSE,
  player_only BOOLEAN NOT NULL DEFAULT TRUE,
  extra_rule_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 宝箱模板扩展表。
-- 奖励池、抽取规则、每日限制等与宝箱强相关的字段都收敛到这里，避免把主表做得过重。
CREATE TABLE IF NOT EXISTS item_box_extra (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id),
  reward_pool_id BIGINT NOT NULL DEFAULT 0,
  select_mode VARCHAR(32) NOT NULL DEFAULT 'random',
  box_open_rule VARCHAR(64) NOT NULL DEFAULT '',
  daily_limit INTEGER NOT NULL DEFAULT 0,
  extra_rule_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_item_box_extra_daily_limit CHECK (daily_limit >= 0)
);

-- 功能道具模板扩展表。
-- 当前主要用于扩容券等系统性道具，后续也可承接改名卡、重置券之类的配置。
CREATE TABLE IF NOT EXISTS item_functional_extra (
  item_id BIGINT PRIMARY KEY REFERENCES item_definition(item_id),
  function_type VARCHAR(32) NOT NULL,
  expand_target VARCHAR(32) NOT NULL DEFAULT '',
  expand_slots INTEGER NOT NULL DEFAULT 0,
  target_rule_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_item_functional_extra_expand_slots CHECK (expand_slots >= 0)
);

-- 玩家容器配置表。
-- 这里一条记录表示某个玩家的一类容器，比如个人背包或个人仓库。
CREATE TABLE IF NOT EXISTS player_container (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  container_type VARCHAR(32) NOT NULL,
  capacity INTEGER NOT NULL DEFAULT 30,
  max_capacity INTEGER NOT NULL DEFAULT 300,
  version BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_container UNIQUE (player_id, container_type),
  CONSTRAINT chk_player_container_capacity_non_negative CHECK (capacity >= 0),
  CONSTRAINT chk_player_container_capacity_max CHECK (capacity <= max_capacity),
  CONSTRAINT chk_player_container_max_capacity CHECK (max_capacity >= 30 AND max_capacity <= 300),
  CONSTRAINT chk_player_container_type CHECK (container_type IN ('bag', 'warehouse'))
);

CREATE INDEX IF NOT EXISTS idx_player_container_player_id ON player_container (player_id);

-- 装备实例表。
-- 任何需要强化、耐久、随机词条、绑定差异的装备，都应通过 item_uid 在该表留存个体状态。
CREATE TABLE IF NOT EXISTS equipment_instance (
  item_uid VARCHAR(64) PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  item_id BIGINT NOT NULL REFERENCES item_definition(item_id),
  enhance_level INTEGER NOT NULL DEFAULT 0,
  star_level INTEGER NOT NULL DEFAULT 0,
  durability BIGINT NOT NULL DEFAULT 0,
  max_durability BIGINT NOT NULL DEFAULT 0,
  bind_type VARCHAR(32) NOT NULL DEFAULT 'none',
  extra_attrs_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  special_effect_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  state VARCHAR(32) NOT NULL DEFAULT 'bag',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_equipment_instance_enhance_level CHECK (enhance_level >= 0),
  CONSTRAINT chk_equipment_instance_star_level CHECK (star_level >= 0),
  CONSTRAINT chk_equipment_instance_durability CHECK (durability >= 0),
  CONSTRAINT chk_equipment_instance_max_durability CHECK (max_durability >= 0),
  CONSTRAINT chk_equipment_instance_durability_order CHECK (durability <= max_durability),
  CONSTRAINT chk_equipment_instance_state CHECK (state IN ('bag', 'warehouse', 'equipped', 'locked', 'deleted'))
);

CREATE INDEX IF NOT EXISTS idx_equipment_instance_player_id ON equipment_instance (player_id);
CREATE INDEX IF NOT EXISTS idx_equipment_instance_player_item ON equipment_instance (player_id, item_id);
CREATE INDEX IF NOT EXISTS idx_equipment_instance_player_state ON equipment_instance (player_id, state);

-- 玩家容器格子表。
-- 普通堆叠物品可只依赖 item_id + quantity；装备等个体化物品则使用 item_uid 指向 equipment_instance。
CREATE TABLE IF NOT EXISTS player_container_item (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  container_type VARCHAR(32) NOT NULL,
  slot_index INTEGER NOT NULL,
  item_id BIGINT NOT NULL REFERENCES item_definition(item_id),
  item_uid VARCHAR(64) NULL REFERENCES equipment_instance(item_uid),
  quantity BIGINT NOT NULL DEFAULT 1,
  is_bound BOOLEAN NOT NULL DEFAULT FALSE,
  expire_at TIMESTAMPTZ NULL,
  instance_data_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  source_type VARCHAR(32) NOT NULL DEFAULT '',
  source_ref_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_player_container_item_slot UNIQUE (player_id, container_type, slot_index),
  CONSTRAINT chk_player_container_item_slot_index CHECK (slot_index >= 1),
  CONSTRAINT chk_player_container_item_quantity CHECK (quantity >= 1),
  CONSTRAINT chk_player_container_item_type CHECK (container_type IN ('bag', 'warehouse'))
);

CREATE INDEX IF NOT EXISTS idx_player_container_item_player_item
ON player_container_item (player_id, container_type, item_id);

CREATE INDEX IF NOT EXISTS idx_player_container_item_uid ON player_container_item (item_uid);
CREATE INDEX IF NOT EXISTS idx_player_container_item_updated
ON player_container_item (player_id, container_type, updated_at);

-- 玩家钱包表。
-- 新体系统一以总铜币存储，展示时再拆分为金币/银币/铜币，避免多列进位与借位逻辑带来的歧义。
CREATE TABLE IF NOT EXISTS player_wallet (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL UNIQUE REFERENCES player(id),
  currency_copper_total BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_player_wallet_currency_non_negative CHECK (currency_copper_total >= 0)
);

-- 物品流水日志。
-- 后续战斗掉落、任务奖励、后台补发、出售、使用道具都应写入这里，便于审计与回滚排查。
CREATE TABLE IF NOT EXISTS item_change_log (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  container_type VARCHAR(32) NOT NULL DEFAULT '',
  slot_index INTEGER NOT NULL DEFAULT 0,
  change_type VARCHAR(32) NOT NULL,
  item_id BIGINT NOT NULL,
  item_uid VARCHAR(64) NULL,
  before_qty BIGINT NOT NULL DEFAULT 0,
  change_qty BIGINT NOT NULL DEFAULT 0,
  after_qty BIGINT NOT NULL DEFAULT 0,
  reason_type VARCHAR(32) NOT NULL,
  reason_ref_id BIGINT NOT NULL DEFAULT 0,
  operator_type VARCHAR(32) NOT NULL DEFAULT 'system',
  operator_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_item_change_log_player_created_at
ON item_change_log (player_id, created_at);

CREATE INDEX IF NOT EXISTS idx_item_change_log_item_created_at
ON item_change_log (item_id, created_at);

CREATE INDEX IF NOT EXISTS idx_item_change_log_reason_created_at
ON item_change_log (reason_type, created_at);

-- 容量变更日志。
-- 背包扩容券和仓库扩容券等行为都应把扩容前后容量记录下来。
CREATE TABLE IF NOT EXISTS container_expand_log (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  container_type VARCHAR(32) NOT NULL,
  before_capacity INTEGER NOT NULL,
  expand_slots INTEGER NOT NULL,
  after_capacity INTEGER NOT NULL,
  item_id BIGINT NOT NULL DEFAULT 0,
  reason_type VARCHAR(32) NOT NULL,
  reason_ref_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_container_expand_log_expand_slots CHECK (expand_slots >= 0)
);

CREATE INDEX IF NOT EXISTS idx_container_expand_log_player_created_at
ON container_expand_log (player_id, created_at);

-- 货币流水日志。
-- 当前统一记录总铜币前后变化，后续客户端如需展示金/银/铜，只通过服务层拆分，不在日志层重复存三列。
CREATE TABLE IF NOT EXISTS currency_change_log (
  id BIGSERIAL PRIMARY KEY,
  player_id BIGINT NOT NULL REFERENCES player(id),
  currency_type VARCHAR(32) NOT NULL DEFAULT 'base_coin',
  before_total_copper BIGINT NOT NULL DEFAULT 0,
  change_total_copper BIGINT NOT NULL DEFAULT 0,
  after_total_copper BIGINT NOT NULL DEFAULT 0,
  reason_type VARCHAR(32) NOT NULL,
  reason_ref_id BIGINT NOT NULL DEFAULT 0,
  operator_type VARCHAR(32) NOT NULL DEFAULT 'system',
  operator_id BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_currency_change_log_player_created_at
ON currency_change_log (player_id, created_at);

CREATE INDEX IF NOT EXISTS idx_currency_change_log_reason_created_at
ON currency_change_log (reason_type, created_at);

-- 为新增表补 updated_at 自动刷新触发器。
-- 这里复用 001_init_schema.sql 中已经创建的 set_updated_at() 函数，保持全库行为一致。
DROP TRIGGER IF EXISTS set_item_definition_updated_at ON item_definition;
CREATE TRIGGER set_item_definition_updated_at
BEFORE UPDATE ON item_definition
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_item_equipment_extra_updated_at ON item_equipment_extra;
CREATE TRIGGER set_item_equipment_extra_updated_at
BEFORE UPDATE ON item_equipment_extra
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_item_box_extra_updated_at ON item_box_extra;
CREATE TRIGGER set_item_box_extra_updated_at
BEFORE UPDATE ON item_box_extra
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_item_functional_extra_updated_at ON item_functional_extra;
CREATE TRIGGER set_item_functional_extra_updated_at
BEFORE UPDATE ON item_functional_extra
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_player_container_updated_at ON player_container;
CREATE TRIGGER set_player_container_updated_at
BEFORE UPDATE ON player_container
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_equipment_instance_updated_at ON equipment_instance;
CREATE TRIGGER set_equipment_instance_updated_at
BEFORE UPDATE ON equipment_instance
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_player_container_item_updated_at ON player_container_item;
CREATE TRIGGER set_player_container_item_updated_at
BEFORE UPDATE ON player_container_item
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS set_player_wallet_updated_at ON player_wallet;
CREATE TRIGGER set_player_wallet_updated_at
BEFORE UPDATE ON player_wallet
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- 为已有玩家初始化两类容器。
-- 这里使用 ON CONFLICT，保证迁移反复执行或在测试环境重跑时仍保持幂等。
INSERT INTO player_container (player_id, container_type, capacity, max_capacity)
SELECT p.id, 'bag', 30, 300
FROM player p
ON CONFLICT (player_id, container_type) DO NOTHING;

INSERT INTO player_container (player_id, container_type, capacity, max_capacity)
SELECT p.id, 'warehouse', 30, 300
FROM player p
ON CONFLICT (player_id, container_type) DO NOTHING;

-- 为已有玩家初始化钱包。
-- 旧版 player.gold 在当前项目里代表“金币”概念，因此这里按 1 金币 = 1,000,000 铜币回填到新钱包。
-- 若后续确认旧 gold 实际语义不同，用户执行迁移前可按正式数据口径调整这段回填逻辑。
INSERT INTO player_wallet (player_id, currency_copper_total)
SELECT p.id, p.gold * 1000000
FROM player p
ON CONFLICT (player_id) DO NOTHING;

-- 旧版 player_item 没有格子概念，因此这里使用按 item_id 升序生成的顺序槽位做一次保守迁移。
-- 由于新结构为 item_id 增加了模板外键，这里只迁移已经在 item_definition 中注册过的物品，避免在模板尚未补齐时迁移失败。
-- 后续若要保留更精细的排序、处理未注册模板物品或做容量超限校验，应在服务端正式切换前补脚本二次整理。
WITH bag_seed AS (
  SELECT
    pi.player_id,
    'bag'::VARCHAR(32) AS container_type,
    ROW_NUMBER() OVER (PARTITION BY pi.player_id ORDER BY pi.item_id, pi.id) AS slot_index,
    pi.item_id::BIGINT AS item_id,
    pi.count AS quantity,
    pi.created_at,
    pi.updated_at
  FROM player_item pi
  INNER JOIN item_definition idef ON idef.item_id = pi.item_id::BIGINT
  WHERE pi.count > 0
)
INSERT INTO player_container_item (
  player_id,
  container_type,
  slot_index,
  item_id,
  quantity,
  created_at,
  updated_at
)
SELECT
  b.player_id,
  b.container_type,
  b.slot_index,
  b.item_id,
  b.quantity,
  b.created_at,
  b.updated_at
FROM bag_seed b
WHERE NOT EXISTS (
  SELECT 1
  FROM player_container_item pci
  WHERE pci.player_id = b.player_id
    AND pci.container_type = b.container_type
    AND pci.slot_index = b.slot_index
);
