extends RefCounted
class_name ChjSprite

## 单帧宽度（像素），来自 CHJ 头部 byte[2]。
var frame_width: int = 0
## 单帧高度（像素），来自 CHJ 头部 byte[3]。
var frame_height: int = 0
## 动画组数量，来自 CHJ 头部 byte[6]。
var animation_count: int = 0
## 每个动画组对应的帧序号列表。
var animations: Array[PackedInt32Array] = []
## 从 CHJ 内嵌 PNG 解码得到的横向条带图集。
var texture: Texture2D = null


## 从 res:// 路径加载并解析 CHJ 文件。
static func load_from_path(path: String) -> ChjSprite:
	var file: FileAccess = FileAccess.open(path, FileAccess.READ)
	if file == null:
		push_error("无法打开 CHJ 文件: %s" % path)
		return null
	return parse(file.get_buffer(file.get_length()))


## 解析 CHJ 二进制：头部元数据 + 动画索引 + 帧序号表 + 内嵌 PNG。
static func parse(data: PackedByteArray) -> ChjSprite:
	if data.size() < 9:
		push_error("CHJ 数据过短: %d 字节" % data.size())
		return null

	var sprite: ChjSprite = ChjSprite.new()
	sprite.frame_width = int(data[2])
	sprite.frame_height = int(data[3])
	sprite.animation_count = int(data[6])

	var offsets_start: int = 7
	var index_list_len: int = int(data[7 + sprite.animation_count])
	var frame_list_start: int = 8 + sprite.animation_count
	var image_offset: int = frame_list_start + index_list_len

	if image_offset > data.size():
		push_error("CHJ imageOffset 越界: %d > %d" % [image_offset, data.size()])
		return null

	var frame_list: PackedInt32Array = PackedInt32Array()
	for i: int in range(frame_list_start, image_offset):
		frame_list.append(int(data[i]))

	for i: int in range(sprite.animation_count):
		var start: int = int(data[offsets_start + i])
		var end: int = int(data[offsets_start + i + 1]) if i + 1 < sprite.animation_count else index_list_len
		var group: PackedInt32Array = PackedInt32Array()
		for j: int in range(start, end):
			if j < frame_list.size():
				group.append(frame_list[j])
		sprite.animations.append(group)

	var png_bytes: PackedByteArray = data.slice(image_offset)
	var image: Image = Image.new()
	var err: Error = image.load_png_from_buffer(png_bytes)
	if err != OK:
		push_error("CHJ PNG 解码失败: %d" % err)
		return null

	sprite.texture = ImageTexture.create_from_image(image)
	return sprite


## 返回指定动画组的帧序号；组无效时回退到第 0 组或 [0]。
func get_animation_frames(action_index: int) -> PackedInt32Array:
	if action_index >= 0 and action_index < animations.size():
		var frames: PackedInt32Array = animations[action_index]
		if frames.size() > 0:
			return frames
	if animations.size() > 0:
		var fallback: PackedInt32Array = animations[0]
		if fallback.size() > 0:
			return fallback
	return PackedInt32Array([0])


## 合并主 CHJ 末尾最后两个动画组的帧序号，供战斗待机循环播放。
func get_battle_idle_frames() -> PackedInt32Array:
	var combined: PackedInt32Array = PackedInt32Array()
	if animation_count >= 2:
		for group_index: int in [animation_count - 2, animation_count - 1]:
			if group_index < 0 or group_index >= animations.size():
				continue
			var group_frames: PackedInt32Array = animations[group_index]
			if group_frames.is_empty():
				continue
			for raw_frame: int in group_frames:
				combined.append(raw_frame)
	if combined.is_empty():
		return get_animation_frames(0)
	return combined


## 截取 CHJ 世界待机首帧为独立贴图；默认取 down idle（动画组 0），供 HUD 头像等静态展示。
func create_world_idle_preview_texture(direction_suffix: String = "down", frame_cursor: int = 0) -> Texture2D:
	if texture == null or frame_width <= 0 or frame_height <= 0:
		return null
	var direction_key: String = direction_suffix.strip_edges().to_lower()
	if direction_key.is_empty():
		direction_key = "down"
	var action_index: int = _resolve_world_idle_action_index(direction_key)
	var force_flip: bool = direction_key == "right"
	var frames: PackedInt32Array = get_animation_frames(action_index)
	if frames.is_empty():
		return null
	var safe_cursor: int = clampi(frame_cursor, 0, frames.size() - 1)
	var raw: int = int(frames[safe_cursor])
	var flip: bool = raw >= 128 or force_flip
	var frame_index: int = raw - 128 if raw >= 128 else raw
	var source_image: Image = texture.get_image()
	if source_image == null:
		return null
	var region: Rect2i = Rect2i(frame_index * frame_width, 0, frame_width, frame_height)
	var image_size: Vector2i = source_image.get_size()
	if region.position.x < 0 or region.position.y < 0:
		return null
	if region.position.x + region.size.x > image_size.x or region.position.y + region.size.y > image_size.y:
		return null
	var frame_image: Image = source_image.get_region(region)
	if flip:
		frame_image.flip_x()
	return ImageTexture.create_from_image(frame_image)


## 解析世界待机 CHJ 动画组索引，与 ChjWorldRenderer.ACTION_MAP 保持一致。
func _resolve_world_idle_action_index(direction_suffix: String) -> int:
	match direction_suffix:
		"up":
			return 2
		"left", "right":
			return 4
		_:
			return 0
