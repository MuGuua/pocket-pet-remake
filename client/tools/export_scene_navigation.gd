class_name ExportSceneNavigation
extends SceneTree

## 世界场景注册表用于读取正式 scene_id、地图资源路径和服务端格到像素的换算倍率。
const WorldSceneRegistryScript: Script = preload("res://scripts/feature/world/world_scene_registry.gd")
## 玩家场景是静态通行判定使用的唯一人物碰撞体来源，避免工具复制碰撞尺寸后与运行时漂移。
const PLAYER_SCENE: PackedScene = preload("res://scenes/world/player.tscn")
## 服务端静态通行位图固定以一个场景格为一个采样单元。
const CELL_SIZE_MILLI: int = 1000
## 地图主层选择顺序必须与 world_controller.gd 保持一致。
const NAVIGATION_LAYER_NAMES: PackedStringArray = ["Collision", "Bottom", "Map", "TileMapLayer", "地图"]
## 单次物理查询上限用于完整检查采样点附近的静态碰撞体，同时不会把动态角色写入位图。
const MAX_PHYSICS_RESULTS: int = 128
## 默认从 P0-05 数据库迁移读取场景矩形边界，避免在客户端维护第二份正式边界数据。
const DEFAULT_BOUNDS_SQL_PATH: String = "res://../backend/server/migrations/119_world_scene_boundaries.sql"

## 命令行解析或导出失败时记录退出码，延迟执行完成后统一退出 Godot。
var _exit_code: int = 0


## Godot 以 --script 启动时进入此方法；延迟一帧后执行，确保项目自动加载和 SceneTree 已初始化。
func _initialize() -> void:
    call_deferred("_run")


## 解析参数并执行单场景或全场景导出；任何失败都会输出明确错误并返回非零退出码。
func _run() -> void:
    var arguments: Dictionary = _parse_arguments(OS.get_cmdline_user_args())
    if bool(arguments.get("help", false)):
        _print_usage()
        quit(0)
        return

    var validation_error: String = _validate_arguments(arguments)
    if not validation_error.is_empty():
        push_error(validation_error)
        _print_usage()
        quit(2)
        return

    var bounds_path: String = str(arguments.get("bounds_sql", DEFAULT_BOUNDS_SQL_PATH))
    var scene_bounds: Dictionary = _load_scene_bounds(bounds_path)
    if scene_bounds.is_empty():
        quit(3)
        return

    var scene_ids: Array[int] = _resolve_scene_ids(arguments)
    var exported_entries: Array[Dictionary] = []
    for scene_id: int in scene_ids:
        var scene_bounds_variant: Variant = scene_bounds.get(scene_id, null)
        if not scene_bounds_variant is Dictionary:
            push_error("场景 %d 缺少 P0-05 矩形边界，已停止导出。" % scene_id)
            _exit_code = 4
            break
        var exported_entry: Dictionary = await _export_scene_navigation(
            scene_id,
            scene_bounds_variant as Dictionary
        )
        if exported_entry.is_empty():
            _exit_code = 5
            break
        exported_entries.append(exported_entry)

    if _exit_code == 0:
        _exit_code = _write_outputs(arguments, exported_entries)
    quit(_exit_code)


## 将 --key=value 和布尔开关解析为字典；未知参数保留给后续校验报告。
func _parse_arguments(raw_arguments: PackedStringArray) -> Dictionary:
    var parsed: Dictionary = {}
    for raw_argument: String in raw_arguments:
        if raw_argument == "--all":
            parsed["all"] = true
            continue
        if raw_argument == "--help" or raw_argument == "-h":
            parsed["help"] = true
            continue
        if not raw_argument.begins_with("--") or not raw_argument.contains("="):
            parsed["unknown"] = raw_argument
            continue
        var separator_index: int = raw_argument.find("=")
        var key: String = raw_argument.substr(2, separator_index - 2).replace("-", "_")
        var value: String = raw_argument.substr(separator_index + 1)
        parsed[key] = value
    return parsed


## 校验命令组合，避免覆盖错误文件或在单场景与全量模式间产生歧义。
func _validate_arguments(arguments: Dictionary) -> String:
    if arguments.has("unknown"):
        return "无法识别参数：%s" % str(arguments.get("unknown", ""))

    var allowed_keys: PackedStringArray = [
        "all",
        "help",
        "scene_id",
        "output",
        "output_dir",
        "sql_output",
        "bounds_sql",
    ]
    for argument_key_variant: Variant in arguments.keys():
        var argument_key: String = str(argument_key_variant)
        if not allowed_keys.has(argument_key):
            return "无法识别参数：--%s" % argument_key.replace("_", "-")

    var export_all: bool = bool(arguments.get("all", false))
    var scene_id_text: String = str(arguments.get("scene_id", "")).strip_edges()
    if export_all == not scene_id_text.is_empty():
        return "必须且只能指定 --all 或 --scene-id={scene_id} 其中一种。"
    if not scene_id_text.is_empty() and (not scene_id_text.is_valid_int() or int(scene_id_text) <= 0):
        return "--scene-id 必须是大于 0 的整数。"

    var output_path: String = str(arguments.get("output", "")).strip_edges()
    var output_dir: String = str(arguments.get("output_dir", "")).strip_edges()
    var sql_output_path: String = str(arguments.get("sql_output", "")).strip_edges()
    if export_all:
        if output_dir.is_empty() and sql_output_path.is_empty():
            return "全量导出至少需要 --output-dir 或 --sql-output。"
        if not output_path.is_empty():
            return "--output 仅支持单场景导出；全量导出请使用 --output-dir。"
    elif output_path.is_empty():
        return "单场景导出必须指定 --output。"
    if not output_dir.is_empty() and not export_all:
        return "--output-dir 仅支持 --all。"
    return ""


## 根据命令行模式返回有序 scene_id；全量模式直接使用注册表键，避免硬编码场景清单。
func _resolve_scene_ids(arguments: Dictionary) -> Array[int]:
    var scene_ids: Array[int] = []
    if bool(arguments.get("all", false)):
        for scene_id_variant: Variant in WorldSceneRegistryScript.SCENE_CONFIGS.keys():
            scene_ids.append(int(scene_id_variant))
        scene_ids.sort()
        return scene_ids
    scene_ids.append(int(str(arguments.get("scene_id", "0"))))
    return scene_ids


## 从 P0-05 迁移 SQL 的 VALUES 数据读取正式场景边界；工具只消费事实来源，不维护重复配置。
func _load_scene_bounds(bounds_path: String) -> Dictionary:
    var absolute_path: String = _absolute_path(bounds_path)
    var sql_file: FileAccess = FileAccess.open(absolute_path, FileAccess.READ)
    if sql_file == null:
        push_error("无法读取场景边界迁移：%s，错误码=%d" % [absolute_path, FileAccess.get_open_error()])
        return {}
    var sql_text: String = sql_file.get_as_text()
    sql_file.close()

    var row_pattern: RegEx = RegEx.new()
    var compile_error: Error = row_pattern.compile(
        "\\(\\s*(\\d+)\\s*,\\s*(-?\\d+)\\s*,\\s*(-?\\d+)\\s*,\\s*(-?\\d+)\\s*,\\s*(-?\\d+)\\s*\\)"
    )
    if compile_error != OK:
        push_error("场景边界 SQL 解析正则编译失败，错误码=%d" % compile_error)
        return {}

    var bounds: Dictionary = {}
    var matches: Array[RegExMatch] = row_pattern.search_all(sql_text)
    for row_match: RegExMatch in matches:
        var scene_id: int = int(row_match.get_string(1))
        var min_x_milli: int = int(row_match.get_string(2))
        var min_y_milli: int = int(row_match.get_string(3))
        var max_x_milli: int = int(row_match.get_string(4))
        var max_y_milli: int = int(row_match.get_string(5))
        if max_x_milli <= min_x_milli or max_y_milli <= min_y_milli:
            continue
        bounds[scene_id] = {
            "min_x_milli": min_x_milli,
            "min_y_milli": min_y_milli,
            "max_x_milli": max_x_milli,
            "max_y_milli": max_y_milli,
        }

    if bounds.is_empty():
        push_error("场景边界迁移中没有解析到有效 VALUES 数据：%s" % absolute_path)
    return bounds


## 实例化正式地图与玩家碰撞体，并按服务端场景格逐点生成行优先、高位优先的通行位图。
func _export_scene_navigation(scene_id: int, bounds: Dictionary) -> Dictionary:
    var scene_config: Dictionary = WorldSceneRegistryScript.get_scene_config(scene_id)
    var scene_path: String = str(scene_config.get("scene_path", "")).strip_edges()
    var grid_to_pixels: float = float(scene_config.get("grid_to_pixels", 0.0))
    if scene_path.is_empty() or grid_to_pixels <= 0.0:
        push_error("场景 %d 注册信息不完整，缺少 scene_path 或 grid_to_pixels。" % scene_id)
        return {}

    var level_scene: PackedScene = load(scene_path) as PackedScene
    if level_scene == null:
        push_error("场景 %d 无法加载地图资源：%s" % [scene_id, scene_path])
        return {}
    var level_instance_variant: Node = level_scene.instantiate()
    if not level_instance_variant is Node2D:
        push_error("场景 %d 地图根节点不是 Node2D：%s" % [scene_id, scene_path])
        level_instance_variant.free()
        return {}
    var level_instance: Node2D = level_instance_variant as Node2D

    var player_instance_variant: Node = PLAYER_SCENE.instantiate()
    if not player_instance_variant is CharacterBody2D:
        push_error("正式玩家场景根节点不是 CharacterBody2D，无法复用运行时碰撞体。")
        player_instance_variant.free()
        level_instance.free()
        return {}
    var player_instance: CharacterBody2D = player_instance_variant as CharacterBody2D

    get_root().add_child(level_instance)
    get_root().add_child(player_instance)
    await process_frame
    _disable_dynamic_obstacles(level_instance)

    var level_rect: Rect2 = _resolve_level_world_rect(level_instance)
    if not level_rect.has_area():
        push_error("场景 %d 缺少可用地图主层或主层没有有效区域：%s" % [scene_id, scene_path])
        _free_export_nodes(level_instance, player_instance)
        return {}
    if not level_rect.position.is_equal_approx(Vector2.ZERO):
        level_instance.global_position -= level_rect.position
    var scene_origin_pixels: Vector2 = level_instance.to_local(Vector2.ZERO)

    await physics_frame
    await physics_frame

    var collision_shape: CollisionShape2D = player_instance.get_node_or_null("CollisionShape2D") as CollisionShape2D
    if collision_shape == null or collision_shape.shape == null:
        push_error("正式玩家场景缺少主 CollisionShape2D，无法导出静态通行数据。")
        _free_export_nodes(level_instance, player_instance)
        return {}

    var origin_x_milli: int = int(bounds.get("min_x_milli", 0))
    var origin_y_milli: int = int(bounds.get("min_y_milli", 0))
    var max_x_milli: int = int(bounds.get("max_x_milli", 0))
    var max_y_milli: int = int(bounds.get("max_y_milli", 0))
    if not _is_cell_aligned(origin_x_milli) \
            or not _is_cell_aligned(origin_y_milli) \
            or not _is_cell_aligned(max_x_milli) \
            or not _is_cell_aligned(max_y_milli):
        push_error("场景 %d 的 P0-05 边界未按 %d milli 对齐，无法无损生成位图。" % [scene_id, CELL_SIZE_MILLI])
        _free_export_nodes(level_instance, player_instance)
        return {}

    var grid_width: int = (max_x_milli - origin_x_milli) / CELL_SIZE_MILLI + 1
    var grid_height: int = (max_y_milli - origin_y_milli) / CELL_SIZE_MILLI + 1
    var data_length: int = (grid_width * grid_height + 7) / 8
    var navigation_data: PackedByteArray = PackedByteArray()
    navigation_data.resize(data_length)
    navigation_data.fill(0)

    var walkable_cell_count: int = 0
    var space_state: PhysicsDirectSpaceState2D = player_instance.get_world_2d().direct_space_state
    for grid_y: int in range(grid_height):
        for grid_x: int in range(grid_width):
            var scene_x_milli: int = origin_x_milli + grid_x * CELL_SIZE_MILLI
            var scene_y_milli: int = origin_y_milli + grid_y * CELL_SIZE_MILLI
            var scene_position: Vector2 = Vector2(
                float(scene_x_milli) / float(CELL_SIZE_MILLI),
                float(scene_y_milli) / float(CELL_SIZE_MILLI)
            )
            var player_world_position: Vector2 = level_instance.to_global(
                scene_origin_pixels + scene_position * grid_to_pixels
            ).round()
            if not _can_player_stand_at(
                space_state,
                player_instance,
                collision_shape,
                player_world_position
            ):
                continue
            var bit_index: int = grid_y * grid_width + grid_x
            var byte_index: int = bit_index / 8
            var bit_offset: int = 7 - bit_index % 8
            navigation_data[byte_index] = navigation_data[byte_index] | (1 << bit_offset)
            walkable_cell_count += 1


    var data_hash: String = _sha256_hex(navigation_data)
    var exported_entry: Dictionary = {
        "scene_id": scene_id,
        "origin_x_milli": origin_x_milli,
        "origin_y_milli": origin_y_milli,
        "grid_width": grid_width,
        "grid_height": grid_height,
        "cell_size_milli": CELL_SIZE_MILLI,
        "navigation_data": navigation_data.hex_encode(),
        "data_hash": data_hash,
        "walkable_cell_count": walkable_cell_count,
        "source_scene_path": scene_path,
    }
    print(
        "[SceneNavigationExport] scene=%d size=%dx%d walkable=%d hash=%s path=%s" % [
            scene_id,
            grid_width,
            grid_height,
            walkable_cell_count,
            data_hash,
            scene_path,
        ]
    )
    _free_export_nodes(level_instance, player_instance)
    return exported_entry


## 关闭地图中动态物理体的碰撞层，确保 NPC、玩家和临时运行时物体不会进入静态位图。
func _disable_dynamic_obstacles(root_node: Node) -> void:
    var pending_nodes: Array[Node] = [root_node]
    while not pending_nodes.is_empty():
        var current_node: Node = pending_nodes.pop_back()
        for child_node: Node in current_node.get_children():
            pending_nodes.append(child_node)
        if current_node is PhysicsBody2D and not current_node is StaticBody2D:
            var physics_body: PhysicsBody2D = current_node as PhysicsBody2D
            physics_body.collision_layer = 0
            physics_body.collision_mask = 0


## 判断目标世界坐标是否没有命中静态地图碰撞；Area2D 与所有动态 PhysicsBody2D 均被排除。
func _can_player_stand_at(
    space_state: PhysicsDirectSpaceState2D,
    player_instance: CharacterBody2D,
    collision_shape: CollisionShape2D,
    player_world_position: Vector2
) -> bool:
    var query: PhysicsShapeQueryParameters2D = PhysicsShapeQueryParameters2D.new()
    query.shape = collision_shape.shape
    query.collide_with_bodies = true
    query.collide_with_areas = false
    query.collision_mask = player_instance.collision_mask
    query.exclude = [player_instance.get_rid()]
    query.transform = Transform2D(0.0, player_world_position) * collision_shape.transform

    var collisions: Array[Dictionary] = space_state.intersect_shape(query, MAX_PHYSICS_RESULTS)
    for collision: Dictionary in collisions:
        var collider_variant: Variant = collision.get("collider", null)
        if collider_variant is StaticBody2D or collider_variant is TileMapLayer:
            return false
    return true


## 按运行时相同的主层选择顺序获取用于地图范围与导航的 TileMapLayer。
func _get_navigation_layer(level: Node2D) -> TileMapLayer:
    for layer_name: String in NAVIGATION_LAYER_NAMES:
        var layer: TileMapLayer = level.get_node_or_null(layer_name) as TileMapLayer
        if layer != null:
            return layer
    return null


## 根据地图主层已使用区域计算加载前的完整世界矩形，算法与 world_controller.gd 保持一致。
func _resolve_level_world_rect(level: Node2D) -> Rect2:
    var navigation_layer: TileMapLayer = _get_navigation_layer(level)
    if navigation_layer == null:
        return Rect2()
    var used_rect: Rect2i = navigation_layer.get_used_rect()
    if not used_rect.has_area() or navigation_layer.tile_set == null:
        return Rect2()

    var tile_size: Vector2 = Vector2(navigation_layer.tile_set.tile_size)
    var top_left_local: Vector2 = navigation_layer.map_to_local(used_rect.position) - tile_size * 0.5
    var bottom_right_cell: Vector2i = used_rect.position + used_rect.size - Vector2i.ONE
    var bottom_right_local: Vector2 = navigation_layer.map_to_local(bottom_right_cell) + tile_size * 0.5
    var top_left_global: Vector2 = navigation_layer.to_global(top_left_local)
    var bottom_right_global: Vector2 = navigation_layer.to_global(bottom_right_local)
    return Rect2(top_left_global, bottom_right_global - top_left_global)


## 检查迁移边界是否能被固定单元格尺寸整除，防止隐式截断改变服务端可达范围。
func _is_cell_aligned(value_milli: int) -> bool:
    return value_milli % CELL_SIZE_MILLI == 0


## 计算原始位图字节的 SHA-256 小写十六进制，供人工校验；服务端上传时仍会独立重算。
func _sha256_hex(data: PackedByteArray) -> String:
    var hashing_context: HashingContext = HashingContext.new()
    var start_error: Error = hashing_context.start(HashingContext.HASH_SHA256)
    if start_error != OK:
        return ""
    var update_error: Error = hashing_context.update(data)
    if update_error != OK:
        return ""
    return hashing_context.finish().hex_encode()


## 根据单场景或全量模式写 JSON 和可选的首批发布 SQL；写入失败时返回非零退出码。
func _write_outputs(arguments: Dictionary, exported_entries: Array[Dictionary]) -> int:
    var output_path: String = str(arguments.get("output", "")).strip_edges()
    if not output_path.is_empty():
        if not _write_json_file(output_path, exported_entries[0]):
            return 6

    var output_dir: String = str(arguments.get("output_dir", "")).strip_edges()
    if not output_dir.is_empty():
        var absolute_output_dir: String = _absolute_path(output_dir)
        var make_dir_error: Error = DirAccess.make_dir_recursive_absolute(absolute_output_dir)
        if make_dir_error != OK:
            push_error("无法创建 JSON 输出目录：%s，错误码=%d" % [absolute_output_dir, make_dir_error])
            return 7
        for exported_entry: Dictionary in exported_entries:
            var scene_id: int = int(exported_entry.get("scene_id", 0))
            var scene_output_path: String = absolute_output_dir.path_join(
                "scene_navigation_%d.json" % scene_id
            )
            if not _write_json_file(scene_output_path, exported_entry):
                return 8

    var sql_output_path: String = str(arguments.get("sql_output", "")).strip_edges()
    if not sql_output_path.is_empty():
        if not _write_text_file(sql_output_path, _build_seed_sql(exported_entries)):
            return 9
    return 0


## 将一个场景的导出数据写为便于后台粘贴上传的格式化 JSON。
func _write_json_file(path: String, data: Dictionary) -> bool:
    return _write_text_file(path, JSON.stringify(data, "  ") + "\n")


## 创建父目录并写入 UTF-8 文本文件。
func _write_text_file(path: String, content: String) -> bool:
    var absolute_path: String = _absolute_path(path)
    var parent_directory: String = absolute_path.get_base_dir()
    if not parent_directory.is_empty():
        var make_dir_error: Error = DirAccess.make_dir_recursive_absolute(parent_directory)
        if make_dir_error != OK:
            push_error("无法创建输出目录：%s，错误码=%d" % [parent_directory, make_dir_error])
            return false
    var output_file: FileAccess = FileAccess.open(absolute_path, FileAccess.WRITE)
    if output_file == null:
        push_error("无法写入导出文件：%s，错误码=%d" % [absolute_path, FileAccess.get_open_error()])
        return false
    output_file.store_string(content)
    output_file.close()
    print("[SceneNavigationExport] wrote=%s" % absolute_path)
    return true


## 生成迁移可直接包含的初始发布数据 SQL；服务端运行时只读取 status=1 的版本。
func _build_seed_sql(exported_entries: Array[Dictionary]) -> String:
    var sql_lines: PackedStringArray = PackedStringArray()
    sql_lines.append("BEGIN;")
    sql_lines.append("")
    sql_lines.append("-- 以下位图由 client/tools/export_scene_navigation.gd 基于正式地图和玩家碰撞体生成。")
    sql_lines.append("INSERT INTO world_scene_navigation (")
    sql_lines.append("    scene_id, version, origin_x_milli, origin_y_milli, grid_width, grid_height,")
    sql_lines.append("    cell_size_milli, navigation_data, data_hash, walkable_cell_count, source_scene_path,")
    sql_lines.append("    status, change_reason, publish_reason")
    sql_lines.append(") VALUES")
    for entry_index: int in range(exported_entries.size()):
        var entry: Dictionary = exported_entries[entry_index]
        var terminator: String = "," if entry_index < exported_entries.size() - 1 else ""
        sql_lines.append(
            "    (%d, 1, %d, %d, %d, %d, %d, decode('%s', 'hex'), '%s', %d, '%s', 1, '%s', '%s')%s" % [
                int(entry.get("scene_id", 0)),
                int(entry.get("origin_x_milli", 0)),
                int(entry.get("origin_y_milli", 0)),
                int(entry.get("grid_width", 0)),
                int(entry.get("grid_height", 0)),
                int(entry.get("cell_size_milli", 0)),
                str(entry.get("navigation_data", "")),
                str(entry.get("data_hash", "")),
                int(entry.get("walkable_cell_count", 0)),
                _escape_sql_literal(str(entry.get("source_scene_path", ""))),
                "P0-06 根据正式 Godot 地图资源初始化静态通行位图",
                "P0-06 首批静态通行数据随迁移发布",
                terminator,
            ]
        )
    sql_lines.append("ON CONFLICT (scene_id, version) DO NOTHING;")
    sql_lines.append("")
    sql_lines.append("-- 所有启用场景必须存在一个已发布导航版本，否则迁移失败，避免服务启动后普通移动全部被拒绝。")
    sql_lines.append("DO $$")
    sql_lines.append("BEGIN")
    sql_lines.append("    IF EXISTS (")
    sql_lines.append("        SELECT 1")
    sql_lines.append("        FROM world_scene_definition AS scene")
    sql_lines.append("        WHERE scene.status = 1")
    sql_lines.append("          AND NOT EXISTS (")
    sql_lines.append("              SELECT 1")
    sql_lines.append("              FROM world_scene_navigation AS navigation")
    sql_lines.append("              WHERE navigation.scene_id = scene.scene_id")
    sql_lines.append("                AND navigation.status = 1")
    sql_lines.append("          )")
    sql_lines.append("    ) THEN")
    sql_lines.append("        RAISE EXCEPTION 'enabled world scene is missing published navigation data';")
    sql_lines.append("    END IF;")
    sql_lines.append("END")
    sql_lines.append("$$;")
    sql_lines.append("")
    sql_lines.append("COMMIT;")
    return "\n".join(sql_lines) + "\n"


## 转义 SQL 单引号；工具只生成迁移文本，不直接连接或写入数据库。
func _escape_sql_literal(value: String) -> String:
    return value.replace("'", "''")


## 将 res://、user://、绝对路径和当前工作目录相对路径统一转换为可读写的绝对路径。
func _absolute_path(path: String) -> String:
    var trimmed_path: String = path.strip_edges()
    if trimmed_path.begins_with("res://") or trimmed_path.begins_with("user://"):
        return ProjectSettings.globalize_path(trimmed_path)
    if trimmed_path.is_absolute_path():
        return trimmed_path
    var current_directory: String = OS.get_environment("PWD").strip_edges()
    if current_directory.is_empty():
        current_directory = ProjectSettings.globalize_path("res://")
    return current_directory.path_join(trimmed_path).simplify_path()


## 立即释放当前导出的地图与玩家节点，防止全量导出时前一张地图碰撞残留到下一张。
func _free_export_nodes(level_instance: Node2D, player_instance: CharacterBody2D) -> void:
    if is_instance_valid(level_instance):
        level_instance.free()
    if is_instance_valid(player_instance):
        player_instance.free()


## 输出无头导出工具的支持参数和最小使用示例。
func _print_usage() -> void:
    print("用法：")
    print("  Godot --headless --path client --script res://tools/export_scene_navigation.gd -- --scene-id=9 --output=/tmp/scene_navigation_9.json")
    print("  Godot --headless --path client --script res://tools/export_scene_navigation.gd -- --all --output-dir=/tmp/navigation --sql-output=/tmp/world_scene_navigation_seed.sql")
    print("可选：--bounds-sql=/absolute/path/to/119_world_scene_boundaries.sql")
