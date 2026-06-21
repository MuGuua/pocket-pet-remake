-- 057_admin_pet_combat_stat_cap_permissions.sql
-- 宠物战斗属性封顶配置后台权限。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('pet_combat_stat_cap:view', '查看宠物战斗属性封顶', 'pet_combat_stat_cap', 'view', '查看 pet_combat_stat_cap 封顶配置', 1),
  ('pet_combat_stat_cap:edit', '编辑宠物战斗属性封顶', 'pet_combat_stat_cap', 'edit', '更新 pet_combat_stat_cap 封顶值与说明', 1)
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
JOIN admin_permission p ON p.permission_key IN ('pet_combat_stat_cap:view', 'pet_combat_stat_cap:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
