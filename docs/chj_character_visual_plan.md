# CHJ 与 PNG 形象共存方案

本文描述如何在 `pocket-pet-remake` 客户端中接入 `.chj` 运行时精灵解析，并与现有 `UnitSkin + SpriteFrames + PNG` 方案共存。

> **运营 / 配置实操**：参数用法、动画新建步骤与示例见 [`形象动画配置指南.md`](./形象动画配置指南.md)。

参考实现：

- Demo 工程：`/Users/wangzhiwei/game/精灵帧demo`
- CHJ 格式说明：`docs/chj_sprite_web_control.md`
- 宠物格步跟随（后续）：`docs/pet_follow_logic.md`

## 1. 背景与目标

### 1.1 现状

当前世界玩家形象有三层回退：

| 优先级 | 路径 | 说明 |
| --- | --- | --- |
| 1 | `CharacterVisual` + `UnitSkin` | 服务端 `skin_id` → `.tres` → `AnimatedSprite2D` + `SpriteFrames` |
| 2 | `player.tscn` Legacy | 无 `skin_id` 时使用 `AnimationPlayer` + `Sprite2D(hframes)` |
| 3 | 战斗覆盖 | 进入战斗时临时切 `战斗待机_004` 等专用皮肤 |

移动方式为客户端连续位移（`velocity = direction * move_speed`），服务端权威坐标为 `Vec2i` 格坐标。

### 1.2 目标

- **有 `.chj` 且可加载**：世界场景 idle/walk 使用 CHJ 运行时切帧（与 demo 一致）。
- **战斗待机**：使用主 CHJ **末尾最后两个动画组**合并后的帧序列循环播放（如 647.chj 的组 10/11）。
- **技能/普攻**：通过 `UnitSkin.chj_skill_path` 指定独立 CHJ，播放第 0 组一次性动画。
- **只有 PNG / SpriteFrames**：保持现有 `AnimatedSprite2D` 逻辑不变。
- **无 skin_id**：继续走 Legacy，不做改动。
- **服务端协议不变**：仍只下发 `skin_id`；CHJ/PNG 由客户端资源配置决定。

## 2. Demo 方案摘要

Demo 拆为两个模块：

```text
ChjSprite       解析 .chj 二进制 → Texture2D + 动画组帧序号
ChjWorldRenderer  按方向/状态手动切 AtlasTexture + flip_h
```

核心映射（四方向 × idle/walk）：

| 方向 | 待机组 | 行走组 | 备注 |
| --- | --- | --- | --- |
| down | 0 | 1 | |
| up | 2 | 3 | |
| left | 4 | 5 | |
| right | 4 | 5 | `force_flip = true` |

帧号规则：

- `raw < 128`：正常绘制第 `raw` 帧。
- `raw >= 128`：绘制第 `raw - 128` 帧并水平翻转。

动画计时：

- 行走：`frame_tick += delta * walk_anim_speed`
- 待机：`idle_tick += delta * idle_anim_speed`（停止时不清零 idle，避免待机卡首帧）
- 取帧：`frames[floor(tick / frame_divisor) % frames.size()]`

Demo 移动为 **16px 格步**；本方案第一期 **只换渲染、不改移动**，降低与 `world_controller`、寻路、服务端的耦合。

## 3. 架构设计

### 3.1 模块划分

```text
client/scripts/feature/character/
├── chj_sprite.gd              # CHJ 二进制解析（RefCounted）
├── chj_world_renderer.gd      # 世界 CHJ 切帧（Sprite2D）
├── character_visual.gd        # 双后端调度：CHJ | PNG
└── character_skin_registry.gd # 不变，仍按 skin_id 加载 UnitSkin

client/scripts/feature/battle/resources/
└── unit_skin.gd               # 新增 chj_path 等字段 + resolve_chj_path()

client/asset/chj/              # CHJ 原始文件目录（按 skin_id 或编号命名）
client/resources/battle/unit_skins/  # 现有 UnitSkin .tres
```

### 3.2 渲染选型流程

```text
apply_skin_id(skin_id)
    ↓
有 chj_path 且可加载？
    ├─ 是 → RENDER_MODE_CHJ（基础渲染）
    │         每个状态播放时检查 sprite_frames 是否有覆盖：
    │           world_animation_map / world_battle_animation 映射的动画名
    │           且 sprite_frames.has_animation(name) → 临时切 PNG 只播该动画
    │           否则 → CHJ（行走/待机/战斗待机末两组/技能 CHJ）
    └─ 否 → 仅 sprite_frames → RENDER_MODE_PNG
              都没有 → 回退 Legacy
```

**局部覆盖示例**：只替换战斗待机

```text
chj_path = "res://asset/chj/2057.chj"        # 行走/日常待机仍走 CHJ
sprite_frames = 仅含「战斗待机」动画
world_battle_animation = "战斗待机"           # 进入 battle 态时用 PNG 覆盖
```

### 3.3 CharacterVisual 场景结构

```text
CharacterVisual (Node2D)
├── AnimatedSprite2D   [%AnimatedSprite2D]  PNG 模式
└── ChjSprite2D        [%ChjSprite2D]       CHJ 模式，脚本 chj_world_renderer.gd
```

### 3.4 与 player.gd 的接口

`player.gd` 已有 `_uses_character_visual` 分支，**不改状态机**，仅调用：

- `apply_skin_id(skin_id)` — 换肤
- `play_world(state, direction_suffix)` — 每帧或状态变化时触发
- `CharacterVisual._process(delta)` — CHJ 模式内部 tick 动画

Legacy 与 PNG 路径行为保持不变。

## 4. UnitSkin 扩展字段

在 `unit_skin.gd` 增加：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `chj_path` | String | 显式 CHJ 路径；留空则尝试 `res://asset/chj/{skin_id}.chj` |
| `chj_display_scale` | Vector2 | CHJ 显示缩放，默认 `(2, 2)` |
| `chj_walk_anim_speed` | float | 行走动画速度，默认 `120` |
| `chj_idle_anim_speed` | float | 待机动画速度，默认 `60` |
| `chj_frame_divisor` | float | 取帧除数，对应 demo 的 `/8`，默认 `8` |
| `chj_skill_path` | String | 技能/普攻专用 CHJ；留空则尝试 `{skin_id}_skill.chj` |

新增方法：

- `resolve_chj_path() -> String`：显式路径优先，其次 `{skin_id}.chj` 约定路径。
- `resolve_chj_skill_path() -> String`：显式路径优先，其次 `{skin_id}_skill.chj`。
- `uses_chj_world_render() -> bool`：`resolve_chj_path()` 非空。

**战斗待机**：由主 CHJ 的 `get_battle_idle_frames()` 自动取最后两组，无需 PNG。

**技能动画**：由 `chj_skill_path` 独立 CHJ 的第 0 组播放，播完回到战斗待机。

## 5. 资源约定

### 5.1 目录

```text
client/asset/chj/
  {skin_id}.chj          # 推荐：与 skin_id 同名
  2057.chj               # 或按原始编号命名，在 .tres 里写 chj_path
```

### 5.2 UnitSkin 配置示例

**纯 PNG（现有）** — 不填 `chj_path`：

```text
skin_id = "初始形象男_001"
sprite_frames = ...
world_animation_map = { idle_down: "待机下", walk_down: "向下走", ... }
```

**CHJ 世界 + 技能 CHJ（推荐）**：

```text
skin_id = "某CHJ形象"
chj_path = "res://asset/chj/xxx.chj"           # 世界 idle/walk + 战斗待机（末两组）
chj_skill_path = "res://asset/chj/xxx_skill.chj"  # 普攻/技能一次性动画（第 0 组）
chj_display_scale = Vector2(2, 2)
```

`sprite_frames` 可省略；纯 PNG 皮肤仍按原方式配置。

### 5.3 服务端

无需迁移。测试时将玩家 `skin_id` 改为 `CHJ测试_2057` 即可。

## 6. 移动模型分阶段

| 阶段 | 内容 | 说明 |
| --- | --- | --- |
| **一期（本次）** | CHJ 渲染 + 连续移动 | `state==walk` 时走 walk 组，`idle` 时走 idle 组 |
| 二期 | CHJ 皮肤格步移动 | 16px 插值，更接近原版 J2ME |
| 三期 | 服务端格步同步 + 宠物跟随 | 对齐 `pet_follow_logic.md` |

一期动画与位移可能略不同步，可接受；二期再统一。

## 7. 实施清单（一期）

- [x] 本文档
- [x] `chj_sprite.gd` — 从 demo 移植解析逻辑
- [x] `chj_world_renderer.gd` — 方向映射 + tick + AtlasTexture
- [x] `unit_skin.gd` — CHJ 字段与 `resolve_chj_path()`
- [x] `character_visual.gd` / `.tscn` — 双后端切换
- [x] `client/asset/chj/` — 放置测试用 `.chj`
- [x] `unit_skins/其他/CHJ测试_2057.tres` — 示例皮肤
- [ ] 本地验证：PNG 皮肤 / CHJ 皮肤 / 无 skin_id Legacy 三路共存

## 8. 风险与回退

| 风险 | 应对 |
| --- | --- |
| CHJ 某动画组为空 | 回退到组 0（正面待机） |
| CHJ 无战斗帧 | 未配置 sprite_frames 时，战斗待机走主 CHJ 末两组；技能走 chj_skill_path |
| 缩放/碰撞偏移不准 | `chj_display_scale` + `world_collision_offset` 按皮肤单独调 |
| `.chj` 缺失 | 自动回退 PNG；PNG 也无则 Legacy |

## 9. 后续扩展

- 宠物/跟随者复用 `ChjWorldRenderer` + 格步控制器（见 `pet_follow_logic.md`）。
- 其他玩家实体（`nearby_entities`）展示层接入同一 `CharacterVisual`。
- 可选：Editor 工具从 `.chj` 预览并导出 `UnitSkin` 碰撞/缩放参数。
