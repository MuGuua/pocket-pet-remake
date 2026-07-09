ALTER TABLE quest_template
  ADD COLUMN IF NOT EXISTS client_icon_id BIGINT NOT NULL DEFAULT 1;

COMMENT ON COLUMN quest_template.client_icon_id IS '客户端本地任务图标注册表使用的图标ID；服务端只下发ID，不下发资源路径。';

UPDATE quest_template
SET client_icon_id = CASE quest_id
  WHEN 1001 THEN 1
  WHEN 1002 THEN 2
  WHEN 1003 THEN 3
  ELSE client_icon_id
END
WHERE quest_id IN (1001, 1002, 1003);
