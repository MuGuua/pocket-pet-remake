# 任务总结

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
- 新增 `PP_REPOSITORY_MODE`、PostgreSQL、Redis 相关配置项，并补充示例环境变量
- 新增 PostgreSQL 版账号、玩家、宠物仓储适配器，统一复用现有模块仓储接口
- 新增 Redis 版 `ws_token` 仓储适配器，使用 key 前缀和一次性消费语义预留真实接入点
- 新增 provider 装配层，统一管理 memory 与 `postgres_redis` 两种仓储模式的依赖绑定
- 当前 `postgres_redis` 模式只完成骨架与接口约束，真实数据库连接、Redis 客户端初始化和驱动导入仍是下一步工作
