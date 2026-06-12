-- 031_admin_scene_wild_encounter_permissions.sql
-- 地图暗雷遭遇配置后台权限。

INSERT INTO admin_permission (permission_key, permission_name, module_key, action_key, description, status)
VALUES
  ('scene_wild_encounters:view', '查看地图暗雷', 'scene_wild_encounters', 'view', '查看地图暗雷遭遇配置列表与详情', 1),
  ('scene_wild_encounters:edit', '编辑地图暗雷', 'scene_wild_encounters', 'edit', '新增、编辑、删除地图暗雷遭遇配置', 1)
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
JOIN admin_permission p ON p.permission_key IN (
  'scene_wild_encounters:view', 'scene_wild_encounters:edit'
)
WHERE r.role_key = 'super_admin'
ON CONFLICT (admin_role_id, admin_permission_id) DO NOTHING;
