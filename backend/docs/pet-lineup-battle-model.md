# 宠物编队与战斗快照模型设计

## 1. 文档目的

- 本文用于指导当前 MVP 阶段的“宠物养成与编队”和“服务端权威战斗结算”后续实现。
- 重点解决 4 个容易混淆的问题：
  - 玩家拥有的宠物实例是什么
  - 当前编队是什么
  - 当前出战宠是什么
  - 战斗中的单位快照是什么
- 本文结论结合了当前仓库实现现状，以及 `backend/docs/kdjl-client-reference.md` 中从原版客户端提炼出的参考逻辑。

## 2. 设计结论

后续实现必须明确拆成 4 层对象，不允许混用：

1. 宠物实例 `PetInstance`
2. 编队 `Lineup`
3. 当前出战宠 `ActivePet`
4. 战斗单位快照 `BattleActorSnapshot`

一句话原则：

- `pet` 模块负责“我拥有哪些宠物”
- `player` 模块负责“我当前带哪几只上阵”
- `battle` 模块负责“这一场战斗里临时怎么打”
- 客户端只保存展示和输入组织所需状态，不保存最终结算结果

## 3. 为什么要这样拆

当前仓库已经有两类事实：

- 服务端 `pet.LineupPet` 只保留了最小编队字段
- 战斗模块 `battle.Service` 已经默认从 `lineup[0]` 取主战宠创建战斗快照

这说明当前骨架已经隐含了“宠物实例”和“战斗单位”不是同一个对象，但这层边界还没有文档化，后续继续写功能时很容易出现几个问题：

- 把宠物实例上的永久属性直接当战斗运行时状态
- 把编队顺序和当前出战状态混成一个字段
- 在客户端本地维护最终 HP 或技能结果
- 在 `battle` 模块里直接回写宠物仓储，破坏模块边界

因此必须先把模型边界定死，再继续实现。

## 4. 四层模型定义

### 4.1 宠物实例 `PetInstance`

含义：

- 玩家拥有的一只真实宠物
- 可持久化
- 脱离战斗后仍然存在

建议字段：

- `pet_uid`
- `player_id`
- `pet_id`
- `level`
- `exp`
- `quality`
- `hp`
- `hp_max`
- `atk`
- `def`
- `spd`
- `skill_ids`
- `state_flags`

职责归属：

- 服务端：`pet` 模块
- 客户端：`GameState.pets`

注意：

- `hp` / `hp_max` 是宠物实例层当前状态，不是战斗事件流本身。
- 后续如果有升级、治疗、死亡、复活，最终落库都应该回到宠物实例层。

### 4.2 编队 `Lineup`

含义：

- 玩家当前准备带入世界或战斗的一组宠物顺序
- 是玩家状态的一部分，不是战斗状态的一部分

建议字段：

- `player_id`
- `slot_index`
- `pet_uid`
- `is_lead`

约束建议：

- 当前 MVP 固定最多 3 只或 4 只即可，先不要扩展复杂阵型
- 同一只宠物不能重复进入编队
- 编队不能为空时，服务端应明确返回错误
- 第一位默认视为主战候选宠

职责归属：

- 服务端主归属：`pet` 模块提供查询与校验
- 服务端状态引用：`player` 模块可缓存当前编队摘要
- 客户端：`GameState.lineup`

注意：

- 编队是“准备出战顺序”，不是“当前正在场上的宠物”。
- 战斗过程中如果发生换宠，不能直接把编队数组当作实时战斗状态使用。

### 4.3 当前出战宠 `ActivePet`

含义：

- 当前玩家在战斗中的实际在场宠物
- 是战斗中的局部状态，不应直接等于编队第一位

来源：

- 战斗开始时，默认由编队第一位生成
- 战斗过程中可因为换宠、死亡、强制替换而变化

建议最小字段：

- `battle_id`
- `player_id`
- `pet_uid`
- `lineup_index`
- `entered_round`

职责归属：

- 服务端：`battle` 模块内部运行态
- 客户端：不单独存永久结构，可由 `battle_state.allies` 推导

注意：

- `ActivePet` 只存在于战斗上下文中。
- 战斗结束时，它的最终生命和状态需要折叠回宠物实例层。

### 4.4 战斗单位快照 `BattleActorSnapshot`

含义：

- 战斗开始时服务端发给客户端、并在战斗过程中持续更新的临时战斗视图对象
- 用于展示和动作校验

当前仓库已有字段：

- `actor_id`
- `actor_type`
- `pet_uid`
- `pet_id`
- `name`
- `hp`
- `hp_max`
- `skill_ids`

后续可扩展但仍建议保持轻量：

- `states`
- `energy`
- `can_act`
- `lineup_index`

职责归属：

- 服务端：`battle` 模块构造与推送
- 客户端：`GameState.battle_state`

注意：

- 它是战斗快照，不是宠物实例。
- 其中 `hp` 的变化只代表本场战斗过程。

## 5. 四层关系图

```text
PetInstance(持久化)
  -> Lineup(世界态准备顺序)
    -> ActivePet(本场当前在场宠物)
      -> BattleActorSnapshot(发给客户端的战斗表现快照)
```

关系原则：

- `PetInstance` 可以不在 `Lineup` 中
- `Lineup` 中的宠物在战斗开始时才会生成 `ActivePet`
- `ActivePet` 会映射为一个或多个 `BattleActorSnapshot` / `BattleActorState`
- 战斗结束后，再把结果折叠回 `PetInstance`

## 6. 当前仓库对应关系

### 6.1 已有能力

- 服务端 `pet.Service.ListLineup()` 已能返回最小编队
- 服务端 `world_handler` 已在 `ENTER_WORLD_RESP` 中返回 `lineup`
- 服务端 `battle.Service.StartPVE()` 已从 `lineup[0]` 生成玩家方战斗单位
- 客户端 `GameState` 已区分：
  - `pets`
  - `lineup`
  - `battle_state`
- 客户端战斗场景已经按服务端下发的 `skill_ids` 生成动作按钮

### 6.2 当前缺口

- `pet` 模块没有完整 `PetInstance` 模型
- 编队修改链路只有协议位，没有完整约束文档
- `battle` 还没有“换宠”和“当前出战宠”显式模型
- 战斗结束后的宠物 HP、经验、掉落、背包变化还没有明确回写闭环
- 客户端 `GameState.upsert_pet()` 目前按 `pet_id` 合并，不适合未来多只同种宠物并存

## 7. 服务端职责划分

### 7.1 `pet` 模块

负责：

- 宠物实例列表
- 宠物模板读取
- 宠物成长数据
- 编队设置合法性校验
- 战斗结算后的宠物实例回写

不负责：

- 世界进入流程编排
- 战斗回合推进
- 掉落生成

### 7.2 `player` 模块

负责：

- 玩家基础资料
- 当前场景、等级、货币
- 对外暴露当前主状态摘要

不负责：

- 宠物细节与战斗运行时

### 7.3 `battle` 模块

负责：

- 根据玩家资料和编队创建战斗运行态
- 管理 `ActivePet`
- 校验技能、目标、换宠、逃跑
- 生成 `BATTLE_START_PUSH`、`BATTLE_STATE_PUSH`、`BATTLE_RESULT_PUSH`
- 产出战斗结束后的结算结果

不负责：

- 直接操作底层仓储
- 绕过 `pet` / `bag` / `player` 模块直接改业务数据

### 7.4 结算编排建议

战斗结束后建议由明确的应用层编排完成：

1. `battle` 生成结算结果
2. `pet` 应用宠物 HP / EXP 变化
3. `bag` 应用道具变化
4. `player` 应用金币 / 经验等变化
5. 最后统一推送结果与必要更新

当前 MVP 可以先简化为：

1. 战斗结束
2. 回写主战宠 HP
3. 必要时推送 `PET_UPDATE_PUSH`
4. 再逐步补经验、掉落、抓宠、物品消耗

## 8. 客户端状态设计

### 8.1 `GameState.pets`

用途：

- 保存玩家完整宠物列表
- 驱动宠物列表页和详情页

建议结构：

```gdscript
[
    {
        "pet_uid": 20001,
        "pet_id": 101,
        "level": 8,
        "hp": 32,
        "hp_max": 32,
        "skill_ids": [1001, 1002],
        "in_lineup": true
    }
]
```

关键要求：

- 合并宠物时必须以 `pet_uid` 为主键，不再使用 `pet_id`
- 允许存在多只相同 `pet_id` 的宠物

### 8.2 `GameState.lineup`

用途：

- 只表达当前编队顺序
- 驱动世界 HUD、宠物编队 UI、进入战斗前的主战宠预览

建议结构：

```gdscript
[
    {
        "slot_index": 0,
        "pet_uid": 20001,
        "pet_id": 101,
        "level": 8,
        "hp": 32,
        "hp_max": 32
    }
]
```

注意：

- `lineup` 是世界态静态摘要，不承载战斗事件流。

### 8.3 `GameState.battle_state`

用途：

- 保存战斗快照和事件流
- 驱动战斗 UI

建议至少包含：

- `battle_id`
- `battle_version`
- `round`
- `allies`
- `enemies`
- `actors`
- `events`
- `active_actor_id`
- `active_pet_uid`

说明：

- `active_pet_uid` 可以作为客户端当前己方出战宠的显示锚点
- 但最终是否合法，仍由服务端下发状态决定

## 9. 协议设计建议

### 9.1 `PET_LIST_RESP`

建议保持：

- 返回完整宠物列表
- 同时返回当前编队摘要

推荐结构：

```json
{
  "pets": [],
  "lineup": []
}
```

### 9.2 `PET_LINEUP_SET_REQ`

建议语义：

- 客户端提交完整编队顺序，而不是“只改某一个槽位”

推荐输入：

```json
{
  "op_id": 1,
  "pet_uids": [20001, 20003, 20002]
}
```

服务端校验点：

- `pet_uid` 都属于该玩家
- 不能重复
- 数量不能超上限
- 处于不可编队状态的宠物不能入队

### 9.3 `PET_LINEUP_SET_RESP`

建议返回：

- `accepted`
- `lineup`
- `reason`

当前如果仅返回 `lineup_pet_uids`，客户端后续仍要二次查详情，联调成本较高。

### 9.4 `BATTLE_START_PUSH`

建议在当前基础上，后续补 2 个字段：

- `active_pet_uid`
- `lineup_index`

原因：

- 客户端可以直接知道当前己方哪只在场
- 为后续换宠留稳定接口

### 9.5 `BATTLE_STATE_PUSH`

建议后续在 `actors` 或额外字段中表达：

- 当前行动方
- 当前己方出战宠是否发生变化
- 哪些技能临时不可用

但不要把结算逻辑下放到客户端。

## 10. 与 `kdjl` 参考逻辑的对应

原版客户端最有价值的参考点，在这里对应为：

- 原版“宠物实例 / 出战宠 / 技能列表”分层
  - 对应本文的 `PetInstance` / `ActivePet` / `BattleActorSnapshot`
- 原版战斗里客户端只选动作和目标
  - 对应本文的“客户端只存快照，不做结算”
- 原版世界层和战斗层切开
  - 对应本文的 `lineup` 不混入 `battle_state`

明确不参考的部分：

- 原版文本战斗协议
- 原版 J2ME 菜单式 UI
- 原版服务端驱动 `<input>/<menu>`

## 11. 建议实现顺序

### 第一步：补齐宠物实例模型

- 服务端 `pet` 模块新增完整 `PetInstance`
- `PET_LIST_RESP` 能返回完整宠物详情
- 客户端 `GameState.pets` 改为按 `pet_uid` 识别

### 第二步：补齐编队设置闭环

- 实现 `PET_LINEUP_SET_REQ/RESP`
- 服务端校验编队数量、重复、归属
- 世界进入时返回最新编队摘要

### 第三步：显式化当前出战宠

- `battle` 模块引入 `ActivePet`
- `BATTLE_START_PUSH` 明确当前出战宠
- 客户端战斗界面按 `active_pet_uid` 组织展示

### 第四步：补战斗结束回写

- 先回写主战宠 HP
- 再补经验、道具和掉落
- 需要时发 `PET_UPDATE_PUSH`

### 第五步：再考虑换宠

- 换宠一定放在前 4 步之后
- 否则模型边界会先乱掉

## 12. 约束清单

后续实现时必须遵守：

- 客户端不能提交最终 HP、伤害、死亡状态
- `battle` 模块不能直接写底层宠物仓储
- `lineup` 不能直接代表战斗当前在场宠物
- `battle_state` 不能替代宠物实例持久化状态
- 宠物唯一标识一律使用 `pet_uid`，不能再以 `pet_id` 代替

## 13. 一句话落地口径

后续实现请始终按下面这条口径推进：

- 宠物实例管“长期拥有”
- 编队管“战前准备”
- 出战宠管“当前在场”
- 战斗快照管“临时表现”
- 最终结果始终由服务端结算并回写
