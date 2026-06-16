-- 036_admin_player_progression_permissions.sql
-- 玩家升级经验与属性转化率后台权限。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('player_progression:view', '查看玩家成长配置', 'player_progression', 'view', '查看玩家等级经验与属性转化率配置', 1),
  ('player_progression:edit', '编辑玩家成长配置', 'player_progression', 'edit', '编辑玩家等级经验与属性转化率配置', 1)
ON CONFLICT (permission_key) DO UPDATE SET
  permission_name = EXCLUDED.permission_name,
  module_key = EXCLUDED.module_key,
  action_key = EXCLUDED.action_key,
  description = EXCLUDED.description,
  status = EXCLUDED.status,
  updated_at = CURRENT_TIMESTAMP;

INSERT INTO admin_role_permission (admin_role_id, admin_permission_id)
SELECT r.id, p.id
FROM admin_role r
JOIN admin_permission p ON p.permission_key IN ('player_progression:view', 'player_progression:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
