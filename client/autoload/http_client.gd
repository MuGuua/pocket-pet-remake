extends Node

const DEFAULT_BASE_URL: String = "http://127.0.0.1:8080"

var _base_url: String = DEFAULT_BASE_URL
var _request: HTTPRequest

func _ready() -> void:
    _request = HTTPRequest.new()
    add_child(_request)

func set_base_url(base_url: String) -> void:
    _base_url = base_url.trim_suffix("/")

func login(account: String, password: String) -> Dictionary:
    return await _request_json(
        "/api/v1/auth/login",
        HTTPClient.METHOD_POST,
        {
            "account": account,
            "password": password,
        }
    )

func _request_json(path: String, method: int, payload: Dictionary = {}) -> Dictionary:
    if _request == null:
        return {
            "code": ERR_UNCONFIGURED,
            "msg": "http client is not ready",
            "data": {},
        }

    var headers := PackedStringArray(["Content-Type: application/json"])
    var body := JSON.stringify(payload)
    var err := _request.request(_base_url + path, headers, method, body)
    if err != OK:
        return {
            "code": err,
            "msg": error_string(err),
            "data": {},
        }

    var result: Array = await _request.request_completed
    if result.size() < 4:
        return {
            "code": ERR_PARSE_ERROR,
            "msg": "invalid http response tuple",
            "data": {},
        }

    var http_status: int = int(result[1])
    var body_bytes: PackedByteArray = result[3]
    var body_text := body_bytes.get_string_from_utf8()
    var parsed: Variant = JSON.parse_string(body_text)
    if parsed is Dictionary:
        var response: Dictionary = parsed
        if not response.has("data"):
            response["data"] = {}
        response["http_status"] = http_status
        return response

    return {
        "code": http_status,
        "msg": body_text,
        "data": {},
    }
