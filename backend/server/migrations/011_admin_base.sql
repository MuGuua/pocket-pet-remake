CREATE TABLE IF NOT EXISTS admin_user (
  id BIGSERIAL PRIMARY KEY,
  account_name VARCHAR(64) NOT NULL UNIQUE,
  password_hash CHAR(64) NOT NULL,
  display_name VARCHAR(64) NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  last_login_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS admin_role (
  id BIGSERIAL PRIMARY KEY,
  role_key VARCHAR(64) NOT NULL UNIQUE,
  role_name VARCHAR(64) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS admin_permission (
  id BIGSERIAL PRIMARY KEY,
  permission_key VARCHAR(128) NOT NULL UNIQUE,
  permission_name VARCHAR(128) NOT NULL,
  module_key VARCHAR(64) NOT NULL,
  action_key VARCHAR(64) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS admin_user_role (
  admin_user_id BIGINT NOT NULL REFERENCES admin_user(id) ON DELETE CASCADE,
  admin_role_id BIGINT NOT NULL REFERENCES admin_role(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (admin_user_id, admin_role_id)
);

CREATE TABLE IF NOT EXISTS admin_role_permission (
  admin_role_id BIGINT NOT NULL REFERENCES admin_role(id) ON DELETE CASCADE,
  admin_permission_id BIGINT NOT NULL REFERENCES admin_permission(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (admin_role_id, admin_permission_id)
);

CREATE TABLE IF NOT EXISTS admin_operation_log (
  id BIGSERIAL PRIMARY KEY,
  admin_user_id BIGINT NOT NULL REFERENCES admin_user(id),
  module_key VARCHAR(64) NOT NULL,
  action_key VARCHAR(64) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  before_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  after_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  extra_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admin_user_status ON admin_user(status);
CREATE INDEX IF NOT EXISTS idx_admin_role_status ON admin_role(status);
CREATE INDEX IF NOT EXISTS idx_admin_permission_module ON admin_permission(module_key, action_key);
CREATE INDEX IF NOT EXISTS idx_admin_operation_log_admin_user ON admin_operation_log(admin_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_operation_log_target ON admin_operation_log(target_type, target_id, created_at DESC);

INSERT INTO admin_role (role_key, role_name, description, status)
VALUES ('super_admin', '超级管理员', '拥有后台全部权限', 1)
ON CONFLICT (role_key) DO UPDATE SET
  role_name = EXCLUDED.role_name,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('dashboard:view', '查看控制台', 'dashboard', 'view', '查看后台控制台首页', 1),
  ('players:view', '查看玩家', 'players', 'view', '查看玩家列表与详情', 1),
  ('players:edit', '编辑玩家', 'players', 'edit', '调整玩家资源与状态', 1),
  ('pets:view', '查看宠物', 'pets', 'view', '查看宠物列表与详情', 1),
  ('pets:edit', '编辑宠物', 'pets', 'edit', '调整宠物属性与状态', 1),
  ('bag:view', '查看背包', 'bag', 'view', '查看玩家背包', 1),
  ('bag:grant', '发放道具', 'bag', 'grant', '给玩家发放物品', 1),
  ('quest:view', '查看任务', 'quest', 'view', '查看任务模板与玩家任务', 1),
  ('quest:edit', '编辑任务', 'quest', 'edit', '调整任务状态', 1),
  ('audit:view', '查看审计', 'audit', 'view', '查看后台操作审计日志', 1)
ON CONFLICT (permission_key) DO UPDATE SET
  permission_name = EXCLUDED.permission_name,
  module_key = EXCLUDED.module_key,
  action_key = EXCLUDED.action_key,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO admin_user (account_name, password_hash, display_name, status)
VALUES ('admin', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', '默认超级管理员', 1)
ON CONFLICT (account_name) DO UPDATE SET
  password_hash = EXCLUDED.password_hash,
  display_name = EXCLUDED.display_name,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO admin_user_role (admin_user_id, admin_role_id)
SELECT u.id, r.id
FROM admin_user u
JOIN admin_role r ON r.role_key = 'super_admin'
WHERE u.account_name = 'admin'
ON CONFLICT (admin_user_id, admin_role_id) DO NOTHING;

INSERT INTO admin_role_permission (admin_role_id, admin_permission_id)
SELECT r.id, p.id
FROM admin_role r
JOIN admin_permission p ON p.status = 1
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
