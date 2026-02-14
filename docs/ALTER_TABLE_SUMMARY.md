# 表结构修改功能实现总结

## 实现概述

已成功为 DBM 数据库管理工具添加完整的表结构修改功能，支持 MySQL、PostgreSQL、SQLite 和 ClickHouse 四种数据库。

## 实现内容

### 1. 数据模型扩展 (`internal/model/database.go`)

新增以下数据结构：

- **AlterTableRequest**: 表结构修改请求
- **AlterTableAction**: 单个修改操作
- **AlterActionType**: 操作类型枚举（7种）
- **ColumnDef**: 列定义
- **IndexDef**: 索引定义

支持的操作类型：
- `ADD_COLUMN`: 添加列
- `DROP_COLUMN`: 删除列
- `MODIFY_COLUMN`: 修改列
- `RENAME_COLUMN`: 重命名列
- `ADD_INDEX`: 添加索引
- `DROP_INDEX`: 删除索引
- `RENAME_TABLE`: 重命名表

### 2. 适配器接口扩展 (`internal/adapter/adapter.go`)

在 `DatabaseAdapter` 接口中新增两个方法：
```go
AlterTable(db *sql.DB, request *model.AlterTableRequest) error
RenameTable(db *sql.DB, database, oldName, newName string) error
```

### 3. MySQL 适配器实现 (`internal/adapter/mysql.go`)

**新增方法**：
- `AlterTable()`: 主入口，支持批量操作
- `buildAlterClause()`: 构建单个 ALTER 子句
- `buildAddColumnClause()`: 构建添加列子句
- `buildModifyColumnClause()`: 构建修改列子句
- `buildColumnType()`: 构建列类型定义
- `buildAddIndexClause()`: 构建添加索引子句
- `RenameTable()`: 重命名表

**特性**：
- 支持所有操作类型
- 支持批量操作（一条 ALTER 语句）
- 支持 `AFTER` 子句指定列位置
- 完整的类型、默认值、注释支持

### 4. PostgreSQL 适配器实现 (`internal/adapter/postgresql.go`)

**新增方法**：
- `AlterTable()`: 主入口，分别执行每个操作
- `buildAddColumnSQL()`: 构建添加列 SQL
- `modifyColumn()`: 修改列（需要多条语句）
- `buildColumnType()`: 构建列类型定义
- `getBaseType()`: 获取基础类型
- `formatDefaultValue()`: 格式化默认值
- `buildAddIndexSQL()`: 构建添加索引 SQL
- `RenameTable()`: 重命名表

**特性**：
- 修改列需要多条 ALTER 语句（类型、可空性、默认值分别设置）
- 支持 SERIAL/BIGSERIAL 自增类型
- 使用双引号引用标识符
- 使用位置参数 `$1, $2...`

### 5. SQLite 适配器实现 (`internal/adapter/sqlite.go`)

**新增方法**：
- `AlterTable()`: 主入口，处理有限的操作
- `addColumn()`: 添加列
- `renameColumn()`: 重命名列
- `buildColumnType()`: 构建列类型定义
- `addIndex()`: 添加索引
- `dropIndex()`: 删除索引
- `RenameTable()`: 重命名表
- `rebuildTable()`: 重建表（用于不支持的操作）
- `buildCreateTableSQL()`: 构建建表 SQL

**特性**：
- 仅支持 ADD COLUMN 和 RENAME COLUMN
- DROP/MODIFY COLUMN 返回错误提示需要重建表
- 提供了 `rebuildTable()` 方法用于复杂场景
- 使用反引号引用标识符

### 6. ClickHouse 适配器实现 (`internal/adapter/clickhouse.go`)

**新增方法**：
- `AlterTable()`: 主入口，检测表引擎类型
- `getTableEngine()`: 获取表引擎类型
- `buildAlterClause()`: 构建单个 ALTER 子句
- `buildAddColumnClause()`: 构建添加列子句
- `buildModifyColumnClause()`: 构建修改列子句
- `buildColumnType()`: 构建列类型定义
- `formatDefaultValue()`: 格式化默认值
- `RenameTable()`: 重命名表（检测复制表）
- `CheckMutationStatus()`: 检查 ALTER 操作状态

**特性**：
- 检测 ReplicatedMergeTree 表，给出警告
- 使用 `Nullable()` 包装类型表示可空
- 不支持传统索引
- RENAME 操作不通过 ZooKeeper 同步
- 提供 mutation 状态查询方法

### 7. API 路由 (`internal/server/handler.go`)

**新增路由**：
```
POST /api/v1/connections/:id/tables/:table/alter?database=<db>
POST /api/v1/connections/:id/tables/:table/rename?database=<db>
```

**新增处理函数**：
- `alterTable()`: 处理表结构修改请求
- `renameTable()`: 处理表重命名请求

**特性**：
- 完整的请求验证
- 统一的错误处理
- RESTful 风格设计

### 8. 文档更新

**CLAUDE.md**：
- 添加表结构修改 API 文档
- 添加请求示例
- 添加数据库特性对比表
- 添加 ClickHouse 特别说明
- 更新 AI 使用指引

**ALTER_TABLE_EXAMPLES.md**：
- 详细的使用示例
- 各种操作类型的示例
- 批量操作示例
- 数据类型示例
- 错误处理说明
- 最佳实践
- 数据库特性对比

### 9. 测试 (`internal/adapter/alter_table_test.go`)

**测试用例**：
- `TestMySQLAlterTable`: 测试 MySQL 各种操作
- `TestPostgreSQLAlterTable`: 测试 PostgreSQL 操作
- `TestColumnTypeBuilder`: 测试列类型构建
- `TestSQLiteAlterTableLimitations`: 测试 SQLite 限制

**测试结果**：✅ 所有测试通过

## 功能特性

### 支持的操作

| 操作 | MySQL | PostgreSQL | SQLite | ClickHouse |
|------|-------|------------|--------|------------|
| 添加列 | ✅ | ✅ | ✅ | ✅ |
| 删除列 | ✅ | ✅ | ❌ | ✅ |
| 修改列 | ✅ | ✅ | ❌ | ✅ |
| 重命名列 | ✅ | ✅ | ✅ | ✅ |
| 添加索引 | ✅ | ✅ | ✅ | ❌ |
| 删除索引 | ✅ | ✅ | ✅ | ❌ |
| 重命名表 | ✅ | ✅ | ✅ | ✅* |

*ClickHouse 复制表的 RENAME 不通过 ZooKeeper 同步

### 安全特性

1. **SQL 注入防护**：使用标识符引用（反引号/双引号）
2. **参数验证**：完整的请求参数验证
3. **错误处理**：详细的错误信息返回
4. **类型检查**：验证列类型和索引定义

### 数据库特性适配

1. **MySQL**：
   - 支持 `AFTER` 子句
   - 支持批量操作
   - 使用反引号引用标识符

2. **PostgreSQL**：
   - 修改列需要多条语句
   - 支持 SERIAL 自增类型
   - 使用双引号引用标识符

3. **SQLite**：
   - 功能受限，仅支持添加列和重命名列
   - 提供重建表方法用于复杂场景
   - 明确的错误提示

4. **ClickHouse**：
   - 检测复制表并给出警告
   - 使用 `Nullable()` 包装类型
   - 不支持传统索引
   - 提供 mutation 状态查询

## 使用示例

### 添加列
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "ADD_COLUMN",
        "column": {
          "name": "email",
          "type": "VARCHAR",
          "length": 255,
          "nullable": false,
          "comment": "用户邮箱"
        }
      }
    ]
  }'
```

### 批量操作
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {"type": "ADD_COLUMN", "column": {...}},
      {"type": "MODIFY_COLUMN", "column": {...}},
      {"type": "ADD_INDEX", "index": {...}}
    ]
  }'
```

### 重命名表
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/old_users/rename?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{"newName": "users"}'
```

## 技术亮点

1. **适配器模式**：统一接口，多数据库支持
2. **类型安全**：完整的类型定义和验证
3. **错误处理**：详细的错误信息和提示
4. **批量操作**：MySQL 和 ClickHouse 支持一条语句执行多个操作
5. **特性检测**：ClickHouse 自动检测表引擎类型
6. **测试覆盖**：单元测试验证核心逻辑

## 注意事项

1. **备份数据**：执行前务必备份
2. **测试环境验证**：先在测试环境验证
3. **大表操作**：可能锁表较长时间
4. **SQLite 限制**：不支持 DROP/MODIFY COLUMN
5. **ClickHouse 复制表**：ALTER 操作异步执行，RENAME 不同步

## 后续优化建议

1. **异步执行**：大表操作支持异步执行
2. **进度监控**：提供操作进度查询接口
3. **变更历史**：记录表结构变更历史
4. **回滚支持**：支持表结构变更回滚
5. **在线 DDL**：集成 pt-online-schema-change 等工具
6. **前端界面**：提供可视化的表结构编辑界面

## 文件清单

### 修改的文件
- `internal/model/database.go` - 新增数据模型
- `internal/adapter/adapter.go` - 扩展接口
- `internal/adapter/mysql.go` - MySQL 实现
- `internal/adapter/postgresql.go` - PostgreSQL 实现
- `internal/adapter/sqlite.go` - SQLite 实现
- `internal/adapter/clickhouse.go` - ClickHouse 实现
- `internal/server/handler.go` - API 路由和处理函数
- `CLAUDE.md` - 项目文档更新

### 新增的文件
- `docs/ALTER_TABLE_EXAMPLES.md` - 使用示例文档
- `internal/adapter/alter_table_test.go` - 单元测试
- `PLAN_ALTER_TABLE.md` - 实现计划
- `docs/ALTER_TABLE_SUMMARY.md` - 本总结文档

## 总结

成功为 DBM 添加了完整的表结构修改功能，支持四种主流数据库，提供了统一的 API 接口和详细的文档。代码经过测试验证，可以投入使用。特别针对 ClickHouse 的复制表场景做了特殊处理，确保用户了解 ZooKeeper 同步机制的影响。