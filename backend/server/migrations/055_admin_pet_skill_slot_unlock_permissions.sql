-- 055_admin_pet_skill_slot_unlock_permissions.sql
-- 神符槽解锁道具映射后台权限。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('pet_skill_slot_unlock:view', '查看神符槽解锁配置', 'pet_skill_slot_unlock', 'view', '查看 pet_skill_slot_unlock_item 道具与槽位映射', 1),
  ('pet_skill_slot_unlock:edit', '编辑神符槽解锁配置', 'pet_skill_slot_unlock', 'edit', '增删改 pet_skill_slot_unlock_item 配置', 1)
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
JOIN admin_permission p ON p.permission_key IN ('pet_skill_slot_unlock:view', 'pet_skill_slot_unlock:edit')
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
