extends Node

signal connection_state_changed(state: String)
signal websocket_opened
signal websocket_closed(code: int, reason: String)
signal raw_packet_received(packet: PackedByteArray)
signal dev_message_received(cmd: int, payload: Dictionary)

const DEFAULT_WS_URL: String = "ws://127.0.0.1:8080/ws"

var dev_json_transport: bool = false

var _socket: WebSocketPeer = WebSocketPeer.new()
var _state: String = "idle"

func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS
    set_process(true)

func get_connection_state() -> String:
    return _state

func connect_to_server(url: String = DEFAULT_WS_URL) -> int:
    _socket = WebSocketPeer.new()
    var err := _socket.connect_to_url(url)
    if err != OK:
        _set_state("error")
        return err

    _set_state("connecting")
    return OK

func disconnect_from_server(code: int = 1000, reason: String = "") -> void:
    if _socket.get_ready_state() != WebSocketPeer.STATE_CLOSED:
        _socket.close(code, reason)
    _set_state("closed")

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

    push_warning(
        "Binary packet encoding is not implemented yet for cmd %d. Hook protobuf + packet header here." % cmd
    )

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

func _handle_text_packet(packet_text: String) -> void:
    var parsed: Variant = JSON.parse_string(packet_text)
    if parsed is Dictionary and parsed.has("cmd"):
        var payload_variant: Variant = parsed.get("payload", {})
        var payload: Dictionary = payload_variant if payload_variant is Dictionary else {}
        var cmd: int = int(parsed.get("cmd", 0))
        dev_message_received.emit(cmd, payload)

func _set_state(next_state: String) -> void:
    if _state == next_state:
        return

    _state = next_state
    connection_state_changed.emit(_state)
