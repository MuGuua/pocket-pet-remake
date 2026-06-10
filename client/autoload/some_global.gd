extends Node

@export var default_player_name: String = "玩家名"
@export var default_player_title: String = "称号"
@export var player_portrait: Texture2D = preload("res://asset/口袋所有形象/imgs/53.png")

var some_character_name: String = "玩家名"
var player_title: String = "称号"

func _ready() -> void:
	process_mode = Node.PROCESS_MODE_ALWAYS
	GameState.session_changed.connect(_refresh_from_game_state)
	GameState.world_snapshot_changed.connect(_refresh_from_game_state)
	_refresh_from_game_state()

func _refresh_from_game_state() -> void:
	some_character_name = str(GameState.player_snapshot.get("name", default_player_name))
	player_title = str(GameState.player_snapshot.get("title", default_player_title))
