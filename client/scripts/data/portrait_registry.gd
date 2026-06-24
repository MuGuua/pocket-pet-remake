class_name PortraitRegistry
extends RefCounted

const CharacterSkinRegistry = preload("res://scripts/feature/character/character_skin_registry.gd")

## 玩家无 skin_id 时的兜底形象，与 HUD 头像逻辑保持一致。
const DEFAULT_PLAYER_SKIN_ID: String = "初始形象男_001"
## 对话角标只展示精灵帧高度的上半部分，用于半身头像效果。
const DIALOGUE_PORTRAIT_UPPER_BODY_RATIO: float = 0.5

## NPC portrait_key 到场景首帧 atlas 区域的映射；需与对应 NPC 场景 AnimatedSprite2D 一致。
const NPC_PORTRAIT_FRAME_BY_KEY: Dictionary = {
	"npc_limeng_normal": {
		"atlas_path": "res://asset/口袋所有形象/imgs/2006.png",
		"region": Rect2(0.0, 0.0, 18.0, 37.0),
	},
	"npc_limeng_smile": {
		"atlas_path": "res://asset/口袋所有形象/imgs/2006.png",
		"region": Rect2(0.0, 0.0, 18.0, 37.0),
	},
	"npc_luosi_normal": {
		"atlas_path": "res://asset/分类/NPC/2000.png",
		"region": Rect2(0.0, 0.0, 18.0, 38.0),
	},
	"npc_luoge_normal": {
		"atlas_path": "res://asset/口袋所有形象/imgs/2005.png",
		"region": Rect2(0.0, 0.0, 19.0, 39.0),
	},
	"npc_doro_normal": {
		"atlas_path": "res://asset/分类/NPC/DORO1.png",
		"region": Rect2(0.0, 0.0, 48.0, 48.0),
	},
	"default": {
		"atlas_path": "res://asset/分类/NPC/2000.png",
		"region": Rect2(0.0, 0.0, 18.0, 38.0),
	},
}

## 当服务端未下发 portrait_key 时，按说话人 / 当前 NPC 名称兜底到稳定立绘 key。
const NPC_PORTRAIT_KEY_BY_NAME: Dictionary = {
	"市场理萌": "npc_limeng_normal",
	"理萌": "npc_limeng_normal",
	"罗思": "npc_luosi_normal",
	"luosi": "npc_luosi_normal",
	"市场罗格": "npc_luoge_normal",
	"罗格": "npc_luoge_normal",
	"doro": "npc_doro_normal",
	"Doro": "npc_doro_normal",
	"DORO": "npc_doro_normal",
}


## 加载对话 UI 半身头像；玩家取待机首帧上半截，NPC 取形象首帧上半截。
static func load_dialogue_portrait(portrait_key: String, is_player: bool) -> Texture2D:
	if is_player:
		return load_player_dialogue_portrait()
	return load_npc_dialogue_portrait(portrait_key)


## 加载玩家对话头像：优先当前 skin_id 的世界待机首帧，再取上半截。
static func load_player_dialogue_portrait() -> Texture2D:
	var skin_id: String = str(GameState.player_snapshot.get("skin_id", "")).strip_edges()
	if skin_id.is_empty():
		skin_id = DEFAULT_PLAYER_SKIN_ID
	var skin: UnitSkin = CharacterSkinRegistry.get_unit_skin(skin_id)
	if skin == null:
		skin = CharacterSkinRegistry.get_unit_skin(DEFAULT_PLAYER_SKIN_ID)
	if skin == null:
		return null
	var idle_frame: Texture2D = skin.resolve_avatar_preview_texture()
	return crop_upper_body_portrait(idle_frame)


## 加载 NPC 对话头像：按 portrait_key 取场景一致的首帧，再取上半截。
static func load_npc_dialogue_portrait(portrait_key: String) -> Texture2D:
	var normalized_key: String = portrait_key.strip_edges()
	if normalized_key.is_empty() or not NPC_PORTRAIT_FRAME_BY_KEY.has(normalized_key):
		normalized_key = "default"
	var frame_config_variant: Variant = NPC_PORTRAIT_FRAME_BY_KEY.get(normalized_key, {})
	if frame_config_variant is not Dictionary:
		return null
	var frame_config: Dictionary = frame_config_variant as Dictionary
	var frame_texture: Texture2D = _load_atlas_frame(
		str(frame_config.get("atlas_path", "")),
		frame_config.get("region", Rect2()) as Rect2
	)
	return crop_upper_body_portrait(frame_texture)


## 对任意单帧贴图裁剪上半截，供对话角标或旧资源兜底复用。
static func crop_upper_body_portrait(source_texture: Texture2D) -> Texture2D:
	if source_texture == null:
		return null
	var source_image: Image = _extract_image_from_texture(source_texture)
	if source_image == null:
		return source_texture
	var frame_height: int = source_image.get_height()
	var upper_height: int = maxi(1, int(float(frame_height) * DIALOGUE_PORTRAIT_UPPER_BODY_RATIO))
	var cropped_image: Image = source_image.get_region(
		Rect2i(0, 0, source_image.get_width(), upper_height)
	)
	return ImageTexture.create_from_image(cropped_image)


## 根据 portrait_key 加载 Texture2D；兼容旧调用，等价于 NPC 对话头像。
static func load_texture_by_key(portrait_key: String) -> Texture2D:
	return load_npc_dialogue_portrait(portrait_key)


## 把后台配置的说话人占位符解析成实际展示名；@player 会替换成当前登录玩家名。
static func resolve_speaker_display(speaker: String, npc_fallback: String) -> String:
	var normalized_speaker: String = speaker.strip_edges()
	if normalized_speaker == "@player" or normalized_speaker == "$player" or normalized_speaker == "{player_name}" or normalized_speaker == "玩家":
		var player_name: String = str(GameState.player_snapshot.get("name", "")).strip_edges()
		if player_name.is_empty():
			return "训练家"
		return player_name
	if normalized_speaker.is_empty():
		return npc_fallback
	return normalized_speaker


## 根据 portrait_key、说话人和当前交互 NPC 名称推导最终立绘 key。
static func resolve_portrait_key(portrait_key: String, speaker: String, npc_name: String) -> String:
	var normalized_key: String = portrait_key.strip_edges()
	if not normalized_key.is_empty():
		return normalized_key
	var normalized_speaker: String = speaker.strip_edges()
	if normalized_speaker == "@player" or normalized_speaker == "$player" or normalized_speaker == "{player_name}" or normalized_speaker == "玩家":
		return "player_default"
	var speaker_key: String = _resolve_npc_portrait_key_by_name(normalized_speaker)
	if not speaker_key.is_empty():
		return speaker_key
	var npc_key: String = _resolve_npc_portrait_key_by_name(npc_name.strip_edges())
	if not npc_key.is_empty():
		return npc_key
	return "default"


## 按说话人或当前交互 NPC 名称映射到预设立绘 key，减少后台漏配 portrait_key 时的客户端退化。
static func _resolve_npc_portrait_key_by_name(name_text: String) -> String:
	var normalized_name: String = name_text.strip_edges()
	if normalized_name.is_empty():
		return ""
	if NPC_PORTRAIT_KEY_BY_NAME.has(normalized_name):
		return str(NPC_PORTRAIT_KEY_BY_NAME.get(normalized_name, ""))
	return ""


## 从 atlas 路径与区域构造单帧 AtlasTexture。
static func _load_atlas_frame(atlas_path: String, region: Rect2) -> Texture2D:
	if atlas_path.is_empty() or region.size == Vector2.ZERO:
		return null
	var atlas_variant: Variant = load(atlas_path)
	if not atlas_variant is Texture2D:
		return null
	var atlas_texture: AtlasTexture = AtlasTexture.new()
	atlas_texture.atlas = atlas_variant as Texture2D
	atlas_texture.region = region
	return atlas_texture


## 将 AtlasTexture / ImageTexture 等统一提取为可裁剪的 Image。
static func _extract_image_from_texture(source_texture: Texture2D) -> Image:
	if source_texture == null:
		return null
	if source_texture is AtlasTexture:
		var atlas_tex: AtlasTexture = source_texture as AtlasTexture
		if atlas_tex.atlas == null:
			return null
		var atlas_image: Image = atlas_tex.atlas.get_image()
		if atlas_image == null:
			return null
		var region: Rect2i = Rect2i(
			int(atlas_tex.region.position.x),
			int(atlas_tex.region.position.y),
			int(atlas_tex.region.size.x),
			int(atlas_tex.region.size.y)
		)
		return atlas_image.get_region(region)
	if source_texture is ImageTexture:
		return (source_texture as ImageTexture).get_image()
	return source_texture.get_image()
