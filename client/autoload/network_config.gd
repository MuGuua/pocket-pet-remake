extends RefCounted
class_name NetworkConfig

## 本地开发环境标识。
const PROFILE_LOCAL: String = "local"
## 远程正式环境标识。
const PROFILE_REMOTE: String = "remote"
## 浏览器同源环境标识。
const PROFILE_BROWSER_ORIGIN: String = "browser_origin"

## 原生端默认走哪套网络环境。
const ACTIVE_NATIVE_PROFILE: String = PROFILE_LOCAL
## Web 导出默认走哪套网络环境。
const ACTIVE_WEB_PROFILE: String = PROFILE_LOCAL

## 本地开发服务地址。
const LOCAL_HOST: String = "127.0.0.1"
const LOCAL_HTTP_PORT: int = 8080
const LOCAL_WS_PORT: int = 8080

## 远程服务地址。
const REMOTE_HOST: String = "touwomugua.cn"
const REMOTE_HTTP_PORT: int = 443
const REMOTE_WS_PORT: int = 443

## 浏览器地址栏里用于切换环境的参数键。
const WEB_PROFILE_QUERY_KEY: String = "server"
## 浏览器地址栏里用于临时覆盖 HTTP 地址的参数键。
const WEB_HTTP_BASE_QUERY_KEY: String = "http_base"
## 浏览器地址栏里用于临时覆盖 WebSocket 地址的参数键。
const WEB_WS_BASE_QUERY_KEY: String = "ws_base"

## 浏览器本地存储中用于持久化环境的键。
const WEB_PROFILE_STORAGE_KEY: String = "pp_server_profile"
## 浏览器本地存储中用于持久化 HTTP 地址的键。
const WEB_HTTP_BASE_STORAGE_KEY: String = "pp_http_base"
## 浏览器本地存储中用于持久化 WebSocket 地址的键。
const WEB_WS_BASE_STORAGE_KEY: String = "pp_ws_base"

## 当前运行时内存里生效的环境覆盖；原生端也可通过它临时切换。
static var _runtime_profile_override: String = ""
## 当前运行时内存里生效的 HTTP 地址覆盖。
static var _runtime_http_base_override: String = ""
## 当前运行时内存里生效的 WebSocket 地址覆盖。
static var _runtime_ws_base_override: String = ""
## 当调试面板主动切服后，当前运行期内忽略地址栏参数，避免又被旧 query 覆盖回去。
static var _runtime_ignore_web_overrides: bool = false

## 返回当前最终生效的 HTTP 基础地址。
static func get_http_base_url() -> String:
	return resolve_http_base_url()

## 返回当前最终生效的 WebSocket 完整地址。
static func get_ws_url() -> String:
	return resolve_ws_url()

## 按当前运行平台和浏览器覆盖规则解析 HTTP 基础地址。
static func resolve_http_base_url() -> String:
	if not _runtime_http_base_override.is_empty():
		return _runtime_http_base_override.trim_suffix("/")

	if not _runtime_ignore_web_overrides:
		var override_http_base: String = _get_web_override_value(WEB_HTTP_BASE_QUERY_KEY, WEB_HTTP_BASE_STORAGE_KEY)
		if not override_http_base.is_empty():
			return override_http_base.trim_suffix("/")

	var profile_name: String = _resolve_active_profile()
	if profile_name == PROFILE_BROWSER_ORIGIN:
		var browser_http_base: String = _build_browser_http_base_url()
		if not browser_http_base.is_empty():
			return browser_http_base

	return _get_profile_http_base_url(profile_name)

## 按当前运行平台和浏览器覆盖规则解析 WebSocket 地址。
static func resolve_ws_url() -> String:
	if not _runtime_ws_base_override.is_empty():
		return _runtime_ws_base_override.trim_suffix("/")

	if not _runtime_http_base_override.is_empty():
		return _build_ws_url_from_http_base(_runtime_http_base_override)

	if not _runtime_ignore_web_overrides:
		var override_ws_url: String = _get_web_override_value(WEB_WS_BASE_QUERY_KEY, WEB_WS_BASE_STORAGE_KEY)
		if not override_ws_url.is_empty():
			return override_ws_url.trim_suffix("/")

		var override_http_base: String = _get_web_override_value(WEB_HTTP_BASE_QUERY_KEY, WEB_HTTP_BASE_STORAGE_KEY)
		if not override_http_base.is_empty():
			return _build_ws_url_from_http_base(override_http_base)

	var profile_name: String = _resolve_active_profile()
	if profile_name == PROFILE_BROWSER_ORIGIN:
		var browser_ws_url: String = _build_browser_ws_url()
		if not browser_ws_url.is_empty():
			return browser_ws_url

	return _get_profile_ws_url(profile_name)

## Web 导出时，根据当前页面协议与 host（含非默认端口）拼出 HTTP 基础地址。
## protocol 传入 window.location.protocol，host 传入 window.location.host。
static func build_web_http_base_url(protocol: String, host: String) -> String:
	var http_scheme: String = "https" if protocol == "https:" else "http"
	var normalized_host: String = host.strip_edges()
	if normalized_host.is_empty():
		return ""
	return "%s://%s" % [http_scheme, normalized_host]

## Web 导出时，根据当前页面协议与 host 拼出 WebSocket 地址。
## protocol 传入 window.location.protocol，host 传入 window.location.host。
static func build_web_ws_url(protocol: String, host: String) -> String:
	var ws_scheme: String = "wss" if protocol == "https:" else "ws"
	var normalized_host: String = host.strip_edges()
	if normalized_host.is_empty():
		return ""
	return "%s://%s/ws" % [ws_scheme, normalized_host]

## 返回当前环境实际使用的环境标识。
static func _resolve_active_profile() -> String:
	if _is_supported_profile(_runtime_profile_override):
		return _runtime_profile_override

	if _runtime_ignore_web_overrides:
		return ACTIVE_WEB_PROFILE if OS.has_feature("web") else ACTIVE_NATIVE_PROFILE

	if not OS.has_feature("web"):
		return ACTIVE_NATIVE_PROFILE

	var profile_name: String = _get_web_override_value(WEB_PROFILE_QUERY_KEY, WEB_PROFILE_STORAGE_KEY)
	if profile_name.is_empty():
		profile_name = ACTIVE_WEB_PROFILE

	if not _is_supported_profile(profile_name):
		return ACTIVE_WEB_PROFILE
	return profile_name

## 按环境标识返回 HTTP 地址。
static func _get_profile_http_base_url(profile_name: String) -> String:
	match profile_name:
		PROFILE_LOCAL:
			return _format_origin_url(_http_scheme_for_port(LOCAL_HTTP_PORT), LOCAL_HOST, LOCAL_HTTP_PORT)
		PROFILE_REMOTE:
			return _format_origin_url(_http_scheme_for_port(REMOTE_HTTP_PORT), REMOTE_HOST, REMOTE_HTTP_PORT)
		PROFILE_BROWSER_ORIGIN:
			return _build_browser_http_base_url()
		_:
			return _format_origin_url(_http_scheme_for_port(REMOTE_HTTP_PORT), REMOTE_HOST, REMOTE_HTTP_PORT)

## 按环境标识返回 WebSocket 地址。
static func _get_profile_ws_url(profile_name: String) -> String:
	match profile_name:
		PROFILE_LOCAL:
			return "%s/ws" % _format_origin_url(_ws_scheme_for_port(LOCAL_WS_PORT), LOCAL_HOST, LOCAL_WS_PORT)
		PROFILE_REMOTE:
			return "%s/ws" % _format_origin_url(_ws_scheme_for_port(REMOTE_WS_PORT), REMOTE_HOST, REMOTE_WS_PORT)
		PROFILE_BROWSER_ORIGIN:
			return _build_browser_ws_url()
		_:
			return "%s/ws" % _format_origin_url(_ws_scheme_for_port(REMOTE_WS_PORT), REMOTE_HOST, REMOTE_WS_PORT)

## 从浏览器环境读取临时覆盖值；优先地址栏，其次本地存储。
static func _get_web_override_value(query_key: String, storage_key: String) -> String:
	if not OS.has_feature("web"):
		return ""

	var query_value: String = _get_web_query_param(query_key)
	if not query_value.is_empty():
		return query_value

	return _get_web_storage_value(storage_key)

## 读取浏览器地址栏中的单个参数。
static func _get_web_query_param(param_name: String) -> String:
	if not OS.has_feature("web"):
		return ""

	var script: String = "(() => { const value = new URLSearchParams(window.location.search).get('%s'); return value ?? ''; })()" % param_name
	return str(JavaScriptBridge.eval(script, true)).strip_edges()

## 读取浏览器本地存储中的单个值。
static func _get_web_storage_value(storage_key: String) -> String:
	if not OS.has_feature("web"):
		return ""

	var script: String = "(() => { const value = window.localStorage.getItem('%s'); return value ?? ''; })()" % storage_key
	return str(JavaScriptBridge.eval(script, true)).strip_edges()

## 写入浏览器本地存储中的单个值；空值时删除，避免残留无意义覆盖。
static func _set_web_storage_value(storage_key: String, value: String) -> void:
	if not OS.has_feature("web"):
		return

	var normalized_value: String = value.strip_edges()
	if normalized_value.is_empty():
		_remove_web_storage_value(storage_key)
		return

	var script: String = "window.localStorage.setItem('%s', '%s')" % [
		storage_key,
		_escape_js_single_quoted_string(normalized_value),
	]
	JavaScriptBridge.eval(script, true)

## 删除浏览器本地存储中的单个值。
static func _remove_web_storage_value(storage_key: String) -> void:
	if not OS.has_feature("web"):
		return

	var script: String = "window.localStorage.removeItem('%s')" % storage_key
	JavaScriptBridge.eval(script, true)

## 根据浏览器当前页面生成 HTTP 地址。
static func _build_browser_http_base_url() -> String:
	if not OS.has_feature("web"):
		return ""

	var protocol: String = str(JavaScriptBridge.eval("window.location.protocol", true))
	var host: String = str(JavaScriptBridge.eval("window.location.host", true))
	if host.strip_edges().is_empty():
		return ""

	return build_web_http_base_url(protocol, host)

## 根据浏览器当前页面生成 WebSocket 地址。
static func _build_browser_ws_url() -> String:
	if not OS.has_feature("web"):
		return ""

	var protocol: String = str(JavaScriptBridge.eval("window.location.protocol", true))
	var host: String = str(JavaScriptBridge.eval("window.location.host", true))
	if host.strip_edges().is_empty():
		return ""

	return build_web_ws_url(protocol, host)

## 当只给了 HTTP 地址时，自动补出同一主机的 WebSocket 地址。
static func _build_ws_url_from_http_base(http_base: String) -> String:
	var normalized_http_base: String = http_base.strip_edges().trim_suffix("/")
	if normalized_http_base.begins_with("https://"):
		return "wss://%s/ws" % normalized_http_base.trim_prefix("https://")
	if normalized_http_base.begins_with("http://"):
		return "ws://%s/ws" % normalized_http_base.trim_prefix("http://")
	return normalized_http_base

## 转义写入浏览器脚本时需要的单引号和反斜杠。
static func _escape_js_single_quoted_string(value: String) -> String:
	return value.replace("\\", "\\\\").replace("'", "\\'")

## 判断传入的环境标识是否受支持。
static func _is_supported_profile(profile_name: String) -> bool:
	return profile_name == PROFILE_LOCAL or profile_name == PROFILE_REMOTE or profile_name == PROFILE_BROWSER_ORIGIN

## 返回当前生效的环境标识，供调试面板展示。
static func get_active_profile() -> String:
	return _resolve_active_profile()

## 返回当前生效的 HTTP 地址，供调试面板展示。
static func get_active_http_base_url() -> String:
	return resolve_http_base_url()

## 返回当前生效的 WebSocket 地址，供调试面板展示。
static func get_active_ws_url() -> String:
	return resolve_ws_url()

## 返回当前手动填写的 HTTP 覆盖值；没有时返回空字符串。
static func get_manual_http_base_override() -> String:
	if not _runtime_http_base_override.is_empty():
		return _runtime_http_base_override
	if _runtime_ignore_web_overrides:
		return ""
	return _get_web_override_value(WEB_HTTP_BASE_QUERY_KEY, WEB_HTTP_BASE_STORAGE_KEY)

## 返回当前手动填写的 WebSocket 覆盖值；没有时返回空字符串。
static func get_manual_ws_url_override() -> String:
	if not _runtime_ws_base_override.is_empty():
		return _runtime_ws_base_override
	if _runtime_ignore_web_overrides:
		return ""
	return _get_web_override_value(WEB_WS_BASE_QUERY_KEY, WEB_WS_BASE_STORAGE_KEY)

## 应用开发调试使用的网络环境切换。
## profile_name 传空表示沿用默认环境；http_base / ws_url 传空表示按环境自动拼接。
static func apply_debug_config(profile_name: String, http_base: String = "", ws_url: String = "") -> void:
	var normalized_profile: String = profile_name.strip_edges()
	_runtime_profile_override = normalized_profile if _is_supported_profile(normalized_profile) else ""
	_runtime_http_base_override = http_base.strip_edges().trim_suffix("/")
	_runtime_ws_base_override = ws_url.strip_edges().trim_suffix("/")
	_runtime_ignore_web_overrides = true

	if OS.has_feature("web"):
		_set_web_storage_value(WEB_PROFILE_STORAGE_KEY, _runtime_profile_override)
		_set_web_storage_value(WEB_HTTP_BASE_STORAGE_KEY, _runtime_http_base_override)
		_set_web_storage_value(WEB_WS_BASE_STORAGE_KEY, _runtime_ws_base_override)

## 清空运行时和浏览器本地存储里的调试切服覆盖。
static func clear_debug_config() -> void:
	_runtime_profile_override = ""
	_runtime_http_base_override = ""
	_runtime_ws_base_override = ""
	_runtime_ignore_web_overrides = false

	if OS.has_feature("web"):
		_remove_web_storage_value(WEB_PROFILE_STORAGE_KEY)
		_remove_web_storage_value(WEB_HTTP_BASE_STORAGE_KEY)
		_remove_web_storage_value(WEB_WS_BASE_STORAGE_KEY)

## 拼接协议 + 主机 + 端口；80/443 等默认端口省略，避免 Web 端出现多余的 :80。
static func _format_origin_url(scheme: String, hostname: String, port: int) -> String:
	if (scheme == "http" or scheme == "ws") and port == 80:
		return "%s://%s" % [scheme, hostname]
	if (scheme == "https" or scheme == "wss") and port == 443:
		return "%s://%s" % [scheme, hostname]
	return "%s://%s:%d" % [scheme, hostname, port]

## 按端口选择 HTTP / HTTPS 协议（桌面端连 443 时使用 https）。
static func _http_scheme_for_port(port: int) -> String:
	return "https" if port == 443 else "http"

## 按端口选择 WS / WSS 协议（桌面端连 443 时使用 wss）。
static func _ws_scheme_for_port(port: int) -> String:
	return "wss" if port == 443 else "ws"
