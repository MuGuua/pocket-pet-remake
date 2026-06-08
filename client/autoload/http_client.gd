extends Node

# 默认请求服务端 HTTP 接口时使用的基础地址。
const DEFAULT_BASE_URL: String = "http://127.0.0.1:8080"

# 当前实际使用的 HTTP 基础地址。
var _base_url: String = DEFAULT_BASE_URL
# 负责发起 HTTP 请求的节点实例。
var _request: HTTPRequest

# 初始化 HTTPRequest 节点并挂到单例下。
func _ready() -> void:
	_request = HTTPRequest.new()
	add_child(_request)

# 更新当前请求服务端时使用的基础地址。
func set_base_url(base_url: String) -> void:
	_base_url = base_url.trim_suffix("/")

# 发起登录接口请求，并返回统一字典结构结果。
func login(account: String, password: String) -> Dictionary:
	return await _request_json(
		"/api/v1/auth/login",
		HTTPClient.METHOD_POST,
		{
			"account": account,
			"password": password,
		}
	)

# 以 JSON 请求体发起 HTTP 请求，并把响应规范化为统一结构。
func _request_json(path: String, method: int, payload: Dictionary = {}) -> Dictionary:
	if _request == null:
		return {
			"code": ERR_UNCONFIGURED,
			"msg": "http client is not ready",
			"data": {},
		}

	# 组装当前请求使用的 HTTP 头。
	var headers := PackedStringArray(["Content-Type: application/json"])
	# 把请求载荷序列化为 JSON 文本。
	var body := JSON.stringify(payload)
	# 发起当前 HTTP 请求。
	var err := _request.request(_base_url + path, headers, method, body)
	if err != OK:
		return {
			"code": err,
			"msg": error_string(err),
			"data": {},
		}

	# 等待 HTTPRequest 返回完整响应元组。
	var result: Array = await _request.request_completed
	if result.size() < 4:
		return {
			"code": ERR_PARSE_ERROR,
			"msg": "invalid http response tuple",
			"data": {},
		}

	# 读取当前 HTTPRequest 的底层执行结果。
	var request_result: int = int(result[0])
	# 读取响应中的 HTTP 状态码。
	var http_status: int = int(result[1])
	# 读取响应中的原始字节内容。
	var body_bytes: PackedByteArray = result[3]
	# 把响应字节转换为 UTF-8 文本。
	var body_text := body_bytes.get_string_from_utf8()

	if request_result != HTTPRequest.RESULT_SUCCESS:
		return {
			"code": request_result,
			"msg": body_text if not body_text.strip_edges().is_empty() else "http request failed with result %d" % request_result,
			"data": {},
			"http_status": http_status,
		}

	if body_text.strip_edges().is_empty():
		return {
			"code": http_status if http_status != 0 else ERR_PARSE_ERROR,
			"msg": "empty http response body",
			"data": {},
			"http_status": http_status,
		}

	# 使用显式 JSON 解析器避免非 JSON 响应直接刷出引擎错误。
	var json := JSON.new()
	# 解析当前响应文本。
	var parse_error := json.parse(body_text)
	if parse_error == OK and json.data is Dictionary:
		# 规范化服务端返回的响应字典。
		var response: Dictionary = json.data
		if not response.has("data"):
			response["data"] = {}
		response["http_status"] = http_status
		return response

	return {
		"code": http_status,
		"msg": body_text,
		"data": {},
	}
