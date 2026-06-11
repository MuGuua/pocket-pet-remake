# 后台管理开发实现方案

本文描述 `pocket-pet-remake` 的 Web 后台管理一期落地方案。

## 1. 目标

- 为运营、策划、客服、测试提供独立于 Godot 客户端的管理后台。
- 所有管理动作都走 Go 服务端权威接口，不直接操作数据库。
- 所有配置、数值、资源改动都可审计、可回滚、可追踪。

## 2. 一期范围

### 2.1 基础能力

- 管理员登录
- 权限壳层
- 菜单路由
- 通用列表 / 详情 / 表单 / 确认弹窗
- 审计日志基础结构

### 2.2 业务模块

- 玩家管理
- 宠物管理
- 背包管理
- 任务管理
- NPC / 场景配置管理入口预留

## 3. 目录规划

```text
admin/
  src/
    app/                # 路由、全局 providers、应用壳层
    layouts/            # 后台布局
    components/         # 通用表格、筛选栏、状态标签、确认弹窗
    pages/
      auth/             # 登录页
      dashboard/        # 控制台首页
      players/          # 玩家管理
    services/           # 请求封装、领域 API
    types/              # DTO、表单类型、权限类型
    styles/             # 全局样式与设计令牌
backend/server/internal/transport/http/admin/
  # 后台 HTTP handler
backend/server/internal/module/admin/
  # 后台鉴权、权限、审计编排
backend/server/migrations/
  # admin 用户、角色、权限、审计日志表
```

## 4. 技术栈建议

### 前端

- React 18
- TypeScript
- Vite
- Ant Design
- React Router
- Zustand

### 后端

- 继续沿用现有 Go HTTP 服务
- `/api/admin/...` 作为后台接口前缀
- REST 接口优先于 WebSocket

## 5. 接口分层原则

- `transport/http/admin`：只处理鉴权、参数校验、HTTP 返回格式
- `module/admin`：处理后台权限、审计、批量操作编排
- 现有 `player` / `pet` / `quest` / `world` 模块：继续负责真实领域能力
- `data/postgres`：负责数据库读写

## 6. 一期页面规划

### 6.1 登录页

- 管理员账号密码登录
- 记住登录态
- 登录错误提示

### 6.2 控制台首页

- 系统状态
- 常用入口
- 最近高风险操作日志
- 待处理异常占位区

### 6.3 玩家管理

- 列表：玩家 ID、昵称、等级、金币、最近登录时间、状态
- 筛选：玩家 ID、昵称、状态
- 详情：基础属性、背包摘要、宠物摘要、任务摘要
- 操作：封禁、解封、资源修正

### 6.4 宠物管理

- 列表：宠物 UID、所属玩家、宠物 ID、等级、是否出战
- 详情：属性、技能、状态、归属关系
- 操作：修正属性、调整状态

### 6.5 背包管理

- 玩家背包列表
- 发放物品
- 删除物品
- 批量操作预览

### 6.6 任务管理

- 任务模板列表
- 玩家任务进度查询
- 任务状态修正

## 7. 权限模型

- 菜单权限：是否可见
- 页面权限：是否可进入
- 操作权限：查看 / 新增 / 编辑 / 删除 / 导出 / 发布
- 高风险权限：改货币、发道具、改任务、改配置

## 8. 审计要求

所有高风险操作至少记录以下字段：

- 操作者 admin_user_id
- 操作模块
- 操作类型
- 目标对象 ID
- 修改前数据快照
- 修改后数据快照
- 操作原因
- 创建时间

## 9. 实施顺序

### 阶段 A：骨架

- 后台前端工程初始化
- 管理端布局、路由、鉴权壳层
- 后端 `/api/admin/healthz`
- 后端 `/api/admin/auth/login` 结构占位

### 阶段 B：玩家管理闭环

- 玩家列表接口
- 玩家详情接口
- 玩家管理页面
- 审计日志入库

#### 当前已落地接口

- `POST /api/admin/auth/login`：管理员登录，返回后台 Bearer Token
- `GET /api/admin/me`：校验登录态并返回管理员权限快照
- `GET /api/admin/players`：玩家列表，支持 `player_id`、`name`、`page`、`page_size`
- `GET /api/admin/players/{player_id}`：玩家详情，返回角色基础属性、战斗属性与技能配置
- `POST /api/admin/players`：创建玩家账号与人物档案
- `PUT /api/admin/players/{player_id}`：编辑玩家属性、状态与技能配置
- `DELETE /api/admin/players/{player_id}`：软删除账号与人物
- `GET /api/admin/pets`：宠物列表，支持 `pet_uid`、`player_id`、`pet_id`、分页
- `GET /api/admin/pets/{pet_uid}`：宠物详情
- `POST /api/admin/pets`：创建宠物
- `PUT /api/admin/pets/{pet_uid}`：编辑宠物属性与技能配置
- `DELETE /api/admin/pets/{pet_uid}`：删除宠物，并同步移出出战阵容
- `GET /api/admin/bags`：背包列表，支持 `record_id`、`player_id`、`item_id`、分页
- `GET /api/admin/bags/{record_id}`：背包记录详情
- `POST /api/admin/bags`：新增背包记录，直接落库到 `player_item`
- `PUT /api/admin/bags/{record_id}`：编辑背包记录归属、道具与数量
- `DELETE /api/admin/bags/{record_id}`：删除背包记录
- `GET /api/admin/quests/templates`：任务模板列表，支持 `quest_id`、`quest_type`、`title`、`status`、分页
- `GET /api/admin/quests/templates/{quest_id}`：任务模板详情
- `POST /api/admin/quests/templates`：新增任务模板
- `PUT /api/admin/quests/templates/{quest_id}`：编辑任务模板
- `DELETE /api/admin/quests/templates/{quest_id}`：停用任务模板
- `GET /api/admin/quests/player-progress`：玩家任务记录列表，支持 `record_id`、`player_id`、`quest_id`、`state`、`tracked`、分页
- `GET /api/admin/quests/player-progress/{record_id}`：玩家任务记录详情
- `POST /api/admin/quests/player-progress`：新增玩家任务记录
- `PUT /api/admin/quests/player-progress/{record_id}`：编辑玩家任务状态与目标进度
- `DELETE /api/admin/quests/player-progress/{record_id}`：删除玩家任务记录
- `GET /api/admin/npcs/entities`：NPC / 世界实体列表，支持 `entity_id`、`scene_id`、`name`、`status`、分页
- `GET /api/admin/npcs/entities/{entity_id}`：实体详情
- `POST /api/admin/npcs/entities`：新增实体配置
- `PUT /api/admin/npcs/entities/{entity_id}`：编辑实体所在地图、坐标与状态
- `DELETE /api/admin/npcs/entities/{entity_id}`：删除实体配置
- `GET /api/admin/npcs/menu-entries`：NPC 菜单项列表，支持 `entity_id`、`entry_id`、`status`、分页
- `GET /api/admin/npcs/menu-entries/{entity_id}/{entry_id}`：菜单项详情
- `POST /api/admin/npcs/menu-entries`：新增 NPC 菜单项
- `PUT /api/admin/npcs/menu-entries/{entity_id}/{entry_id}`：编辑 NPC 菜单项
- `DELETE /api/admin/npcs/menu-entries/{entity_id}/{entry_id}`：删除 NPC 菜单项

### 阶段 C：宠物 / 背包 / 任务

- 宠物接口与页面
- 背包接口与页面
- 任务接口与页面

### 阶段 D：配置与发布

- NPC / 场景配置接口
- 草稿与发布记录
- 回滚机制

## 10. 风险与注意事项

- 不允许后台前端假造“修改成功”结果，必须以服务端返回为准。
- 不允许后台直接绕过服务端写数据库。
- 不允许把游戏玩家端协议结构直接暴露给后台页面；后台应使用更清晰的管理 DTO。
- 高风险操作必须有确认与审计。
