# 修改表结构功能实现计划

## 功能概述

为 DBM 添加完整的表结构修改功能，支持：
- 添加列
- 删除列
- 修改列（类型、默认值、可空性、注释）
- 重命名列
- 添加索引
- 删除索引
- 重命名表

## 实现步骤

### 第一步：扩展数据模型 (internal/model/database.go)

添加表结构修改相关的请求模型：

```go
// AlterTableRequest 修改表结构请求
type AlterTableRequest struct {
    Database string              `json:"database"`
    Table    string              `json:"table"`
    Actions  []AlterTableAction  `json:"actions"`
}

// AlterTableAction 表结构修改操作
type AlterTableAction struct {
    Type   AlterActionType `json:"type"`   // ADD_COLUMN, DROP_COLUMN, MODIFY_COLUMN, etc.
    Column *ColumnDef      `json:"column"` // 列定义（用于添加/修改列）
    OldName string         `json:"oldName"` // 旧名称（用于重命名）
    NewName string         `json:"newName"` // 新名称（用于重命名）
    Index  *IndexDef       `json:"index"`   // 索引定义
}

// AlterActionType 修改操作类型
type AlterActionType string

const (
    AlterActionAddColumn    AlterActionType = "ADD_COLUMN"
    AlterActionDropColumn   AlterActionType = "DROP_COLUMN"
    AlterActionModifyColumn AlterActionType = "MODIFY_COLUMN"
    AlterActionRenameColumn AlterActionType = "RENAME_COLUMN"
    AlterActionAddIndex     AlterActionType = "ADD_INDEX"
    AlterActionDropIndex    AlterActionType = "DROP_INDEX"
    AlterActionRenameTable  AlterActionType = "RENAME_TABLE"
)

// ColumnDef 列定义
type ColumnDef struct {
    Name         string `json:"name"`
    Type         string `json:"type"`
    Length       int    `json:"length,omitempty"`       // 类型长度，如 VARCHAR(255)
    Precision    int    `json:"precision,omitempty"`    // 精度，如 DECIMAL(10,2)
    Scale        int    `json:"scale,omitempty"`        // 小数位数
    Nullable     bool   `json:"nullable"`
    DefaultValue string `json:"defaultValue,omitempty"`
    AutoIncrement bool  `json:"autoIncrement,omitempty"`
    Comment      string `json:"comment,omitempty"`
    After        string `json:"after,omitempty"`        // 在哪个列之后（MySQL）
}

// IndexDef 索引定义
type IndexDef struct {
    Name    string   `json:"name"`
    Columns []string `json:"columns"`
    Unique  bool     `json:"unique"`
    Type    string   `json:"type,omitempty"` // BTREE, HASH, etc.
    Comment string   `json:"comment,omitempty"`
}
```

### 第二步：扩展适配器接口 (internal/adapter/adapter.go)

在 `DatabaseAdapter` 接口中添加新方法：

```go
// 表结构修改
AlterTable(db *sql.DB, request *model.AlterTableRequest) error
RenameTable(db *sql.DB, database, oldName, newName string) error
```

### 第三步：实现 MySQL 适配器 (internal/adapter/mysql.go)

实现 MySQL 的表结构修改逻辑：

```go
func (a *MySQLAdapter) AlterTable(db *sql.DB, request *model.AlterTableRequest) error {
    // 1. 构建 ALTER TABLE SQL 语句
    // 2. 处理不同的操作类型
    // 3. 执行 SQL
    // 4. 返回结果
}

func (a *MySQLAdapter) RenameTable(db *sql.DB, database, oldName, newName string) error {
    // RENAME TABLE 语句
}
```

关键 SQL 模板：
- 添加列：`ALTER TABLE table ADD COLUMN name type [options]`
- 删除列：`ALTER TABLE table DROP COLUMN name`
- 修改列：`ALTER TABLE table MODIFY COLUMN name type [options]`
- 重命名列：`ALTER TABLE table CHANGE COLUMN old_name new_name type`
- 添加索引：`ALTER TABLE table ADD INDEX name (columns)`
- 删除索引：`ALTER TABLE table DROP INDEX name`

### 第四步：实现 PostgreSQL 适配器 (internal/adapter/postgresql.go)

PostgreSQL 语法差异：
- 添加列：`ALTER TABLE table ADD COLUMN name type [options]`
- 删除列：`ALTER TABLE table DROP COLUMN name`
- 修改列类型：`ALTER TABLE table ALTER COLUMN name TYPE type`
- 修改列默认值：`ALTER TABLE table ALTER COLUMN name SET DEFAULT value`
- 修改列可空性：`ALTER TABLE table ALTER COLUMN name SET NOT NULL`
- 重命名列：`ALTER TABLE table RENAME COLUMN old_name TO new_name`
- 添加索引：`CREATE INDEX name ON table (columns)`
- 删除索引：`DROP INDEX name`

### 第五步：实现 SQLite 适配器 (internal/adapter/sqlite.go)

SQLite 限制较多，需要特殊处理：
- 支持：添加列、重命名表、重命名列（3.25.0+）
- 不支持：删除列、修改列（需要重建表）

重建表流程：
1. 创建新表（带新结构）
2. 复制数据
3. 删除旧表
4. 重命名新表

### 第六步：实现 ClickHouse 适配器 (internal/adapter/clickhouse.go)

ClickHouse 语法：
- 添加列：`ALTER TABLE table ADD COLUMN name type`
- 删除列：`ALTER TABLE table DROP COLUMN name`
- 修改列：`ALTER TABLE table MODIFY COLUMN name type`
- 重命名列：`ALTER TABLE table RENAME COLUMN old_name TO new_name`

注意：ClickHouse 不支持传统索引，使用 ORDER BY 和 PRIMARY KEY

### 第七步：添加 API 路由 (internal/server/handler.go)

添加新的 HTTP 处理函数：

```go
// POST /api/v1/connections/:id/tables/:table/alter
func (h *Handler) alterTable(c *gin.Context) {
    // 1. 解析请求参数
    // 2. 获取数据库连接
    // 3. 调用适配器的 AlterTable 方法
    // 4. 返回结果
}

// POST /api/v1/connections/:id/tables/:table/rename
func (h *Handler) renameTable(c *gin.Context) {
    // 重命名表
}
```

注册路由：
```go
tables := v1.Group("/connections/:id/tables")
{
    tables.POST("/:table/alter", h.alterTable)
    tables.POST("/:table/rename", h.renameTable)
}
```

### 第八步：前端集成（可选）

如果需要前端界面支持，需要：
1. 在 `web/src/api/` 添加 API 调用方法
2. 在 `web/src/views/` 创建表结构编辑组件
3. 使用 Element Plus 的表单组件构建 UI

## 技术要点

### 1. SQL 注入防护
- 表名和列名使用标识符引用（MySQL: 反引号，PostgreSQL: 双引号）
- 不直接拼接用户输入
- 使用参数化查询（where 条件）

### 2. 事务处理
- 多个操作应在事务中执行
- 失败时回滚

### 3. 错误处理
- 验证列类型是否合法
- 检查列/索引是否已存在
- 处理数据库特定错误

### 4. 兼容性处理
- 不同数据库的类型映射
- 语法差异适配
- 功能限制提示

## 测试计划

### 单元测试
- 每个适配器的 AlterTable 方法
- SQL 生成逻辑
- 错误处理

### 集成测试
- 完整的表结构修改流程
- 多种操作组合
- 边界情况

### 手动测试
- 使用 Postman/curl 测试 API
- 验证数据库中的实际变更
- 测试各种数据库类型

## 风险与注意事项

1. **数据丢失风险**：删除列、修改列类型可能导致数据丢失，需要警告用户
2. **SQLite 限制**：重建表操作复杂，需要充分测试
3. **类型兼容性**：不同数据库的类型系统差异大
4. **性能影响**：大表修改可能耗时较长，考虑异步处理
5. **锁表问题**：ALTER TABLE 会锁表，影响业务

## 实现优先级

1. **P0（必须）**：
   - 数据模型定义
   - 适配器接口扩展
   - MySQL 适配器实现
   - API 路由

2. **P1（重要）**：
   - PostgreSQL 适配器实现
   - SQLite 适配器实现
   - 错误处理和验证

3. **P2（可选）**：
   - ClickHouse 适配器实现
   - 前端界面
   - 异步处理

## 预估工作量

- 后端开发：4-6 小时
- 测试：2-3 小时
- 前端开发（如需要）：3-4 小时
- 总计：6-13 小时

## 后续优化

1. 添加表结构变更历史记录
2. 支持批量修改
3. 提供表结构对比功能
4. 支持从 SQL 文件导入表结构
5. 添加表结构版本管理
