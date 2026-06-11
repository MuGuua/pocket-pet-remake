class_name UiFormat
extends RefCounted

# 把 UI 与动画文本里出现的浮点数统一转成整数文本，避免界面出现 1.0、23.5 这类展示。
static func value_to_text(value: Variant) -> String:
	match typeof(value):
		TYPE_FLOAT:
			return str(int(round(float(value))))
		TYPE_INT:
			return str(int(value))
		TYPE_STRING:
			return _normalize_numeric_fragments(str(value))
		_:
			return str(value)


# 一些 UI 文案会把数值拼进长字符串里，这里用正则把其中的浮点片段逐个转成整数。
static func normalize_text(text: String) -> String:
	return _normalize_numeric_fragments(text)


static func _normalize_numeric_fragments(text: String) -> String:
	var regex := RegEx.new()
	var compile_error := regex.compile("-?\\d+\\.\\d+")
	if compile_error != OK:
		return text

	var result := text
	var matches := regex.search_all(text)
	for index in range(matches.size() - 1, -1, -1):
		var match: RegExMatch = matches[index]
		if match == null:
			continue
		var matched_text := match.get_string()
		var normalized_text := str(int(round(matched_text.to_float())))
		result = result.substr(0, match.get_start()) + normalized_text + result.substr(match.get_end())
	return result
