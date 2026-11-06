extends SceneTree

## 使用正式远端角色的采样算法，所有网络时间线在测试里确定性生成，不修改运行时人物脚本。
const REMOTE_SCRIPT_PATH: String = "res://scripts/feature/world/player.gd"
## 延迟到自动加载就绪后加载，避免人物脚本解析时找不到游戏状态单例。
var _remote_script: Script
## 每个分支正常结束后累计，防止运行时脚本错误被误判为成功。
var _completed_cases: int = 0
## 回归状态只用于命令行退出码，不进入业务场景或持久化数据。
var _failed: bool = false

## 等自动加载完成后执行回归，不开启网络连接。
func _initialize() -> void:
    call_deferred("_run")

## condition 为预期条件，message 为失败定位；失败最终以非零退出码报告。
func _check(condition: bool, message: String) -> void:
    if not condition:
        _failed = true
        push_error(message)

## 验证抖动、丢包、断流封顶、停步重启、时钟异常及移动合并的时间线兼容。
func _run() -> void:
    _remote_script = load(REMOTE_SCRIPT_PATH)
    if _remote_script == null or not _remote_script.can_instantiate():
        push_error("Cannot load production remote player script")
        quit(1)
        return
    _jitter_timeline(false)
    _jitter_timeline(true)
    _stop_and_outage()
    _snapshot_admission()
    _clock_domain_transition()
    _coalesced_timeline()
    _check(_completed_cases == 6, "Movement regression case did not finish")
    if _failed:
        quit(1)
    else:
        print("PASS movement interpolation: deterministic jitter/loss, outage cap, stop/restart, old/duplicate ticks, reset, bounded buffer and coalesced path")
        quit(0)

## drop_samples 开启每五包丢一包；固定 40～100ms 到达抖动，90px/s 直线不应倒退。
func _jitter_timeline(drop_samples: bool) -> void:
    ## 不挂场景树，避免碰撞、贴图与真实时钟影响确定性采样。
    var remote: Node2D = _remote_script.new()
    remote.set("move_speed", 90.0)
    ## 到达队列只描述测试网络，不复用业务状态。
    var arrivals: Array[Dictionary] = []
    var delays: Array[int] = [40, 100, 70, 90]
    for sequence: int in range(25):
        if drop_samples and sequence % 5 == 4:
            continue
        arrivals.append({"arrival": sequence * 100 + delays[sequence % delays.size()], "t": float(sequence * 100), "pos": Vector2(float(sequence) * 9.0, 0.0), "facing": Vector2.RIGHT, "moving": true})
    ## 固定数据中相邻采样到达顺序未逆转；乱序准入另行调用正式入口检查。
    var history: Array[Dictionary] = []
    var next_arrival: int = 0
    var previous_x: float = 0.0
    for now: int in range(0, 2501, 25):
        while next_arrival < arrivals.size() and int(arrivals[next_arrival]["arrival"]) <= now:
            history.append(arrivals[next_arrival])
            next_arrival += 1
        if history.is_empty():
            continue
        remote.set("_remote_snapshots", history)
        ## 150ms 插值缓冲加 40ms 最快到达偏移，与正式发送方时间域保持一致。
        var render_time: float = maxf(float(now - 190), 0.0)
        var result: Dictionary = remote.call("_sample_remote_presentation", render_time)
        var x: float = (result["pos"] as Vector2).x
        _check(x + 0.001 >= previous_x, "Remote presentation moved backwards under jitter/loss")
        if now >= 300:
            _check(absf(x - render_time * 0.09) < 0.05, "Straight presentation diverged within interpolation/extrapolation budget")
        previous_x = x
    remote.free()
    _completed_cases += 1

## 确认停止段不会滑行，断流后最多外推 200ms，恢复后的时间线仍可继续。
func _stop_and_outage() -> void:
    var remote: Node2D = _remote_script.new()
    remote.set("move_speed", 90.0)
    var history: Array[Dictionary] = [
        {"t": 900.0, "pos": Vector2(81, 0), "facing": Vector2.RIGHT, "moving": true},
        {"t": 1000.0, "pos": Vector2(90, 0), "facing": Vector2.RIGHT, "moving": false},
        {"t": 1500.0, "pos": Vector2(90, 0), "facing": Vector2.RIGHT, "moving": true},
        {"t": 1600.0, "pos": Vector2(99, 0), "facing": Vector2.RIGHT, "moving": true},
    ]
    remote.set("_remote_snapshots", history)
    var stopped: Dictionary = remote.call("_sample_remote_presentation", 1250.0)
    _check((stopped["pos"] as Vector2).is_equal_approx(Vector2(90, 0)) and not bool(stopped["walking"]), "Stop segment slid before restart")
    var resumed: Dictionary = remote.call("_sample_remote_presentation", 1550.0)
    _check((resumed["pos"] as Vector2).is_equal_approx(Vector2(94.5, 0)) and bool(resumed["walking"]), "Restart segment failed to interpolate")
    var outage: Dictionary = remote.call("_sample_remote_presentation", 3000.0)
    _check((outage["pos"] as Vector2).is_equal_approx(Vector2(117, 0)), "Outage extrapolated beyond 200ms cap")
    remote.free()
    _completed_cases += 1

## 实际快照写入入口拒绝重复/旧时间戳，并在发送方重启及长时间流动时有界维护缓冲。
func _snapshot_admission() -> void:
    var remote: Node2D = _remote_script.new()
    remote.call("_push_remote_snapshot", Vector2.ZERO, Vector2.RIGHT, true, 100)
    remote.call("_push_remote_snapshot", Vector2.ONE, Vector2.RIGHT, true, 200)
    remote.call("_push_remote_snapshot", Vector2.ONE, Vector2.RIGHT, true, 200)
    remote.call("_push_remote_snapshot", Vector2.ZERO, Vector2.RIGHT, true, 150)
    var history: Array = remote.get("_remote_snapshots")
    _check(history.size() == 2, "Duplicate/old tick entered snapshot buffer")
    remote.call("_push_remote_snapshot", Vector2.ONE, Vector2.RIGHT, true, 10000)
    remote.call("_push_remote_snapshot", Vector2.ZERO, Vector2.RIGHT, false, 10)
    history = remote.get("_remote_snapshots")
    _check(history.size() == 1 and int(history[0]["t"]) == 10, "Clock reset retained previous timeline")
    for tick: int in range(100, 10100, 100):
        remote.call("_push_remote_snapshot", Vector2.ZERO, Vector2.RIGHT, false, tick)
    history = remote.get("_remote_snapshots")
    _check(history.size() == 32, "Snapshot buffer grew beyond fixed capacity")
    remote.free()
    _completed_cases += 1

## 完整与合并快照在指定拐弯、停止、重启轨迹的 61 个采样点应完全一致。
func _coalesced_timeline() -> void:
    var full: Node2D = _remote_script.new()
    var reduced: Node2D = _remote_script.new()
    var positions: Array[Vector2] = [Vector2(0,0),Vector2(10,0),Vector2(20,0),Vector2(20,10),Vector2(20,20),Vector2(20,30),Vector2(20,40),Vector2(20,50),Vector2(20,60),Vector2(20,60),Vector2(20,70),Vector2(20,80),Vector2(20,90)]
    var history: Array[Dictionary] = []
    var compact: Array[Dictionary] = []
    var keep: Array[int] = [1,3,4,6,7,9,10,11,13]
    for index: int in range(positions.size()):
        var sequence: int = index + 1
        var sample: Dictionary = {"t": float(sequence * 100), "pos": positions[index], "facing": Vector2.RIGHT if sequence < 7 else Vector2.DOWN, "moving": sequence != 10}
        history.append(sample)
        if sequence in keep:
            compact.append(sample.duplicate(true))
    full.set("_remote_snapshots", history)
    reduced.set("_remote_snapshots", compact)
    for tick: int in range(0, 1501, 25):
        var expected: Dictionary = full.call("_sample_remote_presentation", float(tick))
        var actual: Dictionary = reduced.call("_sample_remote_presentation", float(tick))
        _check((expected["pos"] as Vector2).is_equal_approx(actual["pos"]) and expected["facing"] == actual["facing"] and expected["walking"] == actual["walking"], "Coalesced path changed presentation at %d" % tick)
    full.free()
    reduced.free()
    _completed_cases += 1


## 无发送方时钟的初始摘要与后续移动可能只差几秒，不能因小于重启阈值而忽略首批移动。
func _clock_domain_transition() -> void:
    var remote: Node2D = _remote_script.new()
    remote.call("_push_remote_snapshot", Vector2.ZERO, Vector2.RIGHT, false, 0)
    ## 构造已按接收时钟记录的摘要，发送方时钟落后 1500ms，仍应切换基线。
    var received_history: Array[Dictionary] = [{"t": 2000.0, "pos": Vector2.ZERO, "facing": Vector2.RIGHT, "moving": false}]
    remote.set("_remote_snapshots", received_history)
    remote.call("_push_remote_snapshot", Vector2(9, 0), Vector2.RIGHT, true, 500)
    var history: Array = remote.get("_remote_snapshots")
    _check(history.size() == 1 and int(history[0]["t"]) == 500, "First sender-clock movement was rejected as an old arrival-clock snapshot")
    remote.free()
    _completed_cases += 1
