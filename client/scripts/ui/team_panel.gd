extends PanelContainer

# 队伍面板直接展示服务端同步下来的编队宠物，不再保留固定文案。
const MAX_MEMBER_ROWS: int = 4
const UiFormat = preload("res://scripts/common/ui_format.gd")

@onready var tips_label: Label = $RootVBox/BodyPanel/BodyVBox/TipsLabel
@onready var member_labels: Array[Label] = [
	$RootVBox/BodyPanel/BodyVBox/Member1/HBox/Info,
	$RootVBox/BodyPanel/BodyVBox/Member2/HBox/Info,
	$RootVBox/BodyPanel/BodyVBox/Member3/HBox/Info,
	$RootVBox/BodyPanel/BodyVBox/Member4/HBox/Info,
]
@onready var bottom_info_label: Label = $RootVBox/BodyPanel/BodyVBox/BottomInfo


func _ready() -> void:
	if not GameState.pets_changed.is_connected(refresh_panel_data):
		GameState.pets_changed.connect(refresh_panel_data)
	if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.connect(refresh_panel_data)
	refresh_panel_data()


func _exit_tree() -> void:
	if GameState.pets_changed.is_connected(refresh_panel_data):
		GameState.pets_changed.disconnect(refresh_panel_data)
	if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.disconnect(refresh_panel_data)


func refresh_panel_data() -> void:
	var lineup: Array = GameState.lineup
	tips_label.text = UiFormat.normalize_text("当前出战编队 %d / %d" % [lineup.size(), MAX_MEMBER_ROWS])

	for index in range(member_labels.size()):
		var label := member_labels[index]
		if label == null:
			continue
		if index >= lineup.size():
			label.text = "空位，等待服务端编队配置。"
			continue
		var lineup_variant: Variant = lineup[index]
		if lineup_variant is Dictionary:
			label.text = _format_lineup_pet(index, lineup_variant)
		else:
			label.text = "编队数据格式异常。"

	bottom_info_label.text = _build_bottom_summary(lineup)


func _format_lineup_pet(index: int, lineup_pet: Dictionary) -> String:
	var pet_name := _find_pet_name(int(lineup_pet.get("pet_uid", 0)), int(lineup_pet.get("pet_id", 0)))
	return UiFormat.normalize_text("%s  Lv.%d  HP %d/%d  阵位 %d" % [
		pet_name,
		int(lineup_pet.get("level", 0)),
		int(lineup_pet.get("hp", 0)),
		int(lineup_pet.get("hp_max", 0)),
		index + 1,
	])


func _find_pet_name(pet_uid: int, pet_id: int) -> String:
	for pet_variant in GameState.pets:
		if pet_variant is Dictionary and int(pet_variant.get("pet_uid", 0)) == pet_uid:
			return str(pet_variant.get("name", "宠物%d" % pet_id))
	return "宠物%d" % pet_id


func _build_bottom_summary(lineup: Array) -> String:
	if lineup.is_empty():
		return "当前没有从服务端拿到出战编队，请先确认宠物列表返回成功。"
	return UiFormat.normalize_text("当前共同步到 %d 只出战宠物；如果调整编队，服务端回包后这里会自动刷新。" % lineup.size())
