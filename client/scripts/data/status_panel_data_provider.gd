class_name StatusPanelDataProvider
extends RefCounted

const DATA_FILE_PATH := "res://data/status_panel_data.json"


static func get_panel_data(overrides: Dictionary = {}) -> Dictionary:
	var data := _load_panel_data()
	return _merge_dictionary(data, overrides)


static func get_section(section_name: String, overrides: Dictionary = {}) -> Dictionary:
	var data := get_panel_data()
	var section: Variant = data.get(section_name, {})
	if typeof(section) != TYPE_DICTIONARY:
		section = {}
	return _merge_dictionary(section, overrides)


static func _load_panel_data() -> Dictionary:
	if not FileAccess.file_exists(DATA_FILE_PATH):
		return {}

	var file := FileAccess.open(DATA_FILE_PATH, FileAccess.READ)
	if file == null:
		return {}

	var parsed: Variant = JSON.parse_string(file.get_as_text())
	if typeof(parsed) != TYPE_DICTIONARY:
		return {}

	return parsed.duplicate(true)


static func _merge_dictionary(base: Dictionary, overrides: Dictionary) -> Dictionary:
	var merged := base.duplicate(true)
	for key in overrides.keys():
		var override_value: Variant = overrides[key]
		var base_value: Variant = merged.get(key)
		if typeof(base_value) == TYPE_DICTIONARY and typeof(override_value) == TYPE_DICTIONARY:
			merged[key] = _merge_dictionary(base_value, override_value)
		else:
			merged[key] = override_value
	return merged
