extends RefCounted
class_name NetworkConfig

## 本地开发配置；导出 Web 前可保持开启，浏览器运行时会忽略此处端口并改用当前页面 origin。
const ACTIVE_HOST: String = "127.0.0.1"
const ACTIVE_HTTP_PORT: int = 8080
const ACTIVE_WS_PORT: int = 8080

### 桌面客户端连正式服时启用（Web 浏览器会自动读当前域名，无需改这里）。
#const ACTIVE_HOST: String = "touwomugua.cn"
#const ACTIVE_HTTP_PORT: int = 443
#const ACTIVE_WS_PORT: int = 443

## 根据当前启用配置返回 HTTP 基础地址。
static func get_http_base_url() -> String:
	return _format_origin_url(_http_scheme_for_port(ACTIVE_HTTP_PORT), ACTIVE_HOST, ACTIVE_HTTP_PORT)

## 根据当前启用配置返回 WebSocket 完整地址。
static func get_ws_url() -> String:
	return "%s/ws" % _format_origin_url(_ws_scheme_for_port(ACTIVE_WS_PORT), ACTIVE_HOST, ACTIVE_WS_PORT)

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
