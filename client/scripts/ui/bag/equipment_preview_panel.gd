extends MarginContainer
class_name EquipmentPreviewPanel

const CharacterVisualScene: PackedScene = preload("res://scenes/character/character_visual.tscn")
const DEFAULT_PLAYER_SKIN_ID: String = "初始形象男_001"
const WORLD_IDLE_STATE: String = "idle"
const WORLD_IDLE_DIRECTION: String = "down"
const PREVIEW_ANCHOR_ROOT_PATH: NodePath = NodePath("PanelContainer/VBoxContainer/HBoxContainer2/Control/HBoxContainer/Control/HBoxContainer")
const PLAYER_DEMO_CANDIDATE_NAMES: Array[String] = ["Player", "player"]
const PET_DEMO_CANDIDATE_NAMES: Array[String] = ["Pet", "pet"]

@export_group("Preview Position")
## 人物真实形象相对 Player/player 占位节点的额外位置偏移；用于在背包面板实例 Inspector 中微调显示位置。
@export var player_preview_position_offset: Vector2 = Vector2.ZERO
## 宠物真实形象相对 Pet/pet 占位节点的额外位置偏移；用于在背包面板实例 Inspector 中微调显示位置。
@export var pet_preview_position_offset: Vector2 = Vector2.ZERO

## 中央展示区里承载人物/宠物占位节点的父节点；运行时锚点会插入到这里。
@onready var _preview_anchor_root: Node = get_node_or_null(PREVIEW_ANCHOR_ROOT_PATH)
## 中央展示区中用于占位的玩家演示节点；脚本会读取其位置和缩放作为真实形象锚点。
@onready var _player_demo_sprite: AnimatedSprite2D = _find_demo_sprite(PLAYER_DEMO_CANDIDATE_NAMES)
## 中央展示区中用于占位的宠物演示节点；脚本会读取其位置和缩放作为真实形象锚点。
@onready var _pet_demo_sprite: AnimatedSprite2D = _find_demo_sprite(PET_DEMO_CANDIDATE_NAMES)
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
	_player_visual_anchor = _build_visual_anchor(_player_demo_sprite, "PlayerVisualAnchor", player_preview_position_offset)
	_pet_visual_anchor = _build_visual_anchor(_pet_demo_sprite, "PetVisualAnchor", pet_preview_position_offset)
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


## 按候选名称查找占位 AnimatedSprite2D；支持编辑器中使用 Player/Pet 或 player/pet 命名。
func _find_demo_sprite(candidate_names: Array[String]) -> AnimatedSprite2D:
	if _preview_anchor_root == null:
		return null
	for candidate_name: String in candidate_names:
		var candidate_node: Node = _preview_anchor_root.get_node_or_null(candidate_name)
		if candidate_node is AnimatedSprite2D:
			return candidate_node as AnimatedSprite2D
	return null


## 以场景中 Player/Pet 演示节点的位置和缩放创建包装节点，并叠加 Inspector 暴露的位置偏移。
func _build_visual_anchor(demo_sprite: AnimatedSprite2D, anchor_name: String, position_offset: Vector2) -> Node2D:
	if demo_sprite == null:
		return null
	var anchor: Node2D = Node2D.new()
	anchor.name = anchor_name
	anchor.position = demo_sprite.position + position_offset
	anchor.scale = demo_sprite.scale
	anchor.z_index = demo_sprite.z_index
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
