# CLAUDE.md - DBM 数据库管理工具

> 最后更新：2026-02-14

## 变更记录

| 日期       | 变更内容                                      |
|------------|----------------------------------------------|
| 2026-02-14 | 添加 KingBase 支持，完善架构说明；添加表结构修改功能；添加类型映射功能；添加 SQL 导出预览 API |
| 2026-02-13 | 初始文档创建，记录项目架构与开发规范 |

---

## 项目愿景

DBM (Database Manager) 是一个用 Go 语言开发的现代化、轻量级、跨平台的通用数据库管理工具。核心设计目标是提供统一的数据库管理体验，通过单文件部署方式简化开发者和 DBA 的日常工作。

**核心价值**：

- 多数据库统一管理（MySQL、PostgreSQL、SQLite、ClickHouse、KingBase）
- 单文件部署，无额外依赖
- 现代 Web 界面，操作便捷
- 安全的 AES-256-GCM 密码加密存储
- 适配器模式架构，易于扩展
- 连接池管理，提升性能

---

## 架构总览

### 系统分层架构

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                         用户浏览器 (Vue.js SPA)                      │
│                    (嵌入 Go 二进制，通过 embed)                       │
└─────────────────────────────────────────────────────────────────────────┘
                                      │ HTTP/WebSocket
                                      ↓
┌─────────────────────────────────────────────────────────────────────────┐
│                          Gin HTTP 服务器                              │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐     │
│  │   静态资源   │  │   API 路由   │  │   CORS/Recovery      │     │
│  │   (embed)    │  │   (/api/v1/) │  │   中间件             │     │
│  └──────────────┘  └──────────────┘  └────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────┘
                                   ↓
┌─────────────────────────────────────────────────────────────────────────┐
│                          业务逻辑层                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐      │
│  │ 连接管理 │  │ SQL引擎  │  │ 导出引擎 │  │ 适配器工厂  │      │
│  │ Manager  │  │ Service  │  │ Export   │  │  Factory     │      │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────────────────┘
                                   ↓
┌─────────────────────────────────────────────────────────────────────────┐
│                      数据库适配器接口层                                │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌────────┐ ┌──────┐ ┌──────┐       │
│  │ MySQL│ │ PG    │ │SQLite│ │ClickHouse│ │KingBase│ │ MSSQL │       │
│  └──────┘ └──────┘ └──────┘ └────────┘ └──────┘ └──────┘       │
└─────────────────────────────────────────────────────────────────────────┘
                                   ↓
┌─────────────────────────────────────────────────────────────────────────┐
│   MySQL │ PostgreSQL │ SQLite │ ClickHouse │ KingBase │ MSSQL │ Oracle │
└─────────────────────────────────────────────────────────────────────────┘
```

### 目录结构

```text
dbm/
├── cmd/dbm/              # 程序入口
│   └── main.go          # 主程序：命令行参数、配置加载、服务器启动
├── internal/
│   ├── adapter/          # 数据库适配器（插件化设计）
│   │   ├── adapter.go    # DatabaseAdapter 接口定义
│   │   ├── factory.go    # 适配器工厂
│   │   ├── mysql.go      # MySQL 适配器
│   │   ├── postgresql.go # PostgreSQL 适配器
│   │   ├── sqlite.go     # SQLite 适配器
│   │   ├── clickhouse.go # ClickHouse 适配器
│   │   ├── kingbase.go   # KingBase 适配器
│   │   └── gokb/         # KingBase 驱动（本地模块）
│   ├── connection/       # 连接管理
│   │   ├── manager.go    # 连接管理器（连接池、配置持久化）
│   │   ├── crypto.go     # AES-256-GCM 密码加密
│   │   └── errors.go    # 错误定义
│   ├── service/          # 业务服务层
│   │   ├── connection.go # 连接服务
│   │   └── database.go  # 数据库服务
│   ├── export/           # 导出引擎
│   │   ├── csv.go        # CSV 导出器
│   │   ├── sql.go        # SQL 导出器
│   │   └── type_mapper.go # 类型映射器
│   ├── model/            # 数据模型
│   │   ├── connection.go # 连接配置模型
│   │   ├── database.go   # 数据库元数据模型
│   │   └── group.go     # 分组模型
│   ├── server/           # HTTP 服务器
│   │   └── handler.go   # 路由与处理器
│   └── assets/          # 嵌入的前端资源
│       └── assets.go    # embed 文件系统
├── web/                 # 前端项目 (Vue 3 + TypeScript)
│   ├── src/
│   │   ├── views/       # 页面组件
│   │   ├── stores/      # Pinia 状态管理
│   │   ├── router/      # Vue Router 配置
│   │   └── api/         # API 客户端
│   ├── package.json
│   └── vite.config.ts
├── configs/
│   └── config.example.yaml
├── scripts/
│   └── build.sh        # 构建脚本
├── docs/
│   ├── REQUIREMENTS.md   # 需求文档
│   └── DESIGN.md        # 设计文档
├── Makefile             # 构建命令
├── go.mod
└── README.md
```

---

## 模块索引

| 模块路径             | 职责描述                                          |
|---------------------|--------------------------------------------------|
| `internal/adapter` | 数据库适配器接口层，实现多数据库统一抽象 |
| `internal/connection` | 连接管理器，负责连接池、配置持久化、密码加密 |
| `internal/service` | 业务服务层，协调适配器和连接管理器 |
| `internal/export` | 导出引擎，支持 CSV 和 SQL 格式导出 |
| `internal/model` | 数据模型定义（连接配置、表结构、查询结果等） |
| `internal/server` | HTTP 服务器，提供 RESTful API |
| `internal/assets` | 嵌入式前端资源，使用 Go embed |
| `web/` | Vue 3 前端项目 |

---

## 运行与开发

### 环境要求

- Go 1.24+
- Node.js 18+

### 快速启动

```bash
# 开发模式（后端热重载）
make dev

# 启动前端开发服务器
make dev-web

# 完整构建
make build
```

### 调试运行

```bash
# 停止旧进程、编译并运行服务器（日志输出到 server.log）
lsof -t -i:2048 | xargs kill || true && make build && ./dist/dbm > server.log 2>&1 &
```

> **提示**：每次修改代码后使用上述命令重新编译运行，日志会输出到 `server.log` 文件中。

### 构建命令

| 命令              | 说明                                       |
|-------------------|-------------------------------------------|
| `make build` | 构建当前平台 |
| `make build-web` | 仅构建前端 |
| `make build-all` | 构建所有平台（Linux/macOS/Windows） |
| `make clean` | 清理构建产物 |

### 前端开发

```bash
cd web
npm install
npm run dev    # 开发服务器
npm run build  # 生产构建
```

### 测试

```bash
make test  # 运行所有测试
```

### 代码检查

```bash
make lint  # 运行 golangci-lint
```

---

## 测试策略

当前项目使用 Go 标准测试框架。

测试文件应放在与源代码同目录下，命名为 `*_test.go`。

---

## 编码规范

### Go 代码规范

1. **包命名**：使用小写单词，避免下划线和驼峰
2. **接口定义**：优先在 `adapter` 包中定义接口
3. **错误处理**：使用 `internal/connection/errors.go` 中定义的错误变量
4. **并发安全**：`connection.Manager` 已使用 `sync.RWMutex`，调用时无需额外加锁

### 前端代码规范

1. **组件命名**：使用 PascalCase
2. **状态管理**：使用 Pinia stores
3. **API 调用**：通过 `src/api/index.ts` 统一调用

---

## API 路由

### 基础路径

```text
BASE_URL: /api/v1
```

### 连接管理

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| GET  | /connections              | 获取连接列表        |
| POST | /connections              | 创建连接            |
| PUT  | /connections/:id          | 更新连接            |
| DELETE | /connections/:id       | 删除连接            |
| POST | /connections/:id/connect  | 建立连接            |
| POST | /connections/:id/close    | 关闭连接            |
| POST | /connections/:id/test     | 测试连接            |
| POST | /connections/test         | 测试连接配置（未保存） |

### 数据库元数据

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| GET | /connections/:id/databases              | 获取数据库列表 |
| GET | /connections/:id/schemas                | 获取 schema 列表（PostgreSQL） |
| GET | /connections/:id/tables                 | 获取表列表    |
| GET | /connections/:id/tables/:table/schema  | 获取表结构    |
| GET | /connections/:id/views                  | 获取视图列表  |

### SQL 执行

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| POST | /connections/:id/query    | 执行查询       |
| POST | /connections/:id/execute  | 执行非查询 SQL |

### 数据编辑

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| POST   | /connections/:id/tables/:table/data | 创建数据 |
| PUT    | /connections/:id/tables/:table/data | 更新数据 |
| DELETE | /connections/:id/tables/:table/data | 删除数据 |

### 表结构修改

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| POST | /connections/:id/tables/:table/alter  | 修改表结构 |
| POST | /connections/:id/tables/:table/rename | 重命名表   |

**修改表结构请求示例**：

```json
{
  "database": "test_db",
  "table": "users",
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
    },
    {
      "type": "MODIFY_COLUMN",
      "column": {
        "name": "age",
        "type": "INT",
        "nullable": true,
        "defaultValue": "0"
      }
    },
    {
      "type": "RENAME_COLUMN",
      "oldName": "old_name",
      "newName": "new_name",
      "column": {
        "name": "new_name",
        "type": "VARCHAR",
        "length": 100
      }
    },
    {
      "type": "DROP_COLUMN",
      "oldName": "unused_column"
    },
    {
      "type": "ADD_INDEX",
      "index": {
        "name": "idx_email",
        "columns": ["email"],
        "unique": true
      }
    },
    {
      "type": "DROP_INDEX",
      "oldName": "idx_old"
    }
  ]
}
```

**支持的操作类型**：
- `ADD_COLUMN`: 添加列
- `DROP_COLUMN`: 删除列
- `MODIFY_COLUMN`: 修改列
- `RENAME_COLUMN`: 重命名列
- `ADD_INDEX`: 添加索引
- `DROP_INDEX`: 删除索引

**数据库特性差异**：

| 操作 | MySQL | PostgreSQL | SQLite | ClickHouse | KingBase |
|------|-------|------------|--------|------------|----------|
| 添加列 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 删除列 | ✅ | ✅ | ❌ 需重建表 | ✅ | ✅ |
| 修改列 | ✅ | ✅ 需多条语句 | ❌ 需重建表 | ✅ | ✅ 需多条语句 |
| 重命名列 | ✅ | ✅ | ✅ (3.25.0+) | ✅ | ✅ |
| 添加索引 | ✅ | ✅ | ✅ | ❌ 使用 ORDER BY | ✅ |
| 删除索引 | ✅ | ✅ | ✅ | ❌ | ✅ |
| 重命名表 | ✅ | ✅ | ✅ | ✅ (非复制表) | ✅ |

**ClickHouse 特别说明**：
- 对于 `ReplicatedMergeTree` 表，ALTER 操作会通过 ZooKeeper 自动同步到所有副本
- RENAME TABLE 操作不会通过 ZooKeeper 同步，需要在每个副本上分别执行
- ClickHouse 不支持传统索引，使用 ORDER BY 和 PRIMARY KEY 代替

**KingBase 特别说明**：
- KingBase 基于 PostgreSQL 内核，大部分语法与 PostgreSQL 兼容
- 支持标准 SQL 语法和 PostgreSQL 扩展功能
- 连接时使用 PostgreSQL 驱动（通过本地 gokb 模块）

### 导出

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| POST | /connections/:id/export/csv      | CSV 导出              |
| POST | /connections/:id/export/sql      | SQL 导出              |
| POST | /connections/:id/export/sql/preview | SQL 导出类型映射预览 |

### 分组管理

| 方法  | 路径            | 描述         |
|------|-----------------|-------------|
| GET  | /groups         | 获取分组列表 |
| POST | /groups         | 创建分组     |
| PUT  | /groups/:id     | 更新分组     |
| DELETE | /groups/:id   | 删除分组     |

### 监控

| 方法 | 路径                    | 描述             |
|------|-------------------------|----------------|
| GET  | /metrics                | Prometheus 指标 |
| GET  | /api/v1/monitor/stats   | 监控统计        |

---

## 架构设计要点

### 适配器模式

项目采用适配器模式实现多数据库支持：

1. **核心接口** (`adapter.go`)：定义 `DatabaseAdapter` 接口，包含所有数据库操作
2. **工厂模式** (`factory.go`)：根据数据库类型创建对应适配器
3. **具体实现**：各数据库适配器实现核心接口
4. **扩展接口**：`SchemaAwareDatabase` 为 PostgreSQL 等支持 schema 的数据库提供额外方法

### 连接管理设计

- **连接池**：每个数据库连接维护独立的 `sql.DB` 连接池
- **并发安全**：使用 `sync.RWMutex` 保护配置访问
- **密码加密**：AES-256-GCM 加密存储，密钥文件位于 `~/.dbm/.key`
- **配置持久化**：连接配置和分组分别存储在 JSON 文件中

### 前后端交互

- **单文件部署**：前端资源通过 Go `embed` 嵌入二进制
- **API 响应格式**：统一使用 `APIResponse` 结构（code/message/data）
- **SPA 路由**：所有非 API 路径返回 `index.html`，由 Vue Router 处理

### 服务层职责

- **ConnectionService**：连接创建、测试、关闭
- **DatabaseService**：适配器获取、元数据查询、SQL 执行
- **Export**：独立导出器（CSV/SQL），可扩展

---

## AI 使用指引

### 添加新数据库支持

1. 在 `internal/model/connection.go` 添加新的数据库类型常量（如 `DatabaseKingBase`）
2. 在 `internal/adapter/` 创建新的适配器文件，实现 `DatabaseAdapter` 接口
3. 在 `internal/adapter/factory.go` 的 `CreateAdapter` 方法中添加新数据库的 case
4. 在 `SupportedTypes()` 方法中添加新类型
5. 如果需要本地驱动（如 KingBase），使用 `replace` 指令引用本地模块

**示例**（KingBase）：
```text
# go.mod
replace kingbase.com/gokb => ./internal/adapter/gokb
```
```go
// factory.go
case model.DatabaseKingBase:
    return &KingBaseAdapter{BaseAdapter: *NewBaseAdapter()}, nil
```

### 添加新的导出格式

1. 在 `internal/export/` 创建新的导出器文件
2. 在 `DatabaseAdapter` 接口中添加新的导出方法
3. 在各适配器中实现新方法
4. 在 `server/handler.go` 添加对应的 API 路由

### 扩展 API

1. 在 `internal/server/handler.go` 中添加新的路由和处理函数
2. 使用统一的响应格式 `APIResponse`
3. 遵循 RESTful 设计原则

### 修改表结构

表结构修改功能已集成到适配器接口中，支持以下操作：

**添加新的表结构修改操作**：
1. 在 `internal/model/database.go` 中添加新的 `AlterActionType` 常量
2. 在各数据库适配器的 `buildAlterClause` 方法中添加对应的处理逻辑
3. 注意不同数据库的语法差异和功能限制

**数据库特性对比**：
- **MySQL**: 功能最完整，支持所有常见操作
- **PostgreSQL**: 修改列需要多条 ALTER 语句（类型、可空性、默认值分别设置）
- **SQLite**: 功能受限，DROP/MODIFY COLUMN 需要重建表
- **ClickHouse**: 不支持传统索引，复制表的 RENAME 操作不通过 ZooKeeper 同步
- **KingBase**: 基于 PostgreSQL 内核，支持类似 PostgreSQL 的语法和功能

---

## 重要约定

1. **密码安全**：所有密码必须通过 `connection.Manager` 的加密方法处理后再存储
2. **连接管理**：使用 `connection.Manager` 管理所有数据库连接，不要直接创建连接
3. **适配器模式**：所有数据库操作都应通过适配器接口进行
4. **配置持久化**：连接配置保存在 `~/.dbm/connections.json`

---

## 常见问题

### Q: 如何修改默认端口？

A: 启动时使用 `--port` 参数：`./dbm --port 9000`

### Q: 连接配置保存在哪里？

A: 默认保存在 `~/.dbm/connections.json`，密码已 AES-256 加密

### Q: 如何添加新的数据库适配器？

A: 参考 `internal/adapter/mysql.go` 实现 `DatabaseAdapter` 接口

---

## 依赖库

### 后端主要依赖

| 库                                  | 版本     | 用途              |
|------------------------------------|---------|-----------------|
| gin-gonic/gin                      | 1.10.0  | HTTP 框架       |
| go-sql-driver/mysql                | 1.9.3   | MySQL 驱动      |
| lib/pq                             | 1.11.2  | PostgreSQL 驱动 |
| mattn/go-sqlite3                   | 1.14.34 | SQLite 驱动     |
| ClickHouse/clickhouse-go/v2        | 2.43.0  | ClickHouse 驱动 |
| kingbase.com/gokb                   | 1.0.0   | KingBase 驱动（本地模块） |
| google/uuid                        | 1.6.0   | UUID 生成       |

### 前端主要依赖

| 库               | 版本  | 用途         |
|------------------|-------|------------|
| Vue.js           | 3.4+  | 前端框架    |
| Element Plus     | 2.5+  | UI 组件库   |
| Monaco Editor   | 0.45+ | SQL 编辑器  |
| Pinia            | 2.1+  | 状态管理    |
| Axios            | 1.6+  | HTTP 客户端 |
| ECharts          | 5.5+  | 图表组件    |

---

## 类型映射功能

### 功能说明

SQL 导出类型映射功能用于跨数据库迁移时的类型转换，确保不同数据库之间的数据类型兼容性。

### 配置文件

类型映射规则存储在 `configs/type_mapping.yaml` 中，支持以下配置：

```yaml
type_mapping:
  mysql_to_postgresql:
    TINYINT:
      target: "SMALLINT"
      safe_fallback: "INTEGER"
      precision_loss: true
    DATETIME:
      target: "TIMESTAMP"
      precision_loss: false
```

### API 使用

**预览类型映射**：

```bash
POST /api/v1/connections/:id/export/sql/preview
Content-Type: application/json

{
  "tables": ["users", "orders"],
  "targetDbType": "postgresql"
}
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "mapped": {
      "TINYINT": "SMALLINT",
      "DATETIME": "TIMESTAMP",
      "VARCHAR(255)": "VARCHAR(255)"
    },
    "warnings": [
      "TINYINT → SMALLINT (有精度损失)"
    ],
    "summary": {
      "total": 15,
      "direct": 12,
      "fallback": 2,
      "userChoice": 1,
      "lossyCount": 1
    }
  }
}
```

### 扩展类型映射

添加新的数据库类型映射：

1. 在 `configs/type_mapping.yaml` 中添加映射规则
2. 在 `internal/export/type_mapper.go` 中验证映射加载
3. 测试跨数据库导出功能

---

## 许可证

MIT License
