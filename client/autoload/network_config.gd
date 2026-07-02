extends RefCounted
class_name NetworkConfig

## 本地开发配置默认注释；需要连接本机 8080 时，取消这一组注释，并注释掉下方服务器配置。
const ACTIVE_HOST: String = "127.0.0.1"
const ACTIVE_HTTP_PORT: int = 8080
const ACTIVE_WS_PORT: int = 8080

### 服务器配置默认启用；需要连接正式服务器时保持这一组开启。
#const ACTIVE_HOST: String = "117.72.124.51"
### 当前 HTTP 对外访问端口。
#const ACTIVE_HTTP_PORT: int = 9002
### 当前 WebSocket 对外访问端口。
#const ACTIVE_WS_PORT: int = 9002

## 根据当前启用配置返回 HTTP 基础地址。
static func get_http_base_url() -> String:
	return "http://%s:%d" % [ACTIVE_HOST, ACTIVE_HTTP_PORT]

## 根据当前启用配置返回 WebSocket 完整地址。
static func get_ws_url() -> String:
	return "ws://%s:%d/ws" % [ACTIVE_HOST, ACTIVE_WS_PORT]

## Web 导出时，根据当前页面协议与主机名拼出 HTTP 基础地址。
## protocol 传入 window.location.protocol，hostname 传入 window.location.hostname。
static func build_web_http_base_url(protocol: String, hostname: String) -> String:
	var http_scheme: String = "https" if protocol == "https:" else "http"
	return "%s://%s:%d" % [http_scheme, hostname, ACTIVE_HTTP_PORT]

## Web 导出时，根据当前页面协议与主机名拼出 WebSocket 地址。
## protocol 传入 window.location.protocol，hostname 传入 window.location.hostname。
static func build_web_ws_url(protocol: String, hostname: String) -> String:
	var ws_scheme: String = "wss" if protocol == "https:" else "ws"
	return "%s://%s:%d/ws" % [ws_scheme, hostname, ACTIVE_WS_PORT]
