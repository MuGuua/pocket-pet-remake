# 宠物成长与属性加点

本文描述宠物实例的等级经验、升级发点、自由属性点分配与「资质 → 战斗属性」转化体系。公式来源为运营参考表 `docs/风车做资参考表（v6.2）.xlsx` 中 **《资质数据》表9 / 表10 / 表11** 与 **《综合计算器》** 的反推结果；服务端为唯一权威，客户端只展示与提交加点意图。

> 玩家角色成长见 `backend/docs/player-progression.md`。宠物与玩家使用**不同的属性维度与公式**，不可复用 `player_attr_convert_config`。

---

## 1. 设计目标

- 宠物最高 **100 级**；经验满则升级，溢出经验结转到下一级，可连升。
- 升级发放 **自由属性点**；升级/进化/转生还会增加各属性的 **系统自动分配点**（无需玩家操作）。
- 玩家对单只宠物主动分配五项自由点（生命 / 攻击 / 速度 / 法力 / 防御）。
- 战斗五维（`hp_max/atk/spd/mana/def`）由 **资质 + 属性点分配 + 转化率** 服务端重算，不写客户端本地公式。
- 等级曲线与转化率由后台可配；变更后刷新运行时缓存立即生效。
- 战斗发经验、任务发经验等入口统一走 `petprogression.ApplyExp`，结束只累加 `exp` 的旧行为废弃。
- 资质公式内的增幅（贴纸 / 天赋 / 法宝）与里世界潜力 **首期系数为 0**，表结构预留，避免以后改协议。
- 宠物技能中的永久属性被动先作为运行时加成叠到宠物快照；未来宠物装备也应接入同一类运行时加成口径。

---

## 2. 核心概念

| 概念 | 含义 |
|------|------|
| `player_pet.level` | 宠物当前等级，1~100 |
| `player_pet.exp` | 当前等级已累计、尚未升级的经验 |
| `player_pet.base_*_apt` | 发放时从模板快照的 **基础资质**（对应参考表「基础资质」） |
| `player_pet.extra_*_apt` | **红色资质**（提资/roll 超出基础部分；首期 wild_capture 可整段写入 extra） |
| `player_pet.free_attr_points` | 未分配的自由属性点 |
| `player_pet.alloc_*_points` | 玩家已手动分配到五项上的自由点累计 |
| `player_pet.evolution_level` | 进化等级（参考表 D15；首期默认 0，不参与计算） |
| `player_pet.rebirth_level` | 转生次数 0~2（参考表 D14；首期默认 0） |
| `player_pet.hp_max/atk/...` | 公式重算后的最终战斗属性（持久化，供战斗/列表读取） |
| `pet_level_config` | 每级升级所需经验、升级奖励自由点 |
| `pet_attr_convert_config` | 五项属性的「转化率」常数（见 §3.2） |

### 2.1 有效资质

对单项属性（以攻击为例）：

```
有效攻击资质 = 基础攻击资质 × 1.1 × 宠物倍率 + 红色攻击资质 × 1.2 × 宠物倍率
```

| 宠物类型 | 倍率 `aptitude_multiplier` | 说明 |
|----------|------------------------------|------|
| `normal` | 1.0 | 通常宠，默认 |
| `special` | 1.5 | 参考表「其他特殊」 |
| `arctic` | 3.2 | 参考表「北极宠」 |

倍率来自 `pet_definition.aptitude_profile`（首期种子全为 `normal`）。

### 2.2 属性点（分配点数）

单项属性最终参与计算的 **分配点数**：

```
分配点数_属性 = 系统自动点_属性 + 手动分配点_属性
```

**系统自动点**（参考《综合计算器》Q29~Q33，首期实现）：

| 属性 | 系统自动点 |
|------|------------|
| 生命 / 攻击 / 速度 / 法力 | `(level - 1) + floor(evolution_level × 3 / 4) + rebirth_alloc_属性` |
| 防御 | `(level - 1) + rebirth_alloc_防御`（进化不加防御自动点） |

`rebirth_alloc_*` 由 `pet_rebirth_config` 按转生档与等级段配置；**首期 `rebirth_level=0` 时恒为 0**。

**手动分配点**：升级获得的 `free_attr_points` 由玩家通过 WS 分配到 `alloc_hp_points` 等字段。

### 2.3 战斗属性计算公式（首期）

```
增幅总和 = 0   （首期：贴纸/天赋/法宝/里世界均不参与资质公式）

最终属性 = floor( 有效资质 / 转化率 × 分配点数 × (1 + 增幅总和) )
```

> 永久属性型宠物技能不写入 `增幅总和`，而是在宠物快照读取时按显式 `passive_attr_*` 配置叠加到 `hp_max/atk/spd/mana` 等运行时最终值。未来宠物装备接入时也应先聚合成同类运行时加成，再统一应用到宠物快照，避免客户端自行计算。

五项 **转化率**（《资质数据》表9，一点属性对应有效资质）：

| 属性 | 转化率 `convert_rate` | 一点有效资质（分配点=1、无增幅） |
|------|----------------------|----------------------------------|
| 生命 `hp_max` | 27.77 | +0.0360 HP |
| 攻击 `atk` | 277.77 | +0.00360 ATK |
| 速度 `spd` | 2081.51 | +0.000480 SPD |
| 法力 `mana` | 1388.73 | +0.000720 MANA |
| 防御 `def` | 277.77 | +0.00360 DEF |

**示例（攻击，通常宠，无增幅）**

- 基础攻资 12000，红色攻资 285372，等级 100，进化 0，手动攻击点 174  
- 自动攻击点 = 99  
- 分配点数 = 99 + 174 = 273  
- 有效攻资 = 12000×1.1 + 285372×1.2 = 355646.4  
- 攻击 = floor(355646.4 / 277.77 × 273 × 1) = **349,xxx**（与参考表同量级）

**反推（常见问题）**

> 给攻击加 1 点手动属性点，需要多少攻击资质才能 +1 攻击力？

在 **分配点数仅 +1、无增幅、通常宠** 时：

```
Δ攻击 = floor(有效攻资 / 277.77 × 1)  →  要 +1 攻击需有效攻资 ≥ 277.77
```

等价红色攻资约 **231.5**（277.77÷1.2），或基础攻资约 **252.5**（277.77÷1.1）。

### 2.4 防御与里世界（二期）

参考表对防御另有 **里世界潜力** 项：总资质 × (0.18~0.27) 额外转防御，且与法宝/天赋/贴纸叠加。首期不实现，预留：

- `player_pet.inner_world_potential`
- `pet_inner_world_config`

### 2.5 升级规则

1. 收到经验增量后读取 `level`、`exp` 与 `pet_level_config`。
2. 若 `exp + gain >= exp_required`：`exp -= exp_required`，`level++`，`free_attr_points += attr_points`。
3. 循环直到无法升级或 `level == 100`；100 级 `exp_required = 0`。
4. 每次实际升级后：重算五项系统自动点 implied 的最终属性，并将 `hp` 补至新 `hp_max`（若当前 `hp > 0`）。

### 2.6 加点规则

1. 客户端指定 `pet_uid` 与五项增量（可为 0）。
2. 校验：宠物归属当前玩家；`sum(delta) <= free_attr_points`；至少一项 > 0。
3. 持久化：累加 `alloc_*_points`、扣减 `free_attr_points`、按 §2.3 重算 `hp_max/atk/spd/mana/def`。
4. 写入 `pet_attr_allocate_log`（`reason_type=manual_allocate`）。
5. 推送 `PET_UPDATE_PUSH` 携带完整 `PetDetail`。

### 2.4 宠物战斗属性封顶

参考表未给出最终战斗属性硬上限；玩法约定如下（迁移 `053_pet_combat_stat_caps.sql` 写入 `pet_combat_stat_cap`，服务端读写时强制 `min(当前值, cap)`）：

| 属性 | stat_key | 封顶 |
|------|----------|------|
| 生命值 | `hp_max` | 1,500,000 |
| 精力 | `spirit` / `spirit_max` | 1,000 |
| 攻击 | `atk` | 250,000 |
| 防御 | `def` | 250,000 |
| 速度 | `spd` | 30,000 |
| 法力 | `mana` | 50,000 |
| 命中 | `hit_pct` | 250 |
| 闪避 | `dodge_pct` | 200 |
| 致命 | `crit_rate_pct` | 150 |
| 爆伤 | `crit_dmg_pct` | 2000 |
| 物理攻击抗性 | `physical_resist_pct` | 150 |
| 逆物 | `reverse_physical_resist_pct` | 100 |
| 技能攻击抗性 | `skill_resist_pct` | 150 |
| 逆技 | `reverse_skill_resist_pct` | 100 |
| 混乱/昏睡/麻痹/封印/诅咒 | `*_resist_pct` | 700 |
| 抗爆伤 | `crit_dmg_resist_pct` | 1000 |
| 抗致命 | `crit_resist_pct` | 100 |
| 抗人物 | `character_resist_pct` | 100 |
| 抗宠物 | `pet_resist_pct` | 100 |

资质公式重算的五项（`hp_max/atk/def/spd/mana`）在 `petprogression` 落库前截断；其余字段持久化在 `player_pet` 扩展列，默认 0，由后续玩法写入。

---

## 3. 数据库结构

迁移脚本：`backend/server/migrations/051_pet_progression.sql`（权限种子见 `052_admin_pet_progression_permissions.sql`）

### 3.1 配置表

**`pet_level_config`**

| 字段 | 说明 |
|------|------|
| `level` | 主键 1~100 |
| `exp_required` | 该等级升到下一级所需经验；100 级为 0 |
| `attr_points` | 升级奖励自由点；种子建议与玩家一致每级 1 点 |
| `status` | 1=启用 |

**`pet_attr_convert_config`**

| 字段 | 说明 |
|------|------|
| `attr_type` | `hp_max` / `atk` / `spd` / `mana` / `def` |
| `convert_rate` | 转化率（浮点，如 277.77） |
| `status` | 1=启用 |

种子数据使用表9默认值；后台可微调。

**`pet_rebirth_config`**（首期可建表种子为空）

| 字段 | 说明 |
|------|------|
| `rebirth_level` | 0~2 |
| `player_level_min` / `player_level_max` | 适用宠物等级段 |
| `bonus_free_points` | 转生一次性自由点（参考表 U28 公式拆配） |
| `alloc_hp` … `alloc_def` | 转生自动分配到五项的点数 |

### 3.2 `player_pet` 扩展字段

```sql
-- 资质拆分
base_hp_apt, base_atk_apt, base_def_apt, base_spd_apt, base_mana_apt  INTEGER NOT NULL DEFAULT 0
extra_hp_apt, extra_atk_apt, extra_def_apt, extra_spd_apt, extra_mana_apt INTEGER NOT NULL DEFAULT 0

-- 成长状态
free_attr_points INTEGER NOT NULL DEFAULT 0
alloc_hp_points, alloc_atk_points, alloc_spd_points, alloc_mana_points, alloc_def_points INTEGER NOT NULL DEFAULT 0
evolution_level INTEGER NOT NULL DEFAULT 0
rebirth_level INTEGER NOT NULL DEFAULT 0

-- 可选二期
inner_world_potential INTEGER NOT NULL DEFAULT 0
```

**迁移回填**

1. 已有 `hp_apt` 等视为 **总资质**：`base_*_apt` ← `pet_definition` 模板值；`extra_*_apt` ← `max(0, player_pet.*_apt - base_*_apt)`。
2. 按当前 `level` 与 §2.3 重算 `hp_max/atk/spd/mana/def` 覆写实例战斗值（替代模板静态值）。
3. 已有 `exp` 不自动连升，避免一次性升级风暴；迁移后可通过 `POST /api/admin/pet-progression/recalculate-combat-stats` 重算存量战斗属性。

### 3.3 审计表

**`pet_attr_allocate_log`**

| 字段 | 说明 |
|------|------|
| `pet_uid` | 宠物实例 |
| `player_id` | 所属玩家 |
| `delta_hp` … `delta_def` | 本次五项增量 |
| `free_before` / `free_after` | 自由点 |
| `reason_type` | `manual_allocate` / `admin_adjust` |
| `created_at` | 时间 |

---

## 4. 服务端模块

### 4.1 新模块 `module/petprogression`

职责（对齐 `module/progression` 模式）：

| 函数 | 说明 |
|------|------|
| `RefreshRuntimeCache` | 加载等级表、转化率、转生配置 |
| `ApplyExp(ctx, playerID, petUID, gain)` | 经验、连升、发点、重算战斗属性 |
| `AllocateAttrPoints(ctx, playerID, petUID, delta)` | 校验、持久化加点、重算 |
| `RecalculateCombatStats(state)` | 纯函数，供发放/后台改资质后调用 |
| `BuildProgressionState(pet)` | 从仓储组装的计算快照 |

核心纯函数签名建议：

```go
// EffectiveAptitude 计算单项有效资质。
func EffectiveAptitude(baseApt, extraApt uint32, multiplier float64) float64

// AutoAllocatedPoints 返回五项系统自动分配点。
func AutoAllocatedPoints(level, evolutionLevel, rebirthLevel uint32, rebirthCfg) PetAllocatedPoints

// TotalAllocatedPoints 系统自动 + 手动 alloc_*。

// FinalCombatStats 按 §2.3 返回五项 uint32 战斗属性。
func FinalCombatStats(input PetProgressionInput) PetCombatStats
```

仓储：`module/petprogression/repo.go`  
PostgreSQL：`data/postgres/pet_progression_repo.go`

### 4.2 接入点

| 模块 | 变更 |
|------|------|
| `pet.Service.UpdatePetBattleProgress` | 经验部分改调 `petprogression.ApplyExp`；HP 仍更新当前血量 |
| `reward.Service` | 宠物经验奖励走 `ApplyExp` |
| `pet` 发放链路 `insertRuntimePet` | 写入 `base_*_apt` / `extra_*_apt`，按 1 级公式初始化战斗属性 |
| `battle.Service` | 创建宠物 actor 时继续读 `player_pet` 持久化战斗值（已是公式结果） |
| `ws/pet_handler` | 新增 `PET_ALLOCATE_ATTR_REQ/RESP` |
| `transport/http/admin_handlers` | 新增 `/api/admin/pet-progression/...` |

### 4.3 与现有 `hp_apt` 字段关系

- **过渡期**：保留 `hp_apt`…`mana_apt` 作为 `base + extra` 的冗余合计，每次变更 extra 或 base 时同步更新，避免旧代码 NPE。
- **长期**：列表/详情统一读 `base_*_apt` + `extra_*_apt`；旧字段标记 deprecated。

---

## 5. WebSocket 协议

命令号建议（实现时写入 `protocol/command.go` 与 `client/scripts/common/command_ids.gd`）：

| 命令 | ID（建议） | 方向 |
|------|------------|------|
| `PET_ALLOCATE_ATTR_REQ` | 2063 | C→S |
| `PET_ALLOCATE_ATTR_RESP` | 2064 | S→C |

### 5.1 `PetDetail` 扩展

在现有 `PetDetail` 上增加：

```json
{
  "exp_to_next": 660,
  "free_attr_points": 3,
  "alloc_hp_points": 10,
  "alloc_atk_points": 25,
  "alloc_spd_points": 0,
  "alloc_mana_points": 0,
  "alloc_def_points": 0,
  "base_hp_apt": 12,
  "extra_hp_apt": 0,
  "growth_aptitudes": {
    "hp_apt": 12,
    "atk_apt": 11,
    "def_apt": 10,
    "spd_apt": 9,
    "mana_apt": 8
  },
  "auto_hp_points": 99,
  "auto_atk_points": 99
}
```

说明：

- `growth_aptitudes.*` = `base + extra`（兼容旧字段语义）。
- `auto_*_points` 服务端计算下发，供 UI 展示「系统点 + 手动点 = 总分配点」，无需客户端自算。
- 所有浮点中间值 **不下发**；展示用整数。

### 5.2 加点请求 / 响应

**`PetAllocateAttrReq`**

```json
{
  "pet_uid": 10001,
  "hp": 0,
  "atk": 1,
  "spd": 0,
  "mana": 0,
  "def": 0
}
```

**`PetAllocateAttrResp`**

```json
{
  "pet": { }
}
```

错误：`pet not found` / `insufficient attr points` / `invalid allocate attr input`。

### 5.3 战斗结算推送扩展

`BattleResultPush.pet_rewards[]` 每项增加：

```json
{
  "pet_uid": 10001,
  "exp_gained": 120,
  "level_up_count": 1,
  "attr_points_gained": 1,
  "free_attr_points": 4,
  "exp_to_next": 800
}
```

客户端合并后刷新宠物详情，**不本地预判升级**。

---

## 6. 后台管理

### 6.1 权限

| 权限键 | 说明 |
|--------|------|
| `pet_progression:view` | 查看宠物成长配置 |
| `pet_progression:edit` | 编辑配置 |

### 6.2 HTTP 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/pet-progression/level-configs` | 等级经验列表 |
| PUT | `/api/admin/pet-progression/level-configs/{level}` | 更新单级 |
| GET | `/api/admin/pet-progression/attr-convert-configs` | 转化率列表 |
| PUT | `/api/admin/pet-progression/attr-convert-configs/{attr_type}` | 更新单项转化率 |
| POST | `/api/admin/pet-progression/recalculate-combat-stats` | 按公式重算宠物战斗属性（body 可选 `player_id`/`pet_uid` 筛选） |

### 6.3 前端

- 方案 A：在 `admin/src/pages/progression/PlayerProgressionPage.tsx` 增加 Tab「宠物成长」。
- 方案 B：独立路由 `/pet-progression`。
- 玩家宠物详情 `PlayerPetSection` / `PetDefinitionPage` 展示 `base/extra` 资质与五项手动分配点。

---

## 7. 客户端对接

| 文件 | 职责 |
|------|------|
| `command_ids.gd` | `2063/2064` |
| `app.gd` | `request_pet_allocate_attr(pet_uid, delta)` + loading |
| `pet_controller.gd` | 处理 `PET_ALLOCATE_ATTR_RESP`、`PET_UPDATE_PUSH` |
| `game_state.gd` | 合并宠物快照；战斗结算合并 `pet_rewards` 升级字段 |
| 宠物详情 UI（新建或扩展 `status_panel` 宠物页） | 展示经验条、自由点、五项 `+1` |

交互约束（与玩家加点一致）：

1. 所有加点/打开详情请求走 **RequestLoadingOverlay**。
2. 以响应 `pet` 为准刷新，禁止本地公式预判最终攻击/生命。
3. UI 展示整数；中间公式浮点只在服务端。

---

## 8. 经验发放入口

| 入口 | 现状 | 目标 |
|------|------|------|
| PVE 战斗胜利 | `UpdatePetHPAndExpByUID` 只加 exp | 改 `petprogression.ApplyExp` |
| 任务奖励 | 若已有宠物经验 | 同上 |
| 道具「宠物经验」 | 背包使用后 | 同上（后续） |

---

## 9. 实现分期

### 阶段 A：公式与迁移（服务端可测）

- [x] `051_pet_progression.sql`
- [x] `module/petprogression` + 单元测试
- [x] 发放/迁移回填 + `RecalculateCombatStats` 运维入口（Admin `POST /api/admin/pet-progression/recalculate-combat-stats`）

### 阶段 B：升级与 WS 加点

- [x] `ApplyExp` 接入战斗结算
- [x] `PET_ALLOCATE_ATTR_*` + `PetDetail` 扩展
- [x] `pet_attr_allocate_log`

### 阶段 C：后台与客户端

- [x] Admin 成长配置页（`/pet-progression`，权限见 `052_admin_pet_progression_permissions.sql`）
- [x] 客户端宠物加点 UI + loading
- [x] 更新 `backend/docs/protocol.md`
- [x] WS `2063/2064`、Admin `/api/admin/pet-progression/`

### 阶段 D：二期增强

- [ ] 进化 / 转生配置生效
- [ ] 增幅（贴纸/天赋/技能/法宝）系数表
- [ ] 宠物装备运行时加成链路（先聚合装备属性，再与永久被动技能一起应用到宠物最终快照）
- [ ] 里世界潜力防御项
- [ ] 提资修改 `extra_*_apt` 的后台与玩法链路
- [ ] 洗点 / 批量加点

---

## 10. 测试用例（最低集）

1. **公式**：基础 12000 + 红色 285372、分配 273 攻击点 → 与参考表数量级一致（误差仅来自 floor）。
2. **+1 点攻击**：有效攻资 277.77 → 攻击 +1。
3. **升级**：99 级满经验升 100，不再发点；`exp_to_next=0`。
4. **加点**：自由点 5，请求 atk+3+hp+2 成功；atk+3 失败。
5. **发放**：模板宠物 base 资质快照正确，extra=0，1 级属性与公式一致。
6. **wild_capture**：roll 资质进入 extra，重算后战斗值变化。
7. **战斗**：升级后出战宠 `BATTLE_START_PUSH` 中 atk/hp_max 为新值。

验证命令：

```bash
cd backend && go test ./server/internal/module/petprogression/...
cd backend && go test ./server/...
```

---

## 11. 参考

- 运营表：`docs/风车做资参考表（v6.2）.xlsx`
  - 《资质数据》表9：转化率
  - 《资质数据》表10/11：属性点 ↔ 资质反推
  - 《综合计算器》：最终属性公式与属性点组成
- 玩家成长对照：`backend/docs/player-progression.md`
- 宠物实例模型：`backend/docs/pet-lineup-battle-model.md`
