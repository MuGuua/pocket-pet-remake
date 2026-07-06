# 最新变更记录

## 2026-07-06
- Web 本地调试画布改为铺满浏览器窗口：`client/autoload/web_runtime_canvas.gd` 在 Debug Web 运行时不再强制 `780:1440` 竖屏比例，避免桌面浏览器调试时画布宽度过窄、可视视口显示不全；正式 Web 构建仍保留 `780:1440` 移动端比例约束
- Web 画布尺寸策略改为“固定比例、允许同比例缩放”：`client/autoload/web_runtime_canvas.gd` 不再把浏览器 `canvas` 锁死为 `780x1440` 像素，而是统一按 `780:1440` 纵横比在当前可视区域内自适应缩放；登录页与主运行态都会共用同一套全局比例约束，既避免拉伸变形，也兼容不同浏览器窗口大小
- Web 运行时世界渲染改为固定设计尺寸：`client/scripts/feature/world/world_controller.gd` 在 Web 环境下不再把内部 `SubViewport` 跟随 `GameShell` 当前尺寸，而是强制按 `780x1440` 渲染，专门兜底 `tmp_js_export.html` 这类临时调试页把实际内部视口压成 `621x834` 的问题
- Web 导出壳层改为固定比例盒子：`client/export_presets.cfg` 的自定义 CSS 不再强制固定 `#canvas` 像素尺寸，而是统一使用 `780:1440` 纵横比居中显示，确保正式导出页与运行时兜底策略一致
- 客户端新增全局静音背景音保活器：`client/autoload/background_audio_keeper.gd` 通过 `AudioStreamGenerator` 持续播放 0 振幅静音流，并作为自动加载单例贯穿所有场景；主要用于尽量减少切后台时音频上下文被立即回收的概率，但浏览器/移动系统后台冻结仍不保证完全不断线
- 修复主菜单运行态节点绑定报错：`client/scripts/ui/main_menu.gd` 不再只依赖 `%TabsRow` / `%ItemsList`，当 unique-name 绑定因 owner 变化失效时会回退到稳定场景路径，避免 `_ready()` 因 `Node not found` 中断主菜单初始化
- 修复市场地图 TileSet 越界：`client/scenes/maps/fashtown/radiant_market.tscn` 的 `TileSetAtlasSource_rbgk4` 删除超出 `Tilemap_Platform.png` 高度上限的 `0:24`~`0:27` 条目，避免切图到市场时触发 `Cannot create tile` 与 `atlas has no tile` 报错
- 客户端网络入口切换为统一多环境解析：`client/autoload/network_config.gd` 现支持 `local`、`remote`、`browser_origin` 三种环境，并区分原生端默认环境与 Web 默认环境，避免再分别修改 HTTP / WebSocket 脚本
- Web 调试支持灵活切服：浏览器运行时优先读取地址栏参数 `server`、`http_base`、`ws_base`，其次读取 `localStorage` 中的 `pp_server_profile`、`pp_http_base`、`pp_ws_base`；便于本地打开导出页时在本地后端、远程服务和同源部署之间快速切换
- `http_client.gd` 与 `net_client.gd` 不再各自强制改回当前页面同源地址，统一改为读取集中配置解析结果，修复 `http://localhost:8060/tmp_js_export.html` 调试时请求总被锁到 `http://localhost:8060` 的问题
- 登录页新增开发切服面板：`client/scenes/ui/common/dev_server_switcher.tscn` + `client/scripts/ui/common/dev_server_switcher.gd` 支持运行时切换本地后端 / 远程服务 / 浏览器同源，应用后会同步刷新 HTTP / WebSocket 入口并清理旧会话，避免把旧服 token 带到新服

## 2026-07-03
- 奖励弹窗右上角关闭按钮增加专属直连：`reward_popup.gd` 现在对 `%TopCloseButton` 显式绑定 `pressed -> close_popup()`，不再只依赖模态基类的通用关闭按钮链路
- 奖励弹窗模板节点改为可选兜底：`reward_popup.gd` 不再强依赖 `PlainLineTemplate` 等模板节点，场景少某个模板时会自动回退到运行时创建默认行，避免 `_ready()` 因 `Node not found` 直接报错
- 修复奖励弹窗右上角关闭按钮仍无响应：`reward_popup.gd` 改为像确认弹窗一样只走 GUI 点击链路，不再启用基类全局 `_input` 吞事件，并补齐标题面板的鼠标交互
- 修复奖励弹窗标题与正文面板重叠：`reward_popup.tscn` 的正文容器改为在 `VBoxContainer` 中按正常高度参与排版，移除把内容向上顶回标题区域的负偏移，恢复两个边框格子的间距
- 奖励弹窗正文布局改为以场景模板节点为准：`reward_popup.gd` 不再在脚本里硬编码奖励行字号、图标尺寸和行布局，改为复制 `reward_popup.tscn` 中的文本/富文本/物品行模板，后续可直接在场景里调布局
- 战斗进场网格转场放大：`grid_spread.gd` 提高 `spacing`，`grid_spread.tscn` 放大圆点 `QuadMesh` 尺寸，让转场网格更疏、圆圈更大
- 战斗进场网格转场提速：`grid_spread.gd` 下调 `stagger_delay` 与 `scale_duration`，让世界进入战斗时的铺展/揭开动画更利落，减少等待感
- 适配重构后的奖励弹窗场景层级：`reward_popup.gd` 现在按 `reward_popup.tscn` 新结构重新绑定 `DimLayer`、正文 `CenterContainer` 与 `PanelContainer`，恢复遮罩关闭、正文点击拦截和右上角关闭按钮交互
- 修复背包装备预览面板节点路径错误：`equipment_preview_panel.gd` 改为匹配 `equipment_slot_1.tscn` 真实层级，移除不存在的 `MarginContainer` 路径，恢复背包运行态初始化
- 修复战斗结算奖励弹窗右上角关闭按钮无响应：`reward_popup.gd` 现在在打开前显式恢复正文面板与 `TopCloseButton` 的可交互鼠标过滤，并把 `close_popup()` 收敛到 `_dismiss_modal()`，保证点击 `X` 时能正常关闭并广播 `popup_closed`
- 客户端世界场景改为 3 倍放大显示：修正世界相机缩放方向判断后，`player.tscn` 的 `camera_zoom_scale` 调整为 `3.0`，在当前统一 `780x1440` 设计分辨率下，让地图、人物与同层世界内容按旧 `260x480` 视野放大约 3 倍显示，而不影响 UI 分辨率
- 客户端统一设计分辨率切换为 `780x1440`：`project.godot` 的 `viewport` 与 `window override` 同步放大，`world_controller.gd` 的默认渲染基准与 `main.gd` 的战斗弹窗固定尺寸也一并调整到同一口径，保证 UI、地图、角色与战斗继续共用同一套分辨率
- 修复主菜单场景缺少 `TabsRow` / `ItemsList` 内容节点导致 `main_menu.gd` 在 `_ready()` 期间报 `Node not found`；`main_menu.tscn` 已补齐 `MenuFrame/Content` 下的最小容器结构，恢复主菜单初始化

## 2026-07-02
- 新增公开注册接口 `POST /api/v1/auth/register`：当前只要求账号、密码和男女形象选择；服务端复用账号名作为玩家名，男性默认 `初始形象男_001`、女性默认 `初始形象女_002`，并沿用正式玩家初始属性/背包/钱包创建链路
- 客户端登录页新增注册入口与初始形象选择：输入账号密码后可直接注册男女角色；注册成功后自动登录并进入世界；注册表单新增确认密码与基础前端校验，减少误输密码导致的无效注册
- 主世界 HUD 在背包按钮左侧新增“设置”按钮；设置菜单复用通用 `ActionMenuPopup`，提供“返回登录页”“退出游戏”两个入口，其中退出游戏补充二次确认弹窗，分别沿用现有登录页切换清理链路与客户端退出链路
- 客户端网络入口改为集中配置：新增 `client/autoload/network_config.gd`，统一维护 HTTP / WebSocket 地址；默认启用正式服 `117.72.124.51:9002`，并保留注释态本地 `127.0.0.1:8080` 配置，便于开发时通过切换注释在本地与服务器之间快速切换
- `http_client.gd` 与 `net_client.gd` 不再各自硬编码地址；浏览器 Web 导出时也统一从 `network_config.gd` 读取端口，避免只改原生端地址却漏改 Web 入口

## 2026-06-29
- 客户端运行态面板打开链路继续统一 loading：`main.gd` 中「个人状态」快捷键与主菜单入口改为和背包一致，先展示全屏通用 loading，等待 `ENTER_WORLD_REQ + BAG_LIST_REQ + WALLET_QUERY_REQ` 权威数据就绪后再打开 `player_status_panel`；避免旧数据先展示、再闪成新数据
- 客户端面板预加载判定继续收严：`player_status_panel.gd` 与 `bag_panel.gd` 在 opening loading 阶段必须等待各自依赖请求全部成功（人物面板：人物/背包/钱包；背包面板：背包/已穿戴装备）后才允许打开，避免某一条子请求失败时仍带旧快照展示
- 背包已打开后的二次刷新继续收严：普通 `USE_ITEM` 与装备 `REPAIR` 不再在回包到达时立即结束 loading，而是等待后续背包快照真正写入 `GameState` 后再关闭 loading / 提示成功，减少旧数量、旧修复石库存或旧损坏状态闪现

## 2026-06-28
- 强化成功率穿戴等级段：迁移 `070_equipment_enhance_success_required_level_band.sql`（`equipment_enhance_success_config` 复合主键 `target_level + required_level_min`，每10级穿戴段独立配置）；后台「强化成功率」Tab 增加穿戴等级段筛选；强化/预览按装备 `required_level` 解析段并查表；`enhance_preview` 新增 `required_level_band_label`
- 强化材料锻造属性：迁移 `069_equipment_enhance_material_config.sql`（按 `item_id` 配置成功率模式/失败惩罚）；后台物品页在 `item_sub_type=equipment_enhance` 时展示「锻造属性」编辑器；Admin API `GET/PUT /api/admin/equipment-enhance-success-configs` + 物品模板页「强化成功率」Tab 维护 `equipment_enhance_success_config`；强化事务按所选材料计算有效成功率并在失败时分支（损坏 / 降级 / 无惩罚）；`enhance_preview.materials[]` 下发 `effective_success_rate_pct` / `failure_penalty_label`；强化回包新增 `failure_penalty`
- 装备强化失败损坏与修复：迁移 `068_equipment_damaged_repair.sql`（`equipment_instance.is_damaged`、修复宝石 3202、`equipment_repair_cost`）；强化失败时将实例标记为损坏（等级不变、材料仍消耗）；损坏装备不可佩戴/不可强化；WS `2078/2079` 修复接口消耗修复宝石 ×1 清除损坏；背包快照新增 `is_damaged` / `repair_preview`；客户端背包格子红色损坏样式、详情「已损坏」标签、主按钮「修复」+ 确认弹窗

## 2026-06-27
- 消耗品使用效果列表化：后台 `effect_params_json.use_effects[]` 支持人物/宠物/装备/系统多类字段配置；服务端 `use_effects` 通用执行器在同一事务内落地（兼容旧版 `pet_hp_restore` / 扩容 / 神符槽等单效果）；`5021 USE_ITEM_REQ` 新增可选 `target_item_uid`（装备类效果目标）；`5022` 回包新增 `applied_effects` / `needs_wallet_push`；客户端背包新增目标选择弹窗，支持 `pet_single` 选宠与 `equipment_single` 选装备后再 USE_ITEM
- 背包丢弃实例化物品：`DROP_ITEM_REQ` 新增 `item_uid`；服务端按 `player_id + container_type + item_uid` 定位格子并整格丢弃，同步删除 `equipment_instance`；客户端实例化物品（含多件相同装备）丢弃时传 `item_uid`
- 背包丢弃可堆叠物品：确认弹窗新增数量选择（-/+/最大），客户端按所选 `quantity` 部分丢弃；`DROP_ITEM_RESP` 回传 `item_uid` 供提示与刷新
- 战斗/奖励弹窗物品名：4013 结算包 `rewards[]` 在构建时通过 `itemService` 补全缺失的 `item_name`（战斗快照回退路径此前只有 `item_id`）；客户端 `RewardPopup` 在服务端字段缺失时从 `GameState` 背包/已佩戴快照兜底
- 背包丢弃：`5121/5122 DROP_ITEM` 服务端权威丢弃（校验 `can_drop`、写变更日志、推送 `5011`）；客户端「更多 → 丢弃」二次确认后走 loading 请求；背包快照新增 `can_drop` 字段供 UI 控制丢弃按钮
- 系统装备管理「可丢弃」：后台装备模板表单新增 `can_drop` 开关（新建默认开启）；Admin API 与 `item_definition.can_drop` 读写贯通，修复创建时 `can_store`/`is_enabled` 占位错位
- 丢弃确认弹窗：`ConfirmPromptPopup` 关闭基类全局 `_input` 吞事件，修复确定/取消按钮点击无响应
- 丢弃回包刷新：`App._request_cmd_for_response` 补齐 `DROP_ITEM_RESP -> DROP_ITEM_REQ` 映射，修复丢弃成功后 UI 一直等待超时、背包不刷新；`BagController.drop_item_responded` 作为 UI 完成兜底，并修正 `_finish_drop_action` 提前 return
- 装备强化铜币消耗：迁移 `065_equipment_enhance_cost_gold_copper.sql` 为 `equipment_enhance_cost` 增加 `cost_gold_copper`（总铜币真值）；`enhance_preview.cost_gold_copper` 由服务端填充；强化成功/失败均扣铜币并推送 `WALLET_UPDATE_PUSH`；客户端强化页展示消耗铜币与当前金/银/铜余额，余额不足时禁用强化按钮
- 装备强化铜币公式可配置：迁移 `067_item_equipment_enhance_gold_cost.sql` 在 `item_equipment_extra` 增加每件装备独立的强化铜币公式（默认基础 100 铜、每级固定 +200）；运行时按装备模板 `item_id` 计算 `enhance_preview.cost_gold_copper` 与强化扣费；系统装备管理页「编辑装备」弹窗内可配置并预览 +1~+15 消耗

## 2026-06-23
- 客户端新增新版背包一期：主菜单“物品行囊”打开 `bag_panel.tscn`，通过 `BAG_LIST_REQ/USE_ITEM_REQ` 与服务端交互；新增本地 `ItemIconRegistry` 仅映射 `icon_key -> Texture2D`，物品数量/可用行为仍来自服务端快照。
- 客户端左上角头像入口改为打开新版 `player_status_panel.tscn`；旧人物弹窗及其背包/组队/技能旧页资源已移除，新面板直接读取 `GameState` 服务端权威快照展示战斗属性、状态抗性和社会属性。
- 客户端人物信息新面板三个分页按钮补齐 hover 样式，悬停态复用按下态贴图与字体颜色，保证只有普通/按下两种视觉状态。
- 客户端人物信息新面板 `player_status_panel.tscn` 接入分页脚本：打开默认选中“战斗属性”，三个分页按钮保持单选按下/弹起状态，并同步切换战斗属性、状态抗性、社会属性内容面板。

## 2026-06-18
- 战斗伤害公式切换为《口袋伤害计算新表》链路：分子 `(A×SkillMult)×(爆伤链)/100×(1−技能抗性差/100)−D`、分母 `1+Guard×(0.001|0.01)`、综合乘子含天赋/元素/抗类与全局 0.5；删除旧 `def/(def+K)` 与 block 叠乘逻辑；爆伤链直接进分子（不再独立掷暴击骰）；`skillDef` 新增 `skill_mult`/`skill_crit_add`（缺省 `attack_pct/100`）；`actorRuntime` 新增 `guard`/`talent_dmg_pct`/`talent_reduce_pct`/元素字段
- 口袋伤害 DB/Admin：迁移 `062_skill_pocket_damage_fields.sql`（`skill_mult`/`skill_crit_add`）、`063_combat_pocket_damage_stats.sql`（player/player_pet/monster 的 guard/天赋/元素 + 宠物封顶）；技能页/怪物页/宠物实例页可配置上述字段
- 战斗控制双体系：`seal_chance_pct`/`control_chance_pct` 概率无视抗性；`seal_power`/`control_power` 威力对抗控制抗性（差值≥50 稳控，每缩小 1 点降 2%）；迁移 `061_skill_control_power.sql`；文档 `backend/docs/battle-control-effects.md`；Admin 系统技能页「效果」Tab 可配置封印/控制双体系字段
- 人物装备系统 P2 强化：迁移 `060_player_equipment_enhance_cost.sql`（材料表 + 强化石 3201）；WS `2076/2077`；**仅未佩戴且位于背包**时可强化；扣材料 → 掷骰 → 成功升一级（失败不掉级）；客户端背包「强化」按钮；`enhance_preview` 含强化等级行 + 可强化属性行 + `materials` 列表；请求可传 `cost_item_id` 选择强化材料；背包 category `enhance_material` 按 `item_sub_type=equipment_enhance` 筛选
- 人物装备系统 P1：运行时佩戴/卸下 WS `2070`–`2075`；`player_equipment_slot` 事务写入；`equipment/stats.go` 属性重算 + `pet_combat_stat_cap` 截断；背包装备无 `item_uid` 时自动创建 `equipment_instance`；客户端背包「穿戴」、状态页「已佩戴装备/卸下」；`EnterWorld.player.equipped_items`
- 人物装备系统 P0：迁移 `058_player_equipment_foundation.sql`、`059_admin_equipment_permissions.sql`；`module/equipment` Admin CRUD；后台「系统装备管理」页 `/equipment-definitions`
- 玩家人物装备系统设计文档 `backend/docs/player-equipment-system.md`：13 部位、强化成功率、药囊战后恢复、时装纯外观、镶嵌无损取下、属性全额叠加、与宠物共用 `pet_combat_stat_cap`
- 宠物战斗属性封顶 Admin：迁移 `057_admin_pet_combat_stat_cap_permissions.sql`；`/api/admin/pet-combat-stat-caps` GET 列表 + PUT 按 stat_key 更新；后台「战斗属性封顶」页 `/pet-combat-stat-caps`
- 玩家宠物实例独立管理页 `/player-pets`：跨玩家筛选 pet_uid / player_id / pet_id，分页列表 + 详情/编辑/新增/删除/出战开关
- 宠物技能分槽一期：迁移 `054_pet_skill_slots.sql`（天生/神符/普通/法宝槽 + `pet_artifact_equipment`）；设计文档 `backend/docs/pet-skill-slots.md`
- 神符槽道具解锁：`5021 USE_ITEM` 扩展 `effect_type=pet_talisman_slot_unlock`，读取 `pet_skill_slot_unlock_item` 配置；迁移 `055_admin_pet_skill_slot_unlock_permissions.sql` 后台权限
- 法宝装备/卸下 WS：`3031/3032` 装备、`3033/3034` 卸下；技能详情 `3035/3036`；物品 `effect_type=pet_artifact`
- Admin：神符槽解锁配置页 `/pet-skill-slot-unlock`；宠物模板支持编辑 `innate_skill_ids` / `normal_skill_ids`；玩家宠物编辑支持次要战斗属性（精力/命中/抗性等），保存时服务端封顶截断
- 客户端：宠物状态页「查看技能」面板（3035）、背包 usable 道具使用 + 选宠物（5021 + `target_pet_uid`）；法宝装备双入口（背包「装备」3031、技能面板空槽「装备」）
- 可选测试种子：迁移 `056_seed_pet_skill_slot_items.sql`（神符解锁符 3010/3011、法宝 3020/3021）
- 服务端 `module/pet/skill_slots.go` 合并战斗可用技能；`PetDetail.skill_slots` 协议字段（列表页隐藏法宝技 skill_id）
- 宠物战斗属性封顶：迁移 `053_pet_combat_stat_caps.sql` 扩展 `player_pet` 次要战斗字段与 `pet_combat_stat_cap` 配置表；服务端读写与公式重算后强制截断
- 宠物成长一期落地：迁移 `051_pet_progression.sql`、`052_admin_pet_progression_permissions.sql`、`module/petprogression`（经验升级/加点/公式重算）、WS `2063/2064`、Admin `/api/admin/pet-progression/`、后台「宠物成长配置」页
- 客户端状态面板宠物页支持切换宠物、展示成长字段与 +1 加点；战斗结算 `pet_rewards` 扩展升级摘要；新增宠物升级弹窗
- Admin 宠物成长页支持「重算全部宠物战斗属性」运维入口（`POST /api/admin/pet-progression/recalculate-combat-stats`）
- 新增 `backend/docs/pet-progression.md`：基于 `docs/风车做资参考表（v6.2）.xlsx` 反推的宠物升级、自由加点与资质→战斗属性公式
- `backend/docs/protocol.md` 同步 `PetDetail`、`PET_ALLOCATE_ATTR_*`、`BattlePetReward` 扩展字段

## 2026-06-17
- 新增 `docs/形象动画配置指南.md`：UnitSkin 全参数、CHJ/PNG 局部覆盖规则、动画帧新建流程与配置示例
- 客户端 CHJ 战斗待机改为主 CHJ 末尾最后两个动画组合并循环；技能/普攻通过 `UnitSkin.chj_skill_path` 独立 CHJ 补充；`sprite_frames` 可按动画名局部覆盖 CHJ
- 新增 `ChjSprite`、`ChjWorldRenderer`、`CharacterVisual` 双后端；示例皮肤 `CHJ测试_2057` + `client/asset/chj/2057.chj`

## 2026-06-16
- 任务模板后台支持多阶段结构化编辑：每阶段可配置事件类型、目标 NPC/场景、菜单 entry_id、剧情 entry_id 与引导文案；详情页以阶段卡片展示
- 任务运行时按 `objective_id` 顺序推进阶段，未完成前置阶段时不会跳阶段完成后续目标
- `AdminObjectiveInput` 与 `objectives_json.guide` 扩展 `menu_entry_id`、`dialogue_entry_id` 字段，便于运营绑定 NPC 菜单/剧情
- 新增结构化 NPC 剧情系统：迁移 `044_npc_dialogue.sql`、`045_npc_dialogue_more_entries.sql`，引入 `npc_dialogue`、`npc_dialogue_node`、`npc_dialogue_option`、`player_npc_dialogue_session` 表
- 新增 `module/npcdialogue` 领域模块与后台 `/api/admin/npcs/dialogues` CRUD；WebSocket 新增 `2037/2038/2039` 剧情推进协议
- 客户端新增 `NPCDialoguePanel`、`CinematicPlayer`、`PortraitRegistry`、`RequestLoadingOverlay`；NPC 交互/菜单/剧情请求统一走 loading 遮罩后再开面板
- 示例剧情：`93001/dialog_market_intro` 含 action 节点 + `market_limeng_step_aside` 客户端演出；`dialog_market_news`、`dialog_warehouse_intro`、`dialog_trade_tip` 已迁入结构化对话
- 节点 `effects_json` 第一版支持 `notice` 与 `quest_event` 两类服务端副作用
- 新增迁移 `046_npc_shop_goods.sql`：商店商品表 + 市场罗格 `shop_open_market` 改为 `result_type=shop`；`NPC_ACTION_RESP` 增加 `shop` 载荷，客户端接入 `npc_shop_panel` 与 `5101/5102 BUY_ITEM`
- 剧情 `conditions_json` 第一版支持 `quest_id + quest_state` 过滤节点/选项；断线重连 `1022 RECONNECT_RESP` 增加 `active_dialogue` 恢复未结束剧情
- 后台新增 `/npc-dialogues` 独立剧情列表页，复用 `fetchAdminNPCDialogues` 与剧情编辑抽屉
- 后台剧情编辑抽屉支持节点/选项 `conditions_json`（`quest_id + quest_state`）可视化配置
- 后台剧情编辑抽屉支持节点 `effects_json`（`notice + quest_event`）可视化配置，修复后台保存时会清空副作用的问题
- 新增 WebSocket `2042 NPC_MENU_REQ` / `2043 NPC_MENU_RESP`：NPC 菜单拉取与 `2031 INTERACT_REQ` 拆分；对有菜单 NPC 的 INTERACT 请求返回 `use npc menu request`
- 新增迁移 `047_npc_menu_entry_conditions.sql`：菜单项 `conditions_json` 与 `linked_quest_id`；支持按任务状态/分阶段 `objective_id + objective_completed` 过滤可见菜单
- 剧情节点 `effects_json` 扩展 `grant_items`（服务端发物品）与 `accept_quest_id`（进入节点自动接任务）；节点/菜单条件扩展 `objective_id + objective_completed`
- 运行时菜单按玩家任务进度过滤；剧情推进时应用发奖/接任务副作用；菜单动作支持 `quest_accept` / `quest_submit`
- 后台地图 NPC 菜单编辑改为「菜单配置 | 剧情配置」合并 Tab；移除独立 `/npc-dialogues` 导航（旧路由重定向至 `/npcs`）
- 后台剧情表单支持发物品、接任务、任务阶段可见条件可视化编辑
- 新增迁移 `048_npc_scene_only_placement.sql`：`world_entity_definition` 移除坐标/朝向/速度字段，新增 `world_scene_definition` 供后台展示场景中文名；NPC 摆放改由客户端场景资源维护

## 2026-06-15
- 新增玩家成长体系第一版：迁移 `035_player_level_progression.sql`、`036_admin_player_progression_permissions.sql`，引入等级经验表、属性转化率表、玩家自由属性点与 `base_*` 裸装战斗值
- 新增 `module/progression` 领域模块，统一承接经验连升、升级发点、加点校验与战斗属性重算；`player.AddExp`、战斗结算发经验、任务发经验均走该模块
- 新增 WebSocket `PLAYER_ALLOCATE_ATTR_REQ/RESP (2061/2062)`；`PlayerSnapshot` 与 `BattleResultPush` 扩展成长相关字段
- 后台新增 `/player-progression` 页面与 `/api/admin/player-progression/...` 配置接口；玩家详情页展示自由属性点与四维分配值
- 客户端状态面板与加点页已对接服务端权威快照，加点请求带 loading 遮罩
- 设计文档：`backend/docs/player-progression.md`

## 2026-06-11
- 新增迁移 `backend/server/migrations/010_add_player_pet_mana.sql`，为 `player_pet` 表补齐 `mana` 字段，并同步回填演示宠物初始法力，修复 PostgreSQL 模式下进入世界/读取编队时因 `pp.mana` 缺列导致的失败
- 服务端配置加载方式已从 `config.env` 环境变量文件切换为单一 YAML 配置文件：`backend/server/cmd/game-server/main.go` 现在会优先解析 `backend/server/configs/config.yaml`
- `backend/server/internal/config/config.go` 改为从 YAML 结构读取 `http/auth/heartbeat/postgres/redis` 五段配置，并继续复用原有运行时校验逻辑，避免只改加载方式就把启动约束放松
- 新增 `backend/server/internal/config/config_test.go`、`backend/server/internal/config/yamlfile_test.go`，覆盖 YAML 配置解析、默认路径解析与基础校验
- 示例配置文件已从 `backend/server/configs/config.env(.example)` 切换为 `backend/server/configs/config.yaml(.example)`；`PP_CONFIG_FILE` 现仅用于覆盖 YAML 文件路径，不再承载各项运行参数

## 2026-06-10
- 服务端配置已收敛为单一 `PostgreSQL + Redis` 运行路径：`backend/server/internal/config/config.go`、`backend/server/internal/data/provider/` 与示例环境变量已删除 `memory` / `PP_REPOSITORY_MODE` 分支，后续不再维护双仓储模式
- 已新增 `backend/server/migrations/006_seed_postgres_demo_account.sql`，为 `postgres_redis` 模式补齐 `demo / demo123` 演示账号、`DemoTrainer` 玩家、三只起始宠物与默认编队；切到 PostgreSQL 仓储后不再因为数据库缺少演示数据而登录失败
- 服务端已新增数据库驱动的 NPC 配置能力：新增 `backend/server/migrations/004_npc_config.sql`，引入 `world_entity_definition` 与 `npc_menu_entry` 两张表，并预置当前世界里的引导 NPC、市场 NPC 与仓库 NPC 数据
- `backend/server/internal/module/npc/` 已补齐最小 NPC 配置服务边界，`battle_handler.go` 不再通过硬编码 `switch entity_id` 组装菜单与对话，而是统一从 NPC 仓储读取静态菜单项与动作结果
- `backend/server/internal/data/postgres/world_repo.go` 已接管 PostgreSQL 模式下的世界实体查询；后续若要新增一个可交互 NPC，只需在 SQL 迁移或数据库数据中新增实体定义和菜单项，不必再改服务端交互代码
- 内存模式也已同步补上 `NPCRepository` 和 `91001 罗思` 的基础菜单配置，便于当前默认内存模式下继续本地联调

## 2026-06-09
- 已把 `/Users/wangzhiwei/game/dialogue_demo` 中可直接复用的对话与运行态 UI 骨架迁入当前客户端：新增 `client/addons/dialogue_manager/`、`client/dialogue/`、`client/scenes/ui/`、`client/scripts/ui/`、`client/scripts/data/` 与 `client/data/`
- `client/project.godot` 已接入 `DialogueManager` 与 `SomeGlobal` 自动加载，并新增 `open_main_menu`、`open_player_panel`、`open_scene_npc_list` 输入动作；当前可直接呼出 demo 风格主菜单、角色面板、场景 NPC 列表与 NPC 菜单
- `client/scripts/bootstrap/main.gd` 已把新 UI 接入现有联机主运行态：附近 NPC 现在可通过列表选择并向服务端发起交互，服务端返回的 `menu_entries` 会用 demo 风格 NPC 菜单展示；当菜单项为本地 `talk`/`dialogue` 类型时，会直接走 `DialogueManager` 气泡框
- 状态面板头部称号与玩家名现已通过 `SomeGlobal`/`GameState` 联动填充；宠物状态页会优先读取当前联机宠物列表中的首只宠物，其余数值项缺省时继续回退到 demo 默认数据
- `client/scenes/bootstrap/main.tscn` 已移除登录后主运行态底部“世界操作区 / 战斗操作区”面板、按钮组和数据弹层，`GameplayArea` 现恢复为全屏显示世界/战斗内容
- `client/scripts/bootstrap/runtime_hud.gd` 已收敛为最小顶部状态条，只保留连接、场景、玩家文案和隐藏日志输出，不再承载宠物、编队、任务、背包或 NPC 菜单交互
- `client/scripts/bootstrap/main.gd` 已同步删除对底部操作区信号、自动摘要拉取和 NPC 菜单面板的依赖；当前收到相关交互负载时仅记录日志，不再弹出界面
- 本次未改动服务端协议与世界/战斗主链路，只移除客户端运行态中的世界操作区相关界面与脚本耦合

## 2026-05-20
- `client/autoload/http_client.gd` 已补上非 JSON、空响应和底层 HTTP 失败结果的容错处理，避免后端未启动或返回异常内容时直接触发 `JSON.parse_string()` 解析报错
- `client/scripts/common/command_ids.gd` 已为客户端协议消息号常量补齐说明性注释，当前各请求、响应和推送编号的用途更清晰
- `client/autoload/message_router.gd` 已为消息回调注册表、注册/注销和统一分发逻辑补齐说明性注释
- `client/autoload/http_client.gd` 已为基础地址、HTTP 请求节点、登录接口和通用 JSON 请求封装逻辑补齐说明性注释
- `client/scripts/feature/world/player.gd` 已为四方向移动、状态机切换、动画回退和切图/战斗锁定逻辑补齐说明性注释
- `client/scripts/feature/pet/pet_controller.gd` 已为宠物列表响应、宠物更新推送和编队设置响应的状态写回逻辑补齐说明性注释
- `client/scripts/feature/bag/bag_controller.gd` 已为背包列表响应和单物品更新推送的状态写回逻辑补齐说明性注释
- `client/scripts/feature/battle/battle_controller.gd` 已为交互响应、战斗开始/更新/结算推送的状态写回与事件广播逻辑补齐说明性注释
- `client/scripts/bootstrap/main.gd` 已为主运行态场景挂载、消息路由注册、HUD 刷新、世界/战斗视图切换和返回登录页流程补齐说明性注释
- `client/autoload/app.gd` 已为应用层启动编排、HTTP 登录、WebSocket 鉴权、推送处理和战斗动作上报入口补齐说明性注释
- `client/scripts/auth/login_scene.gd` 已为登录按钮链路、演示账号填充、登录页状态刷新和过渡动画流程补齐说明性注释
- `client/scripts/feature/battle/battle_scene.gd` 已为战斗界面刷新、技能按钮绑定、战斗事件文案生成和单位状态读取逻辑补齐说明性注释
- `client/scripts/bootstrap/runtime_hud.gd` 已为运行态 HUD 的常量、信号、节点引用、面板状态字段和主要渲染函数补齐说明性注释，当前宠物/编队/背包面板的职责与交互入口更清晰
- `client/scripts/feature/world/world_controller.gd` 已为场景配置、地图装载、固定镜头布局、门区切图、坐标换算和序列号生成逻辑补齐说明性注释，不改变现有地图切换与权威同步链路
- `client/autoload/net_client.gd` 已为连接状态、心跳调度、二进制封包解包、CRC32 校验和开发态 JSON 收发逻辑补齐说明性注释，便于后续继续维护网络层
- `client/autoload/game_state.gd` 已为会话状态、世界快照、宠物/编队、背包和战斗状态写入逻辑补齐说明性注释，保持现有状态合并与事件广播行为不变

## 2026-05-17
- `world_controller.gd` 已把固定镜头地图的角色出生显示点收敛为“地图可见内容中心”规则：当场景未显式配置 `spawn_local_position` 时，会自动按当前地图内容包围盒中心计算出生显示点
- `scene_id = 1` 的 `roxus_house` 已移除手写出生显示坐标，登录进入世界或权威重同步后，角色会默认显示在地图场景中心；后续新增固定镜头地图时也可直接复用同一规则
- `main.tscn` 的登录后主运行态上下分区已从 `4:1` 调整为 `3:1`：上部 `GameplayArea` 现占 `75%` 高度，下部 `HudRoot` 现占 `25%` 高度，继续保持游戏区与操作区互不遮挡
- 客户端设计分辨率已从 `1080x1920` 收敛回 `360x640`，并继续通过 `window/stretch` 在移动端按整数倍率自动放大适配；`main.tscn`、`world_scene.tscn`、`battle_scene.tscn`、`login_scene.tscn` 与 `runtime_hud.gd` 也已同步回收到小设计分辨率口径
- 客户端设计分辨率曾短暂收敛为 `240x320`，但由于与当前竖屏目标分辨率比例不一致，运行时整数倍率放大后清晰度下降；现已回退为 `360x640`，并同步恢复主运行态 `SubViewport`、世界层默认渲染尺寸、登录页、战斗卡片和底部 HUD 的对应尺寸口径
- 客户端早期占位地图文件已清理，只保留 `roxus_house` 作为当前地图资源；`world_controller.gd` 中对已删除占位地图的加载引用，以及 `roxus_house` 中通往已删除地图的出口门区也已同步移除
- 客户端地图切换现已重新接通 `scene_id = 1 <-> 2`：`world_controller.gd` 新增 `scene_id = 2 -> east_road_of_shanguang_town.tscn` 的地图映射，`roxus_house.tscn` 中门区现为 `portal_id = 1001 -> scene_id = 2`，`east_road_of_shanguang_town.tscn` 中回程门现为 `portal_id = 2001 -> scene_id = 1`
- 为修正正式地图门区无法稳定触发的问题，`scene_id = 2` 的客户端坐标基准现已改为贴合 `east_road_of_shanguang_town.tscn` 当前门区像素位置；服务端内存世界仓储中 `portal_id = 1001` 与 `portal_id = 2001` 的权威入口落点也已同步重标定，`go test ./server/internal/transport/ws` 通过

## 2026-05-16
- 新增 `backend/docs/kdjl-client-reference.md`，梳理逆向原版客户端 `/Users/wangzhiwei/study/kdjl` 中对当前 MVP 有参考价值的流程设计
- 文档聚焦登录前状态机、世界与战斗场景切换、宠物实例/编队/出战宠分层、战斗意图上报与服务端权威结算
- 文档同时明确原版中不应直接迁移的部分，包括文本 UI 协议、WAP 代理联网细节、旧资源协议与敏感信息处理方式
- 新增 `backend/docs/pet-lineup-battle-model.md`，把“宠物实例 / 编队 / 当前出战宠 / 战斗快照”四层模型固定为后续实现口径
- 新文档同步梳理了服务端模块边界、客户端 `GameState` 状态建议、协议补强方向和分步实现顺序
- 补齐 `PET_LIST_REQ/RESP` 与 `PET_LINEUP_SET_REQ/RESP` 的最小双端闭环，服务端新增 `pet_handler`、内存/PostgreSQL 仓储能力和相关 WebSocket 路由
- 客户端 `GameState` 的宠物合并主键从 `pet_id` 调整为 `pet_uid`，并在编队变更后自动同步 `in_lineup` 标记
- 同步更新 `backend/docs/protocol.md` 与 `backend/proto/pet/pet.proto`，使宠物列表和编队设置响应结构与当前实现一致
- 新增 `backend/docs/map-scene-loading.md`，把参考原版客户端后的地图切换加载方案落成当前项目的实现文档
- 文档明确了“世界层常驻、地图资源热切换、服务端权威切图、客户端按 `WORLD_RESYNC_PUSH` 装载地图”的实现口径，并给出分阶段实施顺序
- 客户端 `world_scene.tscn` 新增 `MapMount` 和最小地图加载遮罩，`world_controller.gd` 已接入 `scene_id -> scene_path` 地图挂载/卸载逻辑，并按 `WORLD_RESYNC_PUSH` 切换地图资源
- 新增三张最小地图占位骨架，作为当时地图切换链路与门区接入的早期占位资源
- 服务端内存版 `world_repo` 新增按来源地图决定入口落点的切图逻辑，不再把目标地图统一 `spawnPos` 当作落点，解决切图后角色总出现在地图中心的问题
- 同步更新世界切图测试与协议/设计文档，明确当前最小入口模型为“按来源地图选择目标地图入口落点”；`go test ./server/...` 已通过
- 继续补齐地图门区实例：服务端 `MOVE_INTENT_REQ` 已支持 `portal_id`，客户端地图场景已接入 `Area2D` 门区与 `MapPortal` 脚本，门区触发后会按 `portal_id` 发起权威切图
- 同步更新 `backend/proto/world/world.proto`、`backend/docs/protocol.md` 与 `backend/docs/map-scene-loading.md`，并新增无效 `portal_id` 的服务端测试；相关 GDScript/场景诊断与 `go test ./server/...` 已通过
- 客户端已移除边界触发切图链路，`player.gd` 不再检测地图边缘，`world_controller.gd` 只保留门区 `Area2D` 触发的地图切换
- 继续落地宠物战斗模型：`battle` 模块现已在 `BATTLE_START_PUSH` / `BATTLE_STATE_PUSH` 中显式返回 `active_actor_id`、`active_pet_uid`，并为战斗单位补充 `lineup_index`
- 客户端 `GameState` 与 `battle_scene.gd` 已按当前出战宠字段组织战斗展示和动作提交；`backend/proto/battle/battle.proto` 与协议文档已同步更新，`go test ./server/...` 通过
- 继续补齐核心模型闭环：服务端战斗结算后现已把主战宠最终 HP 回写到 `pet` 模块，并通过 `3011 PET_UPDATE_PUSH` 把更新后的宠物实例同步给客户端
- `pet` 模块新增宠物 HP 更新接口，内存仓储与 PostgreSQL 仓储都已补齐最小实现；客户端继续复用现有 `handle_pet_update()`，无需新增一套路由或 UI
- 战斗链路测试现已覆盖 `PET_UPDATE_PUSH` 与回写后的 `PET_LIST_RESP` 一致性校验，`go test ./server/...` 通过
- 客户端世界地图配置已将 `scene_id = 1` 的加载资源切换为 `client/scenes/maps/fashtown/roxus_house.tscn`，从而直接复用新建的 `roxus_house` 地图场景
- `roxus_house.tscn` 已补接最小门区节点，复用现有 `map_portal.gd` 脚本并配置为 `portal_id = 1001 -> scene_2`，同时增加可见出口标记，便于继续迭代地图细化
- `world_controller.gd` 已为 `roxus_house` 接入固定镜头模式：相机固定在当前视口中心，地图按实际可见内容自动居中并在必要时缩放到可完整显示，角色按地图内本地坐标移动，不再带动镜头平移
- 客户端主场景已拆成“上部游戏显示区 + 下部固定 HUD 区”两段布局：世界地图、地图切换、战斗场景都只在上部区域渲染，下部会永久显示 `client/asset/场景原图/闪光镇/时光小屋.png`
- `main.tscn` 已新增底部固定背景与 HUD 容器，并把现有状态面板、挑战按钮和日志区收敛到底部常驻区域；`main.gd` 与 `world_controller.gd` 已同步支持按上部游戏区域尺寸布局固定镜头地图
- 继续补充 `backend/docs/kdjl-client-reference.md`，新增“登录后主运行态分层布局”和“战斗层与常驻 UI 共存关系”两节，明确原客户端采用单主画布分层承载世界、战斗与底部常驻功能区的结构
- 新增 `backend/docs/main-runtime-ui-layout.md`，把当前项目登录后“上部游戏区 + 下部常驻 HUD 区”的主运行态 UI 结构单独沉淀为实现文档，并明确只覆盖当前 MVP 内的世界、战斗、宠物/编队与背包入口挂点
- 客户端新增 `runtime_hud.gd` 组件并接入 `main.tscn`，底部正式操作区现已补出 MVP 骨架：运行状态区、世界交互按钮、宠物/编队/背包入口按钮与日志区
- `main.gd` 现已通过 `RuntimeHud` 统一驱动底部 HUD，并在首次进入世界后自动请求宠物列表与背包列表，使底部入口计数能同步当前摘要数据
- `RuntimeHud` 现已新增最小数据面板：点击 `宠物`、`编队`、`背包` 按钮会打开对应摘要面板，并随 `GameState` 数据更新自动刷新内容；进入战斗时该面板会自动收起
- `RuntimeHud` 的数据面板已进一步升级为滚动卡片列表样式，并在 `编队` 面板中补上最小交互：支持加入/移除、上移/下移调整顺序，以及通过 `main.gd -> App.set_pet_lineup()` 提交完整编队
- 主场景布局继续保持明确的上下分区：上部 `GameplayArea` 占约 `4/5`，下部 `HudRoot` 占约 `1/5`；当前已改为上部天蓝色纯背景、下部淡红色纯背景，游戏画布和操作区不再互相遮挡
- 上部游戏区现已改为 `SubViewportContainer + SubViewport` 独立渲染：世界层与战斗层挂点都迁入子视口，`main.gd` 会同步子视口尺寸，从而修复根视口清屏色在顶部泄露导致的黑条问题
- 客户端现已按 `1080x1920` 新设计分辨率补齐主运行态适配：`main.tscn` 不再依赖旧的 `320/384/96` 固定尺寸，改为保持 `4:1` 上下比例并放大 HUD 字号与按钮尺寸；`world_controller.gd` 的固定视角地图允许在大屏中自动放大，`battle_scene.tscn` 与 `world_scene.tscn` 的旧加载提示/战斗卡片也已同步扩大
- 登录页 `login_scene.tscn` 也已按 `1080x1920` 适配：新增纯色底板、居中登录卡片，并整体放大标题、输入框、主按钮、状态文字与日志区，保持与当前大屏主运行态一致的可读性

## 2026-05-14
- 新增联机复刻版架构草案，明确客户端、服务端、同步和持久化边界
- 新增实时协议文档，固定包头、消息号分段和 HTTP/WS 令牌策略已定稿
- 新增双端消息路由文档，明确 server/client 消息处理职责
- 新增 `proto/` 初版协议草案，覆盖 auth、world、pet、battle、bag 五类消息
- 新增 PostgreSQL 最小表结构迁移脚本，覆盖账号、玩家、宠物、背包、编队、战斗记录
- 新增 Go 服务端骨架，覆盖 HTTP 登录、JWT 签发、`ws_token` 鉴权、WebSocket 会话与应用层心跳
- 新增内存版账号仓储与 `ws_token` 仓储，用于当前阶段的无数据库联调
- 新增协议包头编解码与基础测试，`go test ./server/...` 已通过
- 新增 `ENTER_WORLD_REQ` 链路，打通 `session -> player -> pet -> world` 的场景快照返回
- 新增内存版 `player/pet/world` 仓储，当前可返回演示角色、编队和单场景快照
- 新增 WebSocket 路由测试，已覆盖已鉴权进入世界与未鉴权拦截场景
- 新增 `MOVE_INTENT_REQ` 链路，已支持移动合法性校验、位置更新、移动回执和世界重同步
- 新增玩家位置更新能力，移动成功后再次进入世界会返回最新坐标
- 新增世界移动测试，已覆盖合法移动、非法越界移动与重同步场景
- 调整根目录结构，现已拆分为 `backend/` 服务端目录和 `client/` 客户端目录
- 当前 Go 工程、协议、文档和迁移脚本已全部迁入 `backend/`
- 新增 Godot 4 客户端骨架，补齐 `client/project.godot`、入口场景和可直接打开的最小工程结构
- 新增客户端 `autoload` 单例：`App.gd`、`HttpClient.gd`、`NetClient.gd`、`MessageRouter.gd`、`GameState.gd`
- 新增世界、宠物、战斗、背包控制器占位脚本，先把客户端模块边界与消息路由挂好
- 新增根目录 `.gitignore`，忽略本地 SkillHub 目录和 Godot 生成的 `.godot/` 目录
- 持久化方案从 MySQL 调整为 PostgreSQL，并同步改写初始化迁移脚本方言与字段定义
- 新增 PostgreSQL、Redis 配置骨架与示例环境变量
- 新增 PostgreSQL 账号/玩家/宠物仓储适配器，以及 Redis `ws_token` 仓储适配器骨架
- 新增仓储 provider 装配层，为后续统一切换到 PostgreSQL/Redis 持久化准备依赖注入入口
- 新增 `backend/server/configs/config.env` 实际配置文件，并支持启动时自动加载本地 env 文件
- 客户端主场景改为最小登录页，支持账号密码输入、状态展示与日志输出
- 客户端补齐 WebSocket 二进制包头编码、CRC32 校验、JSON 消息体编解码与基础心跳
- 客户端打通 `HTTP 登录 -> WS 鉴权 -> 进入世界` 主流程，登录后自动建立实时会话
- 客户端全局状态新增 `session_id`、`reconnect_token`、`heartbeat_sec` 与 `is_ws_authenticated` 追踪
- 客户端新增独立 `login_scene` 登录场景，并将项目启动入口切换到登录场景
- 客户端 `bootstrap/main` 收敛为登录后的世界入口，只负责世界挂载、消息路由与运行态状态展示
- 客户端登录场景与主场景新增淡入淡出切场景过渡，优化登录成功和掉线返回体验
- 客户端主场景顶部状态面板与底部日志区进一步压缩，更适配 `320x480` 小窗口
- 客户端角色在进入战斗场景前补齐三态状态机，当前支持待机、行走、战斗中三种状态
- 客户端战斗消息已驱动角色状态切换，战斗中会锁定移动并优先播放战斗态动画回退
- 客户端新增独立 `battle_scene` 战斗视图场景，收到 `BATTLE_START_PUSH` 后会挂载战斗场景
- 客户端收到 `BATTLE_RESULT_PUSH` 后会卸载战斗场景并返回世界视图，保持现有网络链路不重建
- 服务端新增最小 PvE 战斗模块，当前通过与附近 NPC 交互触发战斗，所有伤害、回合推进和胜负结算均在服务端完成
- 客户端主场景新增“挑战附近NPC”入口，战斗场景新增“普通攻击”按钮，客户端仅发送交互/动作意图，不参与数值计算
- 服务端战斗模块新增最小技能表与技能合法性校验，当前支持按 `skill_id` 驱动不同技能伤害
- 客户端战斗场景改为根据服务端下发的 `skill_ids` 动态展示技能按钮，继续保持只发技能意图
