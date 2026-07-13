# 世界剧情动作场景

固定过场的移动、角色动画和对白顺序全部由客户端动画 Key 场景负责。服务端只下发动画 Key，并等待整段客户端过场完成。

## 新建一段固定过场

1. 在 Godot 中打开 `res://scenes/cinematics/common/world_player_cinematic.tscn`。
2. 使用“新建继承场景”，把新场景保存到 `res://scenes/cinematics/` 或 `res://剧情动画/` 根目录。
3. 文件名就是后台动作节点填写的动画 Key。例如 `story_training_intro.tscn` 对应 `story_training_intro`。
4. 给继承场景根节点附加一个继承 `WorldPlayerCinematic` 的脚本，在 `_run_sequence()` 中固定编排 Tween、动画和对白。

不需要修改动画注册表。以下是“左移 100px、上移 50px、播放两句本地对白”的完整脚本：

```gdscript
extends WorldPlayerCinematic

## 固定编排当前动画 Key 对应的完整客户端过场。
func _run_sequence() -> void:
    if not begin_player_cinematic():
        complete_cinematic()
        return

    var moved_left: bool = await tween_player_offset(
        Vector2(-100.0, 0.0),
        1.0,
        "walk_left"
    )
    if not moved_left:
        return

    var moved_up: bool = await tween_player_offset(
        Vector2(0.0, -50.0),
        0.5,
        "walk_up"
    )
    if not moved_up:
        return

    set_cinematic_player_facing(Vector2.UP)
    await show_local_dialogue("引导员", "到这里就可以了。")
    await show_local_dialogue("玩家", "我明白了。", "", true)
    complete_cinematic()
```

`walk_left`、`walk_up` 需要替换成当前角色资源实际存在的动画名。Godot 坐标中向左和向上分别使用负 X、负 Y。

通用基类提供：

- `begin_player_cinematic()`：取得真实玩家、锁定手动输入并开始过场控制。
- `tween_player_offset(offset, duration, animation)`：按固定像素偏移做线性 Tween，并同时播放指定行走动画。
- `set_cinematic_player_facing(direction)`：设置终点朝向并恢复对应待机方向。
- `show_local_dialogue(...)`：复用现有对话面板展示客户端文本，并等待玩家点击继续；不会请求服务端。
- `complete_cinematic()`：恢复玩家控制、发出 `finished`，整段过场至此才结束。

基础场景仍保留 Inspector 配置方式，适合不编写派生脚本的导航移动：

- `scene_waypoints`：统一场景坐标数组，与服务端 `self_pos` 和 HUD 坐标口径一致；玩家会通过当前地图导航网格依次前往。
- `final_facing_direction`：结束朝向，使用 `(0,-1)` 上、`(0,1)` 下、`(-1,0)` 左、`(1,0)` 右；零向量表示保持。
- `path_timeout_seconds`：路径最长执行时间；动态障碍持续阻挡时会恢复玩家并继续剧情。
- `animation_name`：路径完成后播放的角色动画名，留空表示只移动。
- `animation_frame`：`-1` 正常播放；大于等于 `0` 时暂停在指定 PNG 动画帧。
- `animation_frame_fps`：旧 `AnimationPlayer` 把帧号换算成时间时使用。
- `animation_hold_seconds`：动作或指定帧保持多久，然后恢复世界待机并推进剧情。

## 服务端动作节点

服务端只需要一个阻塞动作节点：

```text
action 动画Key=story_training_intro，阻塞等待=是
```

客户端收到后执行 `story_training_intro.tscn` 中写死的移动、动画和全部对白。只有脚本调用 `complete_cinematic()` 后，客户端才请求服务端推进该 action 后面的节点。

## 约束

- `tween_player_offset()` 直接修改玩家位置，不处理墙体、动态 NPC 或其他玩家碰撞，只能用于地图设计已确认畅通的固定路线。
- 需要避障时继续使用基础场景的 `scene_waypoints`，由 AStar 和 `move_and_slide()` 执行。
- CHJ 形象支持播放动作，但指定单帧只对具有 `SpriteFrames` 的 PNG 动画生效。
- 固定过场对白可以写在动画 Key 脚本中；会影响任务、奖励或剧情分支的权威结果仍必须由服务端处理。
- 包含本地对白的 action 必须开启“阻塞等待”，否则服务端下一节点可能与客户端过场同时展示。
