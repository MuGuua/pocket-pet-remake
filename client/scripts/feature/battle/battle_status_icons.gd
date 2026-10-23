@tool
extends Node2D
class_name BattleStatusIcons

## 战斗单位脚下的状态图标行。
## 外观参数在 scenes/battle/battle_status_icons.tscn 的检查器里调整（编辑器内实时预览）；
## 它作为 StatusIcons 子节点挂在 scenes/battle/battle_unit.tscn 里，拖动该节点即可调整位置。
## 数据只来自服务端：开战快照、每批事件播完后的权威 actors 快照（含剩余回合），
## 以及播放中的「施加状态」事件（让图标与日志同一时刻出现）。客户端不推算状态是否到期，
## 移除一律等权威快照校准，因此不会出现「状态已结束但图标残留」。

## 状态 ID → 图标配置。ID 与服务端 battle/model.go 的 Status* 常量一一对应。
## frames=2 的图集是左右两帧（32×16），运行时交替播放；frames=1 为单帧 16×16。
const STATUS_ICONS: Dictionary = {
	1: {"name": "流血", "path": "res://asset/分类/状态符/两帧/373.png", "frames": 2},
	2: {"name": "封印", "path": "res://asset/分类/状态符/两帧/380.png", "frames": 2},
	3: {"name": "眩晕", "path": "res://asset/分类/状态符/单帧/370.png", "frames": 1},
	4: {"name": "易伤", "path": "res://asset/分类/状态符/两帧/381.png", "frames": 2},
	5: {"name": "破甲", "path": "res://asset/分类/状态符/单帧/368.png", "frames": 1},
	6: {"name": "减速", "path": "res://asset/分类/状态符/两帧/365.png", "frames": 2},
	7: {"name": "暴击提升", "path": "res://asset/分类/状态符/两帧/372.png", "frames": 2},
	8: {"name": "诅咒", "path": "res://asset/分类/状态符/两帧/375.png", "frames": 2},
	9: {"name": "束缚", "path": "res://asset/分类/状态符/单帧/2048.png", "frames": 1},
	10: {"name": "沉睡", "path": "res://asset/分类/状态符/单帧/367.png", "frames": 1},
	11: {"name": "麻痹", "path": "res://asset/分类/状态符/两帧/385.png", "frames": 2},
	12: {"name": "混乱", "path": "res://asset/分类/状态符/两帧/374.png", "frames": 2},
	13: {"name": "光之洗礼", "path": "res://asset/分类/状态符/两帧/369.png", "frames": 2},
	14: {"name": "圣光忏悔", "path": "res://asset/分类/状态符/两帧/376.png", "frames": 2},
	15: {"name": "感电", "path": "res://asset/分类/状态符/两帧/378.png", "frames": 2},
	16: {"name": "混乱之血", "path": "res://asset/分类/状态符/两帧/382.png", "frames": 2},
	17: {"name": "幻影闪击", "path": "res://asset/分类/状态符/两帧/377.png", "frames": 2},
	18: {"name": "魅惑之风", "path": "res://asset/分类/状态符/单帧/2049.png", "frames": 1},
	19: {"name": "恶魔之力", "path": "res://asset/分类/状态符/两帧/383.png", "frames": 2},
}
## 单帧图标源尺寸（像素）。
const ICON_SOURCE_SIZE: int = 16

@export_group("图标")
## 图标显示缩放；单位节点本身还会再乘战斗单位缩放（battle_scene 的 battle_unit_scale）。
@export_range(0.2, 3.0, 0.05) var icon_scale: float = 0.7:
	set(value):
		icon_scale = value
		_rebuild()
## 相邻图标中心间距（单位本地坐标）。
@export_range(4.0, 40.0, 0.5) var icon_spacing: float = 12.0:
	set(value):
		icon_spacing = value
		_rebuild()
## 同时展示的最大图标数；超出时最后一格显示「+N」。
@export_range(1, 10, 1) var max_visible_icons: int = 4:
	set(value):
		max_visible_icons = value
		_rebuild()
## 两帧图标的换帧间隔（秒）。
@export_range(0.1, 2.0, 0.05) var frame_interval_sec: float = 0.45
## 勾选为自上而下竖排（贴在单位外侧），不勾选为以节点为中心水平横排。
@export var vertical: bool = true:
	set(value):
		vertical = value
		_rebuild()

@export_group("回合数字")
## 剩余回合数字字号。
@export_range(4, 20, 1) var rounds_font_size: int = 7:
	set(value):
		rounds_font_size = value
		_rebuild()
## 剩余回合数字相对图标右下角的偏移。
@export var rounds_offset: Vector2 = Vector2(2.5, 3.0):
	set(value):
		rounds_offset = value
		_rebuild()
## 「+N」溢出与无图标兜底字的字号。
@export_range(4, 20, 1) var overflow_font_size: int = 8:
	set(value):
		overflow_font_size = value
		_rebuild()

@export_group("挂在单位上的位置")
## 本节点在 battle_unit.tscn 里的 position 就是「敌方一侧」的基准：
## y 是第一个图标中心的高度，|x| 是离单位中心的最小水平距离；友方运行时自动镜像到左侧。
## 运行时还会按贴图不透明像素宽度把图标往外推（防止被大体型单位挡住），但不超过 max_outer_x。
@export_range(0.0, 200.0, 1.0) var max_outer_x: float = 70.0
## 图标与贴图不透明边缘之间的间隙（单位本地坐标）。
@export_range(0.0, 30.0, 0.5) var body_gap: float = 3.0

@export_group("编辑器预览（运行时忽略）")
## 只在编辑器里显示的示例状态 ID；游戏运行时一律以服务端数据为准。
@export var preview_status_ids: Array[int] = [11, 5, 10]:
	set(value):
		preview_status_ids = value
		_apply_editor_preview()
## 预览用剩余回合；0 表示不显示数字。
@export_range(0, 9, 1) var preview_remaining_rounds: int = 2:
	set(value):
		preview_remaining_rounds = value
		_apply_editor_preview()

## 当前生效状态：按服务端顺序排列的 [{status_id, remaining_rounds}]。
var _statuses: Array[Dictionary] = []
## 已加载的图集缓存，避免每次刷新重复 load。
static var _texture_cache: Dictionary = {}
## 当前帧序号（0 / 1），所有两帧图标同步换帧。
var _frame_index: int = 0
var _frame_elapsed: float = 0.0
## 当前渲染出的图标节点，与 _statuses 前 max_visible_icons 项对应。
var _icon_sprites: Array[Sprite2D] = []


func _ready() -> void:
	_apply_editor_preview()


## 图标单边显示尺寸（单位本地坐标）。
func icon_display_size() -> float:
	return float(ICON_SOURCE_SIZE) * icon_scale


## 仅编辑器：用预览状态渲染示例图标，运行时不生效。
func _apply_editor_preview() -> void:
	if not Engine.is_editor_hint() or not is_inside_tree():
		return
	var preview: Array = []
	for status_id: int in preview_status_ids:
		preview.append({"status_id": status_id, "remaining_rounds": preview_remaining_rounds})
	_statuses = BattleStatusIcons.normalize_statuses(preview)
	_rebuild()


func _process(delta: float) -> void:
	if _icon_sprites.is_empty():
		return
	_frame_elapsed += delta
	if _frame_elapsed < frame_interval_sec:
		return
	_frame_elapsed = 0.0
	_frame_index = 1 - _frame_index
	for sprite: Sprite2D in _icon_sprites:
		var atlas: AtlasTexture = sprite.texture as AtlasTexture
		if atlas != null and int(sprite.get_meta("frames", 1)) > 1:
			atlas.region = _frame_region(_frame_index)


## 用权威快照整体替换状态；statuses 为服务端 statuses 数组，旧服务端只有 status_ids 时传 fallback_ids。
func set_statuses(statuses: Variant, fallback_ids: Variant = []) -> void:
	_statuses = BattleStatusIcons.normalize_statuses(statuses, fallback_ids)
	_rebuild()


## 播放中收到「施加状态」事件时先行加上图标；剩余回合未知，等权威快照校准。
func add_status(status_id: int) -> void:
	if status_id <= 0:
		return
	for status: Dictionary in _statuses:
		if int(status.get("status_id", 0)) == status_id:
			return
	_statuses.append({"status_id": status_id, "remaining_rounds": 0})
	_statuses.sort_custom(func(left: Dictionary, right: Dictionary) -> bool:
		return int(left.get("status_id", 0)) < int(right.get("status_id", 0)))
	_rebuild()


## 清空全部状态（单位死亡或重新初始化时）。
func clear_statuses() -> void:
	_statuses.clear()
	_rebuild()


## 当前状态 ID 列表，供测试与调试使用。
func get_status_ids() -> Array[int]:
	var result: Array[int] = []
	for status: Dictionary in _statuses:
		result.append(int(status.get("status_id", 0)))
	return result


## 状态中文名；未知 ID 返回「状态#ID」，便于发现新增状态未配图标。
static func status_name(status_id: int) -> String:
	var config: Dictionary = STATUS_ICONS.get(status_id, {}) as Dictionary
	return str(config.get("name", "状态#%d" % status_id))


## 把服务端 statuses（优先）或 status_ids（兼容旧服务端）统一成 [{status_id, remaining_rounds}]，
## 过滤非法 ID 并按 ID 升序（与服务端顺序一致）。纯函数，便于测试。
static func normalize_statuses(statuses: Variant, fallback_ids: Variant = []) -> Array[Dictionary]:
	var result: Array[Dictionary] = []
	var seen: Dictionary = {}
	if statuses is Array and not (statuses as Array).is_empty():
		for entry: Variant in statuses as Array:
			if entry is not Dictionary:
				continue
			var status_id: int = int((entry as Dictionary).get("status_id", 0))
			if status_id <= 0 or seen.has(status_id):
				continue
			seen[status_id] = true
			result.append({
				"status_id": status_id,
				"remaining_rounds": maxi(0, int((entry as Dictionary).get("remaining_rounds", 0))),
			})
	elif fallback_ids is Array:
		for id_variant: Variant in fallback_ids as Array:
			var status_id: int = int(id_variant)
			if status_id <= 0 or seen.has(status_id):
				continue
			seen[status_id] = true
			result.append({"status_id": status_id, "remaining_rounds": 0})
	result.sort_custom(func(left: Dictionary, right: Dictionary) -> bool:
		return int(left.get("status_id", 0)) < int(right.get("status_id", 0)))
	return result


## 按当前状态重建图标行：最多 max_visible_icons 个，居中排列；超出部分在最后一格显示「+N」。
func _rebuild() -> void:
	if not is_inside_tree():
		return
	for child: Node in get_children():
		remove_child(child)
		child.queue_free()
	_icon_sprites.clear()
	var total: int = _statuses.size()
	if total == 0:
		return
	var overflow: int = maxi(0, total - max_visible_icons)
	var icon_count: int = total if overflow == 0 else max_visible_icons - 1
	var slot_count: int = icon_count + (1 if overflow > 0 else 0)
	for index: int in range(icon_count):
		var status: Dictionary = _statuses[index]
		_add_icon(int(status.get("status_id", 0)), int(status.get("remaining_rounds", 0)), _slot_position(index, slot_count))
	if overflow > 0:
		_add_centered_text("+%d" % (overflow + 1), _slot_position(icon_count, slot_count), overflow_font_size)


## 第 index 个格子的中心：竖排从原点向下排，横排以原点为中心左右展开。
func _slot_position(index: int, slot_count: int) -> Vector2:
	if vertical:
		return Vector2(0.0, icon_spacing * float(index))
	return Vector2(-icon_spacing * float(slot_count - 1) * 0.5 + icon_spacing * float(index), 0.0)


## 创建一个状态图标；剩余回合 > 0 时在右下角叠加数字。
func _add_icon(status_id: int, remaining_rounds: int, slot_position: Vector2) -> void:
	var config: Dictionary = STATUS_ICONS.get(status_id, {}) as Dictionary
	var source: Texture2D = _load_texture(str(config.get("path", "")))
	if source == null:
		# 未配置图标的新状态：用名字首字兜底，至少让玩家知道身上有状态。
		_add_centered_text(BattleStatusIcons.status_name(status_id).left(1), slot_position, overflow_font_size)
		return
	var frames: int = maxi(1, int(config.get("frames", 1)))
	var atlas: AtlasTexture = AtlasTexture.new()
	atlas.atlas = source
	atlas.region = _frame_region(_frame_index if frames > 1 else 0)
	var sprite: Sprite2D = Sprite2D.new()
	sprite.texture = atlas
	sprite.texture_filter = CanvasItem.TEXTURE_FILTER_NEAREST
	sprite.scale = Vector2(icon_scale, icon_scale)
	sprite.position = slot_position
	sprite.set_meta("frames", frames)
	sprite.set_meta("status_id", status_id)
	add_child(sprite)
	_icon_sprites.append(sprite)
	if remaining_rounds > 0:
		# 剩余回合数字的右下角对齐图标右下角。
		# 先进树再取尺寸：未进树时主题字体未生效，最小尺寸不准。
		var badge: Label = _make_label(str(remaining_rounds), rounds_font_size)
		add_child(badge)
		var badge_size: Vector2 = badge.get_combined_minimum_size()
		var corner: Vector2 = slot_position + Vector2.ONE * icon_display_size() * 0.5
		badge.size = badge_size
		badge.position = corner - badge_size + rounds_offset


## 以 center 为中心放置一段文字（「+N」溢出、无图标兜底字）。
func _add_centered_text(text: String, center: Vector2, font_size: int) -> void:
	var label: Label = _make_label(text, font_size)
	add_child(label)
	label.size = label.get_combined_minimum_size()
	label.position = center - label.size * 0.5


## 创建一个带黑色描边的小号标签，不拦截鼠标。
func _make_label(text: String, font_size: int) -> Label:
	var label: Label = Label.new()
	label.text = text
	label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	label.add_theme_font_size_override("font_size", font_size)
	label.add_theme_color_override("font_color", Color(1.0, 0.95, 0.8, 1.0))
	label.add_theme_color_override("font_outline_color", Color(0.0, 0.0, 0.0, 1.0))
	label.add_theme_constant_override("outline_size", 3)
	label.add_theme_constant_override("line_spacing", 0)
	return label


## 两帧图集中第 frame 帧的区域。
func _frame_region(frame: int) -> Rect2:
	return Rect2(float(ICON_SOURCE_SIZE * frame), 0.0, float(ICON_SOURCE_SIZE), float(ICON_SOURCE_SIZE))


## 加载并缓存图集；路径为空或资源缺失时返回 null。
func _load_texture(path: String) -> Texture2D:
	if path.is_empty():
		return null
	if _texture_cache.has(path):
		return _texture_cache[path] as Texture2D
	var texture: Texture2D = null
	if ResourceLoader.exists(path):
		texture = load(path) as Texture2D
	_texture_cache[path] = texture
	return texture
