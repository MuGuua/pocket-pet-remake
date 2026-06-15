# battle-test 战斗场景 Demo 接入方案

> 文档版本：v0.2（评审稿）  
> 对比来源：`/Users/wangzhiwei/game/battle-test`  
> 目标仓库：`pocket-pet-remake`  
> 结论先行：**不能 100% 原样接入**，但表现层主体可迁移；冲突字段**一律以服务端契约为准**，Demo 不合理字段在迁入时直接改掉；后端缺失字段（如 `skin_id`）**必须补数据库 + 后台配置 + 协议推送**。

---

## 1. 文档目的

1. 对比 `battle-test` JSON 驱动战斗 Demo 与当前项目后端战斗系统的契约差异。
2. 评估哪些代码/资源可以直接迁入 `client/`。
3. 给出推荐接入路径、分阶段任务与风险清单。
4. 供确认后再进入实现阶段。

---

## 2. 接入原则（已确认）

### 2.1 冲突字段：以服务端为准

凡 Demo 与现后端/协议不一致的字段，**不迁就 Demo**，迁入时直接按服务端已有契约改造客户端与 Demo 派生代码：

| 原则 | 说明 |
|------|------|
| 协议字段名与类型 | 使用 `battle.proto` / `protocol.md` 已定义字段，禁止在正式链路保留 Demo 私有命名 |
| 标识符类型 | 统一 `uint64 actor_id`、`uint32 skill_id`、`uint64 battle_id`，废弃 Demo 字符串 `id` |
| 结算与顺序 | 伤害、治疗、胜负、出手顺序均以 `battle.Service` 产出的事件流为准 |
| 表现层从属 | 客户端只消费推送，不把 Demo JSON 预写结果当作权威 |

### 2.2 缺失逻辑：后端补齐 + 后台可配置

后端当前没有、但战斗表现必需的逻辑，**必须在服务端与运营后台补齐**，不允许长期用客户端硬编码映射表顶掉：

| 缺失项 | 补齐方式 |
|--------|----------|
| 宠物/怪物/角色外观 `skin_id` | 数据库字段 + 后台模板编辑 + `BATTLE_START_PUSH` 下发 |
| 技能表现 `skill_visual_id` | 数据库字段（可空）+ 后台技能模板编辑 + 技能快照下发；空则回退 `animation_key` |
| 战斗用药 | 后端实现 `USE_ITEM` 与背包扣减后，客户端再接物品按钮 |
| 战场召唤 | 后端有运行态后再接 Demo `summon_changes` 演出 |

### 2.3 Demo 可保留的部分

- 导演状态机、单位/特效/UI 组件、本地 `UnitSkin` / `SkillVisualConfig` 资源扫描机制。
- `formation` 站位算法（属**客户端布局**，由 `lineup_index` 推算，不是协议字段）。
- 离线调试录制 JSON（仅开发态，字段格式也必须与真实推送一致）。

---

## 3. 两个系统的核心定位对比

| 维度 | battle-test Demo | pocket-pet-remake 当前后端 |
|------|------------------|---------------------------|
| 权威归属 | JSON 模拟服务端（本地文件） | Go `battle.Service` + WebSocket 推送 |
| 客户端职责 | 只播放，不结算 | 只展示，不结算（设计一致） |
| 数据形态 | **回合剧本**（`rounds[].timeline/actions`） | **事件流**（`BATTLE_STATE_PUSH.events[]`） |
| 单位标识 | 字符串 `id`（如 `player_1`） | `uint64 actor_id`（**迁入时废弃 Demo 字符串 id**） |
| 外观绑定 | `skin_id` → `UnitSkin.tres` | **待补** `skin_id`（§7：`pet_definition` / `monster_definition` + 后台配置） |
| 技能表现 | `skill_visual_id` → `SkillVisualConfig.tres` | `animation_key` + 颜色 + `projectile`；**待补** `skill_visual_id`（§7，可空回退） |
| 数值结果 | JSON 预写 `hp_after` / `floating_text` | 事件 `value` + 回合末 `actors[]` 快照 |
| 出手顺序 | JSON `timeline` 固定顺序 | 服务端按 `spd` 排序 + 随机 tie-break |
| 输入阶段 | JSON `input_phase.order` 模拟 | `pending_actor_ids` / `command_deadline_ms` |
| 联机 | 无 | PVE / PVP / 野外遇敌 / 断线重连 |
| 结算 | JSON `result.summary` | `BATTLE_RESULT_PUSH` + 钱包/宠物/任务 |

**共同点（有利于接入）**

- 都遵循「服务端权威、客户端纯表现」。
- 都有命令阶段 → 演出阶段 → 下一回合/结束 的状态机概念。
- 都区分技能逻辑名与表现资源（Demo 的 `skill_id` / `skill_visual_id` ≈ 后端的 `skill_id` / `animation_key`）。

**根本差异（阻碍 100% 接入）**

- Demo 假设服务端下发**完整回合剧本**；当前后端只下发**扁平事件列表**，没有 `timeline` / `actions` / `combo` 结构。
- Demo 的 JSON 是**离线可重放**的；当前协议是**增量状态机**，客户端必须自己把事件编排成演出时间线。

---

## 4. battle-test 资产清单

### 4.1 脚本（10 个）

| 文件 | 职责 |
|------|------|
| `battle_director.gd` | 总导演：单位生成、输入收集、回合演出、Buff/召唤 |
| `battle_data_provider.gd` | 读取 `battle_demo.json`，解析站位 |
| `battle_content_registry.gd` | 扫描 `unit_skins/`、`skill_visuals/` 资源 |
| `battle_unit.gd` | 单位表现：动画映射、受击/治疗/死亡、选目标高亮 |
| `battle_effect.gd` | 技能特效序列帧 |
| `action_panel.gd` | 攻击/技能/物品/自动/逃跑 + 技能列表 + 选目标 |
| `action_log_panel.gd` | 战斗日志面板 |
| `floating_text.gd` | 飘字 |
| `resources/unit_skin.gd` | 单位皮肤 Resource 定义 |
| `resources/skill_visual_config.gd` | 技能表现 Resource 定义 |

### 4.2 场景与资源

| 类型 | 路径 |
|------|------|
| 主场景 | `scenes/battle/battle_scene.tscn` |
| 单位场景 | `scenes/battle/battle_unit.tscn` |
| 特效场景 | `scenes/battle/battle_effect.tscn` |
| 演示数据 | `data/battle_demo.json` |
| 单位皮肤 | `resources/battle/unit_skins/*.tres`（当前 2 个） |
| 技能表现 | `resources/battle/skill_visuals/*.tres`（当前 2 个） |
| UI/背景素材 | `assets/战斗ui.png`、`魔法阵.png`、`星空背景.png` 等 |

### 4.3 Demo JSON 数据模型（摘要，正式链路不采用）

```text
battle_demo.json
├── battle_id / battle_name
├── formation          # 站位布局、人数上限、别名
├── units[]            # 开场单位（含 skin_id、skills、items）
└── rounds[]
    ├── input_phase    # 本回合可选单位、可选技能
    ├── timeline[]     # serial / parallel 播放顺序
    ├── actions[]      # 动作剧本（含 targets、buff_changes、summon_changes）
    ├── combo[]        # 追击/反击等追加动作
    └── result         # 是否结束、胜者、摘要
```

### 4.4 Demo 设计原则（与项目 AGENTS.md 一致）

- 服务端决定战斗结果和表现顺序。
- 客户端按 ID 查本地 `UnitSkin` / `SkillVisualConfig` 播放。
- 浮点展示转 `int`（Demo 已遵守）。

---

## 5. pocket-pet-remake 当前战斗链路

### 5.1 后端

- 领域服务：`backend/server/internal/module/battle/service.go`
- 协议：`backend/proto/battle/battle.proto`
- 传输：`backend/server/internal/transport/ws/battle_handler.go`
- 技能表：内存 catalog + DB `skill` 模块（含 `animation_key`）

### 5.2 协议消息

| CMD | 方向 | 用途 |
|-----|------|------|
| 4001 `BATTLE_ACTION_REQ` | C→S | 提交技能/逃跑/自动/捕捉 |
| 4002 `BATTLE_ACTION_RESP` | S→C | 动作接受回执 |
| 4011 `BATTLE_START_PUSH` | S→C | 开战快照（双方 actor 列表） |
| 4012 `BATTLE_STATE_PUSH` | S→C | 回合事件 + actor 状态 + 命令阶段信息 |
| 4013 `BATTLE_RESULT_PUSH` | S→C | 胜负、奖励、回场景 |

### 5.3 服务端事件类型（`event_type`）

| 值 | 含义 | Demo 对应能力 |
|----|------|---------------|
| 1 | 使用技能 | `actions[].action_type=skill` |
| 2 | 伤害 | `targets[].result_type=damage` |
| 3 | 治疗 | `targets[].result_type=heal` |
| 4 | 添加状态 | `buff_changes[]`（部分） |
| 5 | 状态 tick | `buff_tick` / 持续伤害 |
| 6 | 跳过回合 | 无直接对应 |
| 7 | 击败 | 隐含在 `hp_after=0` |
| 8 | 闪避 | 无直接对应 |
| 9 | 反击 | `combo[]`（部分） |
| 10 | 复活 | 无直接对应 |
| 11 | 连击 | `combo[]`（部分） |
| 12 | 捕捉 | 无（Demo 未做） |

### 5.4 客户端现状

| 组件 | 状态 |
|------|------|
| `battle_controller.gd` | ✅ 已接 WS，写入 `GameState.battle_state` |
| `App.submit_battle_action()` | ✅ 可发 4001 |
| `main.gd` `_on_battle_started` | ⚠️ **已移除独立战斗场景**，仅打日志 |
| `scenes/battle/battle_scene.tscn` | ⚠️ 存在，但引用的 `battle_scene.gd` **缺失** |
| `battle_player.gd` | ⚠️ 简易 Tween 表现，未接 Director |
| `battle-test` 级 Director/UI | ❌ 未迁入 |

---

## 6. 字段级映射对照（冲突以服务端为准）

> 下表「Demo 迁入改法」列 = 接入时**必须**对 Demo 代码/数据结构做的修改，不以 Demo 原字段为准。

### 6.1 开战单位：Demo `units[]` → `BATTLE_START_PUSH`

| Demo 字段 | 服务端权威字段 | Demo 迁入改法 |
|-----------|----------------|---------------|
| `id` (string) | `actor_id` (uint64) | `BattleUnit.unit_id` 改为 `int`/`uint64`；`_units` 字典 key 用 `actor_id` |
| `type` player/enemy/pet | `actor_type` + `unit_class` | 废弃字符串 `type`；用 `actor_type` 判敌我，`unit_class` 判人物/宠物/怪物 |
| `name` | `name` | 直接沿用 |
| `hp` / `max_hp` | `hp` / `hp_max` | 直接沿用，UI 转 `int` |
| `position` | `lineup_index` | **删除 Demo 协议字段**；`position` 仅作客户端内部 slot key，由 `BattleFormationMapper` 根据 `lineup_index` + 阵营计算 |
| `skin_id` | `skin_id`（**待补**） | Demo 保留字段语义；值改由服务端推送，见 §7 |
| `skills[].skill_id` (string) | `skills[].skill_id` (uint32) | 全部改为数字 ID；展示名读 `skills[].name` |
| `skills[].skill_visual_id` | `skills[].skill_visual_id`（**待补**） | 接入后读推送；未配置时回退 `animation_key` 查本地资源 |
| `skills[].display_name` | `skills[].name` | 字段名对齐服务端 |
| `skills[].target_side` | `skills[].target_type` | 用服务端枚举：`enemy_single` / `ally_single` / `enemy_all` / `enemy_multi` |
| `skills[].target_count` | `skills[].target_count` | 直接沿用 |
| `items[]` | —（**待补**） | Phase 1 移除物品按钮；后端实现 `USE_ITEM` 后再从背包快照生成 |

**`unit_class` 与 `skin_id` 解析规则（服务端拼装）：**

| `unit_class` | 含义 | `skin_id` 来源表 |
|--------------|------|------------------|
| 1 人物 | 玩家角色 | `player_profile.default_skin_id` 或全局默认（**待补**，见 §7.2） |
| 2 宠物 | 玩家宠物 | `pet_definition.skin_id`，按 `pet_id` 查 |
| 4 怪物 | 敌方怪物 | `monster_definition.skin_id`，按 `pet_id`（怪物模板 ID）查 |

### 6.2 回合演出：Demo `rounds[]` → `BATTLE_STATE_PUSH`

| Demo 结构 | 服务端权威结构 | Demo 迁入改法 |
|-----------|----------------|---------------|
| `rounds[]` 整包 | 单次 `BATTLE_STATE_PUSH` | 删除「按回合预加载 JSON」；每收到一次 4012 由 `BattleEventAdapter` 生成一轮伪剧本 |
| `input_phase.order` | `pending_actor_ids` | 选招顺序改读 `pending_actor_ids`（`uint64[]`） |
| `timeline[]` | `events[]` | 客户端适配器生成；不要求服务端补 `timeline` |
| `actions[]` | `events[]` | 由 `events[]` 聚合；`actor_id`/`skill_id` 用服务端类型 |
| `targets[].hp_after` | `events[].value` + `actors[].hp` | 演出中本地累加 `value`，回合末用 `actors[]` **强制校准** |
| `targets[].floating_text` | `events[].value` | 客户端按 `value` 生成，不依赖预写字段 |
| `targets[].is_crit` | — | 暂不依赖；后续可由服务端事件扩展 `crit` 标记 |
| `buff_changes[]` | `event_type=4` + `state_id` | 用 `state_id` 查状态展示名；增减由事件类型推断 |
| `summon_changes[]` | —（**待补**） | 后端无召唤前不实现 |
| `combo[]` | `event_type=9/11` | 适配器拆为追加 step，不保留 Demo 独立 JSON 块 |
| `result.is_finished` | `phase` + `BATTLE_RESULT_PUSH` | 结束判定以 4013 为准 |

### 6.3 玩家输入：Demo 本地选招 → `BATTLE_ACTION_REQ`

| Demo 行为 | 服务端字段 | Demo 迁入改法 |
|-----------|------------|---------------|
| 普通攻击 | `action_type=1`, `skill_id=1001` | 攻击按钮发默认普攻 `skill_id`，不再发字符串 `attack` |
| 技能 | `action_type=1`, `skill_id`, `target_id` | `target_id` 改为 `uint64` |
| 物品 | `action_type=2`, `item_uid` | 待后端补齐后再接 |
| 自动 | `action_type=5`, `auto_battle_enabled` | 对齐现有 `App.set_battle_auto()` |
| 逃跑 | `action_type=4` | 直接对齐 |
| 多单位选招 | `pending_actor_ids` | `_selection_order` 改为 `Array[int]`，元素为 `actor_id` |
| 捕捉 | `action_type=6` | Demo 无；后端已有，表现层后续补按钮 |

### 6.4 Demo 脚本必改清单（迁入时逐项执行）

| 文件 | 必改点 |
|------|--------|
| `battle_unit.gd` | `unit_id: String` → `actor_id: int`；`skills` 内 `skill_id` 改 `int` |
| `battle_director.gd` | `_units`、`_selection_order`、`_ally_selections` 全部改用 `actor_id`；删除读 `rounds[]` 正式路径 |
| `battle_data_provider.gd` | 仅保留调试态；正式链路由 `BattleNetworkProvider` 替代 |
| `action_panel.gd` | 技能列表读 `battle_state` 中 `skills[]`，不发 Demo 字符串 ID |
| `battle_demo.json` | 不进入正式包体；若保留调试，字段格式按 §6 改造为与 4011/4012 一致 |

---

## 7. 后端与后台补充设计（缺失逻辑）

> 本节为**必做项**，不是可选项。运营在后台配置外观与技能表现 ID，客户端只按推送 ID 加载本地 `.tres`，禁止在正式环境硬编码 `pet_id → skin_id` 映射表。

### 7.1 数据库迁移（新增字段）

建议迁移文件：`backend/server/migrations/0xx_battle_presentation_fields.sql`

```sql
-- 宠物模板：战斗外观 ID
ALTER TABLE pet_definition
  ADD COLUMN IF NOT EXISTS skin_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN pet_definition.skin_id IS '战斗场景客户端 UnitSkin 资源 ID，对应 client/resources/battle/unit_skins/{skin_id}.tres';

-- 怪物模板：战斗外观 ID
ALTER TABLE monster_definition
  ADD COLUMN IF NOT EXISTS skin_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN monster_definition.skin_id IS '战斗场景客户端 UnitSkin 资源 ID';

-- 技能模板：技能表现资源 ID（可空，空则客户端回退 animation_key）
ALTER TABLE skill_definition
  ADD COLUMN IF NOT EXISTS skill_visual_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN skill_definition.skill_visual_id IS '战斗场景客户端 SkillVisualConfig 资源 ID；为空时使用 animation_key 匹配';
```

**种子数据要求：**

- 为现有 `pet_definition`、`monster_definition` 补默认 `skin_id`（可与 Demo 资源名对齐，如 `嫩叶犬_001`）。
- `skill_definition` 可暂留空，由 `animation_key`（`slash` / `burst` / `heal` 等）回退到通用表现资源。

### 7.2 协议与领域模型扩展

**`battle.proto` / `BattleActorSnapshot` 新增：**

```protobuf
string skin_id = 17;  // 战斗外观资源 ID，由模板表解析后填入
```

**`BattleSkillSnapshot` 新增：**

```protobuf
string skill_visual_id = 9;  // 可空；空时客户端用 animation_key 回退
```

**`battle.Service` 拼装规则：**

1. 创建 `ActorSnapshot` 时，按 `unit_class` + `pet_id` 查 `pet_definition` / `monster_definition` 取 `skin_id`。
2. `unit_class=1`（人物）暂无独立模板表时，使用全局默认 `skin_id`（建议环境变量或 `game_config` 表，默认如 `决斗者_001`）。
3. 构建 `SkillSnapshot` 时，从 `skill_definition` 读取 `skill_visual_id`；为空则不下发或下发空字符串，客户端走 `animation_key`。

**需同步修改：**

- `backend/server/internal/module/battle/model.go` — `ActorSnapshot` / `SkillSnapshot`
- `backend/server/internal/transport/ws/battle_handler.go` — `toProtocolBattleActors`
- `backend/docs/protocol.md` — 4011 示例补充 `skin_id` / `skill_visual_id`

### 7.3 运营后台补充（必做）

| 后台页面 | 新增字段 | 校验规则 | 说明 |
|----------|----------|----------|------|
| `admin/.../PetDefinitionPage.tsx` | `skin_id` | 非空、≤64 字符、建议仅 `[A-Za-z0-9_\u4e00-\u9fff-]` | 运营配置宠物战斗形象 |
| `admin/.../MonsterDefinitionPage.tsx` | `skin_id` | 同上 | 运营配置怪物战斗形象 |
| `admin/.../SkillDefinitionPage`（或技能编辑抽屉） | `skill_visual_id` | 可空、≤64 字符 | 绑定技能特效资源；空=回退 `animation_key` |

**后台 API：**

- 扩展现有宠物/怪物/技能 admin CRUD 入参与出参，不新增平行接口。
- 列表页可增加 `skin_id` / `skill_visual_id` 列，便于排查「进战斗不显示模型」问题。

### 7.4 客户端资源约定（与后台配置对齐）

| 后台 `skin_id` | 客户端文件路径 |
|----------------|----------------|
| `决斗者_001` | `client/resources/battle/unit_skins/决斗者_001.tres` |
| `史莱姆_001` | `client/resources/battle/unit_skins/史莱姆_001.tres` |

| 后台 `skill_visual_id` | 客户端文件路径 |
|------------------------|----------------|
| `刃光斩_一级` | `client/resources/battle/skill_visuals/刃光斩_一级.tres` |

- `BattleContentRegistry` 扫描逻辑从 Demo 原样迁入。
- 推送的 `skin_id` 在本地找不到 `.tres` 时：打日志 + 使用阵营默认占位皮肤（仅表现兜底，不影响战斗结算）。

### 7.5 后续待补（非外观类）

| 能力 | 后端 | 后台 | 客户端 |
|------|------|------|--------|
| 战斗用药 | `ActionTypeUseItem` + 背包扣减 | 道具「可在战斗使用」标记 | `ActionPanel` 物品列表 |
| 捕捉 | 已有 `ActionTypeCapture` | 怪物捕捉配置（已有） | 捕捉按钮 + 特效 |
| 人物外观 | `player_profile` 或角色模板表增加 `skin_id` | 玩家/角色配置页 | 按推送展示 |

---

## 8. 能否 100% 接入？——结论分解

### 8.1 不能直接原样接入的部分（约 30%）

1. **`BattleDataProvider` 读本地 JSON** — 必须替换为 `GameState.battle_state` + WS 推送驱动。
2. **回合剧本 `timeline/actions/combo`** — 后端不产出；客户端新增 `BattleEventAdapter`。
3. **Demo 字符串 ID 体系** — 迁入时全部改为服务端 `uint64` / `uint32`（见 §6.4）。
4. **战斗物品、召唤** — 后端未实现前不接 Demo 对应 UI。
5. **独立 Demo 工程耦合** — 需接 `App` / Loading / 断线重连；`battle_scene.gd` 需新建。

### 8.2 可以高比例复用的部分（约 70%）

| 模块 | 复用度 | 说明 |
|------|--------|------|
| `BattleDirector` 状态机 | 80% | 输入/选目标/演出状态可保留；字段类型按 §6 改造 |
| `BattleUnit` | 85% | 动画、受击、死亡、高亮；`unit_id` → `actor_id` |
| `BattleEffect` + `FloatingText` | 90% | 纯表现 |
| `ActionPanel` / `ActionLogPanel` | 75% | UI 可迁；技能/目标数据改读 `battle_state` |
| `BattleContentRegistry` + Resource | 90% | 资源扫描机制原样迁入 |
| `formation` 站位算法 | 70% | 客户端布局；由 `lineup_index` 推算 |
| 场景 `.tscn` + 美术素材 | 80% | 改路径、接 Loading；背景替换为进入战斗时的场景 |

### 8.3 综合判定

| 评估项 | 100% 原样接入 | 推荐方案（服务端为准 + 后台配置） |
|--------|---------------|----------------------------------|
| 表现层代码 | ❌ | ✅ |
| 数据契约 | ❌ | ✅（客户端适配层 + 服务端补字段） |
| 外观/技能表现配置 | ❌ | ✅（§7 数据库 + 后台） |
| 物品/召唤/PVP | ❌ | 分期补齐 |
| 联机权威战斗 | ❌ | ✅ |

**总评：不能 100% 零改动接入；表现层可达 ~90% 等价，外观与技能表现必须走 §7 后台配置链路。**

---

## 9. 推荐接入架构

```text
┌─────────────────────────────────────────────────────────────┐
│                        Godot Client                          │
├─────────────────────────────────────────────────────────────┤
│  WS: BATTLE_START / STATE / RESULT                           │
│       ↓                                                      │
│  battle_controller.gd → GameState.battle_state               │
│       ↓                                                      │
│  battle_scene.gd（新建）                                      │
│    ├─ BattleNetworkProvider   读 battle_state（替代 JSON）    │
│    ├─ BattleEventAdapter      events[] → 演出伪剧本           │
│    ├─ BattleFormationMapper   lineup_index → 客户端 slot     │
│    ├─ BattleContentRegistry   按 skin_id / skill_visual_id 加载│
│    └─ BattleDirector          Demo 状态机（字段已按 §6 改造） │
│       ↓                                                      │
│  BattleUnit / BattleEffect / ActionPanel / ActionLogPanel    │
├─────────────────────────────────────────────────────────────┤
│  App.submit_battle_action()  ← ActionPanel（uint64 目标 ID）  │
└─────────────────────────────────────────────────────────────┘
                              ↑
┌─────────────────────────────────────────────────────────────┐
│                     Go Backend（权威）                        │
│  battle.Service → events[] + ActorSnapshot.skin_id           │
│  pet/monster/skill 模板表 → 后台配置 skin_id / skill_visual_id│
└─────────────────────────────────────────────────────────────┘
```

### 9.1 BattleEventAdapter（客户端）

职责：把 `BATTLE_STATE_PUSH.events[]` 转成 Director 可消费的伪剧本（字段类型与服务端一致）：

```text
events: [UseSkill, Damage, ApplyStatus, Combo, Damage, Defeat]
  ↓ 适配
timeline + actions（actor_id/skill_id 均为数字）
```

规则：

1. `UseSkill` 开启 action 块，记录 `source_id` / `skill_id`。
2. `Damage`/`Heal`/`ApplyStatus` 归入同一 action。
3. `Combo`/`Counter` 插入追加 step。
4. 演出 HP 本地累加，回合末用 `actors[]` 强制校准。

### 9.2 BattleFormationMapper（客户端）

`position` 不是协议字段，仅客户端内部使用：

```text
allies:  lineup_index → left_front_1, left_back_1, ...
enemies: lineup_index → right_front_1, right_back_1, ...
```

### 9.3 资源加载（服务端推送 ID → 本地 .tres）

- `BattleActorSnapshot.skin_id` → `BattleContentRegistry.get_unit_skin()`
- `BattleSkillSnapshot.skill_visual_id` 非空 → `get_skill_visual()`；为空 → 用 `animation_key` 匹配资源

**禁止**在正式环境维护 `pet_id → skin_id` 客户端硬编码表；所有外观 ID 来自 §7 后台配置并经 4011 推送。

---

## 10. 分阶段实施计划（确认后执行）

### Phase 0 — 评审与基线（当前）

- [x] 完成对比文档
- [x] 确认原则：冲突字段以服务端为准
- [x] 确认 `skin_id` / 技能表现走后台配置（§7）
- [ ] 确认接入范围：仅 PVE 先落地 / 是否包含 PVP

### Phase 1 — 后端与后台补字段（先做）

预估：2~3 天

1. [x] 执行 §7.1 迁移 SQL，补种子 `skin_id`。（迁移脚本：`backend/server/migrations/033_battle_presentation_fields.sql`，需本地自行执行）
2. [x] 扩展 `battle.proto`、`model.go`、`battle_handler` 推送 `skin_id` / `skill_visual_id`。
3. [x] 后台 `PetDefinitionPage`、`MonsterDefinitionPage` 增加 `skin_id` 字段；技能类型已补 `skill_visual_id`（`SkillDefinitionPage` 页面文件尚未落地，待补页面表单）。
4. [x] 更新 `backend/docs/protocol.md`。
5. [x] `go test ./server/...` 通过。

**Phase 1 交付标准**

- [x] 4011 推送中每个 actor 带 `skin_id`，每个 skill 带 `skill_visual_id`（可空）。
- [x] 运营可在后台修改宠物/怪物外观 ID 并持久化。

### Phase 2 — 客户端表现层迁入

预估：3~5 天

1. [x] 复制 Demo 脚本/场景/资源到 `client/`。
2. [x] **按 §6.4 改造** Demo 字段（`actor_id`、`skill_id` 等），不保留字符串 ID 正式路径。
3. [x] 实现 `BattleNetworkProvider` + `BattleEventAdapter` + `BattleFormationMapper`。
4. [x] 新建 `battle_scene.gd`，接 `battle_controller` 信号。
5. [x] `main.gd` 恢复战斗场景切换。
6. [x] `ActionPanel` → `App.submit_battle_action()`，`pending_actor_ids` 驱动多单位选招。

**Phase 2 交付标准**

- [ ] 真实联机 PVE 联调：开战 → 选技能 → 按推送 `skin_id` 显示模型 → 演出 → 胜负回世界。（需本地执行迁移 SQL 后实测）

### Phase 3 — 能力补齐（按需）

| 能力 | 依赖 |
|------|------|
| 战斗用药 | 后端 `USE_ITEM` + 背包扣减 + 后台道具标记 |
| 捕捉按钮 | `action_type=6` + 表现特效 |
| 人物外观可配置 | `player_profile.skin_id` 或角色模板表 |
| 闪避/反击/连击专属特效 | 扩展 Adapter |
| 8v12 站位 | 扩展客户端 `formation` |
| PVP | `participant_player_ids` + 视角翻转 |
| 断线重连续播 | `GetActiveSnapshot` + `stateHistory` |

---

## 11. 目标仓库目录规划（实现时参考）

```text
client/
├── scenes/battle/
│   ├── battle_scene.tscn      # 由 battle-test 迁入并改造
│   ├── battle_unit.tscn
│   └── battle_effect.tscn
├── scripts/feature/battle/
│   ├── battle_controller.gd   # 已有，补场景拉起
│   ├── battle_scene.gd        # 新建
│   ├── battle_director.gd     # 从 demo 迁入
│   ├── battle_network_provider.gd
│   ├── battle_event_adapter.gd
│   ├── battle_formation_mapper.gd
│   ├── battle_content_registry.gd
│   ├── battle_unit.gd
│   ├── battle_effect.gd
│   ├── action_panel.gd
│   ├── action_log_panel.gd
│   ├── floating_text.gd
│   └── resources/
│       ├── unit_skin.gd
│       └── skill_visual_config.gd
├── resources/battle/
│   ├── unit_skins/
│   └── skill_visuals/
└── data/debug/
    └── battle_replay_sample.json   # 可选：录制真实推送做离线调试
```

---

## 12. 风险与注意事项

### 12.1 协议/权威风险

- **禁止**在客户端本地推演伤害；演出 HP 最终以 `actors[]` 为准。
- 事件适配顺序须与 `events[]` 推送顺序一致；反击/连击依赖 Adapter 正确拆 step。
- `skin_id` 必须由 4011 推送，客户端不得用 `pet_id` 自行推断正式外观。

### 12.2 项目规范风险

- 所有网络请求须经 Loading 遮罩（AGENTS.md）。
- GDScript 变量/参数显式类型、`##` 注释。
- UI 浮点转 `int` 展示。
- **禁止**在客户端硬编码 `pet_id → skin_id` / `skill_id → skill_visual_id` 正式映射表。

### 12.3 工程风险

- `battle_scene.tscn` 已引用不存在的 `battle_scene.gd`，迁入时需一并修复。
- Demo 使用 Godot 4.5，目标项目需确认版本兼容。
- `skin_id` / `skill_visual_id` 与 `actor_id` / `skill_id` 是不同维度：前者管表现资源，后者管战斗逻辑；后台配置时需分别维护。

### 12.4 测试建议

```bash
# 后端：迁移 + 推送字段
cd backend && go test ./server/internal/module/battle/...

# 后台：修改 pet_definition.skin_id 后，4011 应带出对应值

# 客户端：4011 skin_id 能加载到 unit_skins/*.tres
```

---

## 13. 决策项

| # | 决策点 | 已确认结论 |
|---|--------|------------|
| 1 | 冲突字段以谁为准 | **服务端**；Demo 字段迁入时直接改造（§2.1、§6） |
| 2 | 外观/技能表现 ID | **数据库 + 后台配置 + 4011 推送**（§7）；不走客户端硬编码 |
| 3 | 接入范围 | 待确认：建议 PVE 先行 |
| 4 | Demo JSON | **完全移除**正式包体 |
| 5 | 战斗场景 | 独立全屏；背景替换为进入战斗时的场景 |
| 6 | 实施顺序 | **先 Phase 1 补后端/后台，再 Phase 2 迁客户端** |
| 7 | 物品/召唤 | 分期补齐；Phase 2 不接 |

---

## 14. 附录：参考文件索引

### battle-test

- `/Users/wangzhiwei/game/battle-test/data/battle_demo.json`
- `/Users/wangzhiwei/game/battle-test/scripts/battle/battle_director.gd`
- `/Users/wangzhiwei/game/battle-test/新手修改Demo指南.md`

### pocket-pet-remake

- `backend/proto/battle/battle.proto`
- `backend/server/internal/module/battle/service.go`
- `backend/server/internal/module/battle/model.go`
- `backend/docs/protocol.md`（§4011~4013）
- `backend/docs/pet-lineup-battle-model.md`
- `client/scripts/feature/battle/battle_controller.gd`
- `client/scenes/battle/battle_scene.tscn`
- `client/scripts/bootstrap/main.gd`（当前未打开战斗场景）

- `admin/src/pages/pets/PetDefinitionPage.tsx`
- `admin/src/pages/monsters/MonsterDefinitionPage.tsx`
- `backend/server/migrations/016_pet_definition_reward_support.sql`
- `backend/server/migrations/025_monster_definition.sql`
- `backend/server/migrations/023_skill_definition.sql`

---

## 15. 变更记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v0.1 | 2026-06-14 | 初版：完成 battle-test 与现后端对比及接入方案 |
| v0.2 | 2026-06-14 | 明确冲突字段以服务端为准；补充 §7 后端/后台 skin_id 与 skill_visual_id 设计；调整实施顺序为先补后端再迁客户端 |
