# 任务总结

## 2026-08-05：东路进入闪光镇传送区出生格修正

- 将闪光镇传送区根场景的 `inbound_from_east_road_scene_position` 显式配置为 `(1,13)`，确保 `portal_id=2003` 的普通门请求提交正确目标入口格。
- 将服务端正式仓储和测试桩中的 `2003 -> scene 8` 门点及来源场景兼容入口同步为 `(1,13)`；服务端仍先验证普通门拓扑，再持久化并回传权威位置。
- 更新普通门坐标契约测试，防止该入口回退到旧坐标；未修改闪光平原入口、其他传送门、协议或数据库结构。
- 验证通过：Godot 4.7 Headless 实例化场景后确认配置值、`portal_id=2003` 解析值和目标瓦片均有效；相关 Go 包测试与 `git diff --check` 均通过。

## 2026-08-03：普通门切图落点开放为场景导出配置

- 在东路、闪光镇传送区和闪耀广场地图脚本中，将普通门入口坐标统一归入 `普通门切图落点（场景格）` 导出分组。
- 在对应 `.tscn` 地图根节点显式保存当前入口值；以后可直接选中地图根节点，在 Godot 检查器中修改各来源地图的落点，不需要改匹配代码。
- 落点仍按地图场景格填写，普通门请求继续提交给服务端验证、持久化和广播；世界地图快速传送逻辑保持不变。
- 验证通过：7 个导出属性值与 `portal_id` 解析结果专项烟测、Godot 4.7 Headless 项目加载和 `git diff --check`。

## 2026-08-03：普通门切图改为客户端目标场景脚本选择落点

- `client/scripts/feature/world/world_controller.gd` 在普通门请求前加载目标地图场景，调用 `get_portal_spawn_scene_position(portal_id)`，并把有效入口格写入 `MOVE_INTENT_REQ.target_pos`；服务端快照返回后仍只消费 `self_pos`，不在本地二次覆盖。
- `backend/server/internal/module/world/service.go` 在仓储完成门关系和等级验证后采用合法的非负客户端入口格；`world_handler.go` 继续统一持久化、回包、重同步和多人进入广播。快速传送分支不调用该入口规则。
- 补齐东路、传送区和闪耀广场的来源门映射，重点修复从闪光镇东路右门进入传送区时落在左侧门附近。
- 更新协议、架构、地图加载和坐标约定文档，并补充普通门落点、旁观者同步、旧客户端回退、快速传送忽略客户端坐标的测试。
- 验证通过：四条客户端入口映射专项烟测（`2003 -> (2,12)`、`8001 -> (9,5)`、`8002 -> (20,12)`、`9001 -> (6,9)`）、`go test ./server/...`、Godot 4.7 Headless 项目加载和 `git diff --check`。

## 2026-08-03：同步时光小屋显隐与地图碰撞

- 保留 `east_road_of_shanguang_town.tscn` 当前的“时光小屋”默认显隐设置，不覆盖已有场景调整。
- 在 `east_road_of_shanguang_town_level.gd` 中监听“时光小屋”的可见性变化：隐藏时禁用碰撞，重新显示时恢复碰撞。
- 继续调用 `NetworkDoorLevelBase._ready()`，不改变闪光镇东路现有联网传送门绑定和服务端权威切图流程。

## 2026-08-03：移除客户端玩家位置调试日志

- 删除 `client/scripts/feature/world/world_controller.gd` 中 `[PlayerPos][Client]` 周期调试输出，以及只服务于该日志的开关、计时和坐标文本组装逻辑。
- 玩家移动上报、服务端权威位置、数据库持久化、远端玩家插值与多人同屏表现均保持不变。

## 2026-08-03：补齐闪光平原区域地图选择与快速传送

- 在 `client/scenes/ui/world/map_teleport_panel.tscn` 新增独立的 `FlashPlainPointButtons`，按闪光平原原图坐标配置 19 个 `52×52` 热点；17 个已落地场景映射 `scene_id=9..25`，“传送门”和“海道”保持不可传送。
- 修改 `client/scripts/ui/world/map_teleport_panel.gd`，按当前地区切换独立节点容器，复用闪光镇的四帧选中光标、上下循环选择、二次点击传送、未开放提示和人物当前位置图标逻辑；脚本已统一转换为 4 空格缩进。
- 新增 `backend/server/migrations/113_shining_plain_map_teleport_nodes.sql`，为 17 张闪光平原地图写入服务端权威快速传送中心格；迁移仅生成，未直接执行。
- 同步 `backend/server/internal/teststub/repos.go` 的测试中心格，并在 `backend/server/internal/transport/ws/world_handler_test.go` 增加传送到 `scene_id=25` 的权威坐标与持久化断言。
- 更新 `backend/docs/map-scene-loading.md`，明确闪光镇与闪光平原节点范围、未开放节点和服务端权威传送配置。
- 验证通过：Godot 4.7 Headless 实例化测试确认 19 个节点、17 个场景 ID、光标、人物图标、开放/未开放节点二次点击信号均正确；`go test ./server/...` 全量通过，并通过 `git diff --check`。

## 2026-08-03：修正世界地图闪光平原地区图标

- 将 `client/scenes/ui/world/map_teleport_panel.tscn` 中“闪光平原”地区按钮从中央 Boss 图标移到红箭头所指的中间绿色节点。
- 位置按 `世界地图.png` 原始坐标 `(112, 144)`、3 倍显示倍率及地图居中边距换算，热点中心调整为场景坐标 `(368, 488)`，保留原有 `52×52` 点击范围。
- 本次只调整客户端世界地图热点位置，不改变服务端权威传送、场景 ID、协议、闪光镇和精灵迷宫节点。
- 使用 Godot 4.7 headless 加载并实例化地图面板，确认场景可正常解析，运行时“闪光平原”热点中心为 `(368, 488)`；同时通过资源引用检查与 `git diff --check`。

## 2026-08-02：补齐闪光平原新增地图脚本

- 新增 `client/scripts/feature/world/scene_levels/shining_plain_level.gd`，复用 `NetworkDoorLevelBase`，为闪光平原新增地图提供统一地图名称、缩放、中心点和默认出生点接口。
- 将共享脚本绑定到 `client/scenes/maps/闪光平原/` 下 16 张原本没有地图级脚本的场景；“闪耀广场”继续保留已有专用脚本。
- 出于服务端权威原则，本次没有硬编码新增地图的场景 ID、传送门 ID 和目标场景；现有 Area2D 传送区域不会在服务端契约落地前发起客户端切图。
- 使用 Godot 4.7 headless 逐一加载并实例化闪光平原目录内 17 张场景，确认全部具备地图名称、中心点和切图信号接口。

## 2026-07-29 多人同屏出生坐标、远端移动与形象修复

本次修复多人同屏的三类问题，核心是把出生坐标的事实来源收敛到服务端：

- 问题一（旁观视角进入者从别的坐标走过来）：直接根因是 `_sync_remote_players` 读取 `precise_pos` 时把缺省空字典也当成有效坐标，`ENTITY_ENTER_PUSH` 和世界快照实体没有该字段，远端人物于是统一创建在 (0,0) 地图左上角，再被后续移动包慢慢拉到真实位置；现改为字段真实存在且非空才使用。深层根因是客户端在切图/登录后用 `get_portal_spawn_scene_position` / `get_login_spawn_position` 本地覆盖站位，而服务端向旁观者广播的是 `worldScenes` 权威落点，两套坐标不一致。`world_controller.gd` 删除两处本地覆盖，`_apply_authoritative_snapshot` 一律使用 `WORLD_RESYNC.self_pos`；服务端 `world_repo.go` 的 portals/entries 逐门对齐地图当前调好的进门站位（如打怪区 `(6,10)→(6,2)`、市场进东路 `(0,4)→(1,7)`）。
- 注册默认出生：`AdminCreatePlayerInput/AdminUpdatePlayerInput.Normalize()` 的时光小屋默认格从 `(4,4)` 改为客户端调好的 `(6,6)`；新增迁移 `backend/server/migrations/110_align_time_house_default_spawn.sql` 把仍停在旧默认格的存量角色对齐（需用户自行执行）。
- 问题二（远端玩家移动卡顿）：`player.gd` 远端插值不再对目标点和逐帧位置 `round()`，保留连续浮点轨迹；到达阈值从 `0.5px` 收紧为 `0.05px`，行走/待机判定同步调整。像素清晰度仍由项目级 `snap_2d_transforms_to_pixel` 与 nearest 过滤保证，与本地玩家 7-28 的连续浮点移动修复口径一致。
- 问题三（互相看不到对方形象）：远端 `player` 实例此前在 `_ready()` 提前返回、从不消费实体摘要的 `skin_id`，只显示场景默认精灵。`player.gd` 新增 `apply_remote_skin_id()`，与本地共用 `_apply_normal_skin_id()`；`world_controller.gd` 在远端节点创建和每次 `ENTITY_ENTER_PUSH` 刷新时应用该形象。宠物侧继续复用 `following_pet` → `WorldPetFollower` 链路。
- 测试：新增 `TestRouterSceneTransferBroadcastsEntryAtPortalSpawn`（旁观者实体进入推送坐标 == `corrected_pos`，且人物/宠物 `skin_id` 非空）与 `TestPortalSpawnPositionsMatchClientMaps`（逐门断言服务端落点）；`go test ./server/...` 全量通过。客户端完成引用与缩进静态检查（本机未安装 Godot，未执行引擎解析）。
- 文档：更新 `docs/world-coordinate-convention.md`（传送门落点改在服务端维护）与 `backend/docs/architecture.md` 多人移动同步小节。

## 2026-07-29 超时空传送特效 Demo

- 相机像素清晰度修复不回退人物连续移动：`Camera2D` 根据当前 zoom 将跟随中心吸附到屏幕整数像素，关闭物理插值，并恢复正式相机为整数倍率 `3.0`；碰撞、服务端坐标和移动速度保持不变。
- 传送后层光效改用同级绘制顺序：特效实例放在人物精灵之前，`BeamHalo` 与 `GroundRingBack` 使用默认层级，避免负层级被地图 TileMap 覆盖；正层级的前半环、核心和粒子仍覆盖在人物前方。
- `MapTeleportEffect` 改为 `player.tscn` 的预置子节点，当前位置和三分之一缩放均保存在人物场景中；控制器不再重设位置或跨层重挂节点，后续可直接在编辑器中拖动调整。
- 正式传送特效定位从人物碰撞体中心改为 `Sprite2D.position=(13,-20)`，使传送阵中心与人物图像正中心精确重合，不改变服务端坐标和人物实际站位。
- 正式世界相机使用 3 倍缩放，而 Demo 只把人物放大 3 倍；因此将正式 `MapTeleportEffect` 场景实例缩小为三分之一，抵消相机对特效的额外放大并恢复 Demo 中的角色比例。
- `world_scene.tscn` 预置正式 `MapTeleportEffect` 实例，`world_controller.gd` 在地图二次点击传送时将它对齐人物脚底并挂入同一角色排序层，使后半传送阵继续被人物精灵正确遮挡。
- 正式时序调整为“点击并发送权威请求 → 播放聚能特效 → `vanish_started` 隐藏人物 → 启动原有黑屏 → 黑屏中点应用权威快照”；同地图与跨地图快速传送共用该流程，普通传送门不变。
- 服务端响应早于演出时只缓存快照，不提前切换画面；拒绝、加载失败或 15 秒快照超时时会停止特效、恢复人物、清理视觉锁并允许再次传送。
- 将 `CrossFlash` 从 `160×160px` 缩小至 `120×120px`，Shader 的十字横纵光臂改用二维高斯衰减，移除明显的矩形端点和棱角，同时保留向中心收束并淡出的后半段节奏。
- 可复用特效新增 `vanish_started` 信号和预置 `CrossFlash` 节点：进度达到 `0.70` 后通知外部隐藏传送对象，同时让原有光柱、传送阵、聚能闪光和粒子快速退出，再由 Shader 绘制逐渐收束并完全淡出的蓝白十字光点。
- Demo 监听消失信号控制人物显隐，不把人物节点耦合进可复用特效；播放结束时人物与特效均不可见，下一轮重播前恢复人物，点击、触摸和键盘重播流程保持一致。
- 将底部传送阵拆为 `GroundRingBack` 与 `GroundRingFront` 两个预置场景节点；Shader 通过 `ring_layer` 分别输出后半环和前半环，后半环使用负层级接受人物精灵遮挡，前半环保留在人物脚前。
- 将底部传送阵 Shader 的最大环半径从 `1.04` 缩小为 `0.625`，最小环半径继续保持 `0.50`，因此由大至小收束时的最大直径为最小直径的 `1.25` 倍。
- 根据运行截图微调 `space_time_teleport_effect.tscn`：两层光柱在上一版 `420px` 基础上继续降低 `64px`，最终统一为 `356px` 高；恢复外层与核心原有宽度、透明度和绘制层级，使蓝色光柱保持清晰，聚能闪光继续覆盖人物形象中心。
- 新增 `space_time_teleport_effect.tscn` 可复用表现组件，节点树预置外层光柱、内层核心、地面能量环、中心聚能闪光以及两组 `GPUParticles2D`，运行时不动态创建节点。
- `space_time_teleport.gdshader` 通过统一 `progress` 参数驱动光柱宽度、浓度、流动条纹、环形收束和最终淡出；Demo 背景也复用同一 Shader 的独立模式。
- `space_time_teleport_effect.gd` 使用单一 Tween 同步全部 Shader 与粒子透明度，重复播放会终止旧时间轴并重启粒子，场景切换中断时可调用 `stop_effect()` 清理残留。
- `space_time_teleport_demo.tscn` 复用现有人物场景，在 `780×1440` 竖屏构图中自动循环展示；桌面点击/空格/确认键和移动端触摸均可重播。
- 正式接入只调整客户端传送演出时序，没有修改服务端权威传送、WebSocket 协议、地图坐标或数据库持久化契约。

## 2026-07-28 背包与地图关闭按钮统一

- 背包标题栏原独立旧图集按钮替换为 `panel_close_button.tscn` 实例，并删除仅供旧按钮使用的图集裁切与样式子资源。
- 地图标题栏原文字“×”按钮替换为同一通用关闭按钮实例，保留 `%CloseButton` 唯一节点引用。
- 背包继续通过原节点路径 `Title/HBoxContainer/Button` 绑定关闭事件，地图继续通过 `%CloseButton` 绑定关闭事件；业务逻辑与信号时序不变。

## 2026-07-28 通用面板关闭按钮换肤

- `panel_close_button.tscn` 移除旧 `UI.png` 图集裁切样式，改为直接引用 `BtnExitOpacity.png` 和 `BtnExitNoOpacity.png`。
- 普通态与禁用态使用半透明素材，悬停态与按下态使用不透明素材；焦点态保持空样式，不叠加第三种视觉。
- 按钮最小和最大尺寸统一为素材原生 `55×55`，所有引用该通用组件的面板自动获得新样式，关闭信号与业务逻辑不变。

## 2026-07-28 世界地图来源地区自动选中

- 地图面板记录当前正在查看的地区按钮，点击“世界地图”返回时自动恢复来源地区为当前选中项。
- 自动恢复内容包括地区名称、按钮焦点和共享四帧选中动画位置；不依赖鼠标悬停状态。
- 来源地区已经处于选中状态，因此再次点击该节点会按既有二次点击规则直接重新进入对应地区地图。

## 2026-07-28 世界地图二次点击与悬停移除

- 闪光镇地图节点和世界地图地区节点的 hover 样式统一替换为空样式，鼠标经过不再显示额外边框或背景。
- 世界地图打开时不预选地区；首次点击节点只更新地区名称并把四帧动画移动到该节点，第二次点击同一节点才进入对应地区地图。
- 首次点击后改点其他地区时只切换选中动画，不会误进入上一个或新点击的地区；地区内地图原有“首次选中、再次传送”逻辑保持不变。

## 2026-07-28 地图节点四帧选中动画

- 使用 `client/asset/分类/ui/选中状态.png` 第一行的四个 `32×32` 帧创建循环 `SpriteFrames`，动画速度为 8 FPS。
- `map_teleport_panel.tscn` 预置一个共享 `AnimatedSprite2D`，运行时只移动到当前节点中心，不为每个热点动态创建重复动画节点。
- 闪光镇地图节点继续由唯一选中索引驱动动画；世界地图地区节点通过焦点变化驱动同一动画，兼容鼠标、键盘和手柄。
- 旧静态按下态和焦点态边框替换为空样式，热点坐标、地区映射和服务端权威传送链路保持不变。

## 2026-07-28 世界地图地区映射修正

- 按用户提供的世界地图标注，闪光镇入口从左上皇冠节点改到左侧绿色节点。
- 精灵迷宫入口从左侧绿色节点改到右下绿色节点，中央 boss 仍作为闪光平原入口。
- 三个热点继续使用 `52×52` 无圆角正方形，并按统一 3 倍地图倍率覆盖对应图标中心。

## 2026-07-28 世界地图节点位置校准

- 通过 Godot MCP 实际运行截图确认，旧世界地图热点仍使用临时换算坐标，选中框没有覆盖图标中心。
- 三个地区热点改为直接使用 `世界地图.png` 的原始像素节点，并统一应用地图 3 倍显示倍率与居中边距。
- 原始坐标统一乘以地图 3 倍显示倍率，并叠加世界地图在 `592×640` 绘制框中的居中边距；选中框继续保持 `52×52` 无圆角正方形。

## 2026-07-28 地图节点选中框统一

- 闪光镇地图节点与世界地图地区节点的选中框统一为 `52×52` 正方形，按下态和键盘焦点态均移除圆角。
- 世界地图三个地区热点围绕现有图标中心从 `72×72` 收敛为 `52×52`，只调整边界，不改变地区对应关系。
- 闪光镇节点原本已经是 `52×52`，继续保持中心位置、人物图标定位和服务端权威传送 ID 不变。

## 2026-07-28 地图标志尺寸统一

- 世界地图、闪光镇、闪光平原和精灵迷宫不再共用一个固定正方形纹理尺寸，避免原始宽高不同的图片被分别拉伸。
- 四张地图各自在场景中配置独立 `TextureRect`，统一使用原始像素 3 倍尺寸：`528×528`、`384×384`、`432×480`、`576×624`。
- 地图切换只改变预置纹理节点的可见性，不在脚本中修改 UI 尺寸；所有地图保持原始宽高比，因此同源地图标志使用一致倍率显示。
- 闪光镇和世界地图点击节点按统一倍率重新对齐，服务端传送场景 ID 与权威坐标链路没有变化。

## 2026-07-28 地图绘制区域整体放大

- 地图纹理显示尺寸由 `256×256` 放大为 `512×512`，外框由 `272×272` 放大为 `544×544`，不使用运行时 `scale`。
- 闪光镇地图节点、世界地图地区节点和人物当前位置图标的中心坐标同步乘以 2；点击区与图标保持上一轮放大后的尺寸，确保覆盖对应图片节点且不会产生大面积热区重叠。
- 地图仍在原 `map_teleport_panel.tscn` 中切换，服务端权威快速传送协议和目标场景配置没有变化。

## 2026-07-28 地图节点与人物图标放大

- 闪光镇 7 个地图节点点击区由 `26×26` 放大为 `52×52`，世界地图 3 个地区节点点击区由 `36×36` 放大为 `72×72`。
- 所有节点均围绕原中心点扩展，节点对应的地图位置、地区入口和服务端 `target_scene_id` 不变。
- 人物当前位置图标由约 `24×25` 放大为 `48×50`，同步调整相对节点偏移，继续显示在当前地图节点左下方。

## 2026-07-28 世界地图地区导航

- `map_teleport_panel.tscn` 删除底部“上一个 / 下一个”按钮，替换为单个“世界地图”按钮，并保持运行时根节点在场景资源中默认隐藏。
- 世界地图、闪光镇、闪光平原和精灵迷宫共用同一地图纹理节点和同一个地图面板场景；脚本只切换预置节点的纹理与可见性，不动态创建 UI。
- 世界地图预置三个地区热点，点击后进入对应地区地图视图；当前仅闪光镇存在客户端场景 ID 和服务端快速传送配置，因此其他地区只展示地图，不提交无效传送请求。
- 闪光镇原有标点二次点击、人物当前位置图标以及 `MOVE_INTENT_REQ(map_teleport=true) -> WORLD_RESYNC_PUSH` 权威链路保持不变。

## 2026-07-28 通用信息弹窗统一

- 删除旧通用信息弹窗场景与脚本，服务端错误、人物/宠物升级和装备修复成功提示统一由 `client/scenes/ui/common/confirm_prompt_popup.tscn` 承载。
- 错误正文继续使用服务端原文并按 BBCode 富文本渲染，客户端不改写服务端错误含义。
- 升级弹窗继续逐项等待玩家关闭后再展示下一项；装备修复成功提示继续保留全局 notice 兜底。
- 关闭交互统一复用确认弹窗链路：确定按钮、右上角关闭、回车、主键盘 `5` 和小键盘 `5` 均能关闭，关闭后由 `ModalPopupLayer` 延迟一帧恢复世界输入。

## 2026-07-28 移动端选项弹窗尺寸调整

- `option_list_panel.tscn` 由旧的约 `240px` 宽小面板改为适配 `780×1440` 视口的 `640×760` 固定弹窗，并显式保持场景资源默认隐藏。
- 头像增大为 `72×72`，标题字号调整为 `28`，选项字号调整为 `26`，同步增大行间距、右上角关闭热区与底部关闭按钮。
- 移除 `main.gd` 中 NPC List、挂机目标、PVP 目标和 NPC 菜单的 `panel_min_width` 运行时覆盖；面板尺寸只在 Godot 场景节点树中维护。

## 2026-07-28 挂机目标选择修复

- 根因是挂机按钮从 `GameState.nearby_entities` 收集怪物，但暗雷怪物不是地图实体，列表恒为空；旧流程只写入已关闭的 HUD 日志，因此表现为点击无反应。
- 服务端暗雷配置新增 `targets`，按地图 `spawn_monster_ids` 去重后读取已启用怪物模板的 `monster_id / monster_name / skin_id`，不下发战斗数值。
- 客户端复用 `OptionListPanel` 展示权威挂机目标，通用列表支持按 `skin_id` 解析待机首帧；配置不完整时显示可见提示。
- 实际开战继续由服务端按后台 `formations` 权重权威选怪，客户端选择只用于挂机目标展示，不改变战斗结算。

## 2026-07-28 人物移动卡顿修复

- 根因是人物 `90` 像素/秒的连续位移在 60Hz 物理帧中被强制取整，小数余量补偿会形成 `2px、1px` 交替步长；世界相机放大 3 倍后表现为明显的不均匀滚动。
- 本地 `CharacterBody2D` 现在保留 `move_and_slide()` 计算出的连续浮点坐标，速度不变，不再用跨帧余量反复改写角色物理位置。
- 像素素材继续复用项目级像素吸附与 nearest 过滤；服务端权威坐标、100ms 表现上报、远端玩家插值和宠物跟随参数均未改变。

## 2026-07-28 世界地图人物当前位置图标

- 复用 `map_teleport_panel.tscn` 中预先放置的“人物图标”，不在脚本中创建或复制 UI 节点。
- 地图面板直接读取服务端世界快照同步到 `GameState.scene_snapshot.scene_id` 的权威当前场景，并通过各标点已有的 `target_scene_id` 查找对应节点，不维护第二份场景映射。
- 人物图标会显示在当前场景标点左下角；世界快照切换场景时立即更新，打开地图时也会再次校准。
- 当前场景没有对应地图标点时隐藏人物图标；图标忽略鼠标输入，不影响地图标点选择和再次点击传送。
- 面板尺寸调整为 `680×1000` 后，将地图纹理与 `272×272` 标点层分别居中到新地图容器，保留原标点坐标及人物图标相对标点的定位关系。
- 模糊背景改为与边框并列铺满 `PanelStack`，不再作为 `PanelContainer` 的内容子节点，避免被边框样式的 `8px` 内容边距向内收缩。

## 2026-07-27 地图后台数据并发与 NPC 菜单批量加载

- 地图切换完成条件收敛为服务端权威 `WORLD_RESYNC_PUSH` 和本地地图资源挂载；`scene_loaded` 不再等待 NPC 菜单，地图遮罩与场景移动锁可以立即结束。
- 新增 `2047/2048 NPC_MENU_BATCH_REQ/RESP`，客户端每次进入地图只发送一次后台请求，服务端一次读取场景实体、任务摘要和全部 NPC 静态菜单后统一返回。
- 批量菜单请求不显示 loading、不设置运行时输入锁；宠物、背包、任务、在线实体、任务更新和场景剧情允许在玩家进入地图后并发返回。
- 菜单缓存继续绑定 `scene_id + entity_id`；切图后的旧批量响应会被丢弃，同地图剧情新增 NPC 时重新批量刷新并合并缓存。
- 旧 `NPC_MENU_REQ/RESP` 保持兼容，仅用于任务领取或交付后需要立即重新打开单个 NPC 菜单的现有流程。

## 2026-07-27 地图传送后移动锁修复

- 根因是地图转场已有角色场景锁，NPC 菜单预加载又叠加了运行时输入锁；权威快照应用后只释放场景锁，残留的 `_runtime_input_locked` 会继续拦截移动和点击寻路。
- NPC 菜单预加载现在只在非地图转场流程中主动锁定世界输入；传送期间复用现有场景切换锁，避免两个不同职责的锁生命周期交叉。
- 首次登录和同地图剧情解锁 NPC 时的菜单预加载仍保留输入保护，服务端传送、菜单权威计算和地图坐标逻辑均未改变。

## 2026-07-27 闪光镇传送地图等比缩小

- `map_teleport_panel.tscn` 的地图纹理由 `512×512` 等比缩小为 `256×256`，地图外框由 `544×544` 调整为 `272×272`。
- 7 个标点按钮由 `52×52` 缩小为 `26×26`，每个标点的横纵坐标同步除以 2，保持与地图位置严格对齐。
- 本次直接修改场景节点尺寸和坐标，没有使用 `scale`，按钮点击区域与视觉尺寸保持一致；传送场景 ID、开放状态和交互逻辑均未改变。

## 2026-07-27 地图 NPC 菜单提前加载

- 地图场景挂载完成后，根据服务端权威世界快照筛选当前场景 NPC，并在开放玩家交互前串行请求每个 NPC 的动态菜单。
- 菜单响应按“场景 ID + NPC 实体 ID”保存在当前地图缓存中；进入碰撞区只打开缓存数据，不再临时请求服务端。
- 切图会立即清空旧地图菜单并建立新的预加载代次，旧异步流程无法把上一地图响应写入当前地图。
- 预加载期间复用现有全屏 loading 并锁定世界输入；未成功预加载的 NPC 不会在碰撞时偷偷走备用请求，避免重新引入临时加载行为。
- 同地图剧情解锁 NPC 时会监听最新世界快照并只补载缺失实体；失败或超时会写入已尝试占位，避免无限重试和输入永久锁定。

## 2026-07-27 人物信息独立查询修复

- 修复 `_wait_player_panel_open_request()` 使用无限循环内直接返回时，Godot 静态分析仍判定存在无返回分支的问题；现在匹配回包后跳出循环，并在函数末尾显式返回结果。
- 排查服务端日志确认玩家 `10001` 数据仍存在；原人物面板把完整进入世界、背包列表和钱包三条请求绑定为一个成功条件，任一子请求失败都会表现为“人物信息获取失败”。
- 新增 `PLAYER_PROFILE_REQ/RESP（2065/2066）`，服务端只读取当前会话人物的权威属性并返回 `PlayerSnapshot`，不查询背包物品、钱包、宠物、地图实体、任务或剧情触发器。
- 客户端人物控制器先合并服务端人物快照，再结束统一 loading；人物面板只等待这一条请求，不再隐式发送背包和钱包请求。
- 人物面板“背包数量”不再读取可能过期的本地背包缓存，改为提示玩家打开背包查看；背包内容继续只在点击背包入口后加载。
- 新增服务端协议路由测试，验证独立人物资料响应包含玩家身份、血量和形象字段。

## 2026-07-26 切图轻量查询与快照优先发送

- 根因确认：地图资源本地加载只需几十毫秒，实际黑屏主要耗时来自位置落库后串行查询在线玩家完整档案、跟随宠物和两轮任务条件；远程数据库下会让 `WORLD_RESYNC_PUSH` 晚约 2～3 秒。
- 跨地图成功后改为“位置落库 -> 更新在线场景索引 -> 基础 `WORLD_RESYNC_PUSH` -> 在线实体增量 -> 任务更新 -> 场景触发”。客户端不再等待多人资料和任务查询才能显示目标地图。
- 玩家和跟随宠物新增服务端批量轻量摘要，世界展示只读取名字、等级、经验、血量、精力、形象和位置；不刷新完整战斗快照，不读取技能、装备或背包。
- `ENTER_SCENE` 任务推进新增轻量场景事实读取，只查询玩家基础状态、宠物等级和剧情标记，明确跳过背包物品表；物品持有条件仍在任务列表和主动接取时通过完整事实校验。
- 背包请求链路没有改变，切图不预取、不缓存背包内容，玩家点击背包时继续使用既有服务端接口获取最新持久化数据。
- 新增测试锁定世界快照优先顺序、已有快照玩家的跟随宠物补齐、玩家/宠物轻量字段映射，以及场景任务事件不调用完整背包事实查询。
- 已通过 `go test ./server/...`、Godot 4.7 主场景无头启动和 `git diff --check`；本次没有数据库结构变化，无需新增迁移。

## 2026-07-26 切图移动回包竞态修复

- 进一步确认切图超时的上游原因是普通移动每 100ms 持续发送、服务端又按连接串行读取并访问远程数据库，导致切图请求排在数十个旧移动请求之后；单纯延长快照等待时间无法解决该队头阻塞。
- 客户端普通移动同步增加单请求在途背压：上一个移动序号未收到回包时不再发送中间位置，回包后下一帧直接补发角色最新坐标、朝向和移动状态，既限制服务端队列长度，也避免重放已经过时的轨迹点。
- 发起传送时至多只会有一个普通移动请求排在前面；该普通回包会解除移动背压，但仍不会改变当前切图状态，随后匹配切图 `move_seq` 的回包和 `WORLD_RESYNC_PUSH` 可正常完成转场。
- 客户端为每次场景切换保存专属 `move_seq`，只有相同序号的 `MOVE_INTENT_RESP` 才能接受或拒绝当前转场。
- 切图前延迟到达的普通 `position synchronized` 回包会被记录为 stale 并忽略，不再提前清空 `pending_target_scene_id`，从而避免客户端继续以上一场景坐标上报并触发连续 `scene mismatch`。
- 服务端已接受切图后客户端继续等待权威 `WORLD_RESYNC_PUSH`；考虑当前远程 PostgreSQL 场景快照耗时，等待窗口由 5 秒调整为 15 秒。
- 本次不改变服务端传送规则、数据库结构或协议字段，仅修正客户端对既有 `move_seq` 的关联消费。

## 2026-07-26 日志关闭与转场超时审查修复

本次按审查结果修复四项回归，不恢复高频协议日志，也不改变服务端权威玩法契约：
- `WorldController` 增加显式切图中止入口；主场景等待 `WORLD_RESYNC_PUSH` 超时后会清空目标场景、传送门、朝向与待应用出生点，取消视觉锁并解除玩家移动锁
- 运行时普通业务提示不再进入空日志函数，改为复用 `main.tscn` 既有 `BottomPanel/LogOutput` 展示单条提示；新提示通过显示代次覆盖旧提示，3 秒后自动隐藏，不保留历史列表
- 登录页日志仍保持关闭，但最新登录状态、参数校验和服务端错误会保存在现有 `HintLabel`，界面刷新不会再用演示账号说明覆盖错误文案
- Web 与 Android 发布预设统一排除 `addons/godot_ai/**`；实际 Web PCK 导出确认不再包含该目录，编辑器插件和运行时游戏代码保持隔离
- 闪光镇东路脚本恢复为原有四空格缩进，本次不保留无业务含义的整文件格式化差异
- 已通过 `go test ./server/internal/transport/ws`、Godot 4.7 主场景无头启动、Web PCK 实际导出和 `git diff --check`

## 2026-07-24 地图切换双端诊断日志

本次只恢复地图切换所需的定向控制台日志，没有重新启用战斗、背包或 HUD 刷屏日志：
- 客户端统一使用 `[SceneTransition][Client][World]` 和 `[SceneTransition][Client][Main]` 前缀，记录传送门请求参数与序列号、`MOVE_INTENT_RESP`、`WORLD_RESYNC_PUSH`、视觉延迟状态、地图资源路径、实际装载耗时、`scene_loaded`、遮罩中点和等待超时
- `project.godot` 仅恢复标准输出用于查看定向 `print()`；错误输出仍保持关闭，避免资源告警重新刷屏
- 服务端统一使用 `[SceneTransition][Server]` 前缀，记录玩家请求场景与数据库权威场景、传送门判定结果、目标出生坐标、位置持久化、响应与世界重同步发送结果
- 日志不包含令牌、密码或完整玩家数据，只输出定位切图卡点所需的玩家 ID、场景 ID、传送门 ID、坐标、请求序列和错误摘要

## 2026-07-24 客户端日志关闭与场景性能修复

本次关闭客户端日志并修复移动、切图链路中的主线程热点和异常：
- Godot 项目统一关闭标准输出与错误输出；登录页和运行时 HUD 日志入口改为空实现，登录状态、服务端错误弹窗和正常业务交互继续保留
- 移除战斗协议 JSON、战斗演出帧、Web 画布尺寸和奖励弹窗点击调试打印，避免调试构建执行无用字符串格式化与 JSON 序列化
- 远端玩家移动包继续更新人物和宠物表现，但不再每 100ms 广播全局世界快照变化，避免连带刷新玩家 HUD、地图全部 NPC 和装备预览
- 本地玩家继续按原频率发送高精度移动表现；仅在整数场景格变化时更新本地玩家快照与坐标 HUD，减少移动期间字典复制和文字布局
- 主场景所有逐帧等待统一先保存并校验 `SceneTree`；退出主场景时取消地图转场代次，避免异步恢复后访问空树
- 地图转场等待服务端快照最多 5 秒，失败或超时都会淡出黑屏并恢复输入；无头测量显示现有地图资源装载约 14–59ms、导航建立约 2–4ms，资源本身不是长时间黑屏原因

## 2026-07-24 地图提示仅首次进入展示

本次收紧服务端权威场景提示的消费时机：
- 仅包含提示文案、没有动画和服务端副作用的地图触发器，在首次进入地图准备下发时就写入玩家持久化剧情标记
- 同一玩家后续重新登录、切回该地图或重复同步世界时，数据库查询会过滤已经记录的提示，不再重复弹出
- 带剧情动画、任务接取或剧情标记副作用的触发器保持原有完成 Ack 时序，避免提示尚未完成就提前推进任务或解锁 NPC
- 增加分类测试，覆盖纯地图提示、动画提示、任务副作用、剧情标记副作用和空触发器

## 2026-07-24 移动同步日志与时光小屋出口落点

本次完成两项小范围调整：
- `CommandIds.should_log_result()` 将高频 `MOVE_INTENT_RESP` 加入静默列表，不再向控制台和 HUD 日志重复输出正常的 `position synchronized`，消息路由和请求完成处理保持不变
- 时光小屋 `portal_id=7001` 返回东路的出生坐标统一为 `(9, 5)`，客户端脚本、地图场景覆盖值、服务端权威配置、测试桩和传送测试使用同一坐标

## 2026-07-23 远端玩家移动连续性修复

本次修复观察端人物在每个 100ms 移动表现包之间反复“追上目标后停等”产生的卡顿：
- 远端人物收到 `moving=true` 后，按服务端归一化朝向和人物配置速度连续推进表现目标
- 新权威表现包到达后直接替换预测基点，既保持连续移动，也会使用现有追赶速度逐步吸收网络误差
- 单包预测最长 180ms，网络断包后自动停住，不会让远端人物无限走离服务端权威位置
- 收到 `moving=false` 时立即停止外推，并平滑回到服务端最终高精度坐标后播放待机动画
- 旧协议没有明确 `facing`、`moving` 时关闭预测，继续沿用整数目标点追赶逻辑，避免兼容客户端在停止后多走

## 2026-07-23 多人移动位置、朝向与宠物表现对齐

本次修复同一玩家在自身客户端与其他玩家客户端中的实时表现偏差，同时保留服务端整数坐标权威和数据库持久化结构：
- 客户端移动期间以 100ms 最小间隔上报千分之一场景格定点坐标，并在跨格、转向、开始或停止移动时立即补发
- `target_pos` 继续作为数据库整数落点；服务端只在整数格变化时持久化，并把 `precise_pos` 限制在权威格周围半格后再广播
- 协议明确同步四方向 `facing` 和 `moving`，观察端不再根据延迟目标坐标猜测人物朝向和动画状态
- 远端人物继续平滑追赶服务端表现坐标，收到停止状态后会先补完剩余短距离再进入待机，避免静止滑动
- 本地和远端宠物在人物转向时都会重置路径记录轴；远端宠物按高精度轨迹累计完整 24px 步距，并使用与人物配置一致的移动速度
- 新字段均为兼容性扩展；旧客户端缺少高精度位置或移动状态时，服务端仍回退到整数格位置和位移方向

## 2026-07-23 点击寻路减少转弯

本次优化客户端四方向点击寻路的路径形态，不改变服务端权威坐标同步和地图碰撞来源：
- AStarGrid2D 继续负责确认目标可达并生成安全基础路径
- 客户端从当前位置尝试连接尽可能靠后的基础路径节点，水平和垂直直线路段必须逐格通过现有碰撞网格校验
- 无障碍区域优先生成长直线或只包含一个拐角的路径；直线路段受阻时才保留必要的 AStar 绕障节点
- 两条单拐角路径都可用时，优先选择与当前移动方向衔接转弯更少的方案；转弯数相同时先走距离更长的轴
- 点击移动和剧情动作场景路径复用同一入口，避免两套寻路表现不一致

## 2026-07-22 同场景远端玩家宠物展示

本次补齐多人同屏时其他玩家首只出战宠物的权威同步与客户端表现：
- 服务端从玩家持久化编队读取首只宠物，将 `pet_uid`、`pet_id`、等级、生命和数据库宠物形象 `skin_id` 写入玩家实体的 `following_pet`
- `ENTER_WORLD_RESP.nearby_entities` 和 `ENTITY_ENTER_PUSH.entity` 共用同一玩家实体组装逻辑，避免后加入与先加入玩家看到的数据不一致
- 玩家在场景内更换编队后，服务端复用实体进入推送向同场景其他玩家刷新跟随宠物；无编队时省略 `following_pet`，客户端同步移除旧宠物
- 客户端复用 `WorldPetFollower` 创建远端宠物节点，按远端玩家每次权威坐标更新记录跟随路径，并保持独立 Y-Sort 排序
- 增加后端测试覆盖进入世界快照中的远端宠物，以及编队变更后的同场景刷新广播
## 2026-07-22 地图最低进入等级配置

本次在现有服务端权威切图链路上增加数据库可配置的地图等级限制：
- 新增迁移 `backend/server/migrations/109_world_scene_required_level.sql`，为 `world_scene_definition` 增加 `required_level`，限制范围为 1~100，并将全部现有地图初始化为 1 级
- `backend/server/internal/data/postgres/world_repo.go` 先校验地图拓扑与传送门合法性，再读取数据库发布状态和最低进入等级，并使用 `player.Profile.Level` 校验；等级不足或地图停用时均不会写入玩家位置
- Admin API 扩展 `GET /api/admin/npcs/scenes` 返回 `required_level`，新增 `PUT /api/admin/npcs/scenes/{scene_id}` 更新最低进入等级；沿用 `npcs:view` / `npcs:edit` 权限
- `admin/src/pages/npcs/NPCConfigPage.tsx` 在原地图 NPC 页面上方增加地图进入等级配置表，使用固定 Modal 修改等级，不新增并行后台模块
- `client/scripts/bootstrap/main.gd` 在切图失败时复用全局 notice 展示服务端 `reason`；等级不足统一提示“前面的路以后再来探索吧”，客户端不参与等级判断
- 已执行受影响 Go 包测试、地图等级边界测试、Admin API 读写测试与后台生产构建；客户端完成脚本引用、提示信号与缩进静态检查（当前环境未安装 Godot，未执行引擎解析）

## 2026-07-22 Web 本地调试尺寸统一

本次修复 Web 本地调试页与 Godot 原生运行项目的画布和内部渲染尺寸不一致：
- `client/project.godot` 的逻辑视口和窗口覆盖统一为 `780x1440`，保证原生运行与 Web 导出使用同一内部渲染缓冲区，不再出现 Canvas 外框比例正确但游戏内容仍按其他尺寸渲染的问题
- `client/autoload/web_runtime_canvas.gd` 移除 Debug Web 铺满浏览器窗口的特殊分支；调试与正式 Web 都固定使用 `13:24` 设计比例，浏览器空间不足时只做等比缩小，空间充足时不超过 `780x1440`
- Godot 默认导出页的 Canvas 直接挂在 `body` 下，运行时现在会保留 `body` 的全屏居中尺寸，只在 Canvas 存在独立包装节点时同步包装尺寸，避免把整个页面缩到左侧游戏宽度后在右侧留下黑色区域
- Godot 编辑器可能继续复用或重新生成带旧 `1080x1440` 规则的临时 HTML；运行时对 Canvas、加载状态层和独立包装节点使用内联 `!important` 覆盖宽高、边界与纵横比，避免旧壳层的 `!important` 阻止 `13:24` 比例生效
- `client/export_presets.cfg` 把旧的 `1080x1440`（3:4）CSS 修正为 `780x1440`（13:24），并同步约束 Canvas 与加载状态层
- 世界 `SubViewport` 继续沿用现有 `780x1440` Web 固定渲染口径；本次不修改客户端与服务端协议、玩法计算或持久化链路
- Web 导出继续使用 `canvasResizePolicy: 1`，让 Godot 在启动和引擎窗口变化时同步内部绘图缓冲区；本次不使用铺满浏览器窗口的策略

## 2026-07-21 任务开启条件配置

本次把后台单一的“前置任务”升级为服务端权威的任务开启条件：
- 复用 `quest_template.accept_conditions` JSONB 字段，支持已完成任务、人物等级、人物最终属性、当前地图、物品持有数量、宠物等级、剧情标记和服务端时间段
- 所有条件按 AND 关系校验，数值比较支持大于等于、等于和小于等于；人物属性读取最终战斗快照，物品、宠物、地图与剧情标记均读取数据库持久化数据
- 任务列表状态、自动接取与主动接取复用同一套条件判断，客户端只展示服务端返回状态
- 后台新增结构化条件编辑器，任务、地图、物品和宠物引用均使用真实数据下拉选择，时间按运营本地时间录入并转换为 RFC3339 保存
- 历史 `pre_quest_ids` 与 `min_player_level` 继续兼容；打开旧任务编辑时会合并显示到新条件列表，保存后自然迁移到 `accept_conditions`

## 2026-07-21 任务完成提示文案配置

本次补齐任务提交成功后的自定义提示链路：
- 新增迁移 `backend/server/migrations/107_quest_completion_prompt_text.sql`，为 `quest_template` 增加 `completion_prompt_text`，用于持久化任务完成提示文案并支持 Godot BBCode 富文本
- 后端任务模板、后台详情/创建/编辑和运行时任务摘要已同步读写 `completion_prompt_text`；`QUEST_SUBMIT_RESP` 顶层也会下发该字段，客户端无需硬编码文案
- 后台任务模板基础信息新增“完成提示文案”富文本编辑入口；为空时不展示额外提示，保存后与其他任务配置一起写入服务端权威接口
- 客户端任务提交结算现在会先展示“任务完成”信息面板，关闭后再继续玩家升级和奖励弹窗；无奖励但有完成提示时也会弹出提示
- `backend/docs/quest-protocol.md` 已补充 `completion_prompt_text` 字段说明，明确其为空时跳过、非空时支持 RichTextLabel BBCode

## 2026-07-21 后台富文本编辑卡片化

本次调整后台统一富文本入口的展示与编辑方式：
- `admin/src/components/RichTextEditor.tsx` 默认合并为单个客户端效果卡片，不再在页面上同时铺开“文本原文”和“刷色编辑”两块区域
- 富文本卡片右上角新增“编辑”按钮；点击后打开固定宽度弹窗，弹窗内上方输入纯文本、下方保留客户端效果预览与刷色笔刷
- 弹窗内修改先写入本地草稿，点击“保存”后再回写 Ant Design Form 字段；取消关闭不会污染当前表单值
- 物品、宠物和玩家名占位符的预览逻辑继续复用现有 `ItemMentionPreview`，服务端保存的 BBCode 契约不变

## 2026-07-13 世界剧情路径与角色动作

本次在现有 NPC 结构化剧情 `line/action/choice/end` 链路上补齐真实世界玩家演出：
- `player.gd` 新增剧情控制模式、自动路径自然完成信号和角色剧情动画接口；剧情期间方向键失效，但脚本路径仍通过 `CharacterBody2D.move_and_slide()` 正常执行
- `world_controller.gd` 新增世界玩家上下文和场景坐标路径构建接口，多段途经点统一经过当前地图 AStar 导航与碰撞可达性检查
- `world_player_cinematic.tscn` 提供 Inspector 配置的通用动作场景，支持路径点、结束朝向、动画名、指定帧、帧率和保持时长
- `CinematicPlayer` 会在场景入树前注入世界控制器，并以单活动实例和播放代次隔离新旧演出；动作结束或失败都会恢复玩家控制，强制关闭时只取消演出，正常完成才沿现有 `NPC_DIALOGUE_NEXT_REQ` 权威推进下一节点
- `CinematicRegistry` 改为按动画 Key 自动解析同名 `.tscn`；新增剧情只需在 `client/scenes/cinematics/` 根目录保存继承场景，并在后台 action 节点填写同名 Key
- 删除 `CharacterVisual` 中未使用的自身场景预加载，避免 Godot 加载角色场景时产生循环资源解析
- `NPCDialoguePanel` 增加本地过场对白模式，复用原有场景、立绘、富文本和打字机效果，但点击继续只唤醒当前客户端过场，不发送服务端剧情请求
- `WorldPlayerCinematic` 增加固定 Tween 像素移动、本地对白等待和完整过场结束接口；每个动画 Key 的派生脚本可以写死移动、动画及对白顺序，服务端仅负责调用 Key
- `初见桃子.gd` 按脚本注释初始化三个角色：桃子播放左向待机，七色羽水平镜像后播放左向待机，场景内玩家锁定输入并保持左向待机；注册表同步支持中文剧情动画目录
- `InteractiveNPCBase` 使用可选节点方式获取交互区域，世界 NPC 继续正常绑定进入/离开信号，固定过场中的纯展示 NPC 则安全跳过交互逻辑
- 通用过场基类新增地图原点归一和世界相机参数同步接口；“初见桃子”先按 TileMap 可用矩形把地图左上角移动到原点，再重算相机限制并承接世界场景的缩放、偏移和位置，保持地图坐标与真实客户端一致
- `初见桃子.gd` 新增按文本顺序串行执行的本地对白与 Tween：桃子右移 50px 后左移复位，再上移 18px，最后与七色羽同步右移 75px 并一起转为左向待机；`NPCDialoguePanel` 的本地模式新增回车和数字 5 推进输入
- `初见桃子.tscn` 内置默认隐藏的备用对话面板，脚本根据 `local_dialogue_requested` 是否已有外部连接自动选择正式客户端桥接或 F6 单场景预览路径
- `npc_dialogue_panel.tscn` 将对话主体宽度从原 `320px` 调整为固定 `720px`，通过节点偏移居中匹配世界地图放大后的屏幕宽度，没有使用脚本动态尺寸或节点 scale
- 对话正文使用 `32px` 字号，使 704px 内宽每行约容纳 22 个汉字；说话人字号改为 `24px`、头像改为 `28px`，继续按钮扩到 `96×44px` 以匹配文字尺寸
- `初见桃子.gd` 复用场景内三层冲击波节点，运行时以七色羽终点左移 30px、上移 40px 定位并重置序列帧，固定以 6 FPS 循环播放 2 秒，同时驱动根节点左移 100px 并在结束后隐藏
- 冲击波方法内部增加 1 秒前置等待，确保角色转身与特效出现之间保留明确停顿
- 使用步骤和约束见 `client/scenes/cinematics/README.md`

## 2026-07-10 技能品质按钮边框

本次新增独立技能品质展示链路：
- 迁移 `backend/server/migrations/097_skill_quality.sql` 为技能模板增加 `skill_quality`，支持 `normal/divine/soul/sacred/peerless`，并按历史技能名称中的神技、魂技、圣技、绝世自动回填
- 后台技能模板新增品质选择、列表列、筛选项和详情字段；新技能默认普通品质，非法品质由服务端拒绝
- 宠物技能详情槽位新增 `skill_quality`，服务端从权威技能运行时缓存下发，客户端不根据技能名称自行猜测
- 宠物技能按钮通过场景配置的五套 Theme 切换边框：普通灰、神技绿、魂技蓝、圣技紫、绝世金；普通技能 A 至 F 共用普通品质边框
- 品质只影响客户端边框，不参与 `activation_mode`、`skill_type`、伤害公式、被动效果或技能释放规则

## 2026-07-10 技能表现资源图标接入

本次改动复用技能模板已有的 `skill_visual_id`，完成服务端技能数据到客户端本地图标的最小闭环：
- `client/scripts/feature/battle/resources/skill_visual_config.gd` 新增导出字段 `icon: Texture2D`，每个技能表现 `.tres` 可在 Godot Inspector 独立指定技能图标
- `client/scripts/feature/battle/battle_content_registry.gd` 新增按表现 ID 获取图标的方法，并通过 `ResourceLoader.list_directory()` 自动注册技能资源目录内的全部 `.tres/.res`
- 宠物技能详情协议的每个技能槽新增 `skill_visual_id`；该值来自数据库技能模板，不新增图标字段或迁移，也不向客户端下发 `res://` 路径
- `client/scenes/ui/pet/pet_status_panel.tscn` 预置技能资源注册表节点，宠物状态面板优先使用 `.tres.icon`，并保留旧图标路径字段与默认技能图标回退
- `client/scripts/feature/battle/action_panel.gd` 复用战斗场景已有资源注册表，让服务端技能列表中的 `skill_visual_id` 同样驱动选择按钮图标；未配置图标时保持原有纯文字按钮
- 已补充协议转换单测，锁定技能表现 ID 能完整进入宠物技能槽响应
- 排查实际运行数据后确认技能 `20191` 的旧种子把 `skill_visual_id` 留空；新增 `backend/server/migrations/096_backfill_signature_skill_visual_id.sql` 只对空值回填 `pet_圣技_幻影闪击`，不会覆盖运营后台已有配置
- 技能运行时快照补充数据库 `skill_code`，历史 `skill_visual_id` 为空时临时用技能编码解析客户端资源；宠物详情和战斗技能快照共用该规则，避免两处展示口径不一致
- 后续新增的 `迅捷A.tres` 与 `致命A.tres` 会由技能资源目录自动注册；两个被动技能只复用资源中的 `icon` 做宠物技能栏展示，不要求配置或播放特效
- 技能资源注册已取消手工路径清单；当前导出配置使用 `all_resources`，新增配置只需放入 `client/resources/battle/skill_visuals/` 并填写唯一 `skill_visual_id`，桌面与 Web 导出包都可通过 `ResourceLoader` 枚举

## 2026-07-09 技能释放期命中加成配置

本次补充聚焦把“某个技能释放时临时提高命中”从字段复用收敛成独立配置：
- 新增迁移 `backend/server/migrations/094_skill_hit_bonus.sql`，为 `skill_definition` 增加 `skill_hit_bonus`，用于保存本次技能释放的命中加成
- `backend/server/internal/module/skill/model.go` 与 `backend/server/internal/data/postgres/skill_repo.go` 已同步接入该字段，后台详情、创建、编辑和运行时技能缓存都能完整读写
- `backend/server/internal/module/battle/skill.go`、`skill_resolver.go` 与 `service.go` 已把命中/闪避判定切到 `SkillHitBonus`，不再使用 `SkillCritAdd` 作为技能命中加成，避免暴伤配置影响命中
- 后台技能效果编辑器新增“命中加成”效果类型，默认值为 `40`，保存时自动合并为 `skill_hit_bonus`
- 已补充 `TestCalculateDodgeChanceUsesSkillHitBonus`，锁定“施法者命中 + 技能命中加成”参与本次闪避概率计算，技能释放结束后不污染角色或宠物基础属性

## 2026-07-07 技能被动属性加成显式配置

本次补充聚焦把“系统技能库的永久属性被动”从旧的技能名前缀推断，收敛成后台可显式配置、数据库可持久化的正式字段：
- 新增迁移 `backend/server/migrations/087_skill_passive_attribute_bonus.sql`，为 `skill_definition` 增加 `passive_attr_key`、`passive_attr_mode`、`passive_attr_value` 三个字段，支持把永久被动属性加成直接存入系统技能模板
- `backend/server/internal/module/skill/model.go`、`service.go` 与 `backend/server/internal/data/postgres/skill_repo.go` 已同步接入这组三字段：后台详情、创建、编辑、运行时缓存与 PostgreSQL 仓储映射都能完整读写
- 服务端已新增最小业务校验：只有 `activation_mode=passive` 的被动技能允许配置永久属性加成；属性字段只支持生命/攻击/速度/法力、暴击、物理抗性、技能抗性和全异常抗性；其中百分比模式仅允许用于生命/攻击/速度/法力
- `backend/server/internal/module/pet/passive_attributes.go` 现优先按显式配置折算宠物最终属性面板与编队战斗基础属性；历史技能若还没补显式配置，仍继续走 `强壮/强力/迅捷/魔心/致命/暴伤/厚甲/坚韧/结界` 的旧前缀兼容规则，避免现网旧数据立即失效
- 后台技能效果编辑器新增“被动属性加成”效果类型：`admin/src/types/skillEffectConfig.ts` 与 `admin/src/components/SkillEffectConfigEditor.tsx` 现支持运营直接选择属性字段、加成方式与数值，保存时自动合并到技能编辑接口需要的扁平字段
- 已补充 `backend/server/internal/module/pet/passive_attributes_test.go` 与 `backend/server/internal/module/skill/service_test.go`，覆盖显式属性被动生效、非法模式拦截与合法配置通过
- 建议本地执行迁移后，再通过后台把现有永久属性类被动逐步从“名字约定”迁移为显式字段，后续即可继续清理旧前缀兼容分支

本次继续补齐“后台玩家宠物编辑 -> 游戏运行时生效”的结构化技能槽链路：
- `backend/server/internal/module/pet/model.go` 的后台玩家宠物输入/详情结构新增 `innate_skill_ids` 与 `normal_skill_ids`，后台不再只有一个旧 `skill_ids` 兼容列表
- `backend/server/internal/data/postgres/pet_repo.go` 的后台玩家宠物创建/编辑/详情读取已同步接入 `player_pet.innate_skill_ids` 与 `player_pet.normal_skill_ids` 持久化，保存时会同时回写兼容 `skill_ids`
- `backend/server/internal/module/pet/service.go` 在后台创建/编辑玩家宠物时，会先按正式技能槽生成运行时 battle skill 列表，再做技能引用校验，避免“后台看起来加了技能，但运行时没进正式技能槽”这一类错位
- `backend/server/internal/module/pet/skill_slots.go` 新增 `MergeBattleSkillIDs()`，把结构化技能槽与兼容 `skill_ids` 去重合并；这样老数据、后台临时补技能和正式技能槽都能兼容承接
- `admin/src/pages/pets/PlayerPetListPage.tsx` 已改为“天生技 / 普通技 / 兼容战斗技能预览”三段式编辑，运营直接维护正式技能槽，兼容 `skill_ids` 改为只读预览
- `admin/src/pages/pets/petInstanceFormUtils.ts` 与 `admin/src/types/pet.ts` 已同步把玩家宠物编辑载荷切到结构化技能槽口径，避免前端只改旧字段而不改正式槽位

## 2026-06-10 回合战斗公式层第一轮补全

本次补充聚焦把回合战斗 MVP 中最简化的直接伤害计算，收敛成一个可继续扩展的独立公式层：
- 新增 `backend/server/internal/module/battle/formula.go`，把基础伤害合成、防御减伤、最终伤害、治疗量、暴击上下限等规则从 `service.go` 中拆出
- 当前公式层已支持攻击/防御/速度/目标当前生命/固定伤害合成，支持穿甲、易伤入口与 90% 防御减伤上限
- 当前暴击链路已补上 100% 暴击率上限、2000% 暴击伤害上限，并明确“纯固定伤害默认不暴击”
- 新增 `backend/server/internal/module/battle/formula_test.go`，覆盖基础伤害构成、防御修正、暴击边界、治疗量和固定伤害不暴击等关键规则
- 已执行 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/internal/module/battle ./server/internal/transport/ws` 与 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/...`，当前通过

本次继续补充第二轮公式能力：
- `backend/server/internal/module/battle/formula.go` 继续接入法力系数、有效属性倍率入口与格挡减伤入口
- `backend/server/internal/module/battle/service.go` 的战斗单位运行态新增 `mana` 与属性倍率/平值修正字段，为后续状态系统和被动系统预留承接点
- 新增 `backend/server/migrations/007_add_pet_mana.sql`，为 `player_pet` 增加 `mana` 列，并回填演示账号宠物的默认法力值
- `backend/server/internal/data/postgres/pet_repo.go` 与 `backend/server/internal/teststub/repos.go` 已同步接入 `mana`，避免 PostgreSQL 仓储和测试桩公式口径漂移

本次继续补充第三轮“状态驱动公式”能力：
- `backend/server/internal/module/battle/model.go` 新增 `易伤 / 破甲 / 减速 / 暴击提升` 状态编号
- `backend/server/internal/module/battle/service.go` 已增加 `refreshStatusDerivedModifiers()`，让状态在施加、覆盖和到期时真实回写到速度倍率、暴击率、破甲与易伤等派生战斗属性
- 当前技能表中，`火花冲击` 会附加易伤，`利爪突袭` 会附加减速，`活力治愈` 会附加暴击提升，便于先验证“状态影响公式”的闭环
- `backend/server/internal/module/battle/formula_test.go` 已新增状态驱动公式测试，覆盖状态生效与到期恢复

本次继续补充第四轮状态能力：
- `backend/server/internal/module/battle/model.go` 新增 `诅咒 / 束缚 / 沉睡 / 麻痹 / 混乱` 状态编号
- `backend/server/internal/module/battle/service.go` 已把诅咒加入持续伤害结算，并把束缚/沉睡/麻痹统一接入“跳过行动”判定
- 当前混乱会在服务端执行阶段强制改写目标，随机命中除自身外的任意存活单位，保持权威结算边界
- `backend/server/internal/module/battle/formula_test.go` 已补充诅咒 tick、控制状态阻断与混乱目标改写测试

本次修正持续伤害结算时机：
- `backend/server/internal/module/battle/service.go` 已将流血、诅咒等被动扣血从全体回合末改为“该单位回合结束”结算，跳过行动也会进入自身回合结束结算
- 持续伤害事件仍使用 `EventTypeStatusTick`，不走 `resolveDamageSkill()`，因此不会触发吸血、反击、连击等命中后被动
- `backend/server/internal/module/battle/formula_test.go` 已补充事件顺序测试，锁定高速度单位行动后立即结算自身流血，再进入下一个单位行动

本次继续补充第五轮被动能力：
- `backend/server/internal/module/battle/service.go` 已新增服务端权威的 `闪避 / 吸血 / 反击` 结算分支，并统一收口到 `resolveDamageSkill()`
- 当前被动仍先使用演示型运行时配置验证链路：101 号宠物默认吸血、102 号宠物默认闪避、敌方默认有一定反击概率
- `backend/server/internal/module/battle/formula_test.go` 已新增被动测试，覆盖闪避拦截命中、吸血自我恢复和反击回打
- `client/scripts/feature/battle/battle_scene.gd` 已补充新事件类型文案，避免移动端战斗面板只显示通用占位文本

本次继续补充第六轮被动能力：
- `backend/server/internal/module/battle/service.go` 已新增服务端权威的 `连击 / 复活 / 控制免疫` 结算分支
- 当前 101 号宠物默认有连击概率，102 号宠物默认具备一次复活与控制免疫，用于验证完整被动主链
- 持续伤害致死现在也会先尝试走复活分支，避免复活只覆盖直伤而漏掉状态结算
- `backend/server/internal/module/battle/formula_test.go` 已补充连击额外攻击、复活打断死亡与控制免疫拦截控制状态测试

## 2026-06-10 回合战斗开发任务清单入库

本次补充聚焦把当前回合战斗的后续开发计划沉淀进仓库，方便重启会话后继续推进：
- 新增 `docs/回合战斗开发任务清单.md`，整理了“当前已完成能力、未完成能力、推荐实施顺序、下一步任务拆分”
- 文档明确当前实现仍是“PVE 回合战斗 MVP”，不是原始战斗规格文档中的全量实现
- 文档把后续开发拆成规则补全、服务端权威托管、PVE 奖励闭环、客户端站位/表现层、PVP/组队 PVP 五个大阶段
- 后续继续开发时，建议先阅读 `docs/回合战斗开发文档.md` 与 `docs/回合战斗开发任务清单.md` 再进入具体代码实现

## 2026-05-20 登录接口异常响应容错修复

本次补充聚焦修正“后端未启动时客户端登录直接刷 JSON 解析错误”的问题：
- `client/autoload/http_client.gd` 已为底层 HTTP 失败结果、空响应体和非 JSON 响应补上显式容错处理，不再直接对异常内容调用 `JSON.parse_string()`
- 当前 `HTTPRequest.request_completed` 会先检查底层请求结果，再检查响应体是否为空，最后才尝试解析 JSON 字典
- 当后端未启动或返回非 JSON 内容时，客户端现在会返回统一错误字典，而不是在控制台刷出 JSON 解析报错
- 宠物模板详情读取现在兼容历史技能列表 JSON 脏数据：`backend/server/internal/data/postgres/json_uint32_array.go` 新增了对数字数组与数字字符串数组的统一解析，后台读取 `normal_skill_ids` / `innate_skill_ids` / `skill_ids` 时不会再因 `["101"]` 这类旧数据直接报 500
- 已补充数据库清洗脚本 `backend/server/migrations/085_backfill_pet_definition_skill_json_numbers.sql`：用户可自行执行该迁移，把 `pet_definition` 历史字符串技能数组安全回填成数字数组，减少后续再依赖兼容读取
- 后台宠物模板详情接口现在会在返回 `load admin pet definition detail failed` 前输出真实服务端错误日志，日志格式包含 `path`、`method`、`pet_id` 和底层 `err`，便于直接判断是迁移缺失还是数据脏值
- 登录页 `DevServerSwitcher` 已精简为单一下拉框：`client/scenes/ui/common/dev_server_switcher.tscn` 不再保留手填 HTTP/WS、地址摘要和“应用/清空”按钮；`client/scripts/ui/common/dev_server_switcher.gd` 会在切换选项时立即应用对应环境
- `client/autoload/network_config.gd` 的原生端与 Web 端默认环境都已调整为 `local`，且登录页切服面板初始化时会主动回到本地环境，避免旧调试覆盖残留导致误连远程
- 统一客户端网络环境配置：`client/autoload/network_config.gd` 新增本地 / 远程 / 浏览器同源三种环境解析，并支持 Web 端通过 URL 参数或 `localStorage` 临时覆盖 HTTP / WebSocket 地址
- `client/autoload/http_client.gd` 与 `client/autoload/net_client.gd` 改为直接读取统一配置解析结果，不再在 Web 运行时强制回退到 `window.location.origin`，便于本地导出页灵活切换到远程后端联调
- 登录页新增 `DevServerSwitcher` 开发切服面板：可直接在 UI 上切换环境或手填 HTTP / WS 地址；应用配置后会刷新 `HttpClient` / `NetClient` 当前入口并清空旧会话，减少本地联调时频繁改 URL 参数或脚本常量
- 新增 `BackgroundAudioKeeper` 自动加载单例：通过 `AudioStreamGenerator` 持续喂入静音帧，在所有场景中保持一个近乎静音的背景音频上下文；这是浏览器后台保活的尝试性增强，不替代断线重连与前后台恢复链路
- `client/export_presets.cfg` 的 Web 自定义 CSS 改为固定 `780:1440` 纵横比，不再强制 `#canvas` 为固定像素尺寸；正式导出页现在会在浏览器内按同比例自适应缩放
- `client/autoload/web_runtime_canvas.gd` 统一接管 Web 运行时画布比例：客户端启动后会在所有场景把 DOM `canvas`、父容器、`body` 与 `html` 约束为 `780:1440` 比例，并在当前浏览器可视区域内自动算出实际显示尺寸，因此登录页与主运行态都能按同一比例显示
- `client/scripts/feature/world/world_controller.gd` 在 Web 环境下固定内部世界 `SubViewport` 为 `780x1440`，即使临时调试壳 `tmp_js_export.html` 继续按浏览器窗口给出较小容器，也不再把世界内部渲染尺寸改成 `621x834`
- `client/scripts/ui/main_menu.gd` 为 `%TabsRow` / `%ItemsList` 增加稳定路径回退，修复运行态 `Node not found: %ItemsList` 导致主菜单初始化中断
- `client/scenes/maps/fashtown/radiant_market.tscn` 删除 `TileSetAtlasSource_rbgk4` 中超出 `Tilemap_Platform.png` 可用高度的越界 atlas 条目，静态检查确认 `fashtown` 全部地图 atlas 坐标均未再超出贴图边界

- 已重新启动 `backend/server/cmd/game-server`，并确认 `POST /api/v1/auth/login` 当前返回 `200` 与标准 JSON 结构

## 2026-05-20 客户端核心脚本注释补齐

本次补充聚焦把客户端核心运行链路中最常看的四个脚本按当前项目规则补上说明性注释：
- `client/scripts/bootstrap/runtime_hud.gd` 已为运行态 HUD 的常量、信号、节点引用、面板状态字段、数据卡渲染和编队编辑流程补齐注释
- `client/scripts/feature/world/world_controller.gd` 已为场景配置、权威快照应用、固定镜头布局、门区切图和本地坐标换算补齐注释
- `client/autoload/net_client.gd` 已为连接状态、心跳调度、正式链路封包解包、CRC32 校验和开发态文本协议分发补齐注释
- `client/autoload/game_state.gd` 已为会话状态、世界快照、附近实体、宠物/编队、背包和战斗状态合并逻辑补齐注释
- 本次没有调整任何协议字段、状态结构、消息流转或游戏表现，仅增加注释说明并通过相关 GDScript 诊断检查

本次继续补充第二批核心脚本的说明性注释：
- `client/scripts/bootstrap/main.gd` 已为主运行态场景挂载、消息路由注册、HUD 刷新、世界/战斗视图切换和返回登录页流程补齐注释
- `client/autoload/app.gd` 已为应用层启动编排、HTTP 登录、WebSocket 鉴权、提示推送和战斗动作上报入口补齐注释
- `client/scripts/auth/login_scene.gd` 已为登录链路、登录页状态刷新、演示账号填充和场景切换过渡补齐注释
- `client/scripts/feature/battle/battle_scene.gd` 已为战斗界面刷新、技能按钮状态、战斗事件文案生成和单位状态读取逻辑补齐注释
- 第二批脚本同样只增加注释说明，不改动现有协议、状态结构和交互行为，并通过相关 GDScript 诊断检查

本次继续补充第三批运行态薄控制脚本的说明性注释：
- `client/scripts/feature/world/player.gd` 已为四方向移动、状态机切换、动画回退和切图/战斗锁定逻辑补齐注释
- `client/scripts/feature/pet/pet_controller.gd` 已为宠物列表响应、宠物更新推送和编队设置响应的状态写回逻辑补齐注释
- `client/scripts/feature/bag/bag_controller.gd` 已为背包列表响应和单物品更新推送的状态写回逻辑补齐注释
- `client/scripts/feature/battle/battle_controller.gd` 已为交互响应、战斗开始/更新/结算推送的状态写回与事件广播逻辑补齐注释
- 第三批脚本继续保持“只补注释、不改行为”的原则，并通过相关 GDScript 诊断检查

本次补充最后一批客户端基础设施脚本的说明性注释：
- `client/scripts/common/command_ids.gd` 已为客户端协议消息号常量补齐注释，明确各请求、响应和推送编号的用途
- `client/autoload/message_router.gd` 已为消息回调注册表、注册/注销和统一分发逻辑补齐注释
- `client/autoload/http_client.gd` 已为基础地址、登录接口和通用 JSON 请求封装逻辑补齐注释
- 最后一批脚本同样只增加注释说明，不改动现有协议、网络链路和状态结构，并通过相关 GDScript 诊断检查

## 2026-05-17 固定镜头地图出生点居中

本次补充聚焦把登录后角色在固定镜头地图中的出生显示点统一收敛到地图场景中心：
- `client/scripts/feature/world/world_controller.gd` 不再把 `scene_id = 1` 的出生显示点写死为单独的 `spawn_local_position`
- 固定镜头地图现在会优先读取显式配置；如果未配置，则自动按当前地图可见内容包围盒中心计算出生显示点
- 因此登录进入世界、收到权威世界重同步、以及后续切回固定镜头地图时，角色都会默认显示在对应地图场景中心
- 非固定镜头地图原有“出生逻辑坐标映射到视口中心”的链路保持不变，没有扩散修改现有服务端世界权威坐标规则

## 2026-05-17 主运行态改为 3:1 上下布局

本次补充聚焦把登录后主运行态的游戏区与操作区比例从 `4:1` 调整为 `3:1`：
- `client/scenes/bootstrap/main.tscn` 中 `GameplayArea.anchor_bottom` 已从 `0.8` 调整为 `0.75`
- `client/scenes/bootstrap/main.tscn` 中 `HudRoot.anchor_top` 已从 `0.8` 调整为 `0.75`
- 当前上部游戏区占 `75%` 高度，下部操作区占 `25%` 高度，世界、战斗与底部 HUD 的现有链路保持不变
- 本次只调整布局比例，不改动世界渲染、战斗挂载、协议、控制器和底部 HUD 交互逻辑

## 2026-05-17 设计分辨率收敛回 360x640

本次补充聚焦把客户端从大设计分辨率切回更适合像素地图编辑的小设计分辨率，并继续依赖运行时自动拉伸：
- `client/project.godot` 的设计分辨率与窗口覆盖尺寸已从 `1080x1920` 调整为 `360x640`，同时继续保留 `canvas_items + expand + integer` 的移动端适配方式
- `client/scripts/feature/world/world_controller.gd` 与 `client/scenes/world/world_scene.tscn` 已同步把世界层默认渲染尺寸收敛为 `360x480`，与当前 `3:1` 的主运行态上部游戏区一致
- `client/scenes/bootstrap/main.tscn`、`client/scripts/bootstrap/runtime_hud.gd`、`client/scenes/battle/battle_scene.tscn` 与 `client/scenes/auth/login_scene.tscn` 已把此前按大屏放大的字号、面板和按钮尺寸同步收回到小设计分辨率口径
- 当前思路改为“编辑期按小设计分辨率绘制像素地图，运行期由 Godot 按整数倍率自动放大”，不再需要为每张地图单独做统一缩放改造

## 2026-05-17 240x320 方案回退为 360x640

本次补充聚焦修正 `240x320` 设计分辨率导致的运行时发糊问题，并把客户端口径恢复为更适合当前竖屏目标分辨率的 `360x640`：
- `client/project.godot` 已把设计分辨率与窗口覆盖尺寸从 `240x320` 回退为 `360x640`，继续保留 `canvas_items + expand + integer` 的移动端整数倍率拉伸方式
- `client/scripts/feature/world/world_controller.gd` 与 `client/scenes/world/world_scene.tscn` 已同步把世界层默认渲染尺寸恢复为 `360x480`，重新匹配当前主运行态 `3:1` 布局下的上部游戏区
- `client/scenes/bootstrap/main.tscn` 与 `client/scripts/bootstrap/runtime_hud.gd` 已把底部 HUD 的字号、按钮高度、边距和数据面板尺寸恢复到 `360x640` 口径，避免 `240x320` 下过度压缩
- `client/scenes/auth/login_scene.tscn` 与 `client/scenes/battle/battle_scene.tscn` 也已同步恢复卡片、输入框、按钮与文本尺寸，使登录页和战斗界面在当前清晰度优先的方案下保持可读性

## 2026-05-17 清理早期占位地图文件

本次补充聚焦把客户端早期联调用的占位地图文件和对应引用一起清理掉：
- 三张早期占位地图场景已从仓库中删除
- `client/scripts/feature/world/world_controller.gd` 已移除对已删除占位地图的 `SCENE_CONFIGS` 加载路径，只保留当前正式接入的 `roxus_house`
- `client/scenes/maps/fashtown/roxus_house.tscn` 中通往已删除占位地图的出口门区也已同步移除，避免客户端继续发起无效切图
- `backend/docs/changelog.md` 与 `backend/docs/map-scene-loading.md` 已同步清理旧文件路径说明，避免文档继续指向已删除资源

## 2026-05-17 重新接通 roxus_house 与 east_road 双向切图

本次补充聚焦把两张正式地图重新接回当前服务端权威门区切图链路：
- `client/scripts/feature/world/world_controller.gd` 已重新补上 `scene_id = 2` 的地图映射，当前会加载 `client/scenes/maps/fashtown/east_road_of_shanguang_town.tscn`
- `client/scenes/maps/fashtown/roxus_house.tscn` 中新增的 `MapPortal` 现已补齐 `portal_id = 1001` 与 `target_scene_id = 2`，踩中后会沿用现有 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH` 链路切到 `east_road`
- `client/scenes/maps/fashtown/east_road_of_shanguang_town.tscn` 中的回程门现已补齐 `portal_id = 2001` 与 `target_scene_id = 1`，踩中后会返回 `roxus_house`
- 本次没有改动客户端门区脚本和协议结构，只是把现有 `MapPortal` 与服务端内存世界仓储中已经存在的 `portal_id 1001/2001` 重新对接起来

## 2026-05-17 正式地图门区入口坐标重标定

本次补充聚焦修正“人物看起来踩进门区，但没有稳定触发切图”的问题：
- `client/scripts/feature/world/world_controller.gd` 中 `scene_id = 2` 的 `spawn` 现已从旧占位地图时代的 `(2,4)` 调整为更贴合 `east_road_of_shanguang_town.tscn` 当前门区像素位置的本地坐标基准
- `backend/server/internal/data/memory/world_repo.go` 中 `portal_id = 1001` 的目标落点已调整到 `east_road` 左侧入口附近，`portal_id = 2001` 的目标落点已调整到 `roxus_house` 底部门区附近
- 同一文件里 `scene 1 <- 2` 与 `scene 2 <- 1` 的兼容入口落点也已同步修正，避免未携带 `portal_id` 的兼容切图仍然落到旧占位地图坐标
- `backend/server/internal/transport/ws/world_handler_test.go` 已把相关断言更新为新的权威落点，并执行 `go test ./server/internal/transport/ws`，当前通过

本次输出聚焦在线复刻版的基础骨架，完成了三部分设计落地：
- 协议层：定义固定包头、cmd 编号、关键消息边界
- 路由层：明确 server/client 双端消息分发与职责归属
- 存储层：给出可直接初始化的 PostgreSQL 最小表结构
- 服务端骨架：落地 HTTP 登录、JWT、`ws_token`、WebSocket 会话、心跳与基础路由
- 进入世界链路：落地 `ENTER_WORLD_REQ`，返回角色、场景、附近实体和编队快照
- 世界移动链路：落地 `MOVE_INTENT_REQ`，支持移动校验、位置更新、移动推送与重同步
- 目录重组：根目录拆分为 `backend/` 和 `client/`，当前后端工程整体归档到 `backend/`

设计上坚持以下约束：
- 客户端只提交意图，不提交结果
- 服务端拥有世界与战斗的最终权威
- 模板配置与玩家实例分离
- 世界同步和战斗同步隔离
- 当前服务端骨架使用内存仓储完成登录与会话验证，后续再切到 PostgreSQL/Redis
- 进入世界阶段只返回静态快照，不提前混入 AOI 广播和移动状态机
- 当前移动阶段只向请求方回推 `ENTITY_MOVE_PUSH`，AOI 对其他玩家的广播仍在下一阶段实现
- 此前 `client/` 仅保留空目录占位，当前已补齐可直接打开的 Godot 客户端骨架

建议的下一步实现顺序：
1. 生成 protobuf 代码，并把当前 auth/session JSON 消息体切换到 protobuf
2. 接入 PostgreSQL driver 与 Redis client，打通 `postgres_redis` 模式并替换当前内存版账号仓储与 `ws_token` 仓储
3. 在已完成的移动基础上，继续落 AOI 可见集和对其他玩家的移动广播
4. 落宠物实例、编队、战斗状态机
5. 落断线重连、限流与统一错误码映射

## 2026-05-14 客户端骨架补充

本次补充聚焦 Godot 客户端最小可开发骨架，目标是让 `client/` 可以直接被 Godot 4 打开并继续迭代：
- 初始化 `client/project.godot`、入口场景、图标和基础目录结构
- 按架构草案落地 `autoload` 层：`App`、`HttpClient`、`NetClient`、`MessageRouter`、`GameState`
- 预留世界、宠物、战斗、背包四个客户端控制器，并把消息号路由挂接到对应模块
- 当前 HTTP 登录已接好 `POST /api/v1/auth/login` 的调用封装
- 当前 WebSocket 只完成连接与开发期 JSON 路由骨架，二进制包头、protobuf 编解码和正式鉴权仍是下一步工作
- 增加 `.gitignore`，避免本地 SkillHub 目录和 Godot 生成目录进入版本库
- 当前持久化方案已统一切到 PostgreSQL，初始化 SQL 脚本已同步改写为 PostgreSQL 方言

## 2026-05-14 存储骨架补充

本次补充聚焦服务端真实存储切换前的骨架准备，先把配置、仓储适配器和装配边界补齐：
- 新增 PostgreSQL、Redis 相关配置项，并补充示例环境变量；后续已进一步收敛为单一 `PostgreSQL + Redis` 运行路径
- 新增 PostgreSQL 版账号、玩家、宠物仓储适配器，统一复用现有模块仓储接口
- 新增 Redis 版 `ws_token` 仓储适配器，使用 key 前缀和一次性消费语义预留真实接入点
- 新增 provider 装配层，统一管理服务端仓储依赖绑定；后续已删除 memory 运行分支
- 当时的 PostgreSQL/Redis 适配器先完成了骨架与接口约束，后续版本已补齐真实数据库连接、Redis 客户端初始化和驱动导入
- 新增 `config.env` 自动加载能力，后续只需要改 `backend/server/configs/config.env` 即可接入真实服务

## 2026-06-11 服务端 YAML 配置切换

本次补充聚焦把服务端启动配置从环境变量文件收敛到 YAML 配置文件：
- `backend/server/cmd/game-server/main.go` 不再先加载 `config.env`，改为解析 `backend/server/configs/config.yaml`；`PP_CONFIG_FILE` 仍可保留为“覆盖配置文件路径”的入口
- `backend/server/internal/config/config.go` 改为读取分段 YAML 结构：`http`、`auth`、`heartbeat`、`postgres`、`redis`，再转换成现有运行时 `Config` 结构，尽量不影响业务层依赖注入
- 示例文件已更新为 `backend/server/configs/config.yaml` 与 `backend/server/configs/config.yaml.example`，后续本地联调或部署时统一改 YAML，不再维护一长串 `PP_*` 配置键值
- 已新增配置加载测试，覆盖 YAML 解析、默认路径选择与基础校验，降低这次加载方式切换对启动链路的破坏风险

## 2026-06-11 PostgreSQL 宠物 mana 字段补齐

本次补充聚焦修复 PostgreSQL 模式下进入世界时报 `pp.mana does not exist` 的结构不一致问题：
- 新增迁移 `backend/server/migrations/010_add_player_pet_mana.sql`，为 `player_pet` 表补充 `mana` 字段，和当前 `pet_repo` / 战斗构建链路保持一致
- 同一迁移里已回填演示宠物 `20001/20002/20003` 的起始法力，保持 PostgreSQL 模式与内存测试仓储的默认值一致
- 这样 `PET_LIST_RESP`、编队读取、人物带宠进入战斗以及断线重连回放都可以继续复用同一个持久化宠物资源字段，不再因为数据库旧结构直接失败

## 2026-05-14 登录页与登录链路补充

本次补充聚焦 Godot 客户端首个可用登录入口，目标是把现有 HTTP 登录骨架升级为可直接联调的完整登录流程：
- 主场景 UI 从调试面板收敛为最小登录页，保留账号、密码、状态、场景、玩家和日志展示
- 登录按钮触发 `HTTP 登录 -> WebSocket 连接 -> WS_AUTH_REQ -> ENTER_WORLD_REQ` 串行流程
- `NetClient` 补齐固定包头编码、CRC32 校验、二进制包解析与按序号发送能力
- `App` 增加 WebSocket 打开后自动鉴权、鉴权成功后缓存会话并启动心跳的编排逻辑
- `GameState` 补充 `session_id`、`reconnect_token`、`heartbeat_sec`、`is_ws_authenticated` 等会话状态字段
- 现阶段服务端登录接口无需调整，客户端已对齐当前后端的 JSON 消息体和二进制包结构
- 已完成 GDScript 诊断检查、服务端 `go test ./...` 验证以及运行期无报错启动检查

## 2026-05-14 登录场景拆分

本次补充聚焦客户端场景职责收敛，把登录流程从主场景拆成独立入口：
- 新增 `res://scenes/auth/login_scene.tscn` 与对应脚本，专职处理账号密码输入、HTTP 登录、WS 连接与鉴权反馈
- 项目启动入口调整为登录场景，应用启动后先进入登录页，再在鉴权成功后切换到主场景
- `bootstrap/main` 不再承担登录表单职责，当前只负责世界场景挂载、消息路由注册、状态展示与进入世界请求
- 场景切换过程中保留已建立的 WebSocket 会话，避免登录成功后重复认证
- 已完成新旧场景和启动配置的诊断检查，当前无新增 GDScript 或场景报错

## 2026-05-14 登录转场与主场景 HUD 微调

本次补充聚焦登录切换体验与小窗口界面密度：
- 登录场景和主场景均新增全屏遮罩过渡层，当前使用轻量淡入淡出转场，不引入额外资源和依赖
- 登录成功切主场景、主场景掉线返回登录场景时都会经过同一套黑场过渡，减少场景切换突兀感
- 主场景顶部状态面板进一步缩小，保留连接、场景、玩家三类核心信息，尽量不遮挡游戏画面
- 主场景底部日志面板高度同步压缩，继续保留联调可见性但降低运行态占屏
- 已完成场景与脚本诊断、运行态重启检查，当前无新增报错

## 2026-05-15 角色三态状态机补充

本次补充聚焦角色在进入战斗场景前的运行态约束，先把世界内角色状态机补齐：
- `player.gd` 从原先仅依赖输入方向的二态逻辑升级为显式三态：待机、行走、战斗中
- 战斗中状态会锁定角色移动输入，并优先尝试播放 `battle_*` 动画；若资源未补齐，则回退到同朝向待机动画
- `GameState` 新增 `is_in_battle`，用于在世界层和角色层共享当前是否处于战斗中的状态
- `battle_controller.gd` 在战斗开始/进行中时置为战斗态，在战斗结果到达时退出战斗态
- `world_controller.gd` 监听战斗状态变化并同步给本地角色，保证角色表现与战斗入口状态一致
- 已完成相关脚本诊断检查，当前无新增报错

## 2026-05-15 战斗视图场景接入

本次补充聚焦“进入战斗场景”和“战斗结束返回世界”的最小可用链路：
- 新增 `res://scenes/battle/battle_scene.tscn` 作为独立战斗视图场景，并配套 `battle_scene.gd` 做基础信息展示
- `battle_controller.gd` 补充 `battle_started`、`battle_finished` 信号，用于通知主场景进入和退出战斗视图
- 主场景新增 `BattleMount` 容器，在收到 `BATTLE_START_PUSH` 时挂载战斗视图，在收到 `BATTLE_RESULT_PUSH` 时卸载并回到世界视图
- 该实现保留主场景根节点和现有消息路由，不使用整棵树 `change_scene`，从而避免战斗期间网络链路和路由中断
- 战斗进行中会隐藏世界层显示，战斗结束后恢复世界层显示，并继续复用已有世界快照状态
- 已完成相关场景/脚本诊断及运行日志检查，当前无新增报错

## 2026-05-15 服务端权威最小战斗闭环

本次补充聚焦“多人联机场景下所有战斗计算必须由服务端负责”的约束，完成了第一版可跑闭环：
- 服务端新增 `battle` 模块，以玩家当前主战宠对战附近 NPC 的最小 PvE 模型管理单场战斗状态
- 世界内通过 `INTERACT_REQ` 申请与附近 NPC 交互开战，服务端校验会话、玩家、阵容和附近实体后返回 `BATTLE_START_PUSH`
- 战斗内客户端只会提交 `BATTLE_ACTION_REQ` 动作意图，当前最小实现支持普通攻击和逃跑，其中伤害、回合推进和结算全部在服务端完成
- 服务端每次动作处理后会返回 `BATTLE_ACTION_RESP`，并按结果推送 `BATTLE_STATE_PUSH` 与 `BATTLE_RESULT_PUSH`
- 客户端主场景新增“挑战附近NPC”入口，战斗视图新增“普通攻击”按钮；客户端仅负责展示状态和提交意图，不做本地数值计算
- `GameState` 的战斗状态同步改为增量合并，保证战斗开始快照与后续状态推送可以共同驱动 UI
- 已补充协议文档、WebSocket 路由测试和 `go test ./...` 验证，当前服务端测试通过，客户端脚本/场景诊断无报错

## 2026-05-15 最小技能模型与技能按钮

本次补充聚焦战斗动作从“单一普通攻击”升级到“按技能意图提交”：
- 服务端 `battle` 模块新增最小技能表，当前内置玩家和敌方各两种技能，并由服务端按 `skill_id` 计算不同伤害值
- 服务端会校验提交的 `skill_id` 是否属于当前出战单位可用技能，非法技能请求将直接拒绝
- 敌方行动改为按回合轮换自身技能表，不再固定使用单一伤害模板
- 客户端战斗场景改为根据 `BATTLE_START_PUSH` 下发的 `skill_ids` 动态展示技能按钮，而不是写死一个攻击按钮
- 技能按钮点击后仅发送 `BATTLE_ACTION_REQ`，本地不做伤害、命中或回合推进推导，继续保持服务端权威
- 已补充战斗路由测试以覆盖多技能快照和技能动作联调，`go test ./...` 通过，战斗场景诊断无报错

## 2026-05-16 原版客户端参考逻辑沉淀

本次补充聚焦把逆向出来的原版客户端 `/Users/wangzhiwei/study/kdjl` 中可复用的流程设计沉淀为当前项目文档：
- 新增 `backend/docs/kdjl-client-reference.md`，只保留与当前 MVP 直接相关的参考逻辑，不扩展公会、交易、活动等边界外能力
- 文档确认原版最值得吸收的是登录前状态机、登录上下文本地持久化、世界/战斗场景切换关系、地图入口意图上报、战斗意图提交与服务端结算边界
- 文档明确原版协议和 UI 技术只适合参考思路，不适合直接迁移，包括文本协议、服务端驱动 `<menu>/<input>`、WAP 代理联网和敏感信息缓存
- 文档补齐了逆向类与当前项目模块的映射，便于后续在 `client` 与 `backend/server/internal/module/*` 中按现有架构落地
- 本次任务只新增文档与记录，不改动现有双端功能链路

## 2026-05-16 宠物编队与战斗快照模型设计

本次补充聚焦把上一步的原版参考结论进一步收敛成可直接指导实现的模型文档：
- 新增 `backend/docs/pet-lineup-battle-model.md`，把后续实现必须区分的四层对象固定为 `PetInstance`、`Lineup`、`ActivePet`、`BattleActorSnapshot`
- 文档结合当前仓库现状，明确 `pet`、`player`、`battle` 三个模块各自负责什么，不允许把宠物持久化状态、编队顺序和战斗运行态混在一起
- 文档补充客户端 `GameState` 的建议状态结构，明确 `pets`、`lineup`、`battle_state` 的边界，并指出当前 `upsert_pet()` 以 `pet_id` 合并的风险
- 文档补充了 `PET_LIST_RESP`、`PET_LINEUP_SET_REQ/RESP`、`BATTLE_START_PUSH`、`BATTLE_STATE_PUSH` 的后续补强方向，便于后面按最小代价逐步落实现有骨架
- 文档给出建议实现顺序：先补完整宠物实例，再补编队闭环，再显式化当前出战宠，最后再做战斗结算回写与换宠

## 2026-05-16 宠物列表与编队设置最小闭环

本次补充聚焦把上一条模型设计落成第一批最小代码改动：
- 服务端新增 `pet_handler.go`，正式接入 `PET_LIST_REQ` 与 `PET_LINEUP_SET_REQ` 两条 WebSocket 链路，并接入路由与应用启动装配
- `pet` 模块补齐了宠物实例模型、宠物列表查询、编队设置校验和仓储接口；内存仓储新增演示宠物列表，PostgreSQL 仓储新增宠物列表查询与编队写入能力
- `PET_LIST_RESP` 现已返回 `pets + lineup`，`PET_LINEUP_SET_RESP` 现已返回 `accepted + lineup + reason`，避免客户端收到编队变更后还要二次查详情
- 客户端 `GameState.upsert_pet()` 改为按 `pet_uid` 合并，解决同种宠物多只并存时被错误覆盖的问题；`set_pets()` / `set_lineup()` 现在会自动同步 `in_lineup`
- 客户端 `App.gd` 新增 `set_pet_lineup()` 发送入口，`pet_controller.gd` 仅在服务端确认成功后才更新本地编队，避免失败响应把本地状态误清空
- 协议文档和 `backend/proto/pet/pet.proto` 已同步更新；已执行 `go test ./server/...`，并完成相关 GDScript 诊断检查，当前无新增报错

## 2026-05-16 地图切换加载方案沉淀

本次补充聚焦把“参考原版客户端如何做地图切换加载”的方案落到当前仓库文档：
- 新增 `backend/docs/map-scene-loading.md`，明确世界层与战斗层分离、地图资源热切换、服务端权威切图、客户端按 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH` 时序装载地图
- 文档对照当前 `world_controller.gd`、`main.gd` 和服务端 `world_handler.go`，说明现有可复用骨架与当前缺口，避免后续为了切图重写整套世界链路
- 文档给出推荐场景结构：`WorldRoot -> MapMount / RemoteEntities / LocalPlayerAnchor`，要求 `main.tscn` 和 `world_scene.tscn` 常驻，只替换地图节点
- 文档给出地图配置、门区切换、加载遮罩和分阶段实施顺序，便于后续按最小代价推进地图绘制与切图接入
- 本次仅新增设计文档和记录，不改动现有双端运行代码

## 2026-05-16 世界地图资源挂载第一阶段

本次补充聚焦把地图切换加载方案先落成客户端第一阶段的最小实现：
- `client/scenes/world/world_scene.tscn` 新增 `MapMount` 挂载点和最小 `MapLoadingOverlay`，保证世界根场景常驻，只替换地图资源节点
- `client/scripts/feature/world/world_controller.gd` 为 `SCENE_CONFIGS` 增加 `scene_path`，并新增地图资源加载、卸载和切图加载态控制逻辑
- 客户端现在会在收到服务端世界快照时按当前 `scene_id` 装载对应地图资源；地图切换仍然沿用 `MOVE_INTENT_REQ -> MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH`，没有改变服务端权威链路
- `client/scripts/feature/world/player.gd` 继续只负责角色移动和战斗锁定，不承担地图切换判定
- 早期曾补三张最小地图占位骨架用于联调地图切换链路；后续正式地图资源接入后，这些占位场景已被清理
- 已对相关 GDScript 和 `.tscn` 文件完成诊断检查，当前无新增报错

## 2026-05-16 地图入口落点修正

本次补充聚焦修正“切图后角色总出现在新地图中心”的问题：
- 根因是服务端内存版 `world_repo` 在场景切换时统一使用目标地图 `spawnPos` 作为落点，导致无论从哪边进入都落在固定中心参考点附近
- 当前最小实现已改为“按来源地图决定目标地图入口落点”：例如 `1 -> 2` 会落在 `2` 号地图左入口，`2 -> 1` 会落在 `1` 号地图右入口，`2 -> 3` 会落在 `3` 号地图左入口
- 这次没有扩协议字段，仍沿用 `target_scene_id`；因为当前每对相邻地图只有一个入口，最小规则足够支撑现阶段地图切换
- 同步更新 `backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，把 `corrected_pos` / `self_pos` 的口径明确为“权威入口落点”，不再写成统一出生点
- 已更新 `world_handler_test.go` 的切图断言，并执行 `go test ./server/...`，当前通过

## 2026-05-16 地图门区与 portal_id 闭环

本次补充聚焦把“入口落点”进一步落成真正的门/入口实例：
- 服务端 `protocol.MoveIntentReq`、`world.Service` 与内存版 `world_repo` 已补充 `portal_id`，当前会优先按门区配置决定目标地图和入口落点；若 `portal_id` 无效则拒绝切图
- `client/scripts/feature/world/map_portal.gd` 新增为最小门区脚本，地图场景中的 `Area2D` 门区进入后会发出 `portal_id + target_scene_id`，再由 `world_controller.gd` 统一走现有权威切图链路
- 三张占位地图场景已接入门区节点：`scene_1` 右门通往 `scene_2`，`scene_2` 左右门分别通往 `scene_1/scene_3`，`scene_3` 左门通往 `scene_2`
- `world_controller.gd` 新增门区绑定与切图冷却，避免玩家刚落在入口附近时立即再次触发反向传送，并彻底移除了边界触发切图逻辑
- 同步更新 `backend/proto/world/world.proto`、`backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，让协议草案、实现文档和当前代码保持一致
- 已新增无效 `portal_id` 的服务端测试，执行 `go test ./server/...` 通过；相关 GDScript 与地图场景诊断无新增报错

## 2026-05-16 当前出战宠显式化

本次补充聚焦把宠物战斗模型文档里“显式化当前出战宠”这一步真正落成代码：
- 服务端 `battle` 模块的运行时快照已补充 `active_actor_id`、`active_pet_uid`，并为 `BattleActorSnapshot` 增加 `lineup_index`，使“当前出战宠”和“战斗单位快照”不再隐含耦合在数组第一位
- `BATTLE_START_PUSH` 与 `BATTLE_STATE_PUSH` 现在都会下发当前出战宠锚点，客户端不需要再默认用 `allies[0]` 猜测当前己方在场宠物
- 客户端 `GameState` 新增 `active_battle_actor()` 辅助方法，`battle_scene.gd` 改为按 `active_actor_id` / `active_pet_uid` 组织我方显示和动作提交，为后续换宠留稳定接口
- 同步更新 `backend/proto/battle/battle.proto` 与 `backend/docs/protocol.md` 的战斗快照结构，确保协议草案、文档说明和当前 JSON 实现一致
- 已补充战斗链路测试，校验 `BATTLE_START_PUSH` 与 `BATTLE_STATE_PUSH` 中的 `active_actor_id`、`active_pet_uid`、`lineup_index`；执行 `go test ./server/...` 通过

## 2026-05-16 战斗结束主战宠 HP 回写

本次补充聚焦把宠物战斗模型文档里“战斗结束回写主战宠 HP”这一步真正落成最小闭环：
- 服务端 `pet` 模块新增宠物 HP 更新接口，`memory` 与 `postgres` 两套仓储均已支持按 `player_id + pet_uid` 回写当前 HP
- 服务端 `battle` 结算结果现已显式带出主战宠 `pet_uid` 与最终 HP，`battle_handler` 会在发送战斗结果时先回写宠物实例，再通过 `3011 PET_UPDATE_PUSH` 推送最新宠物详情
- 客户端继续复用现有 `pet_controller.gd` 的 `handle_pet_update()`，按 `pet_uid` 合并本地宠物实例，不新增额外路由与 UI 逻辑
- 协议文档已补充 `PET_UPDATE_PUSH` 消息体，并明确当前 `BATTLE_RESULT_PUSH` 之后可能继续跟随宠物更新推送
- 已扩展 `world_handler_test.go`，同时校验 `PET_UPDATE_PUSH` 内容与回写后 `PET_LIST_RESP` / `lineup` 的 HP 一致性；执行 `go test ./server/...` 通过

## 2026-05-16 scene_1 地图资源替换

本次补充聚焦把客户端 `scene_id = 1` 对应的地图资源替换为新建的 `roxus_house` 场景：
- `world_controller.gd` 中 `SCENE_CONFIGS[1].scene_path` 已调整为 `res://scenes/maps/fashtown/roxus_house.tscn`
- 本次只替换客户端地图资源映射，不改服务端 `scene_id`、出生点配置和现有地图切换协议
- 当前 `roxus_house.tscn` 本身未接入门区 `Area2D`，因此如果需要保留 `1 -> 2` 的切图出口，还需要后续继续补门区节点与 `portal_id`

## 2026-05-16 roxus_house 门区补齐

本次补充聚焦把刚替换进来的 `roxus_house` 地图接回现有门区切图链路：
- `roxus_house.tscn` 已新增 `ExitPortal` 门区节点，并复用现有 `res://scripts/feature/world/map_portal.gd`
- 当前门区配置为 `portal_id = 1001`、`target_scene_id = 2`，与之前 `scene_1 -> scene_2` 的最小切图链路保持一致
- 同时新增了一个半透明 `ExitMarker` 出口标记，便于在只画了瓦片的阶段快速确认门区位置和触发范围
- 已完成 `roxus_house.tscn` 的 Godot 诊断检查，当前无新增场景错误

## 2026-05-16 roxus_house 固定镜头模式

本次补充聚焦把 `roxus_house` 调整为“相机固定、整图展示、角色在图内移动”的视角模式：
- `world_controller.gd` 为 `scene_id = 1` 新增 `fixed_view` 与 `spawn_local_position` 配置，当前 `roxus_house` 会按固定镜头模式渲染
- 固定镜头模式下，相机会固定在当前视口中心；地图会按场景内可见内容计算包围盒，并自动居中到屏幕可视区域
- 当地图实际尺寸大于当前窗口可视区域时，地图与角色锚点会按同一缩放比例缩小，尽量完整展示当前地图内容
- 角色位置换算从“相对出生点居中”切换为“出生点对应地图内本地落点 + 服务器坐标偏移”，避免角色和地图相对位置错位
- 已完成 `world_controller.gd` 的 Godot 诊断检查，当前无新增脚本错误

## 2026-05-16 主场景上下分区布局

本次补充聚焦把登录后的主运行态调整为“上部跑游戏、下部常驻 HUD”：
- `client/scenes/bootstrap/main.tscn` 已新增 `GameplayArea`，世界地图与战斗场景现在只在上部固定区域内渲染，避免覆盖底部常驻 UI
- 同一主场景新增底部 `HudRoot` 与 `HudBackground`，会永久显示 `client/asset/场景原图/闪光镇/时光小屋.png`，作为运行态底图
- 现有连接状态、场景信息、玩家信息、挑战按钮与日志输出已统一挪到底部 HUD 区，登录成功后会持续保留，不再压在地图上方
- `main.gd` 会把上部游戏区域尺寸同步给 `world_controller.gd`，固定镜头地图改为按游戏显示区大小计算居中与缩放，而不是按整个窗口布局
- 已完成 `main.tscn`、`main.gd` 与 `world_controller.gd` 的 Godot 诊断检查，当前无新增场景或脚本错误

## 2026-05-16 原客户端主运行态分层参考补充

本次补充聚焦把原客户端里和“登录后常驻主界面”最相关的结构继续沉淀到参考文档：
- `backend/docs/kdjl-client-reference.md` 已新增“登录后主运行态的分层布局”小节，明确原客户端采用单主画布承载上部游戏内容、下部常驻功能区和全局弹层
- 文档同时补充了“战斗层与常驻 UI 的共存关系”，说明原客户端世界层切到战斗时会继续复用公共 UI 资源，而不是整棵界面重建
- 当前项目可以继续吸收这条结构原则：`main.tscn` 作为常驻运行态根容器，上部切换世界/战斗显示层，下部保留固定 HUD 和后续操作区
- 本次只更新参考文档与记录，不扩展新的玩法范围，也不改变现有协议和主链路

## 2026-05-16 主运行态 UI 结构文档

本次补充聚焦把当前项目登录后的主运行态 UI 结构进一步沉淀为单独设计文档：
- 新增 `backend/docs/main-runtime-ui-layout.md`，明确主运行态采用“上部游戏显示区 + 下部常驻 HUD 区”的固定分层
- 文档把 `GameplayArea`、`WorldMount`、`BattleMount`、`HudRoot` 等节点职责单独拆开，约束地图切换、战斗切换只影响上部显示层
- 文档同时明确当前 MVP 下底部 HUD 只应承接连接状态、世界交互、战斗摘要以及宠物/编队/背包入口挂点，不直接扩展商城、频道、任务等超范围功能
- 本次没有新增代码逻辑，只补充了后续 UI 实现所需的结构口径与演进顺序

## 2026-05-16 底部正式操作区骨架

本次补充聚焦把主运行态文档中的底部操作区真正落成第一版可运行骨架：
- 新增 `client/scripts/bootstrap/runtime_hud.gd`，把底部常驻 HUD 的状态刷新、按钮事件和日志输出从 `main.gd` 中独立出来
- `main.tscn` 的 `HudRoot` 现已接入 `RuntimeHud`，并补充 `ModeLabel`、`SummaryLabel`、`ChallengeButton`、`PetButton`、`LineupButton`、`BagButton` 等操作区节点
- `main.gd` 现改为通过 `RuntimeHud` 驱动头部状态文本与日志，并接收底部按钮事件后分别复用现有 `App.request_interact()`、`App.request_pet_list()`、`App.request_bag_list()` 链路
- 首次进入世界后会自动同步一次宠物与背包摘要，使底部按钮的宠物数、编队数、背包数能尽快显示当前状态
- 已完成 `runtime_hud.gd`、`main.gd` 与 `main.tscn` 的 Godot 诊断检查，当前无新增场景或脚本错误

## 2026-05-16 底部入口最小弹出面板

本次补充聚焦让底部 `宠物`、`编队`、`背包` 入口不再只是占位按钮：
- `runtime_hud.gd` 已新增统一 `DataPanel` 逻辑，点击 `宠物`、`编队`、`背包` 按钮会打开对应的最小摘要面板，并支持关闭
- 宠物面板当前展示 `pet_uid`、`pet_id`、等级、HP 与是否在编队中；编队面板展示当前编队顺序和 HP 摘要；背包面板展示物品 ID 与数量
- 面板内容会跟随 `GameState.pets_changed`、`GameState.bag_changed`、`GameState.battle_changed` 自动刷新；进入战斗时会自动收起，避免与战斗态 HUD 冲突
- 本次继续复用已有 `App.request_pet_list()` 与 `App.request_bag_list()` 链路，没有新增额外协议或控制器
- 已完成 `runtime_hud.gd`、`main.tscn` 与 `main.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 编队最小交互与卡片面板

本次补充聚焦把底部右侧数据面板从文本摘要升级成更正式、可操作的列表样式：
- `main.tscn` 的 `DataPanel` 已改为“标题栏 + 提示文案 + 滚动列表 + 底部操作栏”结构，为后续继续细化样式保留稳定骨架
- `runtime_hud.gd` 现已按面板类型动态生成卡片列表：宠物面板显示宠物实例卡片，背包面板显示物品摘要卡片，编队面板显示“当前编队 + 可加入宠物”两段结构
- 编队面板已补最小可操作闭环：支持加入宠物、移除宠物、上移、下移和重置当前待提交编队
- 点击“提交编队”后会通过 `RuntimeHud -> main.gd -> App.set_pet_lineup()` 复用既有请求链路，仍然遵循客户端只提交完整编队顺序、服务端最终校验的口径
- 已完成 `runtime_hud.gd`、`main.gd` 与 `main.tscn` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 主场景 4:1 上下布局

本次补充聚焦把登录后主场景调整成更接近原版参考图的上下分区：
- `main.tscn` 现已将上部 `GameplayArea` 调整为约 `384px` 高、下部 `HudRoot` 调整为约 `96px` 高，对应 `320x480` 小窗口下约 `4/5 : 1/5` 的布局比例
- 当前已取消 `时光小屋.png` 作为下部背景，改为上部天蓝色纯背景、下部淡红色纯背景，并保留轻微遮罩，保证操作区与游戏画布上下分离、互不遮挡
- 底部状态区、按钮区和数据面板已同步压缩到更适合 `1/5` 高度的尺寸，日志面板改为隐藏，避免继续占用操作区可视空间
- `main.gd` 与 `world_controller.gd` 继续沿用现有上部游戏区域尺寸同步链路，因此地图和战斗仍只在上部区域渲染
- 已完成 `main.tscn`、`main.gd` 与 `runtime_hud.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 上部游戏区独立子视口

本次补充聚焦修复上部游戏区顶部出现根视口黑色清屏区域的问题：
- `main.tscn` 的上部游戏区已改为 `GameplayArea -> GameplayViewportContainer -> GameplayViewport` 结构，世界层与战斗层挂点均迁入 `SubViewport`
- `GameplayBackground` 继续作为上部区域的底色，而世界地图与战斗界面改为在透明子视口中绘制，避免根视口默认清屏色继续漏到游戏区
- `main.gd` 的 `_sync_world_render_frame()` 现会同步更新 `GameplayViewport.size`，并继续把同一份尺寸传给 `world_controller.gd`
- 本次修复只涉及主场景渲染边界，不改动现有世界、战斗、宠物、编队和背包链路
- 已完成 `main.tscn`、`main.gd` 与 `world_controller.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误

## 2026-05-16 适配 1080x1920 分辨率

本次补充聚焦把此前基于 `320x480` 小窗口假设的主运行态 UI 和固定视角地图，整体迁移到 `1080x1920` 新设计分辨率：
- `main.tscn` 现已改为按锚点保持 `4:1` 的上下比例，上部游戏区会自动占据 `80%` 高度，下部操作区会自动占据 `20%` 高度，不再依赖旧的 `384px/96px` 写死尺寸
- `HudRoot` 内的状态区、操作区、数据面板、按钮和标题字号都已整体放大，使其在 `1080x1920` 下保持可读性和可点击性；`runtime_hud.gd` 动态生成的卡片字体、边距和按钮尺寸也已同步放大
- `world_controller.gd` 的固定视角布局现已允许在大屏上按可见区域自动放大地图，不再强行把缩放结果限制在 `1.0` 以下；同时移除了先前只针对小屏临时加上的固定偏移，改为通过 `view_offset/view_scale` 配置控制
- `world_scene.tscn` 的地图加载蒙层提示与 `battle_scene.tscn` 的战斗卡片尺寸、字体和按钮高度已同步扩展，避免在大分辨率下仍然维持旧小窗比例
- 已完成 `main.tscn`、`runtime_hud.gd`、`world_controller.gd`、`world_scene.tscn`、`battle_scene.tscn` 与 `main.gd` 的 Godot 诊断检查，当前无新增脚本或场景错误
- 登录页 `login_scene.tscn` 也已同步适配：新增浅色纯背景和居中登录卡片，整体放大标题、输入框、登录按钮、状态信息和日志区，使登录前入口在 `1080x1920` 下不再显得过小

## 2026-06-10 服务端权威自动战斗与超时补行动基础

本次补充聚焦把“命令阶段超时后谁来补行动”从客户端兜底改成服务端权威：
- `battle` 模块现已为活动战斗补充 `command_deadline` 与 `autoBattleEnabled` 运行态，`BATTLE_START_PUSH` / `BATTLE_STATE_PUSH` 会同步下发 `command_deadline_ms` 与 `auto_battle_enabled`
- `BATTLE_ACTION_REQ` 新增 `action_type=5` 入口，客户端可以只提交“开启/关闭自动战斗”的意图，具体剩余动作选择仍由服务端决定
- WebSocket 心跳链路现会顺带调用战斗超时推进；当命令阶段超过权威截止时间，服务端会自动为尚未提交的己方单位补默认动作并继续回合结算
- `session` 模块现已补充断线回调，玩家连接关闭或心跳超时后，活动战斗会自动切入服务端托管；后台轮询会继续推进回合并落宠物 HP 等持久化结果，即使客户端已经离线
- 当前客户端战斗场景已改为只展示服务端倒计时与托管状态，不再本地代投默认动作，避免联机时前后端各自推进同一回合
- 已执行 `cd backend && GOCACHE=/private/tmp/pocket-pet-gocache go test ./server/...`，当前服务端测试通过；Godot 侧仅完成静态脚本核对，尚未在引擎内实际联调

## 2026-06-10 PVE 奖励闭环第一版

本次补充聚焦把当前 PVE 从“能打完”推进到“打完会结算成长”：
- `battle` 结果快照现已补充 `reward_gold`、`reward_player_exp` 和按宠物拆分的 `pet_rewards`，服务端会在胜利时按敌方等级和数量生成稳定奖励
- `player` 与 `pet` 仓储已补充战斗结算写回接口，当前会在战斗结束时持久化玩家金币/经验以及各参战宠物的 HP / EXP
- 新增 `battle_record` 仓储接入，当前会按 `battle_id + player_id` 写入唯一奖励记录，作为最小版重复发奖保护
- `BATTLE_RESULT_PUSH` 现已额外返回本场发放的金币、角色经验以及发奖后的玩家累计金币/经验；客户端收到后会同步刷新本地玩家快照并在主日志中提示奖励
- 进一步补入了 `drop_texts` 文本掉落展示：当前会按怪物生成确定性的掉落文案，并在战斗结算日志和战斗详情里展示，但还不会写入背包
- 已补充 WebSocket 联调测试，覆盖战斗结果奖励字段、文本掉落、宠物经验回写和断线托管后的持久化结果；当前仍未扩展真实物品掉落 / 背包落库，也还没有把发奖与 battle_record 写入收敛到数据库事务

## 2026-06-10 技能目标类型扩展

本次补充聚焦把文档中提到的“全体技能 / 指定数量多目标技能”补到当前最小 PVE 闭环里：
- `battle` 技能表现已支持 `enemy_all` 与 `enemy_multi` 两类敌方范围目标规则；其中 `enemy_multi` 会保留客户端主目标选择，再由服务端按 `TargetCount` 自动补齐剩余目标
- 示例技能中，`1002 火花冲击` 已切换为全体敌方技能，`1004 弧光连射` 已作为双目标技能接入技能目录
- 战斗执行链路已补充多目标解析：全体技能不再要求客户端提供 `target_id`，双目标技能则会优先命中主目标并顺序补足其他存活敌方单位
- 客户端战斗界面已按 `skills[].target_type` 展示 `[敌全]` / `[敌二]` 徽标，并对全体技能禁用无意义的切换目标按钮，继续保持移动端简洁交互
- 已补充 `service_test.go` 覆盖全体技能与双目标技能命中多个敌人的服务端结算行为

## 2026-06-10 断线重连恢复第一版

本次补充聚焦把“战斗托管”继续往前推进到“客户端回来后能接上当前战斗态”：
- `session` 模块不再在 socket 断开时立刻销毁会话，而是保留一段短时重连窗口；窗口内可用 `reconnect_token` 把新连接重新绑定到原玩家会话
- `RECONNECT_REQ/RESP (1021/1022)` 已正式接入，服务端会在重连成功后返回新的会话信息，并轮换新的 `reconnect_token`
- `world` 与 `battle` 处理链路已补充重连快照拼装：当前会返回世界全量快照，以及活动战斗的 `battle_start + battle_state` 双快照，方便客户端直接复用已有控制器恢复界面
- 如果断线期间战斗已由服务端托管结束，重连响应还会临时带回一份 `battle_result`，让客户端仍能看到奖励与掉落文本，而不是只看到世界状态突然刷新
- 进一步补入了最近战斗状态缓存与 `last_frame` 协议字段：客户端重连时若仍知道自己停在第几帧，服务端会在缓存窗口内返回 `battle_replay_states`，先补最近几帧，再与当前战斗态对齐
- 客户端 `App` 已补入最小自动重连流程：连接关闭后若本地仍持有 `reconnect_token`，会优先发起 `RECONNECT_REQ`，成功后重建世界快照、恢复战斗界面，并补拉宠物/背包/任务摘要
- 已补充 WebSocket 测试覆盖“断开后重连恢复世界与战斗快照”的服务端闭环；当前版本仍属于全量重同步，不做战斗事件增量补帧与逐帧回放

## 2026-07-07 系统技能库主动/被动技能拆分

本次补充聚焦把“系统技能库中的被动技能”从旧的推断规则升级为显式配置，并继续复用现有被动效果生效链路：
- 新增迁移 `backend/server/migrations/086_skill_activation_mode.sql`，为 `skill_definition` 增加 `activation_mode` 字段，显式区分 `active` / `passive`
- `skill_type` 继续只表示攻击/治疗/辅助等效果类型，不再混用为“主动/被动”语义，避免影响原有公式与表现分类
- 后台技能模板列表、详情、编辑表单与筛选条件已接入 `activation_mode`；运营现在可以直接把技能标记为被动
- 服务端战斗链路已把被动技能从可选技能列表、客户端战斗快照、自动选技和主动施法入口中过滤；若错误提交被动技能，会权威回退为普通攻击
- 现有 `passive_skills.go` 的吸血、反伤、加属性等被动效果逻辑保持不变；被动技能仍会从已学习技能列表中读取并生效
- `module/skill` 已新增最小校验：被动技能不能配置成普攻、不能带主动目标策略、不能带精力消耗

## 2026-07-07 宠物被动属性常驻入面板

本次补充聚焦把“加属性型被动技能”从仅战斗生效，改为直接参与宠物最终属性计算：
- `backend/server/internal/module/pet/passive_attributes.go` 新增宠物常驻被动属性折算逻辑；列表、详情、单宠物刷新、编队读取都会把加生命/攻击/速度/法力、暴击与抗性类被动计入最终属性
- 例如宠物基础速度 `100`，若学会 `迅捷` 类被动且配置 `speed_pct = 50`，那么服务端返回给客户端的 `spd` 将直接变成 `150`
- 生命上限类被动会同步抬升 `hp_max`，并按当前血量百分比同步 `hp`，避免面板出现“血量仍按旧上限显示”的割裂
- 战斗侧已同步调整：宠物进入 battle 时不再重复叠加这类永久属性被动，只保留吸血、连击、复活、反伤等战斗期效果，避免面板和战斗里各算一次

## 2026-07-07 玩家详情页宠物编辑入口对齐正式技能槽

本次补充聚焦把“玩家详情页里的宠物编辑弹窗”与“独立玩家宠物页”统一到同一套技能维护口径：
- `admin/src/pages/players/PlayerPetSection.tsx` 不再编辑旧的技能名称文本串，改为和 `PlayerPetListPage.tsx` 一样的 `天生技 / 普通技 / 兼容战斗技能预览` 三段式列表编辑
- 编辑弹窗会实时根据 `innate_skill_ids + normal_skill_ids` 自动回填只读 `skill_ids` 预览，确保两个后台入口保存到服务端的数据结构一致
- 详情抽屉也同步拆分展示天生技、普通技和兼容战斗技能，便于运营直接判断某个技能当前是正式槽位数据，还是旧兼容列表数据
- 已执行 `cd admin && npm run build`，当前后台前端构建通过

## 2026-07-07 后台玩家宠物详情兼容历史技能字符串数组

本次补充聚焦修复 `GET /api/admin/pets/:pet_uid` 在历史脏数据下直接返回 500 的问题：
- `backend/server/internal/data/postgres/pet_repo.go` 读取 `player_pet.skill_ids`、`innate_skill_ids`、`normal_skill_ids` 时，已不再只接受严格 JSON 数字数组
- 当前统一复用 `backend/server/internal/data/postgres/json_uint32_array.go` 的弹性解析，兼容 `[20001]`、`["20001"]` 以及混合数字/数字字符串数组
- 这样即使旧数据还没完全清洗，后台玩家宠物列表、详情、编队与单宠物读取链路也不会因为 `json: cannot unmarshal string into Go value of type uint32` 直接报错
- 已执行 `cd backend && GOCACHE=/private/tmp/pocketpet-go-cache go test ./server/internal/data/postgres ./server/internal/module/pet ./server/internal/transport/http`，当前通过

## 2026-07-07 玩家宠物技能 JSON 脏数据清洗迁移

本次补充聚焦把运行时兼容过的 `player_pet` 历史脏数据继续往数据库层清干净：
- 新增迁移 `backend/server/migrations/088_backfill_player_pet_skill_json_numbers.sql`
- 该脚本会把 `player_pet.skill_ids`、`innate_skill_ids`、`normal_skill_ids` 中可安全转换的字符串数字数组统一回填成真正的 JSON 数字数组
- 脚本只处理“全部元素都能安全转成数字”的数组；如果某条记录里混入非法字符串，会保留原值不动，避免迁移误伤数据
- 建议用户本地执行该迁移后再重启后端，这样后台玩家宠物详情/列表链路既有代码兼容，也有数据库层清洗兜底

## 2026-07-07 被动技能删除攻击系数/表现后不再被默认补回

本次补充聚焦修复后台技能编辑页中“被动技能删掉攻击系数和战斗表现，保存刷新后又出现”的问题：
- 根因在 `backend/server/internal/module/skill/model.go` 的 `AdminUpsertInput.Normalize()`：此前无论主动/被动，都会在空值时自动补默认 `attack_pct=100`、`animation_key=slash`、默认施法色和命中色
- 这导致后台虽然已经把被动技能的攻击系数条目和表现条目删除，但服务端保存时又把这些默认值重新写回数据库
- 现在已改为：只有主动技能才补默认攻击系数与默认表现；被动技能如果删成空，就保持空值，不再自动回填
- 已补测试 `backend/server/internal/module/skill/service_test.go`，并执行 `cd backend && GOCACHE=/private/tmp/pocketpet-go-cache go test ./server/internal/module/skill ./server/internal/transport/http` 通过

## 2026-07-07 被动技能前端编辑器只保留被动效果项

本次补充聚焦继续收紧后台交互，避免运营在被动技能里再次看到或添加主动技能字段：
- `admin/src/components/SkillEffectConfigEditor.tsx` 现已按 `activation_mode` 过滤可添加效果类型；被动技能下只保留“被动属性加成”
- `admin/src/components/SkillDefinitionEditor.tsx` 在切换为被动技能时，会自动过滤当前 `effect_entries` 中不属于被动的条目
- `admin/src/pages/skills/SkillDefinitionPage.tsx` 在重新打开历史被动技能详情时，也会先过滤旧的攻击系数、表现等主动条目，避免旧脏配置再次出现在编辑器里
- 已执行 `cd admin && npm run build`，当前后台前端构建通过

## 2026-07-07 删除宠物被动技能后展示属性自动回退

本次补充聚焦修复后台玩家宠物编辑页中“删掉被动技能后，之前加上去的生命/攻击仍被保存成基础值”的问题：
- 根因是后台详情页展示的是“被动技能折算后的最终属性”，例如 `hp_max=100` + `强壮 20%` 会展示成 `120`
- 运营如果只删除技能、不改生命字段，表单会把展示值 `120` 原样提交；服务端此前会直接把这个展示值写回 `player_pet.hp_max`，导致删完技能后生命没有回退
- `backend/server/internal/module/pet/service.go` 现在会在保存前先读取该宠物的原始未加成详情，再把“与旧展示值完全相同、说明运营未手改”的属性字段还原回原始基础值后再持久化
- 当前已覆盖 `hp/hp_max/atk/spd/mana` 以及被永久被动直接影响的暴击率、暴伤、物抗、技抗和五类异常抗性，确保删除技能后这些展示属性会按新技能结果重新计算
- 已补测试并执行 `cd backend && GOCACHE=/private/tmp/pocketpet-go-cache go test ./server/internal/module/pet ./server/internal/transport/http` 通过

## 2026-06-10 战斗属性与抗性底座扩展

本次补充聚焦先把“人物单人 PVE”需要的服务端属性底座补齐，但暂时不接等级成长：
- `battle` 运行态的 `actorRuntime` 已新增完整核心属性：生命、精力、攻击、防御、速度、法力、命中、闪避、致命率、爆伤倍率，并补入物理/技能抗性、混乱/昏睡/麻痹/封印/诅咒抗性、抗致命、抗爆伤、抗人物、抗宠物、抗佣兵与通用护盾抗性
- `formula.go` 的有效属性与减伤链路现已识别新的攻击来源与抗性维度：技能/物理抗性、通用护盾抗性、人物/宠物/佣兵来源抗性都会进入最终减伤计算
- 战斗执行链路已补入精力消耗判定：主动技能若精力不足，会在服务端权威降级为普通攻击，避免未来人物单人战斗再单独补一套资源校验
- 状态命中链路现已支持按目标的混乱、昏睡、麻痹、封印、诅咒抗性扣减命中率；暴击结算也会额外吃目标的抗致命与抗爆伤
- 已新增 `buildPlayerCharacterActor()` 作为未来“人物单人 PVE”入口的服务端人物战斗模板；当前尚未接入 `StartPVE()` 正式参战，但人物和敌人的服务端属性模型已经可复用
- 已执行 `cd backend && go test ./server/internal/module/battle/...`，当前 battle 模块测试通过

## 2026-06-10 人物战斗属性入库第一版

本次补充聚焦把上一节的人物战斗属性从“只存在 battle 运行态”推进到“玩家仓储可持久化读取”：
- 新增迁移 `backend/server/migrations/008_add_player_combat_stats.sql`，为 `player` 表补充人物战斗核心字段与抗性字段，包括精力、攻击、防御、速度、法力、命中、闪避、致命、爆伤，以及物理/技能/状态/来源抗性和通用护盾抗性
- `player.Profile` 与 PostgreSQL `player_repo` 现已同步扩展；`FindByPlayerID()` 会把这些持久化字段全部读回，后续人物参战时不再只能依赖 battle 模块里的硬编码模板
- `teststub` 的内存玩家仓储也已补齐同一套人物战斗字段，保证 battle / ws / auth 等本地测试仓储口径一致
- `buildPlayerCharacterActor()` 现已优先使用 `player.Profile` 中的真实持久化人物数值；若旧测试桩或未迁移数据里某些字段仍为 `0`，才会回落到最小默认模板，降低迁移阶段的联调风险
- 敌方构建逻辑保留了现有按玩家等级生成强度的口径，只是在同一条链路中补上了新的抗性和资源字段，避免这次入库改动意外改变现有 PVE 测试节奏
- 已执行 `cd backend && go test ./server/...`，当前服务端测试通过

## 2026-06-10 单人 PVE 人物 actor 正式接入

本次补充把前两节铺好的“人物战斗属性 + 持久化数据”真正接进单人 PVE 开战链路：
- `battle.Service.StartPVE()` 现在会先构建人物 actor，再按原顺序追加宠物编队；即使当前没有上阵宠物，单人 PVE 也可以由人物独立开战
- 新增人物持久化技能字段迁移 `backend/server/migrations/009_add_player_skill_ids.sql`，并把 `player.Profile` / PostgreSQL 仓储 / teststub 一起扩展为读取 `skill_ids`，避免人物技能继续硬编码在 battle 运行态
- 人物 actor 默认使用数据库中的技能顺序；当前人物起手技能为人物专属主动技 `1101: 裂空斩`，服务端自动托管与超时补行动作也会优先选择人物主动技能，精力不足时再权威回退到普通攻击
- `EnterWorldResp` 新增完整 `player` 快照，客户端 `player_snapshot` 现在可以直接读取人物的生命、精力、攻击、防御、速度、法力、命中、闪避、暴击与技能列表；战斗 actor 快照也新增 `unit_class`，客户端可以区分人物 / 宠物 / 怪物
- 战斗结算已改为只回写真实宠物的 HP / 经验，避免把人物 actor 误当成 `pet_uid=0` 的宠物持久化

## 2026-06-15 玩家成长与属性加点第一版

本次补充聚焦把玩家角色从“有经验字段”推进到“可升级、可加点、可后台配表”的完整闭环（宠物升级留二期）：
- 新增迁移 `035_player_level_progression.sql`：引入 `player_level_config`、`player_attr_convert_config`、`player_attr_allocate_log`，并为 `player` 扩展 `free_attr_points`、四维已分配点数与 `base_*` 裸装战斗值
- 新增迁移 `036_admin_player_progression_permissions.sql`：补充 `player_progression:view/edit` 后台权限
- 新增 `module/progression` 领域模块：服务端权威处理经验连升、溢出结转、升级发点、加点校验与战斗属性重算；配置表变更后自动刷新运行时缓存
- `player.AddExp` 与 `reward` 发经验链路已统一委托 `progression.ApplyExp`；升级只增加 `free_attr_points`，不直接改 `atk/def/...`
- 新增 WebSocket `PLAYER_ALLOCATE_ATTR_REQ/RESP (2061/2062)`；`PlayerSnapshot` 扩展 `exp_to_next`、`free_attr_points` 与四维属性；`BATTLE_RESULT_PUSH` 扩展 `level_up_count`、`attr_points_gained`、`free_attr_points`、`exp_to_next`
- 后台新增 `/api/admin/player-progression/...` 等级经验与转化率配置接口，前端页面 `/player-progression`；玩家详情页展示自由属性点与四维分配值
- 客户端状态面板与加点页已对接真实快照；`points_status_panel.gd` 加点请求走通用 loading 遮罩，响应后通过 `GameState.merge_player_snapshot()` 刷新 UI
- 设计说明见 `backend/docs/player-progression.md`；需本地执行迁移 `035`、`036`、`037` 后重启服务

## 2026-07-07 数据库快照版数值系统第一版

本次补充聚焦把“人物/宠物最终属性”从直接信任业务主表中的最终值，收敛成“基础真源 + 服务端统一计算 + 数据库快照读取”的第一版闭环：
- 新增迁移 `backend/server/migrations/089_player_pet_combat_snapshots.sql`，创建 `player_combat_snapshot` 与 `player_pet_combat_snapshot` 两张运行时快照表；当前不引入 Redis，也不要求历史迁移回填，首次读取时会按需刷新生成
- 新增统一主属性公式层 `backend/server/internal/module/combatcalc/formula.go`，把五大主属性计算口径固定为 `（基础值 + 所有加算）× 所有百分比乘算`；首期已优先接入人物最终属性快照计算，后续其他加成来源继续复用同一入口即可
- `backend/server/internal/data/postgres/combat_snapshot_repo.go` 新增 PostgreSQL 快照构建与查询：
  - 人物快照按 `player.base_* + 已分配属性点转换 + 已佩戴装备` 重新计算 `hp_max/atk/def/spd/mana`，并连同命中、闪避、暴击、抗性、技能列表、皮肤一起写入 `player_combat_snapshot`
  - 宠物快照把当前 `player_pet` 运行态数据、技能槽、资质与次级属性统一收敛到 `player_pet_combat_snapshot`，服务层再继续补“永久属性型被动技能”的最终折算，保证对外读取口径一致
- `backend/server/internal/data/postgres/player_repo.go` 的 `FindByPlayerID()` 现已切成“每次读取前刷新人物快照，再返回快照”，因此客户端进入世界、装备操作回包、战斗开战读取、后台详情查看，都会看到实时重新计算后的最终人物属性
- `backend/server/internal/module/pet/service.go` 的 `ListPets()`、`ListLineup()`、`loadPetByUID()` 与后台宠物列表/详情，现已统一改成“先刷新宠物快照，再读快照”，并继续在服务层复用已有被动技能折算逻辑；这样删除/新增被动技能后，下次查询会按最新结果重新显示，而不是沿用旧最终值
- `backend/server/internal/data/postgres/equipment_runtime_repo.go` 不再把装备重算后的最终属性直接回写 `player` 表；佩戴/卸下/刷新模板后只更新人物快照，避免装备链路持续污染 `base_*` 基础值真源
- `backend/server/internal/data/postgres/player_repo.go` 的后台人物编辑与奖励加属性链路已同步收口：
  - 后台编辑人物五大主属性时，会同步维护 `base_hp_max/base_atk/base_def/base_spd/base_mana`
  - 奖励直接增加五大主属性时，也改为累加对应 `base_*` 字段，而不是直接写死最终 `hp_max/atk/def/spd/mana`
- 当前覆盖范围：客户端人物/宠物主查询、后台玩家/宠物详情展示、战斗开战前的人物/编队属性读取，都已收口到数据库快照版最终属性；本次仍未引入 Redis，也未做历史数据迁移
- 已执行验证：`cd backend && GOCACHE=/private/tmp/pocketpet-go-cache go test ./server/...`，当前通过

本次继续补第二轮“数据库快照版运行时视图”能力，目标是把前一版只覆盖人物/宠物最终战斗属性，继续扩展到装备视图、技能进度视图，以及统一刷新入口：
- 新增迁移 `backend/server/migrations/090_runtime_view_snapshots.sql`，创建 `player_equipment_snapshot` 与 `player_skill_progress_snapshot` 两张快照表
- `player_equipment_snapshot` 保存人物当前全身已佩戴装备的运行时视图，包括槽位、实例 UID、模板 ID、名称、图标、强化等级、是否损坏、描述、固定属性加成、武器附加技能和武器类型
- `player_skill_progress_snapshot` 保存人物武器技能学习进度视图，包括 `skill_exp / skill_level / is_learned / learned_at`，供战斗开战前直接判断当前武器技能可用性与学习态
- 新增统一刷新服务 `backend/server/internal/module/runtimeview/service.go`，把以下四类刷新收口到 `RefreshPlayerRuntimeSnapshots()`：
  - 人物最终战斗属性快照 `player_combat_snapshot`
  - 宠物最终战斗属性快照 `player_pet_combat_snapshot`
  - 人物已佩戴装备视图快照 `player_equipment_snapshot`
  - 人物技能进度视图快照 `player_skill_progress_snapshot`
- `backend/server/internal/app/bootstrap.go` 已把这套统一刷新入口注入到 `WorldHandler / EquipmentHandler / BattleHandler`：
  - 进入世界前会统一刷新玩家运行时快照
  - 装备面板拉取前会统一刷新玩家运行时快照
  - 战斗开战前读取人物武器类型、装备附加技能、技能学习进度前，也会统一刷新玩家运行时快照
- `backend/server/internal/data/postgres/runtime_view_snapshot_repo.go` 新增了 PostgreSQL 层的装备/技能视图快照读写：
  - `EquipmentRepository.ListEquipped()` 改为“先刷新，再读 `player_equipment_snapshot`”
  - `PlayerSkillProgressRepository.ListByPlayerID()` 改为“先刷新，再读 `player_skill_progress_snapshot`”
- 装备与技能进度操作链路已补双更：
  - 佩戴、卸下、模板热刷新后，会同步刷新 `player_equipment_snapshot`
  - 战斗结算写入人物技能经验后，会同步刷新 `player_skill_progress_snapshot`
- 至此，当前战斗入口读取人物数据时，已基本从“散落在多张原始表的直接读”收敛为“人物战斗快照 + 宠物战斗快照 + 装备视图快照 + 技能进度视图快照”的数据库快照体系
- 当前仍未做的部分：还没有把“活动战斗中的中间回合状态”持久化成数据库 battle snapshot 表；本次只覆盖开战前权威输入与主要展示视图快照，不改现有战斗进行中的内存态主链路
- 已执行验证：`cd backend && GOCACHE=/private/tmp/pocketpet-go-cache go test ./server/...`，当前通过

## 2026-07-09 装备强化弹窗关闭按钮常驻
- 问题原因：`equipment_enhance_popup.gd` 在强化演出开始时递归禁用弹窗内所有按钮，右上角关闭按钮使用通用 `panel_close_button.tscn`，其 disabled 样式为空，导致强化中看起来像“关闭按钮消失”。
- 修改内容：`client/scripts/ui/bag/equipment_enhance_popup.gd` 新增关闭按钮常驻处理，递归锁定交互时跳过 `_top_close_button`，保持其可见和鼠标命中；实际关闭仍由 `_block_dismiss` 拦截，避免中断强化演出与服务端回包同步。
- 影响范围：仅调整装备强化弹窗的关闭按钮显示状态，不改变强化请求、进度条、材料选择、背包刷新和服务端协议。

## 2026-07-09 背包装备详情背景修正
- 问题原因：`client/scenes/ui/bag/bag_item_detail.tscn` 在详情面板内部复用了 `模糊背景.tscn`，该资源通过 `SCREEN_TEXTURE` 采样屏幕，详情作为背包内层弹窗打开时会采到世界场景，而不是局部背包面板。
- 修改内容：将详情面板内部背景从通用模糊 Shader 改为普通半透明 `ColorRect`，保留原详情布局、描述、按钮和外层详情遮罩逻辑不变。
- 影响范围：仅调整背包物品详情的背景显示，不改变背包面板背景模糊、详情操作按钮、装备强化入口和服务端协议。

## 2026-07-09 任务面板真实数据与交付闭环
- 后端任务摘要补齐 `rewards` 与目标 `event_type` 输出，客户端可以在任务面板展示奖励预览和当前目标进度，不再依赖静态场景文案。
- 后端提交校验收紧：`quest.Service.Submit()` 只允许 `READY_TO_SUBMIT` 且所有目标完成的任务进入完成与发奖流程，避免未做任务时直接交付并把进度强制补满。
- WebSocket 任务错误映射新增 `quest not ready to submit`，领取/交付/追踪仍通过 `QUEST_ACCEPT_REQ`、`QUEST_SUBMIT_REQ`、`QUEST_TRACK_REQ` 走服务端权威链路。
- 客户端 `TaskPanel` 复用场景内已有任务卡片节点，打开前通过 `QUEST_LIST_REQ` 拉取最新列表，打开后监听 `GameState.quests_changed` 实时刷新；卡片主按钮按状态执行“领取 / 追踪 / 交付”。
- `App.accept_quest()`、`App.submit_quest()`、`App.track_quest()` 改为返回请求序列号，方便 UI 在操作回包前锁定按钮，避免重复点击产生并发请求。

## 2026-07-09 任务列表滚动动态卡片与面板领奖

本次补充聚焦任务面板的真实列表展示和面板领奖规则：
- `client/scenes/ui/task/主线任务列表.tscn`、`支线任务列表.tscn`、`日常任务列表.tscn` 的任务区域改为 `ScrollContainer + TaskCardVBox`，支持任务较多时上下滑动。
- `client/scripts/ui/task/task_panel.gd` 改为按 `GameState.quests` 动态实例化 `task_list.tscn`，一条后端任务对应一个卡片；卡片图标节点暂时隐藏，标题、目标描述、进度、奖励提示和按钮状态均来自后端快照。
- 任务进度条统一按服务端目标设置：`max_value=target`、`value=current`、`step=1`，因此杀怪 `3/10`、经验 `1500/4000`、对话 `0/1` 都走同一套渲染规则。
- `backend/server/internal/module/quest/service.go` 将目标完成后的状态统一收敛到 `READY_TO_SUBMIT`；面板可领奖任务点击“领取”后仍通过 `QUEST_SUBMIT_REQ` 由服务端发奖，NPC 领取/交付任务只在面板提示“前往”，不把权威交付下放给客户端。
- 已同步调整 `backend/server/internal/transport/ws/world_handler_test.go`，覆盖自动类任务需先领取奖励后才解锁后续任务的链路。

## 2026-07-09 任务图标 ID 下发与客户端本地映射

本次补充聚焦让任务图标像背包物品图标一样由“服务端 ID + 客户端本地资源表”驱动：
- 新增迁移 `backend/server/migrations/095_quest_client_icon_id.sql`，为 `quest_template` 增加 `client_icon_id`，并给现有 1001/1002/1003 任务分别配置主线、对话、战斗占位图标 ID。
- `quest.Template`、`quest.Summary` 与协议 `QuestSummary` 同步补充 `client_icon_id`，WebSocket 任务列表和任务更新推送都会带上该字段。
- 客户端新增 `client/scripts/ui/task/task_icon_definition.gd`、`client/autoload/task_icons.gd` 与 `client/resources/task_icons/` 图标资源，按显式资源清单加载，兼容 Web 导出。
- `client/scripts/ui/task/task_panel.gd` 在渲染任务卡片时调用 `TaskIcons.resolve_texture(client_icon_id)` 设置 `TextureRect.texture`，如果服务端未配置或客户端未命中，会自动使用默认任务图标。

## 2026-07-09 任务图标改为客户端图标 ID

本次补充把任务图标字段从通用 `icon_id` 收紧为客户端语义更明确的 `client_icon_id`：
- `quest_template.client_icon_id` 只作为客户端本地图标表的引用 ID，不绑定任务模板主键，也不设置唯一约束。
- 多个任务可以配置相同 `client_icon_id`，客户端会统一通过 `TaskIcons.resolve_texture(client_icon_id)` 解析同一张图标。
- 后台任务模板类型、列表、详情、创建、编辑均补充 `client_icon_id`，运营可以直接配置客户端图标 ID。
- 客户端任务面板优先读取 `client_icon_id`，并保留旧 `icon_id` 回退，避免灰度期间旧包缺字段导致图标空白。

## 2026-07-10 后台富文本双栏编辑与可视化刷色

本次改动收口后台所有已声明支持 BBCode 的编辑入口：
- `admin/src/components/RichTextEditor.tsx` 统一改为上方纯文本输入、下方客户端效果与刷色的单列布局。
- 下方预览中拖选文字后，编辑器会把 DOM 选区精确映射回内部 BBCode 偏移，再用所选颜色标签包裹；物品、宠物和玩家名占位符仍保留原 token。
- 六个颜色笔刷使用精确色值：`rgb(42,255,42)`、`rgb(255,255,0)`、`rgb(0,255,255)`、`rgb(255,125,0)`、`rgb(255,100,255)`、`rgb(255,0,0)`。
- `admin/src/utils/richTextBbcode.ts` 新增可见字符与 BBCode 原文偏移转换、指定区间刷色能力；颜色笔刷统一使用精确十六进制色值。
- 已清理装备强化成功、宠物属性上限、宠物技能槽解锁、任务阶段等入口中的 `showPreview={false}`，保证所有富文本编辑都显示右侧效果。
- 客户端 `task_list.tscn` 的任务描述节点保留原节点名与布局，只把类型切换为 `RichTextLabel`；`task_panel.gd` 通过通用富文本工具写入 BBCode。
- 根据最终界面要求，上方仅保留受控 `Input.TextArea`，显示时去除 `[color]` / `[b]` / `[i]` / `[u]` 标签，删除原文标题、格式工具、插入系统模板、插入示例与底部说明。
- 下方颜色区增加常用白 `rgb(255,255,255)` / `#FFFFFF`，当前共七个常用色。
- `admin/src/utils/richTextBbcode.ts` 增加 BBCode 格式字符解析与重建；运营修改纯文本时，未变部分的颜色、加粗、斜体和下划线继续保留，实际表单值仍为服务端和 Godot 可直接识别的 BBCode。
- 重新刷色不再向已有颜色内直接嵌套新标签；刷色时会把源码选区转换为可见字符区间，替换该区间的颜色格式后统一重建 BBCode。
- 已验证三类情况：单个颜色内局部重刷、跨两个颜色重刷、加粗+颜色文字重刷；可见文字和非颜色格式均保持不变。

## 2026-07-13 固定剧情对话名字框与场景头像修正

本次修复“初见桃子”剧情对话的名字框裁切和头像与场景角色不一致问题：
- `client/scenes/ui/npc_dialogue_panel.tscn` 将说话人行固定为 `60px` 高，确保移动端放大字号和 `28px` 头像有稳定布局空间。
- `client/scripts/ui/npc_dialogue_panel.gd` 的动态高度计算计入名字框纹理上下各 `14px` 的边距；本地对白可优先使用直接传入的 `Texture2D`，未传时继续走原有头像注册表。
- `WorldPlayerCinematic -> CinematicPlayer -> main -> NPCDialoguePanel` 本地对白信号链路同步透传场景头像纹理，不改变服务端对话协议。
- `client/剧情动画/初见桃子.gd` 按对白发生时刻提取桃子、七色羽的 `AnimatedSprite2D` 当前帧；玩家优先读取动态皮肤当前帧，并兼容 CHJ 与旧版图集当前帧。
- 已执行项目级 Godot 无头启动和“初见桃子”场景单独启动，均以退出码 `0` 完成；仅存在仓库既有物品图标 ID、macOS CA 和 ObjectDB 退出警告。

## 2026-07-13 初见桃子后续剧情动作与对白

本次继续扩展 `client/剧情动画/初见桃子.gd`：
- 在凤凰神炎冲击波演出结束后隐藏冲击波根节点，避免静止特效残留到后续对白。
- 七色羽向上移动 `18px`；由于其资源只有“向左走/待机左”，通过 `flip_h=true` 镜像得到右向行走和右向待机效果。
- 桃子向左移动 `10px` 后恢复左向待机；玩家向左移动 `10px` 时播放 `walk_left`，到达后通过现有朝向接口切换为向上待机。
- 按需求顺序追加六句对白，桃子、七色羽和玩家均继续使用场景中的实际当前帧头像。
- 已执行“初见桃子”场景 Godot 无头单独启动，退出码为 `0`；`git diff --check` 通过，受影响 GDScript 的 Tab 数量为 `0`。

## 2026-07-13 重复剧情对白正文不显示修复

本次定位并修复“初见桃子”中第二次七色羽相同对白没有正文的问题：
- 原因是 `RichTextLabel.clear()` 会清空内部解析缓冲，但相同字符串再次赋给 `text` 属性时，Godot 将其判断为属性值未变化，不会重建字符缓冲；因此面板保存的正文和 `label.text` 都正确，但 `get_total_character_count()` 为 `0`。
- `client/scripts/ui/npc_dialogue_panel.gd` 改为先 `clear()`，BBCode 使用 `append_text()`、纯文本使用 `add_text()`，确保每次打开对话都重新解析正文。
- 已使用临时 Godot 场景连续两次显示相同的“啾啾～～啾～啾～”，两次解析字符数均为 `8`；临时测试文件验证后已删除。

## 2026-07-14 时光小屋地图脚本接入

本次仅补齐新地图的客户端场景脚本，不扩展服务端场景数据：
- 新增 `client/scripts/feature/world/scene_levels/time_house_level.gd`，沿用 `NetworkDoorLevelBase`，提供与现有地图一致的 HUD 名称、缩放、出生点、居中和出口配置接口。
- `client/scenes/maps/fashtown/时光小屋.tscn` 根节点已绑定该脚本；地图使用多个中文命名 `TileMapLayer`，因此居中点按全部有效图层的合并边界计算。
- `LeftDoor` 固定使用 `portal_id=7001` 请求进入 `scene_id=2`；东路脚本将该 portal 映射到东侧门内的 `(9, 5)`，但 `_get_door_configs()` 仍不包含 `RightPortal`，因此不能从东路返回。
- 客户端 `WorldSceneRegistry` 注册 `scene_id=7`，服务端 PostgreSQL 仓储与测试桩同步配置 `7 -> 2` 单向出口，并新增迁移 `099_time_house_scene.sql`。
- 新增服务端测试覆盖正向传送成功与东路反向返回失败，确保一次性场景约束不会被后续改动破坏。
- 检查世界渲染链路后，将时光小屋承担房间边界和物理碰撞的 TileMapLayer 从“墙壁”改名为 `Collision`；瓦片、位置和碰撞数据保持不变，使其与其他地图使用相同的边界识别规则。

## 2026-07-14 新增地图 NPC 表单优化

本次只优化后台地图 NPC 表单，不改变接口和数据库结构：
- `admin/src/pages/npcs/NPCConfigPage.tsx` 将实体 ID、实体编码、富文本显示名、所属场景和发布状态重新分区，降低新增时的录入歧义。
- NPC 显示名编辑器占用整行，保留现有 BBCode 刷色能力；实体编码增加小写字母、数字、下划线格式校验。
- 场景下拉展示“场景名（Scene ID）”，支持按名称和 ID 检索；实体类型固定展示为地图 NPC（类型 2）。
- 默认新增值只保留 `entity_type=2` 和启用状态，不再预填测试 ID、测试编码、测试名称或固定场景。
- 弹窗复用 `FIXED_FORM_MODAL_STYLES`，内容较多时在弹窗内部滚动，底部创建/取消操作区保持可见。
- 已执行 `cd admin && npm run build`，TypeScript 编译与 Vite 生产构建通过；仅保留项目既有的大包体积提示。
- 新增 `backend/server/migrations/100_npc_entity_auto_identity.sql`，序列从现有最大 `entity_id` 或 `90000` 之后继续，避免与已有实体冲突。
- PostgreSQL NPC 仓储在单条 `INSERT ... RETURNING` 中生成实体 ID，并写入 `npc_{ID}` 编码；并发新增不再依赖前端计算最大值。
- 后台新增表单不再显示 ID/编码输入框，创建成功转为编辑态后只读展示服务端生成结果；更新接口不会修改实体编码。
- HTTP CRUD 测试新增自动身份断言，并验证生成后的实体仍可正常更新和删除。
- 已执行 `cd backend && GOCACHE=/tmp/pocket-pet-go-cache go test ./server/internal/module/npc ./server/internal/data/postgres ./server/internal/transport/http`，测试通过。
- 按最终录入界面要求，删除地图 NPC 表单中的顶部说明、自动生成说明、所有 `extra` 文案及说明型 placeholder；提交字段和服务端自动生成逻辑不变。

## 2026-07-15 初见桃子技能黑闪与震屏

- `client/剧情动画/初见桃子.tscn` 新增默认隐藏的 `ImpactOverlay/ImpactFlash` 全屏黑色场景节点，不使用脚本动态生成 UI。
- `client/剧情动画/初见桃子.gd` 在冲击波第一帧出现后保持黑屏 `0.06s`，再用 `0.08s` 退回透明，随后按四段短偏移震动剧情相机。
- 冲击波自身的两秒移动 Tween 在反馈演出期间继续运行；剧情退出时统一隐藏黑闪层并恢复同步世界相机后的原始 offset。
- 已使用 Godot 4.7 无头加载项目并单独运行“初见桃子”场景，场景与脚本成功解析；仅出现仓库既有物品图标 ID、macOS CA、ADB 和 ObjectDB 退出警告。
- 修复 `Could not find type "PlotImage"`：剧情字段改为 `CanvasLayer`，图片显示/隐藏通过显式方法调用，避免依赖 Godot 全局类缓存；再次单独运行场景后不再出现该编译错误。

## 2026-07-15 背包物品详情显示修复

- `client/scripts/ui/bag/bag_panel.gd` 在打开详情遮罩时显式显示 `BagItemDetail`，关闭时显式隐藏并清空内容。
- 保留 `bag_item_detail.tscn` 根节点默认隐藏，避免运行时弹窗在编辑器或场景预览中常驻显示。
- 已使用 Godot 4.7 无头启动项目，脚本解析通过；仅出现仓库既有物品图标 ID、macOS CA 与 ObjectDB 退出警告。

## 2026-07-15 闪光镇宠物学校固定剧情

- `client/闪光镇宠物学校.gd` 改为继承 `WorldPlayerCinematic`，实现薇安、桃子、玩家的固定站位和完整串行动作。
- 玩家按当前配置先向下移动 55px并转向左侧，移动完成后桃子再向右移动 50px并切换为右向待机；对白结束后桃子依次向右 20px、向上 70px并隐藏，玩家最后向左移动 120px。
- 桃子和玩家对白使用 BBCode 富文本，角色名称按系统名称格式着色，并从当前场景角色帧提取上半身头像。
- `client/剧情动画/闪光镇宠物学校.tscn` 接入默认隐藏的备用对话面板，保证 F6 单独运行与正式剧情播放器使用同一套剧情内容。
- 剧情场景复用 `plot_image.tscn` 显示顶部闪烁“剧情”图片，脚本在正常完成和提前退出时均显式关闭图片。
- 已使用 Godot 4.7 无头单独加载该剧情场景，场景与脚本解析通过；仅出现仓库既有物品图标 ID、macOS CA 与 ObjectDB 退出警告。

## 2026-07-15 桃子与散文固定剧情

- `client/剧情动画/桃子与散文.gd` 改为继承 `WorldPlayerCinematic`，实现北路地图原点归一化、世界相机参数同步和剧情完成信号。
- 按剧情注释实现玩家向上、桃子转身移动、七色羽分段移动以及桃子与七色羽同时向下移动的完整动作序列；七色羽缺少右向动画时复用左向动画水平镜像。
- 全部对白转为 Godot BBCode：比武大会、闪光平原、橙色风声、粉色心声及红色停顿均由现有富文本对话面板渲染。
- `client/剧情动画/桃子与散文.tscn` 增加默认隐藏的备用对话面板、顶部剧情图片和场景节点构建的 `FadeOverlay/FadeToBlack` 渐黑层，没有使用脚本动态创建 UI。
- 结尾先隐藏顶部剧情图片，再用 `1.2s` 渐变为全黑，随后在黑屏上显示“桃子（想）”内心独白；正常结束和中断退出都会清理视觉层。
- 玩家推进完最后一句内心独白后，脚本显式隐藏 `FadeToBlack` 并重置透明度，再调用 `complete_cinematic()`，避免剧情释放前仍保持黑屏。
- 渐黑 Tween 完成后、内心独白显示前，脚本调用 `_hide_cinematic_actors()` 关闭桃子和七色羽；取消黑屏时不会再次露出已结束演出的角色。
- 已使用 Godot 4.7 无头单独加载剧情场景，场景和脚本解析通过；`git diff --check` 通过且脚本无 Tab，仅出现仓库既有物品图标 ID、macOS CA 与 ObjectDB 退出警告。

## 2026-07-15 对话人物名字字号统一

- `client/scenes/ui/npc_dialogue_panel.tscn` 将 `NpcSpeakerLabel` 与 `PlayerSpeakerLabel` 的 `normal_font_size` 从 `24` 统一调整为与 `ContentLabel` 相同的 `32`。
- 未修改对话脚本、信号链路或服务端数据；名字框仍由现有 `_remeasure_speaker_panel()` 根据文字和头像动态计算尺寸。

## 2026-07-15 剧情场景出战宠物跟随

- `client/scenes/cinematics/common/world_player_cinematic.gd` 统一创建 `CinematicPetFollower`，数据直接读取服务端下发并保存在 `GameState.lineup` 的首只出战宠物摘要，不硬编码宠物或皮肤。
- 跟随表现直接复用 `WorldPetFollower`，因此宠物皮肤解析、待机/行走动画和无效 `skin_id` 隐藏逻辑与世界场景一致。
- 剧情基类根据本地 `Player` 的 Tween 实际位移解析四方向，每移动 `PathFollowController.PATH_STEP_SIZE`（24px）记录一个玩家离开的路径点，再调用同一个路径控制器等速推进宠物。
- 宠物初始位置沿用世界场景规则，按玩家朝向放在身后半格；编队为空时隐藏，`pets_changed` 更新时重新绑定首只出战宠物，剧情退出时断开信号。
- “初见桃子”“闪光镇宠物学校”“桃子与散文”三个现有剧情均已通过 Godot 4.7 无头单独加载和脚本解析；`git diff --check` 通过，基类无 Tab。
- 带真实宠物 `skin_id` 的临时运行态冒烟脚本因执行安全策略未获准，未在本轮自动执行；实际登录态仍需进入任一剧情确认宠物视觉跟随效果。

## 2026-07-15 机械熊剧情泛光颜色修正

- 实际发光节点位于 `client/剧情动画/机械熊.tscn`；`WorldEnvironment` 只负责泛光后处理，绿色来自 `发光特效/ColorRect` 使用的 `StyleBoxFlat.bg_color`。
- 将发光源从 `Color(2.1708035, 5.238883, 2.1782057, 1)` 调整为等 RGB 通道的 `Color(5.238883, 5.238883, 5.238883, 1)`，在保持 HDR 亮度的同时输出白色泛光。

## 2026-07-15 机械熊单节点 Shader 泛光

- `client/剧情动画/机械熊.tscn` 删除 `Environment` 子资源和 `发光特效/WorldEnvironment` 节点，泛光不再作为 Viewport 全局后处理。
- 新增场景内 `canvas_item` Shader 与 `ShaderMaterial`，通过径向距离计算白色核心和衰减光晕，并使用 `blend_add` 叠加到背景，仅作用于 `发光特效/ColorRect`。
- 发光节点由 `10x10px PanelContainer` 调整为以原中心点为基准的 `64x64px ColorRect`；可在 Inspector 的 Shader Parameters 中独立调整 `glow_color`、`glow_intensity` 和 `glow_falloff`。

## 2026-07-15 机械熊剧情 NPC 局部模糊

- `client/剧情动画/机械熊.tscn` 新增 `canvas_item` 九点纹理采样模糊 Shader，并由机械熊与七色羽实例下的 `AnimatedSprite2D` 共用一个 `ShaderMaterial`。
- 通过场景实例子节点覆盖挂载材质，同时保留 `client/npc/dong-lu/机械熊.tscn` 与 `client/npc/dong-lu/七色羽.tscn` 原始资源不变，因此其他地图和剧情中的同名 NPC 不会模糊。
- Shader 默认 `blur_radius=1.0`，仅采样当前 NPC 纹理；白色发光特效、东路地图和场景其余节点不使用该材质。

## 2026-07-15 机械熊固定剧情

- `client/剧情动画/机械熊.gd` 改为继承 `WorldPlayerCinematic`，接入东路/罗克萨斯家地图归一化、世界相机同步、顶部剧情图片、场景头像对白和统一剧情完成信号。
- 东路初始站位配置为桃子 `(75,126)`、七色羽 `(105,134)`、玩家 `(45,160)`、机械熊 `(175,126)`；玩家按右 30px、上 30px、右 10px进入对白位置，三人随后按注释并行向右移动。
- 实现一秒上下后左右震屏、机械熊黑屏揭示、两次黑白闪烁、桃子受惊状态非行走击飞 150px并切换死亡状态，以及玩家/七色羽同步退让。
- 七色羽靠近机械熊后才把共享模糊材质从 `blur_radius=0` 调整为 `1`，同步显示白色 Shader 发光、循环黑白闪烁和短促震动，随后渐黑。
- 黑屏对白结束后隐藏东路角色和特效，切换到 `roxus_house.tscn`，把玩家放到 `(115,180)`，渐亮后显示罗克萨斯与玩家对白。
- `client/剧情动画/机械熊.tscn` 新增桃子、玩家、罗克萨斯家、备用对话面板、剧情图片、黑白闪烁/渐黑场景节点和罗克萨斯头像资源；UI 均由场景节点构建。
- 屏幕闪烁函数改为 `_play_black_flash()`，遮罩资源默认色同步改为黑色，不再产生白屏；黑屏显示与间隔仍各保持 `0.07s`。
- `CINEMATIC_MOVE_SPEED` 从 `100` 降为 `50px/s`，并行移动、机械熊接近及桃子击飞段的 Tween 时长同步翻倍；NPC 序列帧与玩家动画节点统一使用 `speed_scale=0.5`。
- 三人右移动作仍分两阶段，但移除共用 `1.4s` 的强制同步：第二阶段为三个独立线性 Tween，桃子与七色羽移动 `40px/0.8s` 后各自停止，玩家移动 `70px/1.4s` 后停止。
- 已使用 Godot 4.7 无头单独加载机械熊剧情场景，场景、脚本和 Shader 解析通过；`git diff --check` 通过且脚本无 Tab，仅出现仓库既有物品图标 ID、macOS CA 与 ObjectDB 退出警告。

## 2026-07-15 机械熊分帧攻击演出调整

- `client/剧情动画/机械熊.gd` 在机械熊解除黑屏并显示时追加短促震屏，不改变后续对白内容。
- 桃子与机械熊先向相反方向同时后退 `15px`，再以两次纯黑闪烁分别切换机械熊资源已有的“攻击第一帧”和“攻击第二帧”，两帧之间保持 `0.5s`。
- 第二帧出现后才启动桃子的受惊退让，以及玩家和七色羽原有的并行退让；角色仍按各自 Tween 时长自然停止。

## 2026-07-16 任务领取与交付动画

- 服务端任务模板、PostgreSQL 仓储、后台 CRUD 和迁移统一增加 `accept_animation_key`、`submit_animation_key`。
- 直接任务请求与 NPC 菜单任务操作都会在成功响应中返回对应动画键，失败响应不触发动画。
- `QuestController` 在成功响应后请求主场景播放动画；提交结算载荷会暂存到动画结束，再沿用现有升级与奖励弹窗链路。
- 主场景复用现有剧情播放器，并对任务动画和场景触发动画做顺序保护；播放期间继续锁定运行时菜单与玩家交互。

## 2026-07-16 桃子 NPC 碰撞与交互补齐

- 桃子 NPC 场景现在与罗思使用相同的交互结构：根节点继续绑定 `InteractiveNPCBase`，`Area2D` 负责进入范围提示，`StaticBody2D` 负责世界移动阻挡。
- 交互矩形为 `16x7px`，实体阻挡矩形为 `14x5px`，二者都放在角色脚底 `y=-3.5px`，避免身体贴图产生过大的不可通行范围。
- 桃子的客户端 `entity_id` 已与迁移 `101_story_scene_entry_trigger.sql` 中的服务端实体 `92001` 对齐，NPC 菜单请求能够命中正确实体。

## 2026-07-16 镇北兔子 NPC

- 创建兔子独立打包场景，根节点配置 `entity_id=92002`、`npc_code=rabbit`、显示名“兔子”，纹理帧保持为空，由开发者后续在 Inspector 中设置。
- 镇北 `bei_lu.tscn` 引用兔子打包场景并放置在 `(120,120)`；交互和阻挡节点与桃子保持一致。
- 服务端迁移仅把兔子注册到场景 `4`；坐标和朝向由 `bei_lu.tscn` 维护，后续移动兔子时无需修改服务端实体定义。

## 2026-07-16 主线任务数据清理脚本

- 生成 `backend/server/migrations/104_delete_main_quest_data.sql`，供开发者手动删除数据库中的全部主线任务配置和玩家主线运行数据。
- 清理范围包括任务模板、玩家任务、玩家目标、任务事件、主线 NPC 菜单绑定、主线场景触发器及其他任务中的前置主线引用。
- 已领取的历史奖励和奖励审计不在删除范围内，避免任务配置清理意外回滚玩家资产。

## 2026-07-16 《主线·旅行的起点》1/5 至 3/5

- 玩家仍从时光小屋出生；首次进入东路播放“初见桃子”，动画结束后显示提示，确认后服务端写入剧情 flag 并开放桃子 NPC。
- `1101` 从桃子接受，经对话引导到市场璃梦；`1102` 在璃梦处以前置任务解锁；`1103` 在罗格处以前置任务解锁。
- 接受、取消、任务对白、目标完成、自动交付、金币奖励和阶段提示全部由 SQL 配置及服务端对话副作用驱动。
- 客户端复用 `NPCDialoguePanel` 的选项能力和 `ConfirmPromptPopup` 的提示能力，没有新增脚本动态 UI。
- 后台 NPC 剧情编辑器同步增加“交付任务ID”，后续可视化编辑不会丢失 `submit_quest_id`。
- 桃子不再只存在于剧情场景：东路世界地图实例化桃子，并由服务端个人可见性快照控制显示、交互区和实体碰撞。

## 2026-07-16 NPC 注册迁移字段修复

- 根据迁移 `048_npc_scene_only_placement.sql` 后的正式表结构，移除 `105_main_journey_start_quests.sql` 对桃子坐标、朝向和速度字段的更新。
- 同步移除 `103_north_road_rabbit_npc.sql` 对相同废弃字段的插入与冲突更新，避免在当前数据库结构执行时报 `42703`。
- NPC 的服务端实体定义仅维护身份、场景归属和状态；桃子与兔子的具体位置分别由东路、镇北客户端地图场景资源维护。

## 2026-07-16 任务模板编辑表单修复

- 移除任务模板页签的隐藏销毁行为，并恢复 Ant Design Form 默认字段保留机制，编辑过程中来回切换页签不会丢失尚未保存的内容。
- 任务基础信息中的起始 NPC、目标 NPC 和阶段编辑器中的目标 NPC 统一复用服务端 NPC 实体列表，支持按名称或 ID 搜索选择。
- NPC 列表通过现有鉴权请求层按每页 100 条完整分页加载；表单最终仍向后端提交实体 ID，不改变现有任务模板契约。

## 2026-07-16 剧情场景统一黑屏转场

- `CinematicPlayer` 绑定主场景资源中已有的 `TransitionOverlay`，不在脚本中动态创建新的 UI，也不要求每个剧情场景重复维护转场节点。
- 播放顺序统一为“世界渐黑 → 全黑挂载独立剧情场景 → 剧情渐亮”；结束顺序统一为“剧情渐黑 → 全黑释放剧情场景 → 世界渐亮”。
- 剧情实例在退出渐黑完成前不会释放，避免世界画面提前闪现；最终 `cinematic_finished` 在世界重新渐亮后才发出，保证服务端 Ack、对话推进和奖励展示时序不抢跑。
- 取消剧情会停止尚未完成的转场补间、释放剧情实例并立即恢复遮罩和世界交互状态。

## 2026-07-16 初见桃子剧情卡死修复

- `初见桃子.gd` 在隐藏顶部剧情图片后显式调用统一完成入口，使 `finished` 信号能够驱动退出渐黑、释放场景和服务端 Ack。
- 主场景收到 `SCENE_TRIGGER_PUSH` 时如果地图仍在转场，会缓存该触发载荷；目标地图加载完成后不执行世界渐亮，保持全黑并直接启动剧情播放器。
- 剧情播放器检测到遮罩已经全黑时直接挂载剧情场景，不重复等待空的渐黑 Tween；首次从时光小屋进入东路不会再闪现东路世界画面。

## 2026-07-16 后台永久删除禁用账号

- 玩家管理页根据状态展示不同高风险操作：非删除状态只能“禁用账号”，`status=0` 时才显示“永久删除”，确认框明确列出全部不可恢复的数据范围。
- 后端新增独立永久删除路由与 `players:purge` 权限；仅靠伪造前端请求无法跳过禁用状态检查或权限检查。
- 仓储根据目标 `player_id` 锁定所属账号和账号下全部玩家，只有账号及全部玩家状态均为 `0` 才执行事务清理。
- 清理以账号为边界，覆盖该账号下所有玩家及玩法数据；装备引用、宠物引用和快照子表先删除，最后删除玩家与账号主记录。
- 已通过玩家 PostgreSQL 仓储、后台 HTTP 包测试及后台 TypeScript/Vite 构建；数据库迁移只生成到迁移目录，未自动执行。

## 2026-07-26 闪光镇地图节点 UI

- 通过 Godot AI MCP 创建地图面板场景和主场景入口，全部尺寸、纹理与标点按钮均保存在场景节点中，没有由脚本动态生成 UI。
- 7 个标点对应现有闪光镇地图素材，点击、屏幕上下按钮以及键盘/手柄方向输入都会同步唯一选中态和地点名称。
- 新增独立地图面板控制器：打开时关闭其它运行时根面板并锁定世界输入，关闭时恢复输入，进入战斗时自动收起。
- 服务端权威快速传送已接入：当前标点再次点击发送 `MOVE_INTENT_REQ(map_teleport=true)`；服务端从迁移 `108_world_map_teleport_nodes.sql` 读取开放状态和目标地图中心格，持久化后沿用 `MOVE_INTENT_RESP -> WORLD_RESYNC_PUSH`。
- 1–6 号场景开放快速传送；未落地地图场景的闪光平原只展示“尚未开放”，一次性时光小屋不进入快速传送表。同地图传送会回到中心格，但不会重复触发场景进入任务事件。
- 已通过 Godot MCP 校验地图中心边界、7 个节点导出配置、二次点击信号和未开放提示；服务端新增跨地图与同地图快速传送测试。

## 2026-08-02 闪光平原人物切换与传送

- 按服务端权威原则为闪光平原 17 张已落地地图分配并注册 `scene_id=9..25`，数据库变更输出为 `backend/server/migrations/112_shining_plain_scenes.sql`，未直接执行数据库迁移。
- 在 `backend/server/internal/data/postgres/world_repo.go` 配置普通门拓扑和安全出生格，并同步 `backend/server/internal/teststub/repos.go` 与逐门单元测试。
- 在 `client/scripts/feature/world/world_scene_registry.gd` 注册客户端地图资源；`shining_plain_level.gd` 统一读取场景资源中的门配置，`shining_square_level.gd` 接通广场六个现有入口。
- 为 `闪光平原宠物学校.tscn`、`办公区.tscn`、`家族会馆.tscn` 增加必要的返回门碰撞节点，其余地图复用已有 `Area2D` 门节点。
- 未接通不存在目标资源的门：商业区到闪光平原传送区、准备区到战斗区、闪光海岸/精灵大厅到海道；这些门不会向服务端发送无效切图请求。
- 已通过闪光平原 17 张地图 Godot headless 加载、32 个启用传送门契约检查和全部目标出生格物理检查；并通过 `go test ./server/internal/data/postgres`、`go test ./server/...`。

## 2026-08-04 闪光平原普通门出生坐标补齐

- 根因是闪光平原通用地图脚本始终返回无效入口坐标，闪耀广场也只识别从闪光镇传送区返回的单一来源门，导致普通门切图请求缺少目标地图出生格。
- 在 `client/scripts/feature/world/scene_levels/shining_plain_level.gd` 增加可由场景资源配置的来源门出生坐标字典，同时兼容 `.tscn` 字符串键以及运行时整数键。
- 在 `client/scripts/feature/world/scene_levels/shining_square_level.gd` 配置六个返回广场的独立入口，并为闪光平原 15 张存在启用入口的地图补齐场景级坐标。
- 保持服务端权威链路不变：客户端只提供目标入口坐标，服务端继续校验 `portal_id` 与目标场景关系、持久化位置并回传权威快照；无需新增迁移或修改协议。
- 已用 Godot headless 实例化 17 个目标场景并校验 33 条普通门入口映射；随后检查相同 33 个出生格均有地图瓦片且无瓦片物理阻挡。

## 2026-08-04 海道场景接入任务总结

- 根因是闪光海岸已有“通往海道”碰撞区，但场景门配置、客户端场景注册和服务端 `worldScenes` 均缺少海道，导致检测区未绑定切图信号或请求被服务端拒绝。
- 复用用户新增的 `client/scenes/maps/闪光平原/海道.tscn`，为根节点配置共享地图脚本、两条门请求和两个来源门出生坐标，没有修改海道地图瓦片、碰撞区位置或边框布局。
- 在闪光海岸与精灵大厅启用海道门，并在 `world_scene_registry.gd` 注册 `scene_id=26`；世界地图海道热点同步启用服务端权威快速传送。
- 服务端 PostgreSQL 世界仓储、测试桩和普通门坐标测试补齐 `23 <-> 26 <-> 25` 拓扑，防止非法门编号或不匹配目标绕过权威校验。
- 新增 `backend/server/migrations/114_shining_plain_seaway.sql`，供用户手动注册海道场景和快速传送中心格；本任务未直接修改数据库。
- 验证结果：Godot 4.7 Headless 确认四个门信号已连接、四个出生格可通行、`scene_id=26` 与海道热点有效，主场景无解析错误；全部 Go 服务端测试通过。

## 2026-08-04 闪光平原地图当前场景定位任务总结

- 根因是地图面板初始化后只保留当前正在查看的地区，`open_menu()` 未根据服务端世界快照重新判断玩家所在地区，因此闪光平原地图打开时可能仍停留在闪光镇。
- 在 `client/scripts/ui/world/map_teleport_panel.gd` 复用各热点按钮已有的 `target_scene_id`，打开面板时从 `GameState.scene_snapshot.scene_id` 查找所属地区并切换地图，没有新增客户端坐标或重复场景映射。
- 地区切换后继续复用现有 `_select_current_scene_or_first_point()` 和 `_refresh_current_scene_icon()`，保证选择动画、地点名称、焦点和人物当前位置图标都定位到当前地图热点。
- 闪光平原 `scene_id=9..26` 的热点继续支持二次点击传送，客户端只提交 `target_scene_id`，目标中心格仍由 `world_map_teleport_node` 数据库配置和服务端权威校验。
- 验证结果：Godot 4.7 Headless 逐场景检查 `scene_id=9..26`，全部自动显示闪光平原地图、选中当前热点、正确定位两类光标，并发出对应传送信号。


## 2026-08-05 闪光平原点击传送保持原场景任务总结

- 根因是当前数据库 `world_map_teleport_node` 只查询到 `scene_id=26`，闪光平原地图热点提交的 `scene_id=9..25` 没有对应的快速传送中心配置；服务端返回 `map teleport unavailable` 后，客户端按既有失败流程保持原场景。
- 新增 `backend/server/migrations/115_repair_shining_plain_map_teleport_nodes.sql`，使用 `INSERT ... ON CONFLICT DO UPDATE` 幂等补齐 `scene_id=9..26` 的中心出生格与开放状态。
- 未修改客户端脚本、场景节点、WebSocket 协议或服务端权威传送逻辑，避免影响普通传送门和现有地图交互。
- 本次未直接执行数据库迁移；用户执行迁移后，闪光平原热点的二次点击会重新走现有 `MOVE_INTENT_REQ(map_teleport=true)` -> `MOVE_INTENT_RESP` -> `WORLD_RESYNC_PUSH` 权威传送链路。

## 2026-08-05 闪光海岸树木遮挡任务总结

- 根因是闪光海岸的树木使用地图根节点下的独立 `TileMapLayer`，人物和跟随宠物则由世界控制器挂入另一个启用 Y-Sort 的 `ActorRoot`；两者不在同一排序分支，因此无法按树根与角色脚底动态排序。
- 保留原 `树`、`树2` 图块层及其碰撞数据，仅关闭固定树木绘制；将左侧边缘、左侧、右侧和中央四棵完整棕榈树作为 `Sprite2D` 放入预置 `ActorRoot`，排序节点位置分别对齐实际树根。
- 没有修改树木碰撞、地图瓦片、人物/宠物脚本、网络协议或服务端逻辑，避免影响现有移动和联机同步。
- Godot 4.7 Headless 专项验证已确认场景可加载、`ActorRoot` 开启 Y-Sort、四棵树使用完整区域且排序原点位于树根，原图块碰撞仍保留。

## 2026-08-05 翡翠梦境树木遮挡任务总结

- 根因与闪光海岸一致：翡翠梦境树木位于地图根节点的独立 `TileMapLayer`，人物与跟随宠物位于世界控制器使用的 Y-Sort `ActorRoot`，两者不在同一排序分支。
- 还原原树层后确认右侧存在两棵棕榈树，树根分别为 `(200,144)` 与 `(184,208)`；右上角另有一个用于维持地图边缘原貌的单格树叶残片。
- 隐藏原树层绘制但保留全部图块和两处树根碰撞，将两棵树以完整裁剪精灵接入预置 `ActorRoot`；没有修改人物、宠物、移动、联网同步或服务端逻辑。
- Godot 4.7 Headless 已确认场景可加载、`ActorRoot` 开启 Y-Sort、树根坐标与裁剪区域正确、碰撞未丢失；重建画面与原画面没有显著像素差异。

## 2026-08-05 报名区进入准备区出生格任务总结

- 根因是 `client/scenes/maps/闪光平原/准备区.tscn` 将来源门 `portal_id=17002` 的出生格误配置为 `(-4,1)`，导致客户端普通门请求提交地图外坐标。
- 根据准备区左侧“通往报名区”门的场景位置和 16px 场景格，修正为门内侧的 `(2,7)`；该坐标与服务端正式仓储、测试桩及既有普通门坐标契约一致。
- 没有修改 `portal_id=17003` 的比武区入口、门拓扑、WebSocket 协议、服务端权威校验或数据库结构。
- Godot 4.7 Headless 已确认场景解析正常、`17002` 返回 `(2,7)`、目标格有地图瓦片且无物理阻挡；相关 Go 仓储测试通过。
