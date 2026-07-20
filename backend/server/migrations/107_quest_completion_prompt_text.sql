-- 任务完成提示文案由后台配置、服务端下发，客户端只负责展示。
ALTER TABLE quest_template
  ADD COLUMN IF NOT EXISTS completion_prompt_text TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN quest_template.completion_prompt_text IS
  '任务提交成功后展示给玩家的完成提示文案，支持 Godot RichTextLabel BBCode；为空时不展示额外提示。';
