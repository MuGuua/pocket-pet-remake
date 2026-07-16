class_name InteractiveNPCBase
extends Node2D

signal interaction_entered(entity_id: int, npc_name: String)
signal interaction_exited(entity_id: int, npc_name: String)

# NPC 在服务端世界配置中的唯一实体标识。
@export var entity_id: int = 0
# NPC 的稳定配置代号；后续如果显示名改动，服务端仍可按这个字段做配置映射。
@export var npc_code: String = ""
# NPC 的前端显示名；菜单标题、日志和气泡文案会使用这个名字。
@export var npc_name: String = "NPC"
## 是否由服务端世界快照控制可见性；剧情中的同一打包角色应保持关闭。
@export var server_visibility_managed: bool = false

## 可选的 NPC 交互区域；固定过场角色可以不配置该节点。
@onready var interact_area: Area2D = get_node_or_null("Area2D") as Area2D

func _ready() -> void:
	if server_visibility_managed:
		if not GameState.world_snapshot_changed.is_connected(_sync_server_visibility):
			GameState.world_snapshot_changed.connect(_sync_server_visibility)
		_sync_server_visibility()
	# 没有交互区域时直接返回，避免空节点在加载期报错。
	if interact_area == null:
		return

	# 缺少稳定唯一标识时直接给出警告，方便尽早发现未配置完成的 NPC 资源。
	if entity_id <= 0:
		push_warning("NPC %s is missing a valid entity_id." % name)
	if npc_code.is_empty():
		push_warning("NPC %s is missing npc_code." % name)

	# 当前项目的玩家与地图碰撞层还在调整中，这里统一放开 mask，
	# 保证 NPC 感应区始终能检测到 Player 进入和离开。
	interact_area.monitoring = true
	interact_area.monitorable = true
	for layer_index in range(1, 33):
		interact_area.set_collision_mask_value(layer_index, true)
	if not interact_area.body_entered.is_connected(_on_body_entered):
		interact_area.body_entered.connect(_on_body_entered)
	if not interact_area.body_exited.is_connected(_on_body_exited):
		interact_area.body_exited.connect(_on_body_exited)

## 退出场景时断开世界快照信号，避免地图切换后旧 NPC 实例继续接收更新。
func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(_sync_server_visibility):
		GameState.world_snapshot_changed.disconnect(_sync_server_visibility)

## 只在服务端快照包含当前实体时显示世界 NPC，剧情实例不启用该开关。
func _sync_server_visibility() -> void:
	var should_show: bool = entity_id > 0 and GameState.nearby_entities.has(entity_id)
	visible = should_show
	for collision_shape: CollisionShape2D in find_children("*", "CollisionShape2D", true, false):
		collision_shape.set_deferred("disabled", not should_show)

func _on_body_entered(body: Node2D) -> void:
	if body == null or body.name != "Player" or entity_id <= 0:
		return
	interaction_entered.emit(entity_id, npc_name)

func _on_body_exited(body: Node2D) -> void:
	if body == null or body.name != "Player" or entity_id <= 0:
		return
	interaction_exited.emit(entity_id, npc_name)

# 返回当前 NPC 的统一身份信息，便于后续扩展日志或调试面板时复用。
func get_identity_payload() -> Dictionary:
	return {
		"entity_id": entity_id,
		"npc_code": npc_code,
		"npc_name": npc_name,
	}
