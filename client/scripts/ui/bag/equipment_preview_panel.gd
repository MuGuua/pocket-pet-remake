extends MarginContainer
class_name EquipmentPreviewPanel

const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")
const DEFAULT_PLAYER_SKIN_ID: String = "初始形象男_001"
const WORLD_IDLE_STATE: String = "idle"
const WORLD_IDLE_DIRECTION: String = "down"
## 中央展示区人物与宠物统一放大倍数；按用户要求相对原展示节点放大 1.5 倍。
const PREVIEW_SCALE_MULTIPLIER: float = 1.5
## 中央展示区底部背景条的中心 y（相对展示容器内部坐标），人物脚底需要对齐到这里。
const SHOWCASE_GROUND_CENTER_Y: float = 55.0
## 玩家形象在脚底对齐基础上的额外垂直偏移；负值表示向上移动（单位：像素）。
const PREVIEW_PLAYER_VERTICAL_OFFSET_Y: float = -3.0

## 中央展示区中用于占位的玩家演示节点；脚本会读取其位置和缩放作为真实形象锚点。
@onready var _player_demo_sprite: AnimatedSprite2D = $PanelContainer/VBoxContainer/HBoxContainer2/Control/HBoxContainer/Control/HBoxContainer/Player
## 中央展示区中用于占位的宠物演示节点；脚本会读取其位置和缩放作为真实形象锚点。
@onready var _pet_demo_sprite: AnimatedSprite2D = $PanelContainer/VBoxContainer/HBoxContainer2/Control/HBoxContainer/Control/HBoxContainer/Pet
## 承载玩家真实形象的运行时包装节点；包装节点负责沿用场景里原本的摆放位置与缩放。
var _player_visual_anchor: Node2D = null
## 承载宠物真实形象的运行时包装节点；包装节点负责沿用场景里原本的摆放位置与缩放。
var _pet_visual_anchor: Node2D = null
## 当前玩家形象运行时实例；统一复用 CharacterVisual，兼容 PNG 与 CHJ。
var _player_visual: CharacterVisual = null
## 当前宠物形象运行时实例；统一复用 CharacterVisual，兼容 PNG 与 CHJ。
var _pet_visual: CharacterVisual = null


## 初始化展示区：创建运行时形象节点、隐藏旧演示图，并订阅玩家/宠物快照变化。
func _ready() -> void:
	_build_runtime_visuals()
	if not GameState.world_snapshot_changed.is_connected(_refresh_runtime_visuals):
		GameState.world_snapshot_changed.connect(_refresh_runtime_visuals)
	if not GameState.pets_changed.is_connected(_refresh_runtime_visuals):
		GameState.pets_changed.connect(_refresh_runtime_visuals)
	_refresh_runtime_visuals()


## 退出树时注销全局快照信号，避免背包面板销毁后继续回调旧节点。
func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(_refresh_runtime_visuals):
		GameState.world_snapshot_changed.disconnect(_refresh_runtime_visuals)
	if GameState.pets_changed.is_connected(_refresh_runtime_visuals):
		GameState.pets_changed.disconnect(_refresh_runtime_visuals)


## 创建玩家/宠物运行时形象，并复用场景中演示节点的摆放参数作为锚点。
func _build_runtime_visuals() -> void:
	_player_visual_anchor = _build_visual_anchor(_player_demo_sprite, "PlayerVisualAnchor", PREVIEW_PLAYER_VERTICAL_OFFSET_Y)
	_pet_visual_anchor = _build_visual_anchor(_pet_demo_sprite, "PetVisualAnchor", 0.0)
	_player_visual = _instantiate_character_visual(_player_visual_anchor)
	_pet_visual = _instantiate_character_visual(_pet_visual_anchor)


## 按最新的服务端权威快照刷新玩家与首只编队宠物形象。
func _refresh_runtime_visuals() -> void:
	_refresh_player_visual()
	_refresh_pet_visual()


## 同步当前玩家 skin_id；服务端未下发时使用项目统一默认形象兜底。
func _refresh_player_visual() -> void:
	if _player_visual == null or _player_visual_anchor == null:
		return
	var skin_id: String = str(GameState.player_snapshot.get("skin_id", "")).strip_edges()
	if skin_id.is_empty():
		skin_id = DEFAULT_PLAYER_SKIN_ID
	if not _player_visual.apply_skin_id(skin_id):
		_player_visual_anchor.visible = false
		return
	_player_visual_anchor.visible = true
	_player_visual.play_world(WORLD_IDLE_STATE, WORLD_IDLE_DIRECTION)


## 同步当前首只编队宠物 skin_id；没有编队或缺少 skin_id 时直接隐藏宠物展示位。
func _refresh_pet_visual() -> void:
	if _pet_visual == null or _pet_visual_anchor == null:
		return
	var lineup_pet: Dictionary = _resolve_first_lineup_pet()
	var skin_id: String = str(lineup_pet.get("skin_id", "")).strip_edges()
	if skin_id.is_empty():
		_pet_visual_anchor.visible = false
		return
	if not _pet_visual.apply_skin_id(skin_id):
		_pet_visual_anchor.visible = false
		return
	_pet_visual_anchor.visible = true
	_pet_visual.play_world(WORLD_IDLE_STATE, WORLD_IDLE_DIRECTION)


## 读取当前首只编队宠物；若编队为空则返回空字典，不退化成假宠物数据。
func _resolve_first_lineup_pet() -> Dictionary:
	if GameState.lineup.is_empty():
		return {}
	var first_lineup_variant: Variant = GameState.lineup[0]
	if first_lineup_variant is Dictionary:
		return (first_lineup_variant as Dictionary).duplicate(true)
	return {}


## 以旧演示节点的位置/缩放创建一个包装节点，并隐藏演示节点本身。
func _build_visual_anchor(demo_sprite: AnimatedSprite2D, anchor_name: String, extra_offset_y: float = 0.0) -> Node2D:
	var anchor: Node2D = Node2D.new()
	anchor.name = anchor_name
	if demo_sprite != null:
		anchor.position = _resolve_preview_anchor_position(demo_sprite)
		anchor.position.y += extra_offset_y
		anchor.scale = demo_sprite.scale * PREVIEW_SCALE_MULTIPLIER
		var parent_node: Node = demo_sprite.get_parent()
		if parent_node != null:
			parent_node.add_child(anchor)
			parent_node.move_child(anchor, demo_sprite.get_index())
		demo_sprite.hide()
	return anchor


## 在包装节点下实例化一个 CharacterVisual，统一走项目已有角色形象渲染链路。
func _instantiate_character_visual(anchor: Node2D) -> CharacterVisual:
	if anchor == null:
		return null
	var character_visual: CharacterVisual = CharacterVisualScene.instantiate() as CharacterVisual
	if character_visual == null:
		return null
	character_visual.position = Vector2.ZERO
	anchor.add_child(character_visual)
	return character_visual


## 统一把预览锚点向下调整：人物脚底对齐到底部背景中心，宠物沿用同样的下移量保持相对站位。
func _resolve_preview_anchor_position(demo_sprite: AnimatedSprite2D) -> Vector2:
	if demo_sprite == null:
		return Vector2.ZERO
	var moved_position: Vector2 = demo_sprite.position
	var player_feet_offset_y: float = SHOWCASE_GROUND_CENTER_Y - _player_demo_sprite.position.y if _player_demo_sprite != null else 0.0
	moved_position.y += player_feet_offset_y
	return moved_position
