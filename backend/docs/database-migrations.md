# 数据库迁移执行与回滚指南

## 1. 执行前准备

### 1.1 检查当前迁移状态

项目使用自管理的 SQL 迁移脚本（`backend/server/migrations/*.sql`），没有自动版本追踪工具。执行前需人工确认当前数据库已经应用了哪些迁移。

```bash
# 从任一稳定表（如 player、account）的注释或字段结构反推已执行的迁移边界
psql -U pocket_pet_user -d pocket_pet_game -c "\d+ player"
# 查看迁移目录中的文件与上述结构的对应关系
ls backend/server/migrations/*.sql | tail -10
```

### 1.2 备份受影响的数据

**单张表的结构变更**（如增加字段、修改默认值）：
```bash
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
pg_dump -U pocket_pet_user -d pocket_pet_game \
  -t player -t player_quest -t player_quest_objective \
  > /tmp/pocket_pet_pre_migration_124_${TIMESTAMP}.dump
```

**全库备份**（多表结构变更或有外键级联风险时）：
```bash
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
pg_dump -U pocket_pet_user -d pocket_pet_game \
  > /tmp/pocket_pet_full_backup_${TIMESTAMP}.dump
```

备份文件保留在本地临时目录，至少保留到验证通过后。

### 1.3 停服（可选，按迁移风险评估）

**不需要停服的场景**：
- 新增可选字段（有默认值或允许 NULL）
- 新增独立表
- 新增索引（PostgreSQL 支持 `CREATE INDEX CONCURRENTLY`）

**需要停服的场景**：
- 修改已有字段类型
- 增加 `NOT NULL` 约束
- 删除字段或表
- 修改外键关系
- 大批量数据修复（锁表时间较长）

停服流程：
```bash
# 向运行中的服务发送 SIGTERM，优雅等待 WebSocket 关闭、worker 停止和数据连接关闭
kill -TERM $(pgrep -f pocket-pet-game)
# 等待进程退出
wait
```

## 2. 执行迁移

### 2.1 单文件迁移

适用于独立变更，或已确认依赖的迁移已执行。

```bash
psql -U pocket_pet_user -d pocket_pet_game \
  -f backend/server/migrations/124_repair_main_journey_quest_flow.sql
```

执行后观察输出：
- `CREATE TABLE`、`ALTER TABLE` 成功提示
- `INSERT` / `UPDATE` 受影响行数
- 如果有语法错误或约束冲突，事务会回滚（前提是迁移脚本已用 `BEGIN/COMMIT` 包裹）

### 2.2 批量执行

适用于多个互相依赖的迁移，或首次从零初始化数据库。

```bash
# 严格按文件名顺序执行
for migration in backend/server/migrations/{117..123}_*.sql; do
  echo "Executing $migration"
  psql -U pocket_pet_user -d pocket_pet_game -f "$migration"
  if [ $? -ne 0 ]; then
    echo "Migration $migration failed, stopping."
    exit 1
  fi
done
```

**注意**：批量执行时，任一 SQL 错误会停止后续迁移。如果某个迁移已经执行过，应跳过或改用 `IF NOT EXISTS` / `IF EXISTS` 保证幂等。

### 2.3 确认迁移结果

```bash
# 确认字段存在
psql -U pocket_pet_user -d pocket_pet_game -c "\d+ player_quest"

# 确认数据修复结果
psql -U pocket_pet_user -d pocket_pet_game -c "
  SELECT quest_id, 
         jsonb_array_length(objectives_json) AS obj_count,
         jsonb_array_length(rewards_json) AS reward_count
  FROM quest_template 
  WHERE quest_id IN (1101, 1102, 1103);
"

# 确认玩家目标已补齐
psql -U pocket_pet_user -d pocket_pet_game -c "
  SELECT player_id, quest_id, objective_type, target_id, current_progress, target_progress
  FROM player_quest_objective
  WHERE quest_id = 1101;
"
```

## 3. 重启服务

```bash
# 重新编译并启动服务
cd backend/server
go build -o ../../bin/pocket-pet-game ./cmd/game
../../bin/pocket-pet-game &

# 检查日志中的启动阶段缓存加载
tail -f logs/game.log | grep -E "movement_config|scene_boundaries|navigation"
```

确认运行时缓存正常加载，端口正常监听。

## 4. 回滚（迁移失败或验证不通过时）

### 4.1 停服

同 1.3 节。

### 4.2 使用备份恢复

**单表恢复**：
```bash
psql -U pocket_pet_user -d pocket_pet_game < /tmp/pocket_pet_pre_migration_124_20260818.dump
```

**全库恢复**：
```bash
# 先删除当前数据库（危险操作，仅在确认备份有效且业务已停止的情况下执行）
psql -U postgres -c "DROP DATABASE pocket_pet_game;"
psql -U postgres -c "CREATE DATABASE pocket_pet_game OWNER pocket_pet_user;"
psql -U pocket_pet_user -d pocket_pet_game < /tmp/pocket_pet_full_backup_20260817.dump
```

### 4.3 手动回滚 DDL

如果迁移没有备份，且结构变更可逆（如新增字段可删除、修改默认值可还原），手动编写回滚 SQL：

```sql
BEGIN;

-- 回滚示例：删除新增的字段
ALTER TABLE player DROP COLUMN IF EXISTS position_version;

-- 回滚示例：恢复旧的默认值
ALTER TABLE world_movement_config ALTER COLUMN max_speed_milli SET DEFAULT 90000;

COMMIT;
```

### 4.4 数据修复回滚

如果迁移只修改了数据（如 `UPDATE` / `INSERT`），回滚需要按备份恢复对应行，或根据业务逻辑重新计算。

**示例**：恢复被修改的 `objectives_json`
```sql
BEGIN;

-- 从备份表或手动值重建
UPDATE quest_template 
SET objectives_json = '[]'::jsonb
WHERE quest_id IN (1101, 1102, 1103);

-- 删除自动补齐的玩家目标
DELETE FROM player_quest_objective
WHERE quest_id = 1101 AND objective_type = 'TALK_TO_NPC';

COMMIT;
```

### 4.5 重启服务

同 3 节，确认回滚后的结构与运行时缓存正常。

## 5. 发布流程建议

- **本地开发**：迁移在本地 PostgreSQL 执行，通过测试后提交代码。
- **测试环境**：部署前执行迁移，验证业务功能与回滚流程。
- **正式环境**：按 1 → 2 → 3 流程执行，迁移与代码发布保持一致，避免"新代码 + 旧结构"或"旧代码 + 新结构"的中间状态。

**特殊情况**：
- 新增可选字段：可以先执行迁移，再发布代码；老代码不读新字段，不会报错。
- 删除字段：必须先发布不再使用该字段的新代码，再执行迁移删除；否则老代码会因字段缺失失败。
- 修改字段类型：需要停服，迁移与代码发布必须同时完成。

## 6. 常见问题

### 6.1 迁移脚本没有事务包裹，执行一半失败怎么办？

手动检查数据库当前状态，补齐失败步骤之前的 DDL，或从备份恢复。

**建议**：所有迁移 SQL 统一使用 `BEGIN` / `COMMIT` 包裹，确保原子性。对于 PostgreSQL 不支持事务的操作（如 `CREATE INDEX CONCURRENTLY`），单独拆成独立迁移文件。

### 6.2 多个迁移互相依赖，执行顺序错了怎么办？

按文件名前缀（如 `117_`、`118_`）的升序执行。如果已经乱序执行导致外键或约束冲突，从备份恢复后重新按序执行。

### 6.3 迁移执行成功，但运行时缓存加载失败？

检查服务端代码与迁移的一致性：
- `movement_config` 的字段名与 Go 结构体是否对应
- `scene_boundaries` 是否为所有启用场景初始化了数据
- `scene_navigation` 是否有已发布版本

如果缺少数据，补齐迁移或通过后台接口初始化。

### 6.4 如何确认当前数据库已执行的迁移版本？

项目未使用自动迁移工具，需手动对照表结构与迁移文件。建议在数据库中新增 `schema_migrations` 表记录已执行的迁移文件名和执行时间。

示例：
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  executed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 执行迁移后插入记录
INSERT INTO schema_migrations (version) VALUES ('124_repair_main_journey_quest_flow');
```

## 7. 相关文档

- `backend/docs/architecture.md`：数据层架构与运行时缓存装配
- `backend/docs/changelog.md`：每次迁移执行的业务背景与验证结果
- `backend/docs/task-summary.md`：迁移执行的详细操作记录
