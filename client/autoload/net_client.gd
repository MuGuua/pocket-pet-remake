extends Node

# 当网络连接状态变化时向外广播新的状态值。
signal connection_state_changed(state: String)
# WebSocket 连接成功建立后向外广播。
signal websocket_opened
# WebSocket 连接关闭后向外广播关闭码与原因。
signal websocket_closed(code: int, reason: String)
# 收到原始二进制包后向外广播底层字节数据。
signal raw_packet_received(packet: PackedByteArray)
# 收到开发态协议消息后向外广播消息号、序列号、错误码与载荷。
signal dev_message_received(cmd: int, seq: int, code: int, payload: Dictionary)

# 默认连接服务端时使用的 WebSocket 地址。
const DEFAULT_WS_URL: String = "ws://127.0.0.1:8080/ws"
# 当前二进制协议固定包头的总字节长度。
const HEADER_SIZE: int = 26
# CRC32 校验计算使用的多项式常量。
const CRC32_POLYNOMIAL: int = 0xEDB88320

# 是否切换到纯文本 JSON 调试传输模式。
var dev_json_transport: bool = false

# 当前持有的 WebSocket 连接对象。
var _socket: WebSocketPeer = WebSocketPeer.new()
# 当前网络层记录的连接状态文本。
var _state: String = "idle"
# 发送命令时使用的自增序列号。
var _next_seq: int = 1
# 标记当前连接是否已完成业务鉴权。
var _authenticated: bool = false
# 当前会话配置的心跳间隔秒数。
var _heartbeat_interval_sec: int = 0
# 上一次发送心跳时的毫秒时间戳。
var _last_heartbeat_sent_ms: int = 0

# 让单例始终参与帧循环，以便持续轮询底层连接。
func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS
    set_process(true)

# 返回当前网络层记录的连接状态文本。
func get_connection_state() -> String:
    return _state

# 建立到服务端的 WebSocket 连接，并重置旧会话状态。
func connect_to_server(url: String = DEFAULT_WS_URL) -> int:
    disconnect_from_server()
    _socket = WebSocketPeer.new()
    _authenticated = false
    _heartbeat_interval_sec = 0
    _last_heartbeat_sent_ms = 0
    var connect_url: String = _resolve_connect_url(url)
    var err: int = _socket.connect_to_url(connect_url)
    if err != OK:
        _set_state("error")
        return err

    _set_state("connecting")
    return OK

## Web 导出包使用当前页面主机拼出 WebSocket 地址，方便本机和局域网手机调试。
func _resolve_connect_url(url: String) -> String:
    if not OS.has_feature("web") or url != DEFAULT_WS_URL:
        return url

    var protocol: String = str(JavaScriptBridge.eval("window.location.protocol", true))
    var hostname: String = str(JavaScriptBridge.eval("window.location.hostname", true))
    if hostname.strip_edges().is_empty():
        return url

    var ws_scheme: String = "wss" if protocol == "https:" else "ws"
    return "%s://%s:8080/ws" % [ws_scheme, hostname]

# 主动关闭当前连接，并清空鉴权与心跳相关状态。
func disconnect_from_server(code: int = 1000, reason: String = "") -> void:
    if _socket != null and _socket.get_ready_state() != WebSocketPeer.STATE_CLOSED:
        _socket.close(code, reason)
    _authenticated = false
    _heartbeat_interval_sec = 0
    _last_heartbeat_sent_ms = 0
    _set_state("closed")

# 更新当前连接的鉴权状态，并在掉鉴权时清空心跳状态。
func set_authenticated(authenticated: bool) -> void:
    _authenticated = authenticated
    if not authenticated:
        _heartbeat_interval_sec = 0
        _last_heartbeat_sent_ms = 0

# 配置服务端要求的心跳间隔，并重置发送基准时间。
func configure_heartbeat(interval_sec: int) -> void:
    _heartbeat_interval_sec = max(interval_sec, 0)
    _last_heartbeat_sent_ms = _now_ms()

# 发送一条业务命令，按当前模式选择文本或二进制封包，并返回本次请求序列号。
func send_command(cmd: int, payload: Dictionary) -> int:
    if _socket.get_ready_state() != WebSocketPeer.STATE_OPEN:
        push_warning("WebSocket is not connected.")
        return 0

    # 先预留一个稳定序列号，后续等待回包和加载态时都依赖它做匹配。
    var seq := _take_next_seq()

    if dev_json_transport:
        # 组装开发态文本协议使用的 JSON 信封。
        var envelope := JSON.stringify({
            "cmd": cmd,
            "seq": seq,
            "payload": payload,
        })
        _socket.send_text(envelope)
        return seq

    # 生成正式链路使用的二进制数据包。
    var encoded := _encode_json_packet(cmd, seq, 0, payload)
    if encoded.is_empty():
        push_warning("Failed to encode packet for cmd %d." % cmd)
        return 0
    _socket.send(encoded)
    return seq

# 每帧轮询底层连接状态，并处理收包和心跳逻辑。
func _process(_delta: float) -> void:
    if _socket == null:
        return

    _socket.poll()
    match _socket.get_ready_state():
        WebSocketPeer.STATE_CONNECTING:
            pass
        WebSocketPeer.STATE_OPEN:
            if _state != "open":
                _set_state("open")
                websocket_opened.emit()
            _send_heartbeat_if_needed()
            _drain_packets()
        WebSocketPeer.STATE_CLOSING:
            _set_state("closing")
        WebSocketPeer.STATE_CLOSED:
            if _state != "closed":
                websocket_closed.emit(_socket.get_close_code(), _socket.get_close_reason())
                _set_state("closed")

# 连续取出当前帧缓存的所有网络包并分发到对应处理分支。
func _drain_packets() -> void:
    while _socket.get_available_packet_count() > 0:
        # 读取当前待处理的原始网络包。
        var packet := _socket.get_packet()
        if _socket.was_string_packet():
            _handle_text_packet(packet.get_string_from_utf8())
        else:
            raw_packet_received.emit(packet)
            _handle_binary_packet(packet)

# 解析开发态文本协议，并把合法载荷转为统一事件。
func _handle_text_packet(packet_text: String) -> void:
    # 尝试把文本内容解析为 JSON 结构。
    var parsed: Variant = JSON.parse_string(packet_text)
    if parsed is Dictionary and parsed.has("cmd"):
        # 提取消息体载荷字段。
        var payload_variant: Variant = parsed.get("payload", {})
        # 规范化消息体载荷为字典结构。
        var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
        # 读取当前消息对应的协议号。
        var cmd: int = int(parsed.get("cmd", 0))
        var seq := int(parsed.get("seq", 0))
        dev_message_received.emit(cmd, seq, 0, payload)

# 解析正式链路二进制协议，并把合法载荷转为统一事件。
func _handle_binary_packet(packet: PackedByteArray) -> void:
    # 把原始二进制包解码为统一结构。
    var decoded := _decode_packet(packet)
    if decoded.is_empty():
        return

    # 提取解码结果中的消息体载荷字段。
    var payload_variant: Variant = decoded.get("payload", {})
    # 规范化消息体载荷为字典结构。
    var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
    dev_message_received.emit(
        int(decoded.get("cmd", 0)),
        int(decoded.get("seq", 0)),
        int(decoded.get("code", 0)),
        payload
    )

# 当连接已鉴权且超过间隔时主动发送一次心跳包。
func _send_heartbeat_if_needed() -> void:
    if not _authenticated or _heartbeat_interval_sec <= 0:
        return

    # 读取当前系统毫秒时间戳。
    var now_ms := _now_ms()
    if now_ms - _last_heartbeat_sent_ms < _heartbeat_interval_sec * 1000:
        return

    _last_heartbeat_sent_ms = now_ms
    send_command(CommandIds.HEARTBEAT_REQ, {"client_time_ms": now_ms})

# 把消息号、序列号和 JSON 载荷编码为正式链路二进制数据包。
func _encode_json_packet(cmd: int, seq: int, code: int, payload: Dictionary) -> PackedByteArray:
    # 把业务载荷序列化为 JSON 文本。
    var body_text := JSON.stringify(payload)
    # 把 JSON 文本转换为 UTF-8 字节数组。
    var body := body_text.to_utf8_buffer()
    # 记录当前封包使用的时间戳。
    var timestamp_ms := _now_ms()
    # 按协议约定计算当前数据包的校验值。
    var checksum := _crc32(_build_checksum_bytes(cmd, seq, timestamp_ms, body))

    # 创建按大端序写入二进制数据的缓冲区。
    var writer := StreamPeerBuffer.new()
    writer.big_endian = true
    writer.put_u32(HEADER_SIZE + body.size())
    writer.put_u16(cmd)
    writer.put_u32(seq)
    writer.put_u64(timestamp_ms)
    writer.put_u32(code)
    writer.put_u32(checksum)
    if not body.is_empty():
        writer.put_data(body)
    return writer.data_array

# 把正式链路二进制数据包解码为统一字典结构。
func _decode_packet(packet: PackedByteArray) -> Dictionary:
    if packet.size() < HEADER_SIZE:
        push_warning("Received packet shorter than header size.")
        return {}

    # 创建按大端序读取二进制数据的缓冲区。
    var reader := StreamPeerBuffer.new()
    reader.big_endian = true
    reader.data_array = packet

    # 依次读取协议头中的总长度字段。
    var packet_length := int(reader.get_u32())
    # 读取当前包的消息号。
    var cmd := int(reader.get_u16())
    # 读取当前包的序列号。
    var seq := int(reader.get_u32())
    # 读取当前包生成时的时间戳。
    var timestamp_ms := int(reader.get_u64())
    # 读取业务错误码字段。
    var code := int(reader.get_u32())
    # 读取包头里的 CRC32 校验值。
    var checksum := int(reader.get_u32())
    if packet_length != packet.size():
        push_warning("Received packet with mismatched length.")
        return {}

    # 取出包头之后的消息体字节数组。
    var body := packet.slice(HEADER_SIZE, packet.size())
    # 重新计算消息体对应的期望校验值。
    var expected_checksum := _crc32(_build_checksum_bytes(cmd, seq, timestamp_ms, body))
    if checksum != expected_checksum:
        push_warning("Received packet with invalid checksum.")
        return {}

    # 保存解析后的 JSON 业务载荷。
    var payload: Dictionary = {}
    if not body.is_empty():
        # 尝试把消息体字节解码为 JSON 结构。
        var parsed: Variant = JSON.parse_string(body.get_string_from_utf8())
        if parsed is Dictionary:
            payload = parsed

    return {
        "cmd": cmd,
        "seq": seq,
        "code": code,
        "payload": payload,
    }

# 构造参与 CRC32 计算的字节序列。
func _build_checksum_bytes(cmd: int, seq: int, timestamp_ms: int, body: PackedByteArray) -> PackedByteArray:
    # 创建按大端序写入校验字段的缓冲区。
    var writer := StreamPeerBuffer.new()
    writer.big_endian = true
    writer.put_u16(cmd)
    writer.put_u32(seq)
    writer.put_u64(timestamp_ms)
    if not body.is_empty():
        writer.put_data(body)
    return writer.data_array

# 对给定字节序列执行 CRC32 校验计算。
func _crc32(bytes: PackedByteArray) -> int:
    # 以全 1 作为 CRC32 的初始值。
    var crc := 0xFFFFFFFF
    for value in bytes:
        crc ^= int(value)
        for _bit in range(8):
            # 按最低位决定是否与多项式进行异或运算。
            if (crc & 1) == 1:
                crc = (crc >> 1) ^ CRC32_POLYNOMIAL
            else:
                crc >>= 1
            crc &= 0xFFFFFFFF
    return (~crc) & 0xFFFFFFFF

# 返回下一个可用的协议序列号，并在上限后回绕。
func _take_next_seq() -> int:
    # 暂存当前可返回的序列号。
    var seq := _next_seq
    _next_seq += 1
    if _next_seq > 0x7FFFFFFF:
        _next_seq = 1
    return seq

# 返回当前系统时间的毫秒级时间戳。
func _now_ms() -> int:
    return int(Time.get_unix_time_from_system() * 1000.0)

# 更新连接状态文本，并在状态变化时向外广播。
func _set_state(next_state: String) -> void:
    if _state == next_state:
        return

    _state = next_state
    connection_state_changed.emit(_state)
