# 最新变更记录

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
- 新增 `PP_REPOSITORY_MODE`、PostgreSQL、Redis 配置骨架与示例环境变量
- 新增 PostgreSQL 账号/玩家/宠物仓储适配器，以及 Redis `ws_token` 仓储适配器骨架
- 新增仓储 provider 装配层，默认仍走内存模式，并预留 `postgres_redis` 模式的依赖注入入口
- 新增 `backend/server/configs/config.env` 实际配置文件，并支持启动时自动加载本地 env 文件
