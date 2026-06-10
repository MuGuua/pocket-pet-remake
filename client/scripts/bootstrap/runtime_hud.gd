extends Control
class_name RuntimeHud

@onready var status_label: Label = %StatusLabel
@onready var scene_label: Label = %SceneLabel
@onready var player_label: Label = %PlayerLabel
@onready var log_output: RichTextLabel = %LogOutput

func set_header_texts(status_text: String, scene_text: String, player_text: String) -> void:
	status_label.text = status_text
	scene_label.text = scene_text
	player_label.text = player_text

func append_log(message: String) -> void:
	if log_output != null:
		log_output.append_text(message + "\n")
