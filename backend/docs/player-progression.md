# 玩家成长与属性加点

本文描述首期「仅玩家角色」的等级经验、升级发点、自由属性点分配与战斗属性转化体系。宠物升级与宠物加点属于二期，当前宠物 `exp` 仍只累加、不触发升级。

## 1. 设计目标

- 玩家最高 100 级；经验满则升级，溢出经验结转到下一级，可连升。
- 升级发放自由属性点，并按 `player_level_config` 的 `bonus_*` 累加裸装基础战斗值（`base_atk/base_hp_max/base_spd/base_mana`）。
- 玩家主动分配四维属性点（力量 / 体质 / 敏捷 / 灵力），服务端按配置转化率重算战斗属性。
- 等级经验曲线与属性转化率由后台可配，变更后服务端刷新运行时缓存立即生效。
- 所有成长结算在服务端权威完成，客户端只展示与提交加点意图。

## 2. 核心概念

| 概念 | 含义 |
|------|------|
| `player.level` | 玩家当前等级，范围 1~100 |
| `player.exp` | 当前等级已累计、尚未升级的经验 |
| `player_level_config.level` | **处于该等级时**升到下一级所需经验与升级奖励 |
| `player.free_attr_points` | 未分配的自由属性点 |
| `player.strength/vitality/agility/mind` | 已分配到四维上的点数 |
| `player.base_*` | 裸装基础战斗值（不受加点影响） |
| `player.atk/def/...` | 最终战斗属性 = `base_*` + 加点转化加成 |

### 2.1 升级规则

1. 服务端收到经验增量后，读取当前 `level`、`exp` 与 `player_level_config`。
2. 若 `exp + gain >= exp_required`，则扣除 `exp_required`、`level++`，发放 `attr_points` 到 `free_attr_points`，并将当前等级行的 `bonus_atk/bonus_hp_max/bonus_spd/bonus_mana` 累加到 `base_*`。
3. 循环直到经验不足升级或达到 100 级。
4. 100 级时 `exp_required = 0`，不再升级；多余经验保留在 `exp` 字段中。

### 2.2 加点规则

1. 客户端通过 WebSocket 提交本次要分配的四维增量（可为 0）。
2. 服务端校验：`delta 总和 <= free_attr_points`，且至少有一个维度 > 0。
3. 持久化：累加已分配四维、扣减 `free_attr_points`、按转化率重算 `atk/def/spd/mana/hp_max/hit_pct/dodge_pct`。
4. 写入 `player_attr_allocate_log` 审计记录（`reason_type=manual_allocate`）。

### 2.3 战斗属性计算公式

```
最终属性 = base_属性 + Σ(已分配源属性点数 × convert_rate)
```

默认种子转化率（可在后台调整）：

| 源属性 | 目标属性 | 默认转化率 |
|--------|----------|------------|
| strength | atk | 3 |
| vitality | hp_max | 50 |
| vitality | def | 2 |
| agility | spd | 2 |
| agility | dodge_pct | 1 |
| mind | mana | 4 |

迁移 `035` 会把现有 `atk/def/...` 复制到 `base_*`，保证未加点玩家战力不变。

## 3. 数据库结构

迁移脚本：

- `backend/server/migrations/035_player_level_progression.sql`
- `backend/server/migrations/036_admin_player_progression_permissions.sql`
- `backend/server/migrations/043_player_level_combat_bonus.sql`

### 3.1 配置表

**`player_level_config`**

| 字段 | 说明 |
|------|------|
| `level` | 主键，1~100 |
| `exp_required` | 该等级升到下一级所需经验；100 级为 0 |
| `attr_points` | 从该等级升到下一级时发放的自由属性点；默认种子为每级 1 点（100 级为 0） |
| `bonus_atk` | 从该等级升到下一级时累加到 `base_atk` 的裸装加成；默认种子 1~99 级为 7 |
| `bonus_hp_max` | 累加到 `base_hp_max`；默认种子 1~99 级为 38 |
| `bonus_spd` | 累加到 `base_spd`；默认种子 1~99 级为 2 |
| `bonus_mana` | 累加到 `base_mana`；默认种子 1~99 级为 1 |
| `status` | 1=启用，0=停用 |

**`player_attr_convert_config`**

| 字段 | 说明 |
|------|------|
| `id` | 主键 |
| `source_attr` | `strength` / `vitality` / `agility` / `mind` |
| `target_attr` | `atk` / `def` / `spd` / `mana` / `hp_max` / `hit_pct` / `dodge_pct` |
| `convert_rate` | 每 1 点源属性转化的目标属性增量 |
| `status` | 1=启用，0=停用 |

### 3.2 玩家表扩展字段

`player` 新增：

- `free_attr_points`
- `strength`, `vitality`, `agility`, `mind`
- `base_hp_max`, `base_atk`, `base_def`, `base_spd`, `base_mana`, `base_hit_pct`, `base_dodge_pct`

### 3.3 审计表

**`player_attr_allocate_log`**：记录每次加点的四维增量、`free_before/free_after`、操作者与原因。

## 4. 服务端模块

### 4.1 `module/progression`

职责：

- 从数据库加载等级配置与转化率配置到内存缓存（`RefreshRuntimeCache`）
- `ApplyExp`：经验结算、连升、溢出结转
- `AllocateAttrPoints`：校验并持久化加点
- `UpsertAdminLevelConfig` / `UpsertAdminAttrConvertConfig`：后台改配置后自动刷新缓存

仓储接口：`module/progression/repo.go`  
PostgreSQL 实现：`data/postgres/progression_repo.go`

### 4.2 接入点

| 模块 | 行为 |
|------|------|
| `player.Service.AddExp` | 委托 `progression.ApplyExp` |
| `player.Service.AllocateAttrPoints` | 委托 `progression.AllocateAttrPoints` 并返回最新 `Profile` |
| `reward.Service` | 发经验走 `AddExp`，返回 `LevelUpCount`、`AttrPointsGained` |
| `ws/battle_handler` | `BATTLE_RESULT_PUSH` 附带升级与属性点字段 |
| `ws/player_handler` | 处理 `PLAYER_ALLOCATE_ATTR_REQ/RESP` |

## 5. WebSocket 协议

### 5.1 玩家快照扩展（`PlayerSnapshot`）

`EnterWorldResp.player` 与加点响应均携带：

```json
{
  "level": 12,
  "exp": 340,
  "exp_to_next": 660,
  "free_attr_points": 5,
  "strength": 3,
  "vitality": 2,
  "agility": 1,
  "mind": 0,
  "atk": 33,
  "def": 16,
  "hp_max": 200
}
```

`exp_to_next` 由服务端按当前等级配置计算：`exp_required - exp`（满级为 0）。

### 5.2 属性点分配

| 命令 | ID | 方向 |
|------|-----|------|
| `PLAYER_ALLOCATE_ATTR_REQ` | 2061 | 客户端 → 服务端 |
| `PLAYER_ALLOCATE_ATTR_RESP` | 2062 | 服务端 → 客户端 |

**请求体 `PlayerAllocateAttrReq`**

```json
{
  "strength": 1,
  "vitality": 0,
  "agility": 0,
  "mind": 0
}
```

**响应体 `PlayerAllocateAttrResp`**

```json
{
  "player": { }
}
```

`player` 为完整 `PlayerSnapshot`，客户端应调用 `GameState.merge_player_snapshot()` 合并。

**错误码**

| 场景 | 错误 |
|------|------|
| 自由点不足 | `insufficient attr points` |
| 增量全为 0 或非法 | `invalid allocate attr input` |

### 5.3 战斗结算推送扩展（`BattleResultPush`）

胜利结算新增字段：

```json
{
  "reward_player_exp": 120,
  "player_level": 13,
  "level_up_count": 1,
  "attr_points_gained": 5,
  "free_attr_points": 10,
  "exp_to_next": 800
}
```

客户端 `GameState.apply_battle_player_rewards()` 会合并上述字段到 `player_snapshot`。

## 6. 后台管理

### 6.1 权限

| 权限键 | 说明 |
|--------|------|
| `player_progression:view` | 查看成长配置 |
| `player_progression:edit` | 编辑成长配置 |

### 6.2 HTTP 接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/admin/player-progression/level-configs` | view | 等级经验列表 |
| PUT | `/api/admin/player-progression/level-configs/{level}` | edit | 更新单级配置 |
| GET | `/api/admin/player-progression/attr-convert-configs` | view | 转化率列表 |
| PUT | `/api/admin/player-progression/attr-convert-configs/{id}` | edit | 更新单条转化率 |

**更新等级配置请求体**

```json
{
  "exp_required": 1200,
  "attr_points": 6,
  "status": 1
}
```

**更新转化率请求体**

```json
{
  "source_attr": "strength",
  "target_attr": "atk",
  "convert_rate": 3,
  "status": 1
}
```

### 6.3 前端页面

- 路由：`/player-progression`
- 页面：`admin/src/pages/progression/PlayerProgressionPage.tsx`
- 两个 Tab：等级经验表、属性转化率表；均支持行内编辑弹窗

玩家详情页（`/players`）现已展示 `free_attr_points` 与四维已分配点数。

## 7. 客户端对接

| 文件 | 职责 |
|------|------|
| `client/scripts/common/command_ids.gd` | `2061/2062` 命令号 |
| `client/autoload/app.gd` | `request_allocate_attr_points()` |
| `client/autoload/game_state.gd` | `merge_player_snapshot()`、`apply_battle_player_rewards()` |
| `client/scripts/feature/player/player_controller.gd` | 处理 `PLAYER_ALLOCATE_ATTR_RESP` |
| `client/scripts/ui/status_panel.gd` | 展示经验、自由点、四维属性 |
| `client/scripts/ui/points_status_panel.gd` | `+1` 加点按钮 + loading 遮罩 |

加点交互要求：

1. 点击 `+1` 前显示通用 loading（请求超过 1 秒才露出遮罩，避免闪屏）。
2. 等待 `PLAYER_ALLOCATE_ATTR_REQ` 对应序号完成后再关闭 loading。
3. 不本地预判结果；以响应中的 `player` 快照为准刷新 UI。

## 8. 经验发放入口

当前已接入：

- PVE 战斗胜利结算（`battle` → `reward` → `player.AddExp`）
- 任务奖励发经验（`reward` 模块）

所有入口统一走 `progression.ApplyExp`，避免客户端或服务端其他模块各自实现升级逻辑。

## 9. 二期范围（当前未做）

- 宠物等级上限、宠物升级曲线、宠物自由属性点
- 洗点、重置属性、批量加点
- 升级推送独立消息（当前通过战斗结算字段或 `EnterWorld` 快照同步）
- 后台直接给玩家加减自由属性点（需另补审计接口）

## 10. 部署与验证

1. 执行迁移 `035`、`036`、`037`（若已执行过 `035`，单独执行 `037` 即可把升级属性点改为每级 1 点）。
2. 重启 `game-server`（启动时会 `RefreshRuntimeCache` 加载配置）。
3. 后台打开 `/player-progression` 确认 100 级配置与转化率种子数据。
4. 客户端进入世界 → 状态面板 → 加点页，确认 `+1` 后战斗属性按转化率变化。
5. 打完一场 PVE，确认结算日志与 `level_up_count` / `attr_points_gained` 展示正确。

验证命令：

```bash
cd backend && go test ./server/...
```
