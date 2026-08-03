# 地图切换加载方案

## 1. 文档目的

- 本文用于把“参考原版客户端的地图切换加载方案”沉淀为当前项目可直接实现的设计文档。
- 本文只覆盖当前 MVP 范围内的世界地图切换与加载，不扩展开放世界实时流式加载、多层 AOI 分片或跨服地图。
- 实现目标是：
  - 保持现有登录、世界、战斗主链路稳定
  - 继续遵循服务端权威切图
  - 让客户端在 Godot 中可以按 `scene_id` 正常装载和切换地图资源

## 2. 参考原版后的核心结论

从原版客户端 `kdjl` 提炼出来、对当前项目最有价值的不是旧协议或旧 UI，而是下面三条结构性原则：

1. 世界层和战斗层分离
2. 地图切换由服务端确认，客户端只负责加载表现
3. 世界主场景常驻，地图内容热切换，而不是每次整棵树重建

对应参考依据见 [kdjl-client-reference.md](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/docs/kdjl-client-reference.md#L131-L181)。

一句话口径：

- 不复刻原版的 J2ME 菜单形式
- 只吸收它“世界层常驻 + 地图切换由服务端驱动 + 客户端做加载表现”的核心思想

## 3. 当前项目现状

当前项目已经有一条最小地图切换链路：

- 客户端在 [world_controller.gd](file:///Users/wangzhiwei/study/pocket-pet-remake/client/scripts/feature/world/world_controller.gd#L84-L117) 中通过 `MOVE_INTENT_REQ` 请求地图切换
- 服务端在 [world_handler.go](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/server/internal/transport/ws/world_handler.go#L83-L147) 中校验 `target_scene_id`，并返回 `MOVE_INTENT_RESP` 与 `WORLD_RESYNC_PUSH`
- 协议文档里，`MOVE_INTENT_REQ` 已被定义为“申请切换到目标地图”，见 [protocol.md](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/docs/protocol.md#L226-L284)
- 主运行态场景 `main.gd` 保持世界层与战斗层分开挂载，见 [main.gd](file:///Users/wangzhiwei/study/pocket-pet-remake/client/scripts/bootstrap/main.gd#L43-L79) 与 [main.gd](file:///Users/wangzhiwei/study/pocket-pet-remake/client/scripts/bootstrap/main.gd#L212-L231)

当前缺口在于：

- 客户端还没有真正按 `scene_id` 去装载地图资源
- `world_controller.gd` 现在只是在做坐标换算和场景 ID 状态切换
- 地图切换的加载表现、地图节点挂载/卸载、传送点抽象还没有形成完整方案

## 4. 设计目标

本方案希望把地图切换实现成下面这个效果：

- 世界运行态始终留在同一个根场景
- 当前地图资源挂载在固定的 `MapMount`
- 玩家踩中地图里的门区或传送点时，客户端只发切图意图
- 服务端确认地图拓扑、发布状态和最低进入等级后，才更新玩家场景与入口落点
- 地图最低进入等级保存在 `world_scene_definition.required_level`，由运营后台维护；玩家等级使用服务端数据库档案，不接受客户端传值
- 等级不足时服务端拒绝切图并保留原场景，客户端提示“前面的路以后再来探索吧”
- 客户端收到服务端同步后再真正替换地图资源
- 战斗开始时只隐藏世界层并挂载战斗层，不销毁当前地图运行态

## 5. 总体方案

### 5.1 场景结构

推荐保持如下世界结构：

```text
Main
  ├─ WorldMount
  │   └─ WorldRoot
  │       ├─ MapMount
  │       ├─ RemoteEntities
  │       └─ LocalPlayerAnchor
  └─ BattleMount
```

职责说明：

- `WorldRoot`：世界控制根节点，常驻
- `MapMount`：当前地图资源挂载点，只负责当前地图
- `RemoteEntities`：附近实体容器，不跟地图资源脚本耦合
- `LocalPlayerAnchor`：本地角色锚点，常驻
- `BattleMount`：战斗层容器，和世界层分离

### 5.2 地图资源形式

每张地图推荐单独一个 Godot 场景，例如：

- `client/scenes/maps/fashtown/roxus_house.tscn`
- `client/scenes/maps/<map_name>.tscn`

每张地图场景只负责表现：

- 地板和障碍
- 装饰物
- 传送门/出口标记
- 可选碰撞辅助层

不要把下面这些逻辑写进地图场景：

- WebSocket 发包
- 地图切换判定
- 战斗切换逻辑
- 全局状态管理

## 6. 客户端职责

客户端只负责下面 6 件事：

1. 监听本地角色踩中地图里的门区或传送点
2. 把目标 `scene_id` 或入口意图上报服务端
3. 在等待期间锁定移动输入并显示加载态
4. 收到服务端权威结果后装载目标地图场景
5. 根据服务端 `self_pos` 放置本地角色
6. 恢复输入并刷新附近实体表现

客户端不负责：

- 最终判定目标地图是否合法
- 决定入口落点
- 决定能否切图
- 自己提前改权威场景状态

## 7. 服务端职责

服务端继续负责：

1. 校验当前玩家是否允许切图
2. 计算目标地图和目标入口落点
3. 更新玩家当前 `scene_id` 与位置
4. 返回 `MOVE_INTENT_RESP`
5. 下发 `WORLD_RESYNC_PUSH`

服务端不负责：

- 客户端地图资源装载
- 本地过场动画
- Godot 场景树切换

## 8. 推荐时序

### 8.1 地图切换时序

推荐按下面时序实现：

```text
1. 本地角色触发门区或传送点
2. 客户端锁定输入，显示“切换中”
3. 客户端发送 MOVE_INTENT_REQ(scene_id, target_scene_id)
4. 服务端校验并返回 MOVE_INTENT_RESP
5. 服务端下发 WORLD_RESYNC_PUSH
6. 客户端收到 WORLD_RESYNC_PUSH 后：
   - 卸载旧地图
   - 挂载新地图
   - 以服务端 self_pos 放置玩家
   - 刷新 AOI/附近实体
   - 关闭加载态，恢复输入
```

### 8.2 异常时序

如果服务端拒绝切图：

```text
1. 收到 MOVE_INTENT_RESP.accepted = false
2. 客户端取消 pending 状态
3. 解锁玩家输入
4. 提示切图失败原因
5. 保持原地图不变
```

## 9. 当前协议如何承接

当前协议无需推翻，只需要继续沿用：

### 9.1 `MOVE_INTENT_REQ`

作用：

- 表达“我要切到哪个目标地图”

当前定义见 [protocol.md](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/docs/protocol.md#L226-L237)。

### 9.2 `MOVE_INTENT_RESP`

作用：

- 返回服务端是否接受此次切图意图
- 如果接受，带回权威目标 `scene_id` 与 `corrected_pos`
- `corrected_pos` 应表达“从哪个入口进入目标地图后的落点”，而不是固定地图中心

当前定义见 [protocol.md](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/docs/protocol.md#L239-L258)。

### 9.3 `WORLD_RESYNC_PUSH`

作用：

- 作为切图后的权威世界快照
- 客户端应以它作为真正替换地图资源和重置玩家位置的依据

当前定义见 [protocol.md](file:///Users/wangzhiwei/study/pocket-pet-remake/backend/docs/protocol.md#L260-L284)。

## 10. `world_controller.gd` 的落地改造点

当前 [world_controller.gd](file:///Users/wangzhiwei/study/pocket-pet-remake/client/scripts/feature/world/world_controller.gd) 已有最小切图骨架，建议沿现有结构扩展，而不是重写。

### 10.1 保留现有内容

保留这些已有能力：

- `request_scene_transition()`
- `handle_move_intent_response()`
- `handle_world_resync()`
- `_pending_target_scene_id`

这些部分已经符合“先发意图，再等服务端确认”的原则。

### 10.2 需要新增的内容

建议新增：

- `_map_mount` 节点引用
- `_current_map_node`
- `_load_scene_map(scene_id: int)`
- `_unload_scene_map()`
- `_apply_scene_snapshot(scene_id: int, self_pos: Vector2)`
- `_set_transition_loading(active: bool)`

### 10.3 推荐改造口径

不要在 `MOVE_INTENT_RESP.accepted = true` 时立即切地图。  
正确做法是：

- 先进入“等待世界重同步”状态
- 只有在 `WORLD_RESYNC_PUSH` 到来后，再真正装载目标地图

原因：

- `MOVE_INTENT_RESP` 表示请求被受理
- `WORLD_RESYNC_PUSH` 才是完整的切图后权威快照
- 普通坐标同步必须保持最多一个请求在途；服务端按连接串行处理消息时，这能防止高频移动请求在慢数据库链路中排到切图请求前面
- 普通移动回包只解除移动同步背压，只有与当前切图 `move_seq` 相同的回包才能改变转场状态

## 11. 推荐的地图配置结构

当前 `SCENE_CONFIGS` 只保留：

- `scene_path`
- `spawn`

后续建议扩成：

```gdscript
const SCENE_CONFIGS := {
    1: {
        "scene_path": "res://scenes/maps/fashtown/roxus_house.tscn",
        "spawn": Vector2(8, 6)
    },
    2: {
        "scene_path": "res://scenes/maps/<map_name>.tscn",
        "spawn": Vector2(2, 4)
    }
}
```

第一阶段建议只做到这里，不要过早抽象成复杂资源系统。

## 12. 入口点口径

### 12.1 为什么不能统一落在中心

如果切图后统一落在地图中心，会带来 3 个问题：

1. 玩家体感上像“瞬移到房间中心”，没有经过门口进入的连续感
2. 后续做城镇门、洞口、楼梯、传送阵时，无法保证玩家出现在正确入口
3. 客户端地图表现会和服务端切图语义脱节

所以当前项目的最小口径应改为：

- 服务端只确认本次切图是否允许
- 客户端加载目标场景后，按目标场景脚本里 `portal_id -> 出生场景坐标` 配置摆放玩家
- 不再假设每张地图只有一个统一出生点

### 12.2 当前最小实现

当前最小链路已经补上显式 `portal_id`。  
因此当前阶段采用下面这条最低成本规则：

- 玩家进入地图里的 `Area2D` 门区后，客户端上报 `target_scene_id + portal_id`
- 服务端按 `portal_id` 确认目标地图可达，不再负责客户端最终出生场景坐标
- 客户端在目标场景的 `*_level.gd` 中按来源 `portal_id` 读取出生点；如果未配置，则回退使用服务端 `self_pos` 换算后的坐标

例如：

- `1 -> 2` 时，玩家落在 `2` 号地图的左侧入口
- `2 -> 1` 时，玩家落在 `1` 号地图的右侧入口
- `2 -> 3` 时，玩家落在 `3` 号地图的左侧入口

### 12.3 后续升级方向

如果未来出现“同一张地图上门很多，且希望策划或地图资源独立配置”的情况，当前硬编码 `portal_id` 仍然不够灵活。  
那时应升级为：

- 客户端继续上报 `portal_id` 或 `entry_id`
- 服务端只校验场景切换可达性
- 客户端地图场景为每个来源门维护对应出生点
## 13. 门区切换

### 13.1 当前做法

地图画出来后建议补充：

- 每张地图场景内放置 `Area2D` 门区
- 每个门区关联 `portal_id` 与目标 `scene_id`
- 玩家进入时由 `world_controller` 接管并统一调用 `request_scene_transition()`

### 13.2 统一原则

所有门区切换都统一走一条协议链路：

- 本地触发
- 请求服务端
- 服务端确认
- 客户端加载目标地图

## 14. 视觉加载表现

推荐最小实现，不额外引入复杂动画资源：

- 黑色半透明遮罩
- “地图加载中”文字
- 玩家输入锁定
- 切图完成后淡入恢复

当前项目已经在 [main.gd](file:///Users/wangzhiwei/study/pocket-pet-remake/client/scripts/bootstrap/main.gd#L246-L260) 使用了 `transition_overlay` 做场景淡入淡出，这套思路可以复用到世界层切图。

建议：

- 世界切图遮罩优先放在 `WorldRoot` 或 `Main` 中统一管理
- 不要每张地图自己维护一套加载遮罩

## 15. 为什么不建议整树切场景

地图切换不要使用整棵树 `change_scene`，原因有 4 点：

1. 会打断已有网络与消息路由稳定性
2. 不利于保持世界态和战斗态分层
3. 容易把地图切换和登录/主场景切换混在一起
4. 与原版“世界层常驻”的可参考结构不一致

所以建议始终保持：

- `main.tscn` 常驻
- `world_scene.tscn` 常驻
- 地图资源节点热替换

## 16. 分阶段实现顺序

### 第一阶段：最小切图可用

- 新建最小地图场景资源
- 为 `SCENE_CONFIGS` 增加 `scene_path`
- 在 `world_controller.gd` 中实现地图节点挂载/卸载
- 收到 `WORLD_RESYNC_PUSH` 后真正切换地图资源

### 第二阶段：加载体验补齐

- 增加切图遮罩
- 增加切图期间输入锁定
- 增加失败提示

### 第三阶段：传送点系统

- 在地图场景中补 `Area2D` 传送门
- 统一门区触发逻辑
- 支持地图入口点和固定出生点

### 第四阶段：配置外置

- 将地图路径、邻接关系、传送点从硬编码迁到配置文件或资源

## 17. 实现时必须遵守的约束

- 客户端不能自行确认切图成功
- 客户端不能自行决定入口落点
- 服务端必须是最终 `scene_id` 和 `self_pos` 的权威来源
- 地图资源加载不能打断当前 WebSocket 会话
- 战斗层切换不能与地图层切换耦合

## 18. 一句话实施口径

后续实现地图切换加载时，请始终按下面这条口径执行：

- 世界根场景常驻
- 地图资源按 `scene_id` 热切换
- 客户端只在踩中门区后发切图意图
- 服务端确认目标地图和入口落点
- 客户端收到权威快照后再真正装载地图

## 19. 闪光镇地图节点面板

- 主运行态 HUD 提供“地图”入口，打开 `map_teleport_panel.tscn`；面板资源默认隐藏，运行时才显示。
- 面板使用 `res://asset/kdjl-map-points/闪光镇.png` 作为地图底图，7 个按钮按配套标点 JSON 的像素坐标覆盖在地图节点上。
- 地图绘制外框固定为 `592×640`；世界地图、闪光镇、闪光平原和精灵迷宫分别按原图 3 倍显示为 `528×528`、`384×384`、`432×480`、`576×624`，保持原始宽高比与统一图标倍率。
- 闪光镇标点按钮统一为 `52×52`，世界地图地区按钮统一为 `72×72`；按钮中心按原始图片坐标的 3 倍倍率定位。
- 玩家可以直接点击标点，或使用键盘及手柄 `ui_up/ui_down` 循环切换唯一选中态；原屏幕“上一个 / 下一个”按钮已删除。
- 地区地图底部提供单个“世界地图”按钮，在同一个 `map_teleport_panel.tscn` 中切换到 `世界地图.png`，不会卸载或创建新的顶层 UI 场景。
- 世界地图中的闪光镇、闪光平原和精灵迷宫热点分别进入对应地区地图视图；只有已具备客户端场景资源和服务端快速传送配置的地区内部节点可以发起传送。
- 第一次点击标点只切换选中态；再次点击当前选中标点才发送 `MOVE_INTENT_REQ(map_teleport=true)`。上下切换只移动选择，不自动传送。
- `world_map_teleport_node` 保存开放状态和目标地图中心出生格；客户端只提交 `target_scene_id`，服务端校验并持久化权威场景及坐标后，继续通过 `MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH` 驱动黑屏转场。
- 当前开放场景为 `scene_id=1..6`。未落地客户端地图的“通往闪光平原”保持可选但不发请求；一次性剧情场景 `scene_id=7` 不写入快速传送配置。
- 传送到当前地图时仍会移动到数据库配置的中心格，但不会重复触发离开/进入 AOI 或 `ENTER_SCENE` 任务事件。

## 20. 跨地图快速快照时序

- 服务端接受跨地图请求并持久化目标场景坐标后，先更新进程内场景索引，再立即发送基础 `WORLD_RESYNC_PUSH`；该快照包含地图实体和暗雷配置，但不等待在线玩家资料。
- 客户端以基础 `WORLD_RESYNC_PUSH` 作为解除黑屏和加载地图的权威信号。同场景在线玩家及其首只跟随宠物随后通过 `ENTITY_ENTER_PUSH` 增量补齐。
- 在线玩家和跟随宠物分别使用一次批量轻量查询，只读取位置、名字、等级、经验、血量、精力和形象等世界展示字段，不读取背包，也不刷新完整战斗快照。
- `ENTER_SCENE` 任务推进发生在基础世界快照之后，并使用不含 `player_container_item` 的场景事实查询；任务更新和 `SCENE_TRIGGER_PUSH` 允许晚于地图显示到达。
- 背包保持按需加载：切图、玩家同屏和任务场景事件均不请求背包物品，只有玩家点击背包入口时才走既有背包请求。

## 21. NPC 菜单预加载

- 客户端收到基础 `WORLD_RESYNC_PUSH` 并完成地图场景挂载后，立即结束转场等待和场景移动锁；NPC 菜单、在线玩家、任务更新和场景剧情均允许在玩家进入地图后继续返回。
- 客户端根据当前权威 `scene_id` 只发送一次 `NPC_MENU_BATCH_REQ`，服务端一次读取场景实体和任务摘要，并以一条批量 SQL 读取全部 NPC 静态菜单，再返回 `NPC_MENU_BATCH_RESP`。
- 当前地图菜单按 `scene_id + entity_id` 缓存在客户端；玩家进入 NPC 碰撞区或从“周围 NPC”列表选择目标时只读取该缓存，不再临时发起菜单请求。
- 批量菜单属于后台缓存请求，不显示全屏 loading，不设置运行时输入锁，也不参与 `_scene_map_new_scene_ready`；旧地图迟到响应会按 `scene_id` 丢弃。
- 同地图剧情解锁出新 NPC 后，会再发送一次当前地图批量菜单请求并合并最新结果，不会退回逐 NPC 串行请求。
- `NPC_ACTION_REQ`、任务领取与交付等会改变菜单状态的操作仍可主动刷新对应 NPC 菜单，菜单内容及可见条件始终由服务端权威计算。

## 闪光平原场景注册与普通传送门

- 闪光平原当前已落地地图使用连续的服务端 `scene_id=9..25`；其中 `9` 为闪耀广场，`10..25` 由迁移 `112_shining_plain_scenes.sql` 注册到 `world_scene_definition`。
- 客户端 `WorldSceneRegistry` 只负责将服务端 `scene_id` 映射到对应 `.tscn`，地图脚本只提交 `target_scene_id + portal_id` 切图意图。
- 门的可达关系与切图出生格统一维护在服务端 `worldScenes`；客户端收到 `MOVE_INTENT_RESP` / `WORLD_RESYNC_PUSH` 后按服务端坐标展示，不能用本地门位置覆盖权威出生格。
- 已对全部启用传送门的目标出生格执行 TileMap 物理检查；报名区进入准备区使用 `(2, 7)`，闪光南路进入家族会馆使用 `(12, 10)`，避免角色落在地图外或阻挡格。
- 报名区场景中的“通往准备区”和“通往比武区”当前都进入已落地的准备区资源；准备区内“通往战斗区”因目标场景尚未落地而保持禁用。
- 商业区“通往闪光平原传送区”、闪光海岸/精灵大厅“通往海道”同样因目标 `.tscn` 尚未落地而不绑定切图请求，避免进入不存在的客户端场景。
