extends Node

## 静音保活播放器使用的采样率；沿用常见 44.1kHz，减少浏览器/设备兼容性问题。
const SAMPLE_RATE: int = 44100
## 每次往播放缓冲里补多少帧静音数据，避免一次性写入过多造成无意义堆积。
const FEED_CHUNK_FRAMES: int = 2048
## 缓冲低于该阈值时继续补静音，保证播放器不断流。
const MIN_BUFFER_FRAMES: int = 4096
## 用户要求“声音大小为 0”；这里使用接近静音的最小 dB，避免真实发声。
const SILENT_VOLUME_DB: float = -80.0

# 全局常驻的音频播放器；作为自动加载节点的子节点存在于所有场景之上。
var _player: AudioStreamPlayer = null
# 静音流生成器资源；不依赖外部音频文件。
var _generator: AudioStreamGenerator = null
# 当前静音流的底层回放对象，用于持续喂入零振幅采样。
var _playback: AudioStreamGeneratorPlayback = null

## 初始化全局静音播放器。
## 浏览器环境可能要求用户先产生一次交互后才允许音频真正启动，因此这里会在 ready 和后续输入时都尝试恢复。
func _ready() -> void:
    process_mode = Node.PROCESS_MODE_ALWAYS
    set_process(true)
    set_process_input(true)
    _ensure_player_ready()
    _start_silent_playback()

## 每帧维护静音缓冲，避免播放器因缓冲耗尽而停止。
func _process(_delta: float) -> void:
    _ensure_player_ready()
    _start_silent_playback()
    _feed_silence_if_needed()

## 捕获一次用户输入后再次尝试启动音频，兼容浏览器自动播放限制。
func _input(event: InputEvent) -> void:
    if event == null:
        return
    if event is InputEventMouseButton or event is InputEventScreenTouch or event is InputEventKey:
        _start_silent_playback()

## 创建并配置静音播放器。
func _ensure_player_ready() -> void:
    if _player != null and is_instance_valid(_player):
        return

    _generator = AudioStreamGenerator.new()
    _generator.mix_rate = SAMPLE_RATE
    _generator.buffer_length = 0.5

    _player = AudioStreamPlayer.new()
    _player.name = "BackgroundAudioKeeperPlayer"
    _player.bus = "Master"
    _player.stream = _generator
    _player.volume_db = SILENT_VOLUME_DB
    _player.process_mode = Node.PROCESS_MODE_ALWAYS
    add_child(_player)

## 启动静音播放，并抓取底层回放对象。
func _start_silent_playback() -> void:
    if _player == null or _generator == null:
        return
    if not _player.playing:
        _player.play()

    if _playback == null:
        var playback_variant: Variant = _player.get_stream_playback()
        if playback_variant is AudioStreamGeneratorPlayback:
            _playback = playback_variant as AudioStreamGeneratorPlayback

## 当缓冲偏低时持续补入 0 振幅采样，形成真正“无声但持续播放”的音频流。
func _feed_silence_if_needed() -> void:
    if _playback == null:
        return

    var frames_available: int = _playback.get_frames_available()
    if frames_available >= MIN_BUFFER_FRAMES:
        return

    var frames_to_push: int = max(FEED_CHUNK_FRAMES, MIN_BUFFER_FRAMES - frames_available)
    var index: int = 0
    while index < frames_to_push:
        _playback.push_frame(Vector2.ZERO)
        index += 1
