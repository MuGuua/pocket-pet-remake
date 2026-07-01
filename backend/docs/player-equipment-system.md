# 玩家人物装备系统

本文描述**玩家角色**可穿戴装备的模板配置、实例化、佩戴、强化、镶嵌与套装效果。服务端为唯一权威；客户端只展示与提交操作意图。

> 宠物法宝装备见 `backend/docs/pet-skill-slots.md`，与本系统独立。  
> 物品模板通用字段见 `docs/背包系统开发文档.md` §10。  
> 玩家裸装 + 加点公式见 `backend/docs/player-progression.md`。

---

## 1. 设计目标

- 人物装备 **13 个部位**，每部位 **最多 1 件**。
- 装备有 **名称、介绍、基础属性、强化加成、镶嵌加成、套装效果**。
- 部分装备可强化至 **+15**，部分 **不可强化**；强化为 **线性叠加**（例：每级攻击 +100，+3 共 +300）。
- 部分装备支持 **镶嵌孔**，嵌入宝石/符文类物品获得额外属性。
- 装备有 **佩戴等级**（`required_level`），不足等级不可佩戴。
- 所有模板走 **数据库 + Admin 配置**，不在代码硬编码正式装备。
- 佩戴/卸下/强化/镶嵌后，服务端 **重算并持久化** 玩家最终战斗属性，战斗与世界读取权威快照。

---

## 2. 装备部位

| slot_key | 中文 | 说明 |
|----------|------|------|
| `weapon` | 武器 | 主武器 |
| `hat` | 帽子 | 头部防具 |
| `clothes` | 衣服 | 上身防具 |
| `pants` | 裤子 | 下身防具 |
| `shoes` | 鞋子 | 脚部防具 |
| `necklace` | 项链 | 饰品 |
| `ring` | 戒指 | 饰品 |
| `hero_ring` | 英雄之戒 | 特殊戒指槽，通常高阶 |
| `medicine_pouch` | 药囊 | **战后自动回满**人物/宠物生命与精力；未佩戴则保留上一场战斗剩余值（见 §6.5） |
| `class_badge` | 职业徽章 | 职业限定饰品 |
| `class_weapon` | 职业武器 | 与 `weapon` 并存；职业专属第二武器槽；**属性与主武器全额叠加** |
| `costume` | 时装 | **仅改变外观**，不参与战斗属性重算 |
| `element_bracelet` | 元素手镯 | 元素属性向饰品 |

约束：

- `(player_id, equip_slot)` **唯一**，同一槽位只能挂 1 个 `item_uid`。
- `weapon` 与 `class_weapon` 可同时佩戴，互不替换；两者属性 **均为固定数值加法，全额叠加**，不做乘算或折扣。
- `costume` 固定 `appearance_only=true`：只写 `appearance_skin_id` 影响展示层，**零战斗属性**、不参与 `stats.go` 聚合。
- 所有装备、强化、镶嵌、套装效果均为 **+固定数值** 叠加，禁止出现乘积型属性合成。

---

## 3. 与现有表的关系

项目迁移 `013_bag_wallet_foundation.sql` 已预留：

| 已有表 | 用途 | 本系统用法 |
|--------|------|------------|
| `item_definition` | 物品模板主表 | `item_type = 'equipment'` |
| `item_equipment_extra` | 装备模板扩展 | **扩展字段**（见 §4.1） |
| `equipment_instance` | 装备实例 | `enhance_level`、镶嵌状态、绑定 |
| `player_container_item` | 背包格子 | 未佩戴时 `item_uid` 在 bag/warehouse |

**新增**：

| 新表 | 用途 |
|------|------|
| `player_equipment_slot` | 玩家已佩戴映射 |
| `equipment_enhance_rule` | 强化每级属性增量（可按模板或品质复用） |
| `equipment_enhance_success_config` | 穿戴等级段 + 强化目标等级 → 成功率（见 §6.4） |
| `equipment_enhance_cost` | 强化材料消耗（按目标等级） |
| `equipment_socket_config` | 模板镶嵌孔数量与允许宝石类型 |
| `equipment_instance_socket` | 实例各孔当前嵌入物 |
| `equipment_set_definition` | 套装定义 |
| `equipment_set_piece` | 套装 ↔ 装备模板 |
| `equipment_set_effect` | 件数 → 属性加成 |
| `item_gem_extra` | 宝石/镶嵌物模板扩展（可选，与 functional 分离） |
| `item_medicine_pouch_extra` | 药囊模板：战后恢复范围（人物/宠物、hp/spirit 等） |

> **战斗属性封顶**：人物与宠物 **共用** 现有 `pet_combat_stat_cap` 表（字段完全一致）。实现期可将 Admin 文案改为「战斗属性封顶（人物/宠物共用）」；后续迁移可 rename 为 `combat_stat_cap`，首期不拆两套封顶。

---

## 4. 数据库设计

### 4.1 扩展 `item_equipment_extra`

在现有字段基础上增加（迁移 `058_player_equipment_foundation.sql` 建议）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `equip_slot` | VARCHAR(32) | 已有；取值见 §2 |
| `can_enhance` | BOOLEAN | 是否可强化，默认 true |
| `max_enhance_level` | INT | 最大强化等级，不可强化时为 0；可强化默认 15 |
| `set_id` | BIGINT | 所属套装 ID，0=无套装 |
| `appearance_skin_id` | VARCHAR(64) | 时装专用：覆盖玩家外观资源 ID |
| `base_hit_pct` … | BIGINT | 扩展基础战斗属性（与玩家/宠物次要属性对齐） |
| `enhance_rule_id` | BIGINT | 关联 `equipment_enhance_rule`，0=使用默认规则 |
| `socket_config_id` | BIGINT | 关联 `equipment_socket_config`，0=不可镶嵌 |
| `extra_rule_json` | JSONB | 已有；扩展 `career_required`、`gender_limit`、`appearance_only` 等 |

基础五维仍用已有 `base_hp/base_mana/base_atk/base_def/base_spd`；次要属性放新增 `base_*_pct` 列或统一 `base_stats_json JSONB`（推荐 JSONB 便于 Admin 扩展，首期可先列化常用项）。

### 4.2 `player_equipment_slot`

```sql
CREATE TABLE player_equipment_slot (
  player_id   BIGINT NOT NULL REFERENCES player(id),
  equip_slot  VARCHAR(32) NOT NULL,
  item_uid    VARCHAR(64) NOT NULL REFERENCES equipment_instance(item_uid),
  equipped_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (player_id, equip_slot),
  UNIQUE (item_uid)
);
```

- 佩戴：校验等级、职业、槽位 → 写入本表 → `equipment_instance.state = 'equipped'` → 从背包格子移除引用。
- 卸下：删本表行 → `state = 'bag'` → 写回背包空位。

### 4.3 `equipment_enhance_rule`

定义「每强化 1 级」增加的属性（线性叠加）：

| 字段 | 说明 |
|------|------|
| `rule_id` | 主键 |
| `rule_code` | 运营编码，如 `weapon_atk_100` |
| `rule_name` | 展示名 |
| `per_level_stats_json` | 例：`{"atk":100}` 或 `{"atk":100,"def":50}` |
| `status` | 1=启用 |

**强化总加成**：

```
enhance_bonus[stat] = per_level_stats_json[stat] × enhance_level
```

例：`per_level atk=100`，`enhance_level=3` → 攻击 +300。

### 4.4 镶嵌

**`equipment_socket_config`**

| 字段 | 说明 |
|------|------|
| `config_id` | 主键 |
| `socket_count` | 孔数 0~N |
| `allowed_gem_types_json` | 允许嵌入的 gem 子类型，如 `["attack","defense"]` |

**`equipment_instance_socket`**

| 字段 | 说明 |
|------|------|
| `item_uid` | 装备实例 |
| `socket_index` | 孔位 0..N-1 |
| `gem_item_id` | 嵌入物模板 ID；NULL=空孔 |
| `gem_quantity` | 通常 1 |

**`item_gem_extra`**（`item_type = 'gem'`）

| 字段 | 说明 |
|------|------|
| `item_id` | 主键 |
| `gem_type` | 攻击石 / 防御石等 |
| `stats_json` | 嵌入后加成，如 `{"atk":80}` |

镶嵌消耗背包中 1 个宝石物品；**取下镶嵌无损**：宝石回到背包，孔位清空，属性重算（见 §7.5）。

### 4.6 药囊 `item_medicine_pouch_extra`

| 字段 | 说明 |
|------|------|
| `item_id` | 主键，对应 `equip_slot=medicine_pouch` 的装备 |
| `restore_player_hp` | 战后是否回满玩家 `hp` → `hp_max` |
| `restore_player_spirit` | 战后是否回满玩家 `spirit` → `spirit_max` |
| `restore_player_vigor` | 战后是否回满玩家 `vigor` → `vigor_max`（默认 true，与 spirit 一并恢复） |
| `restore_pet_hp` | 战后是否回满**当前出战宠物** `hp` → `hp_max` |
| `restore_pet_spirit` | 战后是否回满出战宠物 `spirit` → `spirit_max` |
| `restore_lineup_pets` | 是否恢复整队宠物（false 时仅出战宠；首期默认 false） |

默认种子建议五项全 true。药囊 **不提供战斗属性**，仅提供战后恢复被动。

### 4.5 套装

**`equipment_set_definition`**

| 字段 | 说明 |
|------|------|
| `set_id` | 主键 |
| `set_code` | 唯一编码 |
| `set_name` | 套装名 |
| `description` | 套装介绍 |
| `max_pieces` | 套装件数上限（用于 UI） |
| `status` | 启用 |

**`equipment_set_piece`**

| 字段 | 说明 |
|------|------|
| `set_id` | 套装 |
| `item_id` | 装备模板 |
| PRIMARY KEY (set_id, item_id) |

**`equipment_set_effect`**

| 字段 | 说明 |
|------|------|
| `set_id` | 套装 |
| `piece_count` | 激活件数，如 2 / 4 / 6 |
| `effect_name` | 效果名 |
| `effect_desc` | 效果说明（UI） |
| `stats_json` | 属性加成 |
| `special_effect_json` | 预留：触发技能、被动等 |

同一件装备通过 `item_equipment_extra.set_id` 或 `equipment_set_piece` 关联套装；**以 piece 表为准**，模板 `set_id` 作冗余展示。

**套装激活件数**：统计玩家当前已佩戴、且 `item_id` 属于该 `set_id` 的 **不同模板件数**（同模板不重复计数）。

---

## 5. 属性合成公式

与 `player-progression` 衔接，推荐 **分层加法叠加**（禁止乘积）：

```
progression_part = f(base_*, 四维加点, player_attr_convert_config)
equipment_part   = Σ(单件装备贡献)
set_part         = Σ(已激活套装效果)

最终属性 = progression_part + equipment_part + set_part
```

**单件装备贡献**（`costume`、`medicine_pouch` **不参与** 本公式）：

```
piece = template.base_stats
      + enhance_rule.per_level_stats × enhance_level
      + Σ(socket.gem.stats)
```

重算时机：

- 佩戴 / 卸下
- 强化成功
- 镶嵌 / 移除镶嵌
- Admin 修改玩家装备或模板（运维）
- 玩家升级导致 `required_level` 变化时 **不自动卸装**；仅阻止新佩戴，已有佩戴保持（可调策略）

持久化策略：

- `player.base_*` 仍只存裸装 + 升级 bonus + 加点转化前的 base 层（不变）。
- 最终 `player.atk/hp_max/...` 在重算后写回，与现有 `progression` 模块一致。
- 可选审计表 `player_equipment_recalc_log` 记录前后 diff（运维排查）。

**封顶**：人物与宠物共用 `pet_combat_stat_cap`（`stat_key` 字段一致）。重算最终属性后对 `hp_max/atk/def/...` 及全部次要抗性字段执行 `ClampCombatStats`；宠物模块已有实现，人物装备重算后走同一套 `CombatStatCaps` 加载逻辑。

---

## 6. 强化规则

| 项 | 规则 |
|----|------|
| 等级范围 | `0 ~ max_enhance_level`，默认最大 15 |
| 不可强化 | `can_enhance=false` 或 `max_enhance_level=0`；**时装固定不可强化** |
| 加成方式 | **线性**：每级固定增量，总加成 = 增量 × 当前等级 |
| 实例字段 | `equipment_instance.enhance_level`（已有） |
| 失败惩罚 | **等级不降**；已消耗材料不返还 |
| 成功率 | 按 **穿戴等级段 + 目标强化等级** 查表（见 §6.4） |

### 6.4 强化成功率（已确认）

从 `enhance_level = N-1` 强化到 `N` 的成功概率，需同时匹配：

1. 装备 `required_level` 所属穿戴段（1~10、11~20、21~30…，每10级一段）
2. 目标强化等级 N

**1~10 级穿戴段默认成功率：**

| 目标等级 N | 成功率 |
|------------|--------|
| +1 | 100% |
| +2 | 100% |
| +3 | 100% |
| +4 | 90% |
| +5 | 90% |
| +6 | 90% |
| +7 | 75% |
| +8 | 75% |
| +9 | 65% |
| +10 | 55% |
| +11 | 45% |
| +12 | 35% |
| +13 | 25% |
| +14 | 15% |
| +15 | 10% |

持久化到 `equipment_enhance_success_config(target_level, required_level_min, success_rate_pct)`；后台 **物品模板 → 强化成功率** Tab 先选穿戴等级段再编辑（API：`GET /api/admin/equipment-enhance-success-configs?required_level_min=`、`PUT /api/admin/equipment-enhance-success-configs/{required_level_min}/{target_level}`）。迁移 `070` 会为 11~91 段生成默认种子（在 1~10 段基础上每升一段整体 -5%，下限 1%）。

服务端 `Enhance` 流程：扣材料 → 掷骰 → 成功则 `enhance_level++` 并重算属性 → 响应 `{ success, new_level, rate_pct }`。

### 6.5 药囊与战后生命/精力（已确认）

**佩戴药囊时**（战斗正常结算后，在 `battle` 模块写回战斗结果 **之后** 执行）：

1. 读取 `player_equipment_slot.medicine_pouch` 对应模板的 `item_medicine_pouch_extra`。
2. 按配置将玩家 `hp/spirit/vigor` 补至上限；将**出战宠物**（首期）的 `hp/spirit` 补至上限。
3. 持久化写回 `player` 与 `player_pet`，并随 `BattleResultPush` 或紧随其后的快照推送刷新客户端。

**未佩戴药囊时**：

- 战斗结束时 **仅持久化战斗中剩余** 的 `hp`、`spirit`（宠物同理）；**不**自动回满。
- 下次进入战斗（含 PVE/PVP）从数据库读取 **上一场结束时的剩余值** 作为初始 `hp/spirit`。
- 世界场景内展示的血条/精力条与库内一致。

**注意**：药囊不影响 `hp_max/spirit_max` 等上限属性，只影响战后当前值恢复；上限仍由成长 + 装备 + 套装 + 封顶决定。

Admin 配置示例：

- 武器 A：`enhance_rule_id=1`，`per_level={"atk":100}`
- 时装 B：`can_enhance=false`，无 base_stats
- 药囊 D：仅 `item_medicine_pouch_extra`，无 base_stats
- 项链 C：`max_enhance_level=10`，`per_level={"hp_max":500}`

---

## 7. 运行时流程

### 7.1 佩戴 EQUIP

1. 客户端：背包点「佩戴」或装备面板空槽 → 选背包中 `item_type=equipment` 的实例。
2. 服务端校验：
   - 实例归属、`state=bag`
   - `item_equipment_extra.equip_slot` 与目标槽一致
   - `player.level >= item_definition.required_level`
   - `career_limit` / `class_badge` 等 `extra_rule_json` 限制
3. 若该槽已有装备 → 先卸下旧装备回背包（或返回「槽位已满」由客户端先卸）。
4. 事务：写 `player_equipment_slot`、更新 instance.state、删/改 container 行。
5. `equipment.RecalculatePlayerStats(player_id)` → 推送 `PLAYER_SNAPSHOT` / 属性变更。

### 7.2 卸下 UNEQUIP

逆过程；`item_uid` 回到背包第一个可用格。

### 7.3 强化 ENHANCE

1. **仅允许未佩戴状态**：实例 `state=bag`、不在 `player_equipment_slot`、且位于背包 `player_container_item`；已佩戴装备须先卸下。
2. 校验 `can_enhance`、`enhance_level < max_enhance_level`、`is_damaged=false`、材料足够。
3. 读取 `equipment_enhance_success_config[required_level_band_min, enhance_level+1]` 掷骰（缺失时回退 1~10 段，再回退代码默认）。
4. 成功：`enhance_level++`；失败：按所选强化材料的 `failure_penalty` 处理，默认 `damage` 会写入 `equipment_instance.is_damaged=true`，`level_down` 降级，`none` 不改变等级与损坏状态。
5. 材料与铜币无论成败均消耗；强化本身不重算玩家属性，佩戴/卸下时再重算。

### 7.4 修复 REPAIR

1. **仅允许未佩戴状态**：实例 `state=bag`、不在 `player_equipment_slot`、且位于背包 `player_container_item`；已佩戴装备须先卸下。
2. 校验 `equipment_instance.is_damaged=true`，并读取启用状态的 `equipment_repair_cost`。
3. 消耗 `item_sub_type=equipment_repair` 的修复材料后，将 `equipment_instance.is_damaged` 清为 `false`。
4. 响应返回修复后的装备快照；客户端再按当前页码与筛选条件刷新背包，确保损坏标记和修复材料数量均来自服务端最新快照。

### 7.5 镶嵌 SOCKET

1. 校验孔位空闲、`gem` 类型允许、背包有宝石。
2. 写 `equipment_instance_socket`，消耗宝石，重算属性。

### 7.6 取下镶嵌 UNSOCKET

1. 校验该孔已有 `gem_item_id`。
2. 清空孔位，**将宝石完整退回背包**（无损、不降级、不损坏）。
3. 重算属性；响应更新后装备详情与背包快照。

---

## 8. WebSocket 协议（建议）

| CMD | 方向 | 说明 |
|-----|------|------|
| `2070` | C→S | `PLAYER_EQUIPMENT_LIST_REQ` 拉取已佩戴 + 各件详情 |
| `2071` | S→C | `PLAYER_EQUIPMENT_LIST_RESP` |
| `2072` | C→S | `PLAYER_EQUIP_REQ` `{ bag_slot_index }` 或 `{ item_uid, equip_slot }` |
| `2073` | S→C | `PLAYER_EQUIP_RESP` + 更新后属性 |
| `2074` | C→S | `PLAYER_UNEQUIP_REQ` `{ equip_slot }` |
| `2075` | S→C | `PLAYER_UNEQUIP_RESP` |
| `2076` | C→S | `PLAYER_EQUIPMENT_ENHANCE_REQ` `{ item_uid, cost_item_id? }`；`cost_item_id` 须为 `item_sub_type=equipment_enhance` 的强化材料 |
| `2077` | S→C | `PLAYER_EQUIPMENT_ENHANCE_RESP` |
| `2078` | C→S | `PLAYER_EQUIPMENT_REPAIR_REQ` `{ item_uid }`；仅修复未佩戴且位于背包的损坏装备 |
| `2079` | S→C | `PLAYER_EQUIPMENT_REPAIR_RESP`，返回 `item` 与 `all_equipped` |
| `2080+` | 预留 | 镶嵌 / 取下镶嵌后续实现时另行分配，禁止复用已占用的 `2078/2079` |

`EnterWorld` / `PlayerSnapshot` 增加可选字段 `equipped_items[]`（slot + item_uid + item_id + enhance_level + is_damaged + 展示用属性摘要），避免进场景后再发一次列表。

---

## 9. Admin 后台

### 9.1 菜单结构

| 路由 | 菜单名 | 权限建议 |
|------|--------|----------|
| `/equipment-definitions` | **系统装备管理** | `equipment:view` / `equipment:edit` |
| `/equipment-sets` | 套装管理 | `equipment_set:view` / `equipment_set:edit` |
| `/gem-definitions` | 镶嵌石管理 | `gem:view` / `gem:edit` |
| `/equipment-enhance-rules` | 强化规则 | `equipment_enhance:view` / `equipment_enhance:edit` |

首期最小闭环：**系统装备管理 + 套装管理**；强化规则可内嵌在装备编辑表单的 Tab 中。

### 9.2 系统装备管理页

列表筛选：`item_id`、`item_name`、`equip_slot`、`set_id`、`required_level`、`can_enhance`、`is_enabled`。

编辑表单 Tab：

1. **基础** — 对接 `item_definition`：名称、介绍、图标、品质、佩戴等级、绑定、价格等。
2. **部位与限制** — `equip_slot`、职业/性别、`player_only`、时装外观 ID。
3. **基础属性** — 五维 + 次要战斗属性表单（整数展示）。
4. **强化** — 开关、`max_enhance_level`、选择或内联 `per_level_stats`。
5. **镶嵌** — 孔数、允许宝石类型。
6. **套装** — 所属套装、在套装中的序号（仅展示）。

详情页展示：**最终预览** = 基础 + 满强化 + 满镶嵌示例（便于运营验数）。

### 9.3 API（REST）

```
GET    /api/admin/equipment-definitions
GET    /api/admin/equipment-definitions/{item_id}
POST   /api/admin/equipment-definitions
PUT    /api/admin/equipment-definitions/{item_id}
DELETE /api/admin/equipment-definitions/{item_id}   -- 软删 is_enabled=false 优先

GET    /api/admin/equipment-sets
POST   /api/admin/equipment-sets
PUT    /api/admin/equipment-sets/{set_id}
GET    /api/admin/equipment-sets/{set_id}/effects
PUT    /api/admin/equipment-sets/{set_id}/effects
```

创建装备时 **同一事务** 写 `item_definition` + `item_equipment_extra`，保证 item_id 一致。

---

## 10. 模块边界

```
module/equipment/
  model.go          -- 槽位枚举、DTO、错误码
  repo.go           -- 佩戴/实例/套装读写的接口
  stats.go          -- 聚合单件 + 套装 + 重算玩家属性
  service.go        -- Equip/Unequip/Enhance/Socket/Admin CRUD
  admin_model.go

data/postgres/equipment_repo.go
transport/ws/equipment_handler.go
transport/http/admin_equipment_handlers.go
```

依赖：

- `module/bag` — 容器格子、物品发放
- `module/player` / `module/progression` — 重算后写回 player 战斗字段
- `module/battle` — 开战读取玩家 Profile（含装备加成）；**战后结算**触发药囊恢复或写回剩余 hp/spirit

---

## 11. 客户端

- **状态面板 · 装备页**：13 槽位网格 + 点击空槽从背包装备；已佩戴显示图标、+N、套装角标。
- **背包装备**：`item_type=equipment` 显示「佩戴」；自动匹配槽位。
- **装备详情**：名称、介绍、基础属性、强化预览（+1~+15）、镶嵌孔、套装件数进度。
- **强化/镶嵌**：独立子面板，请求走 `RequestLoadingOverlay`。
- 数值 UI **全部 int 展示**，禁止 float64 直出。

---

## 12. 分期实施建议

| 阶段 | 内容 | 交付 |
|------|------|------|
| **P0 模板 + Admin** | 迁移扩展表；Admin 系统装备管理 CRUD；无运行时佩戴 | 运营可配装备 |
| **P1 佩戴** | `player_equipment_slot`、Equip/Unequip WS、属性重算、客户端装备页 | 可穿戴、战斗生效 |
| **P2 强化** | 成功率表、Enhance WS、材料消耗 | +15 线性加成 + 概率 |
| **P3 镶嵌** | 宝石模板、孔位、Socket/Unsocket WS | 镶嵌属性，无损取下 |
| **P4 套装** | 套装表、件数激活、UI 套装进度 | 套装效果 |

---

## 13. 配置示例

### 13.1 可强化武器

| 字段 | 值 |
|------|-----|
| item_name | 新手长剑 |
| equip_slot | weapon |
| required_level | 10 |
| base_atk | 500 |
| can_enhance | true |
| max_enhance_level | 15 |
| per_level | atk +100 |

+15 时额外攻击 +1500，总攻击 = 500 + 1500 = 2000（不含镶嵌/套装）。

### 13.2 时装（仅外观）

| 字段 | 值 |
|------|-----|
| item_name | 夏日泳装 |
| equip_slot | costume |
| can_enhance | false |
| appearance_skin_id | 泳装_001 |
| appearance_only | true |
| base_atk 等 | 全部为 0，不参与重算 |

### 13.3 药囊

| 字段 | 值 |
|------|-----|
| item_name | 回血药囊 |
| equip_slot | medicine_pouch |
| can_enhance | false |
| restore_player_hp / spirit / vigor | true |
| restore_pet_hp / spirit | true |

### 13.4 套装「龙鳞」

| piece_count | 效果 |
|-------------|------|
| 2 | def +200 |
| 4 | hp_max +2000 |
| 6 | atk +500, physical_resist_pct +20 |

---

## 14. 已确认规则（2026-06-18）

| # | 议题 | 结论 |
|---|------|------|
| 1 | 药囊 | 战斗结束后自动回满人物与出战宠物的 **生命 + 精力**（玩家含 vigor/spirit）；**未佩戴**则保留上一场剩余值进入下次战斗 |
| 2 | 时装 | **仅改变外观**，零战斗属性 |
| 3 | 强化成功率 | +1~+3 100%；+4~+6 90%；+7~+8 75%；+9 65%；+10 55%；+11 45%；+12 35%；+13 25%；+14 15%；+15 10%；失败 **不掉级** |
| 4 | 镶嵌取下 | **无损**退回背包 |
| 5 | 属性叠加 | 全身装备 **全额 +固定数值** 叠加，含主武器 + 职业武器；禁止乘积 |
| 6 | 属性封顶 | 与宠物 **共用** `pet_combat_stat_cap`，字段一致 |

可按 §12 从 **P0** 编写迁移 `058_player_equipment_foundation.sql` 与 Admin「系统装备管理」。
