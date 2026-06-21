# 宠物技能槽位

本文定义宠物实例的技能分类、槽位数量、默认开关与战斗可用技能合并规则。服务端为唯一权威；客户端查看/配置 UI 只展示协议下发的 `skill_slots`。

---

## 1. 技能分类

| 分类 | 槽位数 | 默认状态 | 说明 |
|------|--------|----------|------|
| 天生技 `innate` | 最多 5 | 模板配置即生效 | 来自 `pet_definition.innate_skill_ids`，发放时快照到实例 |
| 主动神符技 `active_talisman` | 1 | **关闭** | 需消耗指定道具开启 `active_talisman_enabled` 后槽位才生效 |
| 神符技·英雄 `talisman_hero` | 1 | **关闭** | 需对应道具开启 `talisman_hero_enabled` |
| 神符技【1】`talisman_1` | 1 | **关闭** | 需对应道具开启 `talisman_1_enabled` |
| 神符技【2】`talisman_2` | 1 | **关闭** | 需对应道具开启 `talisman_2_enabled` |
| 神符技【3】`talisman_3` | 1 | **关闭** | 需对应道具开启 `talisman_3_enabled` |
| 普通技 `normal` | 3 | **开启** | 三个槽位默认可用 |
| 法宝技 `artifact` | 3 | 视装备 | 需装备法宝；技能来自法宝镶嵌，**仅在查看宠物技能详情时下发** |

道具与槽位解锁映射见配置表 `pet_skill_slot_unlock_item`（可选种子迁移 `056_seed_pet_skill_slot_items.sql`）。

---

## 2. 持久化

### 2.1 `player_pet` 扩展（实例）

| 字段 | 类型 | 说明 |
|------|------|------|
| `innate_skill_ids` | JSONB | 最多 5 个 skill_id，0 表示空槽 |
| `normal_skill_ids` | JSONB | 固定 3 槽 |
| `active_talisman_skill_id` | INTEGER | 主动神符技 |
| `talisman_hero_skill_id` | INTEGER | 神符技·英雄 |
| `talisman_slot_1/2/3_skill_id` | INTEGER | 神符技【1~3】 |
| `active_talisman_enabled` | BOOLEAN | 主动神符技是否已开启 |
| `talisman_hero_enabled` | BOOLEAN | 英雄神符是否已开启 |
| `talisman_slot_1/2/3_enabled` | BOOLEAN | 神符【1~3】是否已开启 |

兼容字段 `skill_ids`：保留为**战斗合并结果缓存**，由服务端在读写后重算，客户端不应自行维护。

### 2.2 `pet_artifact_equipment`（法宝装备）

| 字段 | 说明 |
|------|------|
| `pet_uid` / `player_id` | 所属实例 |
| `slot_index` | 0~2，共 3 槽 |
| `skill_id` | 法宝镶嵌技能；0 表示空槽 |

### 2.3 `pet_definition` 模板

| 字段 | 说明 |
|------|------|
| `innate_skill_ids` | 模板天生技（最多 5） |
| `normal_skill_ids` | 模板默认普通技（最多 3） |

发放实例时：复制模板天生/普通技；神符槽默认 `enabled=false`、skill_id=0。

---

## 3. 战斗可用技能合并

进入战斗时，按以下顺序去重合并（跳过 skill_id=0 或未开启槽位）：

1. 天生技（最多 5）
2. 已开启的普通技（3）
3. 已开启的主动神符技（1）
4. 已开启的神符·英雄 / 【1】【2】【3】（各 1）
5. 已装备法宝技（3，来自 `pet_artifact_equipment`）

结果写入 `LineupPet.skill_ids` / 战斗 actor，**不包含**未开启神符槽与未装备法宝槽。

---

## 4. 协议 `PetDetail.skill_slots`

查看宠物列表/详情时下发结构化槽位；`artifact` 数组仅在拉取完整宠物详情（含技能面板）时填充，列表页可省略或全 0。

```json
{
  "skill_slots": {
    "innate": [{"slot_index": 0, "skill_id": 1001}],
    "active_talisman": {"slot_index": 0, "skill_id": 0, "enabled": false},
    "talisman_hero": {"slot_index": 0, "skill_id": 0, "enabled": false},
    "talisman_1": {"slot_index": 0, "skill_id": 0, "enabled": false},
    "talisman_2": {"slot_index": 0, "skill_id": 0, "enabled": false},
    "talisman_3": {"slot_index": 0, "skill_id": 0, "enabled": false},
    "normal": [
      {"slot_index": 0, "skill_id": 1001, "enabled": true},
      {"slot_index": 1, "skill_id": 1002, "enabled": true},
      {"slot_index": 2, "skill_id": 0, "enabled": true}
    ],
    "artifact": [
      {"slot_index": 0, "skill_id": 0},
      {"slot_index": 1, "skill_id": 0},
      {"slot_index": 2, "skill_id": 0}
    ]
  },
  "skill_ids": [1001, 1002]
}
```

`skill_ids` 仍为战斗合并后的扁平列表，兼容现有战斗 UI。

---

## 5. 待实现（后续）

- 后台配置 `pet_skill_slot_unlock_item` 管理页（表已建，运营可先 SQL 录入）
- 客户端宠物技能面板按 `skill_slots` 分区展示
- 背包使用 UI 支持选择目标宠物（`target_pet_uid`）
