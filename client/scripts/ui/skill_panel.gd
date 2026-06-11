extends PanelContainer

# 技能面板当前先展示玩家快照中的服务端 skill_ids，避免继续显示伪造技能名。
const MAX_VISIBLE_SKILLS: int = 3
const UiFormat = preload("res://scripts/common/ui_format.gd")

@onready var skill_labels: Array[Label] = [
	$RootVBox/BodyPanel/BodyVBox/SkillList/Skill1/HBox/Info,
	$RootVBox/BodyPanel/BodyVBox/SkillList/Skill2/HBox/Info,
	$RootVBox/BodyPanel/BodyVBox/SkillList/Skill3/HBox/Info,
]
@onready var desc_label: Label = $RootVBox/BodyPanel/BodyVBox/DescLabel


func _ready() -> void:
	if not GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.connect(refresh_panel_data)
	if not GameState.session_changed.is_connected(refresh_panel_data):
		GameState.session_changed.connect(refresh_panel_data)
	refresh_panel_data()


func _exit_tree() -> void:
	if GameState.world_snapshot_changed.is_connected(refresh_panel_data):
		GameState.world_snapshot_changed.disconnect(refresh_panel_data)
	if GameState.session_changed.is_connected(refresh_panel_data):
		GameState.session_changed.disconnect(refresh_panel_data)


func refresh_panel_data() -> void:
	var skill_ids: Array = []
	var skill_ids_variant: Variant = GameState.player_snapshot.get("skill_ids", [])
	if skill_ids_variant is Array:
		skill_ids = skill_ids_variant

	for index in range(skill_labels.size()):
		var label := skill_labels[index]
		if label == null:
			continue
		if index >= skill_ids.size():
			label.text = "等待服务端同步技能槽位。"
			continue
		label.text = _format_skill_line(index, skill_ids[index])

	desc_label.text = _build_desc(skill_ids)


func _format_skill_line(index: int, skill_id_variant: Variant) -> String:
	var skill_id := int(skill_id_variant)
	return UiFormat.normalize_text("技能槽 %d\n技能ID %d\n来源：player_snapshot.skill_ids" % [index + 1, skill_id])


func _build_desc(skill_ids: Array) -> String:
	if skill_ids.is_empty():
		return "当前人物快照里还没有 skill_ids，请先确认 EnterWorldResp.player 已正确下发。"
	if skill_ids.size() <= MAX_VISIBLE_SKILLS:
		return UiFormat.normalize_text("当前共同步到 %d 个角色技能，后续若补充技能配置表，可在此替换成正式技能名和描述。" % skill_ids.size())
	return UiFormat.normalize_text("当前共同步到 %d 个角色技能，仅展示前 %d 个。其余技能 ID：%s" % [
		skill_ids.size(),
		MAX_VISIBLE_SKILLS,
		str(skill_ids.slice(MAX_VISIBLE_SKILLS, skill_ids.size())),
	])
