# 表结构修改功能使用示例

本文档提供 DBM 表结构修改功能的详细使用示例。

## API 端点

### 修改表结构
```
POST /api/v1/connections/:id/tables/:table/alter?database=<database_name>
```

### 重命名表
```
POST /api/v1/connections/:id/tables/:table/rename?database=<database_name>
```

---

## 基础示例

### 1. 添加列

**请求**：
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
          "defaultValue": "",
          "comment": "用户邮箱地址"
        }
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` ADD COLUMN `email` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '用户邮箱地址'
```

### 2. 删除列

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "DROP_COLUMN",
        "oldName": "unused_field"
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` DROP COLUMN `unused_field`
```

### 3. 修改列类型

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "MODIFY_COLUMN",
        "column": {
          "name": "age",
          "type": "INT",
          "nullable": true,
          "defaultValue": "0",
          "comment": "用户年龄"
        }
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` MODIFY COLUMN `age` INT NULL DEFAULT '0' COMMENT '用户年龄'
```

**生成的 SQL (PostgreSQL)**：
```sql
ALTER TABLE "test_db"."users" ALTER COLUMN "age" TYPE INT;
ALTER TABLE "test_db"."users" ALTER COLUMN "age" DROP NOT NULL;
ALTER TABLE "test_db"."users" ALTER COLUMN "age" SET DEFAULT '0';
```

### 4. 重命名列

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "RENAME_COLUMN",
        "oldName": "user_name",
        "newName": "username",
        "column": {
          "name": "username",
          "type": "VARCHAR",
          "length": 100,
          "nullable": false
        }
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` CHANGE COLUMN `user_name` `username` VARCHAR(100) NOT NULL
```

**生成的 SQL (PostgreSQL)**：
```sql
ALTER TABLE "test_db"."users" RENAME COLUMN "user_name" TO "username"
```

### 5. 添加索引

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "ADD_INDEX",
        "index": {
          "name": "idx_email",
          "columns": ["email"],
          "unique": true,
          "type": "BTREE",
          "comment": "邮箱唯一索引"
        }
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` ADD UNIQUE INDEX `idx_email` (`email`) USING BTREE COMMENT '邮箱唯一索引'
```

**生成的 SQL (PostgreSQL)**：
```sql
CREATE UNIQUE INDEX "idx_email" ON "test_db"."users" ("email") USING BTREE
```

### 6. 删除索引

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "DROP_INDEX",
        "oldName": "idx_old"
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users` DROP INDEX `idx_old`
```

**生成的 SQL (PostgreSQL)**：
```sql
DROP INDEX "test_db"."idx_old"
```

### 7. 重命名表

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/old_users/rename?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "newName": "users"
  }'
```

**生成的 SQL (MySQL)**：
```sql
RENAME TABLE `test_db`.`old_users` TO `test_db`.`users`
```

**生成的 SQL (PostgreSQL)**：
```sql
ALTER TABLE "test_db"."old_users" RENAME TO "users"
```

---

## 批量操作示例

### 一次性执行多个操作

**请求**：
```bash
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/tables/users/alter?database=test_db" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "ADD_COLUMN",
        "column": {
          "name": "phone",
          "type": "VARCHAR",
          "length": 20,
          "nullable": true,
          "comment": "手机号码"
        }
      },
      {
        "type": "ADD_COLUMN",
        "column": {
          "name": "created_at",
          "type": "TIMESTAMP",
          "nullable": false,
          "defaultValue": "CURRENT_TIMESTAMP",
          "comment": "创建时间"
        }
      },
      {
        "type": "MODIFY_COLUMN",
        "column": {
          "name": "status",
          "type": "TINYINT",
          "nullable": false,
          "defaultValue": "1",
          "comment": "用户状态"
        }
      },
      {
        "type": "ADD_INDEX",
        "index": {
          "name": "idx_phone",
          "columns": ["phone"],
          "unique": false
        }
      }
    ]
  }'
```

**生成的 SQL (MySQL)**：
```sql
ALTER TABLE `test_db`.`users`
  ADD COLUMN `phone` VARCHAR(20) NULL COMMENT '手机号码',
  ADD COLUMN `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  MODIFY COLUMN `status` TINYINT NOT NULL DEFAULT '1' COMMENT '用户状态',
  ADD INDEX `idx_phone` (`phone`)
```

---

## 数据类型示例

### MySQL 数据类型

```json
{
  "actions": [
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "int_col",
        "type": "INT",
        "nullable": false,
        "defaultValue": "0"
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "varchar_col",
        "type": "VARCHAR",
        "length": 255,
        "nullable": true
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "decimal_col",
        "type": "DECIMAL",
        "precision": 10,
        "scale": 2,
        "nullable": false,
        "defaultValue": "0.00"
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "text_col",
        "type": "TEXT",
        "nullable": true
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "datetime_col",
        "type": "DATETIME",
        "nullable": false,
        "defaultValue": "CURRENT_TIMESTAMP"
      }
    }
  ]
}
```

### PostgreSQL 数据类型

```json
{
  "actions": [
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "serial_col",
        "type": "INTEGER",
        "autoIncrement": true,
        "nullable": false
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "json_col",
        "type": "JSONB",
        "nullable": true
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "array_col",
        "type": "TEXT[]",
        "nullable": true
      }
    }
  ]
}
```

### ClickHouse 数据类型

```json
{
  "actions": [
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "uint64_col",
        "type": "UInt64",
        "nullable": false,
        "defaultValue": "0"
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "string_col",
        "type": "String",
        "nullable": true
      }
    },
    {
      "type": "ADD_COLUMN",
      "column": {
        "name": "datetime_col",
        "type": "DateTime",
        "nullable": false,
        "defaultValue": "now()"
      }
    }
  ]
}
```

---

## 错误处理

### 常见错误

**1. 列已存在**
```json
{
  "code": 500,
  "message": "Error 1060: Duplicate column name 'email'"
}
```

**2. 列不存在**
```json
{
  "code": 500,
  "message": "Error 1054: Unknown column 'nonexistent' in 'users'"
}
```

**3. 数据类型不兼容**
```json
{
  "code": 500,
  "message": "Error 1265: Data truncated for column 'age' at row 1"
}
```

**4. SQLite 不支持的操作**
```json
{
  "code": 500,
  "message": "SQLite does not support DROP/MODIFY COLUMN directly, table rebuild required"
}
```

**5. ClickHouse 复制表重命名**
```json
{
  "code": 500,
  "message": "RENAME TABLE is not supported for replicated tables via ZooKeeper. You must rename on each replica separately"
}
```

---

## 最佳实践

### 1. 备份数据
在执行表结构修改前，务必备份数据：
```bash
# 导出表结构和数据
curl -X POST "http://localhost:8080/api/v1/connections/conn-123/export/sql" \
  -H "Content-Type: application/json" \
  -d '{
    "database": "test_db",
    "tables": ["users"],
    "includeCreateTable": true,
    "includeDropTable": false
  }' > backup.sql
```

### 2. 测试环境验证
先在测试环境执行，确认无误后再在生产环境执行。

### 3. 分批执行
对于大表，建议分批执行操作，避免长时间锁表。

### 4. 监控执行状态
对于 ClickHouse 复制表，可以查询 `system.mutations` 表监控进度：
```sql
SELECT * FROM system.mutations
WHERE database = 'test_db' AND table = 'users'
ORDER BY create_time DESC;
```

### 5. 注意数据类型兼容性
修改列类型时，确保现有数据可以转换为新类型。

### 6. 索引命名规范
- 普通索引：`idx_<column_name>`
- 唯一索引：`uk_<column_name>`
- 复合索引：`idx_<col1>_<col2>`

---

## 数据库特性对比

| 特性 | MySQL | PostgreSQL | SQLite | ClickHouse |
|------|-------|------------|--------|------------|
| 添加列 | ✅ 支持 AFTER | ✅ | ✅ | ✅  支持 AFTER |
| 删除列 | ✅ | ✅ | ❌ 需重建表 | ✅ |
| 修改列类型 | ✅ MODIFY | ✅ ALTER TYPE | ❌ 需重建表 | ✅ MODIFY |
| 修改列默认值 | ✅ | ✅ SET DEFAULT | ❌ | ✅ |
| 修改列可空性 | ✅ | ✅ SET/DROP NOT NULL | ❌ | ✅ |
| 重命名列 | ✅ CHANGE | ✅ RENAME | ✅ (3.25.0+) | ✅ RENAME |
| 添加索引 | ✅ | ✅ | ✅ | ❌ 使用 ORDER BY |
| 删除索引 | ✅ | ✅ | ✅ | ❌ |
| 重命名表 | ✅ | ✅ | ✅ | ✅ (非复制表) |
| 批量操作 | ✅ 一条语句 | ❌ 需多条语句 | ❌ 需多条语句 | ✅ 一条语句 |
| 事务支持 | ✅ | ✅ | ✅ | ❌ |

---

## 注意事项

### MySQL
- 使用 `MODIFY COLUMN` 修改列时，必须指定完整的列定义
- `CHANGE COLUMN` 可以同时重命名和修改列
- 大表 ALTER 操作可能锁表较长时间，考虑使用 `pt-online-schema-change`

### PostgreSQL
- 修改列需要多条 ALTER 语句（类型、默认值、可空性分别设置）
- 添加 NOT NULL 约束前，确保列中没有 NULL 值
- 使用 `CONCURRENTLY` 选项可以在线创建索引

### SQLite
- 不支持 DROP COLUMN 和 MODIFY COLUMN
- 需要重建表的操作会自动在事务中执行
- 重建表时会保留数据，但可能丢失触发器和视图

### ClickHouse
- ReplicatedMergeTree 表的 ALTER 操作会通过 ZooKeeper 同步
- RENAME TABLE 不会通过 ZooKeeper 同步
- ALTER 操作是异步的，可能需要时间完成
- 不支持传统索引，使用 ORDER BY 和 PRIMARY KEY