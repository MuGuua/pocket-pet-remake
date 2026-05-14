extends Node

signal connection_state_changed(state: String)
signal websocket_opened
signal websocket_closed(code: int, reason: String)
signal raw_packet_received(packet: PackedByteArray)
signal dev_message_received(cmd: int, payload: Dictionary)

const DEFAULT_WS_URL: String = "ws://127.0.0.1:8080/ws"
const HEADER_SIZE: int = 26
const CRC32_POLYNOMIAL: int = 0xEDB88320

var dev_json_transport: bool = false

var _socket: WebSocketPeer = WebSocketPeer.new()
var _state: String = "idle"
var _next_seq: int = 1
var _authenticated: bool = false
var _heartbeat_interval_sec: int = 0
var _last_heartbeat_sent_ms: int = 0

func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS
    set_process(true)

func get_connection_state() -> String:
    return _state

func connect_to_server(url: String = DEFAULT_WS_URL) -> int:
    disconnect_from_server()
    _socket = WebSocketPeer.new()
    _authenticated = false
    _heartbeat_interval_sec = 0
    _last_heartbeat_sent_ms = 0
    var err := _socket.connect_to_url(url)
    if err != OK:
        _set_state("error")
        return err

    _set_state("connecting")
    return OK

func disconnect_from_server(code: int = 1000, reason: String = "") -> void:
    if _socket != null and _socket.get_ready_state() != WebSocketPeer.STATE_CLOSED:
        _socket.close(code, reason)
    _authenticated = false
    _heartbeat_interval_sec = 0
    _last_heartbeat_sent_ms = 0
    _set_state("closed")

func set_authenticated(authenticated: bool) -> void:
    _authenticated = authenticated
    if not authenticated:
        _heartbeat_interval_sec = 0
        _last_heartbeat_sent_ms = 0

func configure_heartbeat(interval_sec: int) -> void:
    _heartbeat_interval_sec = max(interval_sec, 0)
    _last_heartbeat_sent_ms = _now_ms()

func send_command(cmd: int, payload: Dictionary) -> void:
    if _socket.get_ready_state() != WebSocketPeer.STATE_OPEN:
        push_warning("WebSocket is not connected.")
        return

    if dev_json_transport:
        var envelope := JSON.stringify({
            "cmd": cmd,
            "payload": payload,
        })
        _socket.send_text(envelope)
        return

    var encoded := _encode_json_packet(cmd, _take_next_seq(), 0, payload)
    if encoded.is_empty():
        push_warning("Failed to encode packet for cmd %d." % cmd)
        return
    _socket.send(encoded)

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

func _drain_packets() -> void:
    while _socket.get_available_packet_count() > 0:
        var packet := _socket.get_packet()
        if _socket.was_string_packet():
            _handle_text_packet(packet.get_string_from_utf8())
        else:
            raw_packet_received.emit(packet)
            _handle_binary_packet(packet)

func _handle_text_packet(packet_text: String) -> void:
    var parsed: Variant = JSON.parse_string(packet_text)
    if parsed is Dictionary and parsed.has("cmd"):
        var payload_variant: Variant = parsed.get("payload", {})
        var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
        var cmd: int = int(parsed.get("cmd", 0))
        dev_message_received.emit(cmd, payload)

func _handle_binary_packet(packet: PackedByteArray) -> void:
    var decoded := _decode_packet(packet)
    if decoded.is_empty():
        return

    var payload_variant: Variant = decoded.get("payload", {})
    var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
    dev_message_received.emit(int(decoded.get("cmd", 0)), payload)

func _send_heartbeat_if_needed() -> void:
    if not _authenticated or _heartbeat_interval_sec <= 0:
        return

    var now_ms := _now_ms()
    if now_ms - _last_heartbeat_sent_ms < _heartbeat_interval_sec * 1000:
        return

    _last_heartbeat_sent_ms = now_ms
    send_command(CommandIds.HEARTBEAT_REQ, {"client_time_ms": now_ms})

func _encode_json_packet(cmd: int, seq: int, code: int, payload: Dictionary) -> PackedByteArray:
    var body_text := JSON.stringify(payload)
    var body := body_text.to_utf8_buffer()
    var timestamp_ms := _now_ms()
    var checksum := _crc32(_build_checksum_bytes(cmd, seq, timestamp_ms, body))

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

func _decode_packet(packet: PackedByteArray) -> Dictionary:
    if packet.size() < HEADER_SIZE:
        push_warning("Received packet shorter than header size.")
        return {}

    var reader := StreamPeerBuffer.new()
    reader.big_endian = true
    reader.data_array = packet

    var packet_length := int(reader.get_u32())
    var cmd := int(reader.get_u16())
    var seq := int(reader.get_u32())
    var timestamp_ms := int(reader.get_u64())
    var code := int(reader.get_u32())
    var checksum := int(reader.get_u32())
    if packet_length != packet.size():
        push_warning("Received packet with mismatched length.")
        return {}

    var body := packet.slice(HEADER_SIZE, packet.size())
    var expected_checksum := _crc32(_build_checksum_bytes(cmd, seq, timestamp_ms, body))
    if checksum != expected_checksum:
        push_warning("Received packet with invalid checksum.")
        return {}

    var payload: Dictionary = {}
    if not body.is_empty():
        var parsed: Variant = JSON.parse_string(body.get_string_from_utf8())
        if parsed is Dictionary:
            payload = parsed

    return {
        "cmd": cmd,
        "seq": seq,
        "code": code,
        "payload": payload,
    }

func _build_checksum_bytes(cmd: int, seq: int, timestamp_ms: int, body: PackedByteArray) -> PackedByteArray:
    var writer := StreamPeerBuffer.new()
    writer.big_endian = true
    writer.put_u16(cmd)
    writer.put_u32(seq)
    writer.put_u64(timestamp_ms)
    if not body.is_empty():
        writer.put_data(body)
    return writer.data_array

func _crc32(bytes: PackedByteArray) -> int:
    var crc := 0xFFFFFFFF
    for value in bytes:
        crc ^= int(value)
        for _bit in range(8):
            if (crc & 1) == 1:
                crc = (crc >> 1) ^ CRC32_POLYNOMIAL
            else:
                crc >>= 1
            crc &= 0xFFFFFFFF
    return (~crc) & 0xFFFFFFFF

func _take_next_seq() -> int:
    var seq := _next_seq
    _next_seq += 1
    if _next_seq > 0x7FFFFFFF:
        _next_seq = 1
    return seq

func _now_ms() -> int:
    return int(Time.get_unix_time_from_system() * 1000.0)

func _set_state(next_state: String) -> void:
    if _state == next_state:
        return

    _state = next_state
    connection_state_changed.emit(_state)
