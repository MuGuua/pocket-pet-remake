extends SceneTree

## 固定服务端字节夹具，覆盖空对象、UTF-8、负数、布尔值和非零业务码；仅用于协议回归。
const SERVER_PACKET_FIXTURES: Array[Dictionary] = [
    {"encoded": "0000001c07dd0000000700000000075bcd15000000002a1af4b87b7d", "cmd": 2013, "seq": 7, "code": 0, "payload": {}},
    {"encoded": "0000004607de0000000800000000075bcd15000000011bf5e7677b226d6f76696e67223a747275652c2274657874223a22e7b2bee781b5e88dafe6b0b4222c2278223a2d317d", "cmd": 2014, "seq": 8, "code": 1, "payload": {"moving": true, "text": "精灵药水", "x": -1}},
    {"encoded": "0000005e07df0000000900000000075bcd15000000021e31ed967b226d6f76696e67223a66616c73652c2274657874223a22e995bfe69687e69cace6b58be8af955c6e5b636f6c6f723d677265656e5d2b3330305b2f636f6c6f725d227d", "cmd": 2015, "seq": 9, "code": 2, "payload": {"moving": false, "text": "长文本测试\n[color=green]+300[/color]"}},
]

## 等待自动加载初始化后运行；不发起任何网络连接或数据库请求。
func _initialize() -> void:
    call_deferred("_run")

## 逐位算法保留为测试参照；bytes 为待校验原始字节，返回无符号 CRC32/IEEE 值。
func _legacy_crc(bytes: PackedByteArray) -> int:
    ## 校验寄存器按旧协议以全 1 初始化。
    var crc: int = 0xFFFFFFFF
    for value: int in bytes:
        crc ^= value
        for _bit: int in range(8):
            crc = ((crc >> 1) ^ 0xEDB88320) if (crc & 1) == 1 else (crc >> 1)
            crc &= 0xFFFFFFFF
    return (~crc) & 0xFFFFFFFF

## 运行独立校验参照、固定服务端包解码与客户端封包回读；失败以非零退出码结束。
func _run() -> void:
    ## 直接实例化正式网络脚本，隔离其帧循环和连接生命周期。
    var codec: Node = load("res://autoload/net_client.gd").new()
    if int(codec.call("_crc32", "123456789".to_utf8_buffer())) != 0xCBF43926:
        _fail(codec, "CRC32/IEEE standard vector mismatch")
        return
    for size: int in [0, 1, 7, 255, 256, 4096, 65536]:
        ## 确定性样本包含所有高位字节，避免只测试 ASCII。
        var sample: PackedByteArray = PackedByteArray()
        sample.resize(size)
        for index: int in range(size):
            sample[index] = (index * 37 + 11) & 0xFF
        if int(codec.call("_crc32", sample)) != _legacy_crc(sample):
            _fail(codec, "CRC differs from legacy at size %d" % size)
            return
    for fixture: Dictionary in SERVER_PACKET_FIXTURES:
        ## 固定字节来自服务端，避免只验证客户端自身往返而遗漏两端不一致。
        var decoded: Dictionary = codec.call("_decode_packet", String(fixture["encoded"]).hex_decode())
        if not _matches_fixture(decoded, fixture):
            _fail(codec, "Server packet is incompatible with client decoder")
            return
        ## 新封包仍使用正式入口，以检查发送与接收的共用校验表。
        var encoded: PackedByteArray = codec.call("_encode_json_packet", int(fixture["cmd"]), int(fixture["seq"]), int(fixture["code"]), fixture["payload"])
        decoded = codec.call("_decode_packet", encoded)
        if not _matches_fixture(decoded, fixture):
            _fail(codec, "Client packet round trip mismatch")
            return
    codec.free()
    ## 独立应用实例只调用诊断函数，不进入场景树或启动网络/业务逻辑。
    var application: Node = load("res://autoload/app.gd").new()
    for sequence: int in range(1, 101):
        application.call("_begin_request_timing", 5001, sequence, Time.get_ticks_usec() - 1000)
        application.call("_finish_request_timing", 5002, sequence, 0, Time.get_ticks_usec() - 500)
    ## 本地诊断窗口与待上报队列分别有界，取走上报不能清空本地统计。
    var upload: Array = application.call("take_request_timing_upload")
    var stats: Dictionary = application.call("get_request_timing_stats", 5001)
    if upload.size() != 32 or int(stats.get("sample_count", 0)) != 64 or not (application.call("take_request_timing_upload") as Array).is_empty():
        application.free()
        push_error("Request timing upload/window bound mismatch")
        quit(1)
        return
    ## 超过诊断期限的请求单列一次，不影响业务等待或污染正常耗时分位。
    application.call("_begin_request_timing", 5001, 200, Time.get_ticks_usec() - 31000000)
    var expired: Array = application.call("take_request_timing_upload")
    if expired.size() != 1 or not bool(expired[0].get("timed_out", false)):
        application.free()
        push_error("Request timing timeout was lost or mixed into completed latencies")
        quit(1)
        return
    application.free()
    print("PASS request timing: 32 upload samples, 64 local samples, no duplicate upload")
    print("PASS network codec: IEEE vector, 7 binary sizes, 3 fixed server packets and client round trips")
    quit(0)

## decoded 为正式解码结果，fixture 为固定预期；返回头部字段与业务载荷是否一致。
func _matches_fixture(decoded: Dictionary, fixture: Dictionary) -> bool:
    ## JSON 数字解析为浮点，预期值按同一格式规范化后比较业务内容。
    var expected_payload: Dictionary = JSON.parse_string(JSON.stringify(fixture["payload"]))
    return not decoded.is_empty() and int(decoded["cmd"]) == int(fixture["cmd"]) and int(decoded["seq"]) == int(fixture["seq"]) and int(decoded["code"]) == int(fixture["code"]) and decoded["payload"] == expected_payload

## 释放 codec 并报告 message，以错误退出码让命令行验证能够检测失败。
func _fail(codec: Node, message: String) -> void:
    codec.free()
    push_error(message)
    quit(1)
