ALTER TABLE quest_template
  ADD COLUMN accept_animation_key VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN submit_animation_key VARCHAR(128) NOT NULL DEFAULT '';

COMMENT ON COLUMN quest_template.accept_animation_key IS '任务领取成功后客户端播放的动画注册键；空字符串表示不播放';
COMMENT ON COLUMN quest_template.submit_animation_key IS '任务交付成功后客户端播放的动画注册键；空字符串表示不播放';
