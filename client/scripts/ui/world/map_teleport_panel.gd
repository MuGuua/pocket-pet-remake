extends CanvasLayer
class_name MapTeleportPanel

## 地图面板关闭后通知主运行态解除世界输入锁。
signal menu_closed
## 玩家通过点击或上下切换选中地图标点时广播展示名称。
## point_name 是场景资源中对应按钮配置的地点名称。
signal map_point_selected(point_name: String)
## 玩家再次点击当前选中的开放标点时，请求上层发起服务端权威传送。
## target_scene_id 是标点配置的目标场景 ID；point_name 是用于提示和日志的地点名称。
signal teleport_requested(target_scene_id: int, point_name: String)
## 玩家再次点击尚未开放的地图标点时，请求上层展示提示。
## message 是需要展示给玩家的简体中文提示正文。
signal notice_requested(message: String)

## 面板右上角关闭按钮。
@onready var close_button: Button = %CloseButton
## 面板标题，随世界地图或地区地图视图同步更新。
@onready var title_label: Label = %Title
## 选中地点名称标签。
@onready var selected_name_label: Label = %SelectedNameLabel
## 返回世界地图的唯一底部导航按钮。
@onready var world_map_button: Button = %WorldMapButton
## 闪光镇地图纹理节点，按原始图片尺寸的统一三倍倍率配置。
@onready var map_texture_rect: TextureRect = %MapTexture
## 世界地图纹理节点，保持原始宽高比和统一三倍倍率。
@onready var world_map_texture_rect: TextureRect = %WorldMapTexture
## 闪光平原地图纹理节点，保持原始宽高比和统一三倍倍率。
@onready var flash_plain_map_texture_rect: TextureRect = %FlashPlainMapTexture
## 精灵迷宫地图纹理节点，保持原始宽高比和统一三倍倍率。
@onready var spirit_maze_map_texture_rect: TextureRect = %SpiritMazeMapTexture
## 闪光镇地图标点的场景容器；按钮顺序就是上下切换顺序。
@onready var point_button_container: Control = %PointButtons
## 闪光平原地图标点的独立场景容器，避免切换地区时混入闪光镇节点。
@onready var flash_plain_point_button_container: Control = %FlashPlainPointButtons
## 世界地图的地区入口按钮容器，按钮均预先配置在场景节点树中。
@onready var world_region_button_container: Control = %WorldRegionButtons
## 世界地图中的闪光镇地区入口。
@onready var flash_town_region_button: Button = %FlashTownRegionButton
## 世界地图中的闪光平原地区入口。
@onready var flash_plain_region_button: Button = %FlashPlainRegionButton
## 世界地图中的精灵迷宫地区入口。
@onready var spirit_maze_region_button: Button = %SpiritMazeRegionButton
## 当前选中节点使用的四帧循环动画；场景中只保留一个实例并跟随节点中心移动。
@onready var selection_animation: AnimatedSprite2D = %SelectionAnimation

## 人物图标相对地图标点左上角的偏移；该值对应场景中已调整好的左下角视觉位置。
@export var current_scene_icon_offset: Vector2i = Vector2i(-34, 8)
## 当前参与选择的地图标点按钮列表。
var _point_buttons: Array[Button] = []
## 当前选中标点在按钮列表中的索引。
var _selected_index: int = 0
## 当前是否展示世界地图；地区地图内部才允许选择具体传送节点。
var _is_world_map_visible: bool = false
## 世界地图当前选中的地区按钮；再次点击同一按钮才进入地区地图。
var _selected_world_region_button: Button = null
## 当前正在查看的地区按钮；返回世界地图时自动恢复到该地区节点。
var _current_region_button: Button = null
## 当前地区正在使用的地图标点容器；无可选节点的地区保持为空。
var _active_point_button_container: Control = null
## 当前地区的人物位置图标；位置只根据服务端权威场景快照更新。
var _current_scene_icon: TextureRect = null


## 绑定场景中预先放置的按钮，并保持运行时面板默认隐藏。
func _ready() -> void:
	hide()
	if close_button != null and not close_button.pressed.is_connected(close_menu):
		close_button.pressed.connect(close_menu)
	if world_map_button != null and not world_map_button.pressed.is_connected(show_world_map):
		world_map_button.pressed.connect(show_world_map)
	_bind_world_region_button(flash_town_region_button)
	_bind_world_region_button(flash_plain_region_button)
	_bind_world_region_button(spirit_maze_region_button)
	if not GameState.world_snapshot_changed.is_connected(_refresh_current_scene_icon):
		GameState.world_snapshot_changed.connect(_refresh_current_scene_icon)
	_show_flash_town_map()


## 离开场景时断开全局世界快照信号，避免已释放面板继续接收切图更新。
func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(_refresh_current_scene_icon):
		GameState.world_snapshot_changed.disconnect(_refresh_current_scene_icon)


## 打开地图面板，并把输入焦点放到当前选中的地图标点。
func open_menu() -> void:
	_refresh_current_scene_icon()
	show()
	if _active_point_button_container == null or not _active_point_button_container.visible:
		return
	_apply_selection(_selected_index, false)
	if not _point_buttons.is_empty():
		_point_buttons[_selected_index].grab_focus()


## 关闭地图面板，并通知主运行态恢复世界交互。
func close_menu() -> void:
	var was_visible: bool = visible
	hide()
	if was_visible:
		menu_closed.emit()


## 处理键盘或手柄的上下选择与取消输入。
## event 是模态地图显示期间优先接收的输入事件，处理后会阻止底层世界继续消费。
func _input(event: InputEvent) -> void:
	if not visible:
		return
	if _active_point_button_container != null and _active_point_button_container.visible and event.is_action_pressed("ui_up"):
		select_previous_point()
		get_viewport().set_input_as_handled()
		return
	if _active_point_button_container != null and _active_point_button_container.visible and event.is_action_pressed("ui_down"):
		select_next_point()
		get_viewport().set_input_as_handled()
		return
	if event.is_action_pressed("ui_cancel"):
		close_menu()
		get_viewport().set_input_as_handled()
		return
	if event is InputEventKey or event is InputEventJoypadButton:
		get_viewport().set_input_as_handled()


## 切换到世界地图视图；该操作只改变同一面板中的展示层，不请求服务端切换游戏场景。
func show_world_map() -> void:
	_is_world_map_visible = true
	_selected_world_region_button = _current_region_button
	if title_label != null:
		title_label.text = "世界地图"
	if selected_name_label != null:
		selected_name_label.text = "请选择地区" if _current_region_button == null else _current_region_button.tooltip_text.strip_edges()
	_show_map_texture(world_map_texture_rect)
	_hide_all_point_button_containers()
	if world_region_button_container != null:
		world_region_button_container.show()
	if world_map_button != null:
		world_map_button.disabled = true
	if _current_region_button != null:
		_current_region_button.grab_focus()
		_show_selection_for_button(_current_region_button)
	elif selection_animation != null:
		selection_animation.hide()


## 点击世界地图中的闪光镇节点后，在当前地图场景内显示闪光镇地区地图。
func _show_flash_town_map() -> void:
	_current_region_button = flash_town_region_button
	_show_region_map("闪光镇", map_texture_rect, point_button_container)


## 点击世界地图中的闪光平原节点后，在当前地图场景内显示对应地区地图。
func _show_flash_plain_map() -> void:
	_current_region_button = flash_plain_region_button
	_show_region_map("闪光平原", flash_plain_map_texture_rect, flash_plain_point_button_container)


## 点击世界地图中的精灵迷宫节点后，在当前地图场景内显示对应地区地图。
func _show_spirit_maze_map() -> void:
	_current_region_button = spirit_maze_region_button
	_show_region_map("精灵迷宫", spirit_maze_map_texture_rect, null)


## 应用地区地图视图，并仅激活该地区预先配置的地图标点。
## region_name 是地区展示名称；texture_rect 是预设尺寸的地区地图节点；scene_point_container 是该地区可选择的标点容器。
func _show_region_map(region_name: String, texture_rect: TextureRect, scene_point_container: Control) -> void:
	_is_world_map_visible = false
	if title_label != null:
		title_label.text = region_name
	_show_map_texture(texture_rect)
	if world_region_button_container != null:
		world_region_button_container.hide()
	_activate_point_button_container(scene_point_container)
	if world_map_button != null:
		world_map_button.disabled = false
	if scene_point_container != null:
		_select_current_scene_or_first_point()
		_refresh_current_scene_icon()
		return
	if selection_animation != null:
		selection_animation.hide()
	if selected_name_label != null:
		selected_name_label.text = "%s地区地图" % region_name


## 只显示目标地图纹理节点；每张纹理的尺寸都预先保存在场景中，运行时不拉伸或改写 UI 尺寸。
## target_texture_rect 是本次需要展示的地图纹理节点。
func _show_map_texture(target_texture_rect: TextureRect) -> void:
	var texture_rects: Array[TextureRect] = [
		map_texture_rect,
		world_map_texture_rect,
		flash_plain_map_texture_rect,
		spirit_maze_map_texture_rect,
	]
	for texture_rect: TextureRect in texture_rects:
		if texture_rect != null:
			texture_rect.visible = texture_rect == target_texture_rect


## 隐藏所有地区地图标点，保证世界地图或其他地区不会残留上一个地区的热点。
func _hide_all_point_button_containers() -> void:
	if point_button_container != null:
		point_button_container.hide()
	if flash_plain_point_button_container != null:
		flash_plain_point_button_container.hide()
	_hide_all_current_scene_icons()


## 激活目标地区的标点容器，并重新建立该地区独立的选择列表。
## scene_point_container 是本次地区地图使用的场景节点容器，空值表示该地区暂时没有可选节点。
func _activate_point_button_container(scene_point_container: Control) -> void:
	_hide_all_point_button_containers()
	_active_point_button_container = scene_point_container
	_current_scene_icon = null
	_point_buttons.clear()
	_selected_index = 0
	if scene_point_container == null:
		return
	scene_point_container.show()
	_current_scene_icon = scene_point_container.get_node_or_null("人物图标") as TextureRect
	_collect_point_buttons(scene_point_container)


## 按场景树顺序收集指定地区的地图标点按钮并绑定点击事件。
## scene_point_container 是当前地区地图中预先放置所有按钮的容器。
func _collect_point_buttons(scene_point_container: Control) -> void:
	if scene_point_container == null:
		return
	for child: Node in scene_point_container.get_children():
		var point_button: Button = child as Button
		if point_button == null:
			continue
		_point_buttons.append(point_button)
		var pressed_callback: Callable = _on_point_button_pressed.bind(point_button)
		if not point_button.pressed.is_connected(pressed_callback):
			point_button.pressed.connect(pressed_callback)


## 优先选中服务端当前场景对应节点；当前场景不属于该地区时选中第一个节点。
func _select_current_scene_or_first_point() -> void:
	if _point_buttons.is_empty():
		if selection_animation != null:
			selection_animation.hide()
		return
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	var current_point_button: MapTeleportPointButton = _find_point_button_by_scene_id(current_scene_id)
	var target_index: int = 0
	if current_point_button != null:
		target_index = _point_buttons.find(current_point_button)
	_apply_selection(target_index, false)


## 绑定世界地图地区按钮的点击事件；第一次点击选中，第二次点击同一节点才进入地区地图。
## region_button 是场景中预置的地区热点按钮。
func _bind_world_region_button(region_button: Button) -> void:
	if region_button == null:
		return
	var pressed_callback: Callable = _on_world_region_button_pressed.bind(region_button)
	if not region_button.pressed.is_connected(pressed_callback):
		region_button.pressed.connect(pressed_callback)


## 处理世界地图地区节点的二次点击：首次只选中，再次点击当前节点才进入地区地图。
## region_button 是本次点击的世界地图地区热点。
func _on_world_region_button_pressed(region_button: Button) -> void:
	if region_button == null:
		return
	if _selected_world_region_button != region_button:
		_selected_world_region_button = region_button
		if selected_name_label != null:
			selected_name_label.text = region_button.tooltip_text.strip_edges()
		_show_selection_for_button(region_button)
		return
	if region_button == flash_town_region_button:
		_show_flash_town_map()
		return
	if region_button == flash_plain_region_button:
		_show_flash_plain_map()
		return
	if region_button == spirit_maze_region_button:
		_show_spirit_maze_map()


## 把共享四帧动画移动到目标按钮中心，并从第一帧开始循环播放。
## point_button 是当前选中的地区或地图节点按钮。
func _show_selection_for_button(point_button: Button) -> void:
	if selection_animation == null or point_button == null:
		return
	selection_animation.position = point_button.position + point_button.size / 2.0
	selection_animation.show()
	selection_animation.play(&"selected")


## 隐藏所有地区的人物位置图标，避免地区切换后残留旧场景标记。
func _hide_all_current_scene_icons() -> void:
	var town_scene_icon: TextureRect = null
	var flash_plain_scene_icon: TextureRect = null
	if point_button_container != null:
		town_scene_icon = point_button_container.get_node_or_null("人物图标") as TextureRect
	if flash_plain_point_button_container != null:
		flash_plain_scene_icon = flash_plain_point_button_container.get_node_or_null("人物图标") as TextureRect
	if town_scene_icon != null:
		town_scene_icon.hide()
	if flash_plain_scene_icon != null:
		flash_plain_scene_icon.hide()


## 根据服务端权威世界快照，把人物图标移动到当前场景对应地图标点的左下角。
func _refresh_current_scene_icon() -> void:
	_hide_all_current_scene_icons()
	if _current_scene_icon == null:
		return
	if _is_world_map_visible or _active_point_button_container == null or not _active_point_button_container.visible:
		return
	var current_scene_id: int = int(GameState.scene_snapshot.get("scene_id", 0))
	var current_point_button: MapTeleportPointButton = _find_point_button_by_scene_id(current_scene_id)
	if current_point_button == null:
		return
	_current_scene_icon.position = current_point_button.position + Vector2(current_scene_icon_offset)
	_current_scene_icon.show()


## 在当前地区的地图标点中查找目标场景，不额外维护重复的场景映射表。
## scene_id 是服务端世界快照中的当前场景 ID。
func _find_point_button_by_scene_id(scene_id: int) -> MapTeleportPointButton:
	if scene_id <= 0:
		return null
	for point_button: Button in _point_buttons:
		var teleport_button: MapTeleportPointButton = point_button as MapTeleportPointButton
		if teleport_button != null and teleport_button.target_scene_id == scene_id:
			return teleport_button
	return null


## 选中当前标点的上一个标点；到达首项后循环到末项。
func select_previous_point() -> void:
	if _point_buttons.is_empty():
		return
	_apply_selection(posmod(_selected_index - 1, _point_buttons.size()), true)


## 选中当前标点的下一个标点；到达末项后循环到首项。
func select_next_point() -> void:
	if _point_buttons.is_empty():
		return
	_apply_selection(posmod(_selected_index + 1, _point_buttons.size()), true)


## 响应地图标点点击；首次点击只选中，再次点击当前项才请求服务端传送。
## point_button 是本次被玩家点击的场景按钮。
func _on_point_button_pressed(point_button: Button) -> void:
	var point_index: int = _point_buttons.find(point_button)
	if point_index < 0:
		return
	if point_index == _selected_index:
		_request_selected_point_teleport(point_button)
		return
	_apply_selection(point_index, true)


## 根据当前标点的场景配置发起传送；未开放标点只展示提示，不发送无效请求。
## point_button 是当前已选中的地图标点按钮。
func _request_selected_point_teleport(point_button: Button) -> void:
	var teleport_button: MapTeleportPointButton = point_button as MapTeleportPointButton
	var point_name: String = point_button.tooltip_text.strip_edges()
	if teleport_button == null or not teleport_button.teleport_enabled or teleport_button.target_scene_id <= 0:
		notice_requested.emit("%s尚未开放传送。" % point_name)
		return
	teleport_requested.emit(teleport_button.target_scene_id, point_name)


## 应用唯一选中态并刷新地点名称。
## point_index 是目标标点索引；should_focus 表示是否同步移动键盘焦点。
func _apply_selection(point_index: int, should_focus: bool) -> void:
	if _point_buttons.is_empty():
		if selected_name_label != null:
			selected_name_label.text = ""
		return
	_selected_index = clampi(point_index, 0, _point_buttons.size() - 1)
	for button_index: int in range(_point_buttons.size()):
		var point_button: Button = _point_buttons[button_index]
		point_button.set_pressed_no_signal(button_index == _selected_index)
	var selected_button: Button = _point_buttons[_selected_index]
	_show_selection_for_button(selected_button)
	var point_name: String = selected_button.tooltip_text.strip_edges()
	if selected_name_label != null:
		selected_name_label.text = point_name
	if should_focus:
		selected_button.grab_focus()
	map_point_selected.emit(point_name)
