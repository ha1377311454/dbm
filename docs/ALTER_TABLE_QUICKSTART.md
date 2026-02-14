# 表结构修改功能快速开始

## 快速示例

### 1. 添加一个新列

```bash
curl -X POST "http://localhost:8080/api/v1/connections/your-conn-id/tables/users/alter?database=mydb" \
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

### 2. 修改列类型

```bash
curl -X POST "http://localhost:8080/api/v1/connections/your-conn-id/tables/users/alter?database=mydb" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "MODIFY_COLUMN",
        "column": {
          "name": "age",
          "type": "INT",
          "nullable": true,
          "defaultValue": "0"
        }
      }
    ]
  }'
```

### 3. 添加索引

```bash
curl -X POST "http://localhost:8080/api/v1/connections/your-conn-id/tables/users/alter?database=mydb" \
  -H "Content-Type: application/json" \
  -d '{
    "actions": [
      {
        "type": "ADD_INDEX",
        "index": {
          "name": "idx_email",
          "columns": ["email"],
          "unique": true
        }
      }
    ]
  }'
```

### 4. 重命名表

```bash
curl -X POST "http://localhost:8080/api/v1/connections/your-conn-id/tables/old_table/rename?database=mydb" \
  -H "Content-Type: application/json" \
  -d '{
    "newName": "new_table"
  }'
```

## 操作类型

| 类型 | 说明 | 必需字段 |
|------|------|---------|
| `ADD_COLUMN` | 添加列 | `column` |
| `DROP_COLUMN` | 删除列 | `oldName` |
| `MODIFY_COLUMN` | 修改列 | `column` |
| `RENAME_COLUMN` | 重命名列 | `oldName`, `newName`, `column` |
| `ADD_INDEX` | 添加索引 | `index` |
| `DROP_INDEX` | 删除索引 | `oldName` |

## 数据库兼容性

| 操作 | MySQL | PostgreSQL | SQLite | ClickHouse |
|------|:-----:|:----------:|:------:|:----------:|
| 添加列 | ✅ | ✅ | ✅ | ✅ |
| 删除列 | ✅ | ✅ | ❌ | ✅ |
| 修改列 | ✅ | ✅ | ❌ | ✅ |
| 重命名列 | ✅ | ✅ | ✅ | ✅ |
| 添加索引 | ✅ | ✅ | ✅ | ❌ |
| 删除索引 | ✅ | ✅ | ✅ | ❌ |

## 注意事项

⚠️ **重要提示**：

1. **备份数据**：执行前务必备份数据
2. **测试环境**：先在测试环境验证
3. **SQLite 限制**：不支持删除列和修改列
4. **ClickHouse 复制表**：ALTER 操作会通过 ZooKeeper 同步，RENAME 不会

## 更多示例

查看完整文档：
- [详细使用示例](./ALTER_TABLE_EXAMPLES.md)
- [实现总结](./ALTER_TABLE_SUMMARY.md)