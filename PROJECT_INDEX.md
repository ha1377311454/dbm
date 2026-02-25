# PROJECT_INDEX.md - DBM 数据库管理工具

> 生成时间：2026-02-25
> 项目版本：dev
> 最后更新：2026-02-25（新增 MongoDB 数据库支持）

## 项目概览

**DBM (Database Manager)** 是一个用 Go 语言开发的现代化、轻量级、跨平台的通用数据库管理工具。核心设计目标是提供统一的数据库管理体验，通过单文件部署方式简化开发者和 DBA 的日常工作。

### 基本信息

| 属性 | 值 |
|------|-----|
| **项目名称** | DBM Database Manager |
| **项目类型** | Go 后端 + Vue 3 前端 |
| **Go 版本** | 1.24.2 |
| **主入口** | [cmd/dbm/main.go](./cmd/dbm/main.go) |
| **配置文件** | [go.mod](./go.mod) |
| **前端配置** | [web/package.json](./web/package.json) |
| **默认端口** | 2048 |
| **监听地址** | 0.0.0.0 |

### 技术栈

**后端核心依赖**：
- **gin-gonic/gin** (v1.10.0) - HTTP 框架
- **go-sql-driver/mysql** (v1.9.3) - MySQL 驱动
- **lib/pq** (v1.11.2) - PostgreSQL 驱动
- **mattn/go-sqlite3** (v1.14.34) - SQLite 驱动
- **ClickHouse/clickhouse-go/v2** (v2.43.0) - ClickHouse 驱动
- **kingbase.com/gokb** (v1.0.0) - KingBase 驱动（本地模块）
- **gitee.com/chunanyong/dm** (v1.8.22) - 达梦数据库驱动
- **google/uuid** (v1.6.0) - UUID 生成
- **golang.org/x/crypto** (v0.47.0) - AES-256-GCM 密码加密
- **gopkg.in/yaml.v3** (v3.0.1) - YAML 配置解析
- **prometheus/client_golang** (v1.23.2) - Prometheus 监控

**前端核心依赖**：
- **Vue.js** (v3.4+) - 前端框架
- **Element Plus** (v2.5+) - UI 组件库
- **Monaco Editor** (v0.45+) - SQL 编辑器
- **Pinia** (v2.1+) - 状态管理
- **Axios** (v1.6+) - HTTP 客户端
- **ECharts** (v5.5+) - 图表组件

### 支持的数据库类型

- MySQL / MariaDB (5.7+, 8.0+)
- PostgreSQL (12+, 14+, 15+)
- SQLite (3.x)
- ClickHouse (22.3+)
- KingBase (ES V8)
- 达梦 DM (8+)
- MongoDB (4.0+)

### 项目目录结构

```
dbm/
├── cmd/dbm/              # 程序入口
├── internal/
│   ├── adapter/          # 数据库适配器（插件化设计）
│   │   ├── config/       # 监控指标配置（YAML，embed）
│   │   └── gokb/         # KingBase 驱动（本地模块）
│   ├── connection/       # 连接管理
│   ├── service/          # 业务服务层
│   ├── export/           # 导出引擎（含类型映射）
│   ├── model/            # 数据模型
│   ├── server/           # HTTP 服务器
│   ├── monitor/          # Prometheus 监控采集（含嵌入配置）
│   └── assets/          # 嵌入的前端资源
├── web/                 # Vue 3 前端项目
│   └── src/
│       ├── api/         # API 客户端
│       ├── components/  # Vue 组件
│       ├── router/      # 路由配置
│       ├── stores/      # Pinia 状态管理
│       ├── types/       # TypeScript 类型
│       └── views/       # 页面组件
├── configs/             # 配置文件示例
│   └── type_mapping.yaml # 类型映射规则
├── scripts/             # 构建脚本
├── docs/                # 项目文档
├── Makefile             # 构建命令
└── README.md
```

---

## 核心文件列表

### 后端核心文件

| 文件路径 | 行数 | 功能描述 |
|---------|------|---------|
| [cmd/dbm/main.go](./cmd/dbm/main.go) | 132 | 主程序入口：命令行参数、配置加载、服务器启动 |
| [internal/server/handler.go](./internal/server/handler.go) | 1212 | HTTP 路由与处理器，定义所有 API 端点 |
| [internal/connection/manager.go](./internal/connection/manager.go) | 301 | 连接管理器：连接池、配置持久化、并发安全 |
| [internal/connection/crypto.go](./internal/connection/crypto.go) | ~100 | AES-256-GCM 密码加密 |
| [internal/adapter/adapter.go](./internal/adapter/adapter.go) | 72 | DatabaseAdapter 接口定义 |
| [internal/adapter/factory.go](./internal/adapter/factory.go) | ~80 | 适配器工厂模式实现 |
| [internal/service/connection.go](./internal/service/connection.go) | 114 | 连接服务层 |
| [internal/service/database.go](./internal/service/database.go) | 31 | 数据库服务层 |

### 数据库适配器文件

| 文件路径 | 数据库类型 |
|---------|-----------|
| [internal/adapter/mysql.go](./internal/adapter/mysql.go) | MySQL 适配器实现 |
| [internal/adapter/postgresql.go](./internal/adapter/postgresql.go) | PostgreSQL 适配器实现（支持 schema） |
| [internal/adapter/sqlite.go](./internal/adapter/sqlite.go) | SQLite 适配器实现 |
| [internal/adapter/clickhouse.go](./internal/adapter/clickhouse.go) | ClickHouse 适配器实现 |
| [internal/adapter/kingbase.go](./internal/adapter/kingbase.go) | KingBase 适配器实现 |
| [internal/adapter/dm.go](./internal/adapter/dm.go) | 达梦数据库适配器实现 |
| [internal/adapter/mongodb.go](./internal/adapter/mongodb.go) | MongoDB 适配器实现 |
| [internal/adapter/gokb/](./internal/adapter/gokb/) | KingBase 驱动（本地模块） |

### 导出引擎文件

| 文件路径 | 行数 | 功能描述 |
|---------|------|---------|
| [internal/export/csv.go](./internal/export/csv.go) | 100 | CSV 导出器 |
| [internal/export/sql.go](./internal/export/sql.go) | 234 | SQL 导出器 |
| [internal/export/type_mapper.go](./internal/export/type_mapper.go) | 187 | 类型映射器（跨数据库迁移） |

### 监控模块文件

| 文件路径 | 行数 | 功能描述 |
|---------|------|---------|
| [internal/monitor/init.go](./internal/monitor/init.go) | 123 | 监控初始化、数据库支持注册、配置 embed |
| [internal/monitor/collector.go](./internal/monitor/collector.go) | 241 | Prometheus Collector 实现 |
| [internal/monitor/scraper.go](./internal/monitor/scraper.go) | 31 | 数据库指标采集接口 |
| [internal/monitor/default_scraper.go](./internal/monitor/default_scraper.go) | 155 | 默认指标采集实现 |
| [internal/monitor/common.go](./internal/monitor/common.go) | 139 | 监控公共函数 |

#### 监控配置文件（嵌入在 monitor 模块）

| 文件路径 | 功能描述 |
|---------|---------|
| [internal/monitor/config/mysql.yaml](./internal/monitor/config/mysql.yaml) | MySQL 监控指标配置（运行时长、会话连接、最大连接数） |
| [internal/monitor/config/pg.yaml](./internal/monitor/config/pg.yaml) | PostgreSQL 监控指标配置（运行时长、会话、连接、数据库状态、缓存命中率） |
| [internal/monitor/config/kingbase.yaml](./internal/monitor/config/kingbase.yaml) | KingBase 监控指标配置（基于 PostgreSQL 内核，含缓存命中率） |
| [internal/monitor/config/dm.yaml](./internal/monitor/config/dm.yaml) | 达梦数据库监控指标配置（运行时长、会话连接、最大连接数） |

### 数据模型文件

| 文件路径 | 行数 | 功能描述 |
|---------|------|---------|
| [internal/model/connection.go](./internal/model/connection.go) | 56 | 连接配置模型、DatabaseType 枚举 |
| [internal/model/database.go](./internal/model/database.go) | 151 | 数据库元数据模型、AlterTable 请求 |
| [internal/model/group.go](./internal/model/group.go) | 8 | 分组模型 |

### 前端核心文件

| 文件路径 | 功能描述 |
|---------|---------|
| [web/package.json](./web/package.json) | 前端依赖配置 |
| [web/vite.config.ts](./web/vite.config.ts) | Vite 构建配置 |
| [web/src/main.ts](./web/src/main.ts) | 前端入口 |
| [web/src/api/index.ts](./web/src/api/index.ts) | API 客户端封装 |
| [web/src/router/index.ts](./web/src/router/index.ts) | Vue Router 配置 |
| [web/src/stores/connections.ts](./web/src/stores/connections.ts) | 连接状态管理 |
| [web/src/stores/query.ts](./web/src/stores/query.ts) | 查询状态管理 |
| [web/src/views/connections.vue](./web/src/views/connections.vue) | 连接管理页面 |
| [web/src/views/query.vue](./web/src/views/query.vue) | SQL 查询页面 |
| [web/src/views/tables.vue](./web/src/views/tables.vue) | 表数据浏览页面 |
| [web/src/views/schema-editor.vue](./web/src/views/schema-editor.vue) | 表结构编辑页面 |
| [web/src/views/export.vue](./web/src/views/export.vue) | 数据导出页面 |
| [web/src/views/monitor.vue](./web/src/views/monitor.vue) | 监控面板页面 |
| [internal/assets/assets.go](./internal/assets/assets.go) | 嵌入式前端资源 |

### 构建与配置文件

| 文件路径 | 功能描述 |
|---------|---------|
| [Makefile](./Makefile) | 构建命令定义 |
| [scripts/build.sh](./scripts/build.sh) | 构建脚本 |
| [configs/config.example.yaml](./configs/config.example.yaml) | 配置示例 |
| [configs/type_mapping.yaml](./configs/type_mapping.yaml) | 类型映射配置（跨数据库迁移） |
| [CLAUDE.md](./CLAUDE.md) | AI 开发指南 |
| [docs/DESIGN.md](./docs/DESIGN.md) | 设计文档 |
| [docs/CHANGELOG.md](./docs/CHANGELOG.md) | 变更日志 |

---

## 模块功能分解

### 1. adapter 模块

**位置**：[internal/adapter/](./internal/adapter/)

**功能**：数据库适配器接口层，实现多数据库统一抽象

**核心文件**：
- [adapter.go](./internal/adapter/adapter.go) - 定义 DatabaseAdapter 接口
- [factory.go](./internal/adapter/factory.go) - 适配器工厂
- [mysql.go](./internal/adapter/mysql.go) - MySQL 实现
- [postgresql.go](./internal/adapter/postgresql.go) - PostgreSQL 实现（支持 SchemaAwareDatabase 接口）
- [sqlite.go](./internal/adapter/sqlite.go) - SQLite 实现
- [clickhouse.go](./internal/adapter/clickhouse.go) - ClickHouse 实现
- [kingbase.go](./internal/adapter/kingbase.go) - KingBase 实现（基于 PostgreSQL）
- [dm.go](./internal/adapter/dm.go) - 达梦数据库实现
- [mongodb.go](./internal/adapter/mongodb.go) - MongoDB 实现
- [mongodb.go](./internal/adapter/mongodb.go) - MongoDB 实现
- [gokb/](./internal/adapter/gokb/) - KingBase 驱动（本地模块）

**依赖**：
- `internal/model` - 数据模型定义
- `database/sql` - Go 标准 SQL 接口

**对外接口**：
```go
type DatabaseAdapter interface {
    // 连接管理
    Connect(config *model.ConnectionConfig) (*sql.DB, error)
    Close(db *sql.DB) error
    Ping(db *sql.DB) error

    // 元数据查询
    GetDatabases(db *sql.DB) ([]string, error)
    GetTables(db *sql.DB, database string) ([]model.TableInfo, error)
    GetTableSchema(db *sql.DB, database, table string) (*model.TableSchema, error)
    GetViews(db *sql.DB, database string) ([]model.TableInfo, error)
    GetIndexes(db *sql.DB, database, table string) ([]model.IndexInfo, error)
    GetProcedures(db *sql.DB, database string) ([]model.RoutineInfo, error)
    GetFunctions(db *sql.DB, database string) ([]model.RoutineInfo, error)
    GetViewDefinition(db *sql.DB, database, viewName string) (string, error)
    GetRoutineDefinition(db *sql.DB, database, routineName, routineType string) (string, error)

    // SQL 执行
    Execute(db *sql.DB, query string, args ...interface{}) (*model.ExecuteResult, error)
    Query(db *sql.DB, query string, opts *model.QueryOptions) (*model.QueryResult, error)

    // 数据编辑
    Insert(db *sql.DB, database, table string, data map[string]interface{}) error
    Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error
    Delete(db *sql.DB, database, table, where string) error

    // 导出
    ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *model.CSVOptions) error
    ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error

    // 建表语句生成
    GetCreateTableSQL(db *sql.DB, database, table string) (string, error)

    // 表结构修改
    AlterTable(db *sql.DB, request *model.AlterTableRequest) error
    RenameTable(db *sql.DB, database, oldName, newName string) error
}

// SchemaAwareDatabase 支持 schema 的数据库接口（PostgreSQL、KingBase）
type SchemaAwareDatabase interface {
    GetSchemas(db *sql.DB, database string) ([]string, error)
    GetTablesWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error)
    GetTableSchemaWithSchema(db *sql.DB, database, schema, table string) (*model.TableSchema, error)
    GetViewsWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error)
    GetProceduresWithSchema(db *sql.DB, database, schema string) ([]model.RoutineInfo, error)
    GetFunctionsWithSchema(db *sql.DB, database, schema string) ([]model.RoutineInfo, error)
    GetViewDefinitionWithSchema(db *sql.DB, database, schema, viewName string) (string, error)
    GetRoutineDefinitionWithSchema(db *sql.DB, database, schema, routineName, routineType string) (string, error)
}
```

---

### 2. connection 模块

**位置**：[internal/connection/](./internal/connection/)

**功能**：连接管理器，负责连接池、配置持久化、密码加密

**核心文件**：
- [manager.go](./internal/connection/manager.go) - 连接管理器实现（并发安全）
- [crypto.go](./internal/connection/crypto.go) - AES-256-GCM 加密
- [errors.go](./internal/connection/errors.go) - 错误定义

**依赖**：
- `sync` - 并发安全（RWMutex）
- `encoding/json` - 配置序列化
- `os` - 文件系统操作
- `golang.org/x/crypto` - AES-256-GCM 加密

**数据存储**：
- 配置文件：`~/.dbm/connections.json`
- 分组文件：`~/.dbm/groups.json`
- 加密密钥：`~/.dbm/.key`

---

### 3. service 模块

**位置**：[internal/service/](./internal/service/)

**功能**：业务服务层，协调适配器和连接管理器

**核心文件**：
- [connection.go](./internal/service/connection.go) - 连接服务（创建、测试、关闭）
- [database.go](./internal/service/database.go) - 数据库服务（元数据、SQL 执行）

**依赖**：
- `internal/adapter` - 数据库适配器
- `internal/connection` - 连接管理器
- `internal/model` - 数据模型

---

### 4. server 模块

**位置**：[internal/server/](./internal/server/)

**功能**：HTTP 服务器，提供 RESTful API

**核心文件**：
- [handler.go](./internal/server/handler.go) - 路由与处理器、CORS 中间件、Prometheus 集成

**依赖**：
- `github.com/gin-gonic/gin` - HTTP 框架
- `internal/service` - 业务服务层
- `internal/connection` - 连接管理器
- `internal/monitor` - Prometheus 监控
- `internal/assets` - 嵌入式前端资源

**API 基础路径**：`/api/v1`

---

### 5. export 模块

**位置**：[internal/export/](./internal/export/)

**功能**：导出引擎，支持 CSV、SQL 格式导出，含类型映射功能

**核心文件**：
- [csv.go](./internal/export/csv.go) - CSV 导出器
- [sql.go](./internal/export/sql.go) - SQL 导出器（INSERT 语句）
- [type_mapper.go](./internal/export/type_mapper.go) - 类型映射器（跨数据库迁移）

**依赖**：
- `internal/adapter` - 数据库适配器接口
- `encoding/csv` - CSV 编码
- `gopkg.in/yaml.v3` - YAML 配置解析

---

### 6. model 模块

**位置**：[internal/model/](./internal/model/)

**功能**：数据模型定义

**核心文件**：
- [connection.go](./internal/model/connection.go) - 连接配置模型、DatabaseType 枚举
- [database.go](./internal/model/database.go) - 数据库元数据模型、AlterTable 请求定义
- [group.go](./internal/model/group.go) - 分组模型

**关键类型**：
```go
// DatabaseType 数据库类型
type DatabaseType string

const (
    DatabaseMySQL      DatabaseType = "mysql"
    DatabasePostgreSQL DatabaseType = "postgresql"
    DatabaseSQLite     DatabaseType = "sqlite"
    DatabaseClickHouse DatabaseType = "clickhouse"
    DatabaseKingBase   DatabaseType = "kingbase"
    DatabaseDM         DatabaseType = "dm"
    DatabaseMongoDB    DatabaseType = "mongodb"
)

// RoutineInfo 存储过程与函数信息
type RoutineInfo struct {
    Name     string `json:"name"`
    Type     string `json:"type"` // PROCEDURE or FUNCTION
    Database string `json:"database"`
    Schema   string `json:"schema"`
    Comment  string `json:"comment"`
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
)
```

**依赖**：
- `time` - 时间戳
- `database/sql` - 数据库连接

---

### 7. monitor 模块

**位置**：[internal/monitor/](./internal/monitor/)

**功能**：Prometheus 指标采集

**核心文件**：
- [init.go](./internal/monitor/init.go) - 监控初始化、数据库支持注册、配置 embed
- [collector.go](./internal/monitor/collector.go) - Prometheus Collector 实现
- [scraper.go](./internal/monitor/scraper.go) - 数据库指标采集接口
- [default_scraper.go](./internal/monitor/default_scraper.go) - 默认指标采集实现
- [common.go](./internal/monitor/common.go) - 监控公共函数

**配置文件**（通过 Go embed 嵌入）：
- [config/mysql.yaml](./internal/monitor/config/mysql.yaml) - MySQL 监控指标配置
- [config/pg.yaml](./internal/monitor/config/pg.yaml) - PostgreSQL 监控指标配置
- [config/kingbase.yaml](./internal/monitor/config/kingbase.yaml) - KingBase 监控指标配置
- [config/dm.yaml](./internal/monitor/config/dm.yaml) - 达梦数据库监控指标配置

**支持的数据库监控**：
- MySQL（运行时长、会话连接、最大连接数）
- PostgreSQL（运行时长、会话、连接、数据库状态、缓存命中率）
- KingBase（基于 PostgreSQL 内核，含特有缓存命中率指标）
- 达梦 DM（运行时长、会话连接、最大连接数）

**添加新数据库监控**：
1. 在 `config/` 目录添加 YAML 配置文件
2. 在 `init.go` 的 `dbTypeConfigFile` 映射表中注册
3. 调用 `InitAllScrapers()` 或 `InitMetricsForTypes()` 初始化

**依赖**：
- `embed` - Go 嵌入文件系统（用于 config/*.yaml）
- `prometheus/client_golang` - Prometheus 客户端
- `internal/adapter` - 数据库适配器

---

### 8. assets 模块

**位置**：[internal/assets/](./internal/assets/)

**功能**：嵌入式前端资源，使用 Go embed

**核心文件**：
- [assets.go](./internal/assets/assets.go) - embed 文件系统

**依赖**：
- `embed` - Go 嵌入文件系统
- `net/http` - HTTP 文件系统

---

### 9. web 模块

**位置**：[web/](./web/)

**功能**：Vue 3 前端项目

**核心目录**：
- `src/views/` - 页面组件
  - [connections.vue](./web/src/views/connections.vue) - 连接管理页面
  - [query.vue](./web/src/views/query.vue) - SQL 查询页面
  - [tables.vue](./web/src/views/tables.vue) - 表数据浏览页面
  - [schema-editor.vue](./web/src/views/schema-editor.vue) - 表结构编辑页面
  - [export.vue](./web/src/views/export.vue) - 数据导出页面
  - [monitor.vue](./web/src/views/monitor.vue) - 监控面板页面
- `src/components/` - 可复用组件
  - [TypeMappingDialog.vue](./web/src/components/TypeMappingDialog.vue) - 类型映射对话框
- `src/stores/` - Pinia 状态管理
  - [connections.ts](./web/src/stores/connections.ts) - 连接状态
  - [query.ts](./web/src/stores/query.ts) - 查询状态
- `src/router/` - Vue Router 配置
- `src/api/` - API 客户端
- `src/types/` - TypeScript 类型定义

**依赖**：
- Vue 3.4+
- Element Plus 2.5+
- Monaco Editor 0.45+
- Pinia 2.1+
- Axios 1.6+
- ECharts 5.5+

---

## 依赖关系图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              cmd/dbm/main.go                           │
│                    (程序入口、配置初始化、密钥生成)                     │
└─────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ↓
┌─────────────────────────────────────────────────────────────────────────┐
│                         internal/server/handler.go                     │
│                   (HTTP 服务器、Gin 路由、中间件、CORS)                │
└─────────────────────────────────────────────────────────────────────────┘
                    │                           │
        ┌───────────┴───────────┐   ┌───────────┴───────────┐
        ↓                       ↓   ↓                       ↓
┌──────────────────┐   ┌──────────────────┐   ┌─────────────────────┐
│ internal/service │   │ internal/monitor │   │ internal/assets/    │
│                  │   │                  │   │ (前端资源 embed)    │
├──────────────────┤   ├──────────────────┤   └─────────────────────┘
│ connection.go    │   │ collector.go     │
│ database.go      │   │ scraper.go       │
└──────────────────┘   └──────────────────┘
        │                       │
        ↓                       │
┌───────────────────┐           │
│ internal/adapter/ │◄──────────┘
│                   │
├───────────────────┤     ┌─────────────────────┐
│ factory.go        │────►│ internal/connection/│
├───────────────────┤     │                     │
│ mysql.go          │     │ manager.go          │
│ postgresql.go     │     │ crypto.go           │
│ sqlite.go         │     └─────────────────────┘
│ clickhouse.go     │              │
│ kingbase.go       │              ↓
│ dm.go             │     ┌─────────────────────┐
└───────────────────┘     │ internal/model/     │
         │                 │                     │
         │                 │ connection.go      │
         ↓                 │ database.go         │
┌──────────────────┐       │ group.go            │
│ internal/export/ │       └─────────────────────┘
│                  │
│ csv.go           │
│ sql.go           │
│ type_mapper.go   │
└──────────────────┘
```

---

## API 路由

### 基础路径

```
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
| POST | /connections/test         | 测试连接配置（未保存） |
| POST | /connections/:id/test     | 测试已保存连接      |

### 数据库元数据

| 方法  | 路径                               | 描述                   |
|------|-----------------------------------|----------------------|
| GET | /connections/:id/databases              | 获取数据库列表 |
| GET | /connections/:id/schemas                | 获取 schema 列表（PostgreSQL、KingBase） |
| GET | /connections/:id/tables                 | 获取表列表    |
| GET | /connections/:id/tables/:table/schema  | 获取表结构    |
| GET | /connections/:id/views                  | 获取视图列表  |
| GET | /connections/:id/views/:view/definition | 获取视图定义 |
| GET | /connections/:id/procedures             | 获取存储过程列表 |
| GET | /connections/:id/functions              | 获取函数列表 |
| GET | /connections/:id/routines/:routine/definition | 获取存储过程或函数定义 |

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
        "nullable": false
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
    }
  ]
}
```

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

## 启动编译参数

### 环境要求

- Go 1.24+
- Node.js 18+
- npm 或 yarn
- Make (可选)

### 快速启动

```bash
# 开发模式（后端热重载）
make dev

# 启动前端开发服务器
make dev-web

# 完整构建
make build
```

### 构建命令

| 命令              | 说明                                       |
|-------------------|-------------------------------------------|
| `make build` | 构建当前平台 |
| `make build-web` | 仅构建前端 |
| `make build-all` | 构建所有平台（Linux/macOS/Windows） |
| `make build-linux` | 构建 Linux 版本（amd64、arm64） |
| `make build-darwin` | 构建 macOS 版本（amd64、arm64） |
| `make build-windows` | 构建 Windows 版本（amd64） |
| `make clean` | 清理构建产物 |
| `make test`  | 运行所有测试 |
| `make lint`  | 运行 golangci-lint |

### 前端开发

```bash
cd web
npm install
npm run dev    # 开发服务器
npm run build  # 生产构建
```

### 调试运行

```bash
# 停止旧进程、编译并运行服务器（日志输出到 server.log）
lsof -t -i:2048 | xargs kill || true && make build && ./dist/dbm > server.log 2>&1 &
```

### 服务器启动参数

```bash
./dbm [选项]

选项：
  --host string        监听地址 (默认 "0.0.0.0")
  --port int          监听端口 (默认 2048)
  --config string      配置文件路径
  --data string       数据目录路径 (默认 ~/.dbm)
  --version           显示版本信息
  --config-path       显示配置路径
```

---

## 快速定位

### 常用工具/库/API 位置表

| 功能/需求 | 位置 | 文件 |
|----------|------|------|
| 添加新数据库支持 | [internal/adapter/](./internal/adapter/) | 创建新适配器文件，实现 DatabaseAdapter 接口 |
| 添加数据库监控支持 | [internal/monitor/](./internal/monitor/) | 在 config/ 添加 YAML 配置，在 init.go 注册 |
| 修改 API 路由 | [internal/server/handler.go](./internal/server/handler.go) | setupRoutes() 方法 |
| 修改连接配置模型 | [internal/model/connection.go](./internal/model/connection.go) | ConnectionConfig 结构体 |
| 修改密码加密方式 | [internal/connection/crypto.go](./internal/connection/crypto.go) | Encryptor 结构体 |
| 添加新的导出格式 | [internal/export/](./internal/export/) | 创建新的导出器文件 |
| 修改类型映射规则 | [configs/type_mapping.yaml](./configs/type_mapping.yaml) | YAML 配置文件 |
| 修改表结构修改逻辑 | [internal/adapter/](./internal/adapter/) | 各适配器的 AlterTable 方法 |
| 修改前端 UI 组件 | [web/src/views/](./web/src/views/) | Vue 组件文件 |
| 修改 API 客户端 | [web/src/api/](./web/src/api/) | API 调用封装 |
| 修改构建配置 | [Makefile](./Makefile) | 构建目标定义 |
| 修改依赖版本 | [go.mod](./go.mod) | Go 依赖 |
| 修改前端依赖 | [web/package.json](./web/package.json) | npm 依赖 |

### 数据库适配器实现位置

| 数据库类型 | 适配器文件 | 驱动包 | 特殊说明 |
|-----------|-----------|--------|---------|
| MySQL | [internal/adapter/mysql.go](./internal/adapter/mysql.go) | go-sql-driver/mysql | 完整支持 |
| PostgreSQL | [internal/adapter/postgresql.go](./internal/adapter/postgresql.go) | lib/pq | 支持 SchemaAwareDatabase 接口 |
| SQLite | [internal/adapter/sqlite.go](./internal/adapter/sqlite.go) | mattn/go-sqlite3 | 部分功能受限（如 DROP COLUMN） |
| ClickHouse | [internal/adapter/clickhouse.go](./internal/adapter/clickhouse.go) | ClickHouse/clickhouse-go/v2 | 不支持传统索引 |
| KingBase | [internal/adapter/kingbase.go](./internal/adapter/kingbase.go) | kingbase.com/gokb (本地) | 基于 PostgreSQL 内核 |
| 达梦 DM | [internal/adapter/dm.go](./internal/adapter/dm.go) | gitee.com/chunanyong/dm | 语法类似 Oracle |
| MongoDB | [internal/adapter/mongodb.go](./internal/adapter/mongodb.go) | go.mongodb.org/mongo-driver/v2 | NoSQL 文档数据库 |

### 常见任务快速定位

| 任务 | 位置 | 说明 |
|------|------|------|
| 修改默认端口 | [cmd/dbm/main.go:26](./cmd/dbm/main.go#L26) | 修改 flag.Int("port", 2048, ...) |
| 添加新的 API 端点 | [internal/server/handler.go:82](./internal/server/handler.go#L82) | 在 setupRoutes() 中添加路由 |
| 修改连接池配置 | [internal/connection/manager.go](./internal/connection/manager.go) | 修改 SetMaxOpenConns 等参数 |
| 修改 CORS 配置 | [internal/server/handler.go](./internal/server/handler.go) | corsMiddleware() 函数 |
| 修改响应格式 | [internal/server/handler.go](./internal/server/handler.go) | APIResponse 结构体 |
| 前端开发服务器配置 | [web/vite.config.ts](./web/vite.config.ts) | Vite 配置文件 |
| 注册新数据库监控 | [internal/monitor/init.go](./internal/monitor/init.go) | dbTypeConfigFile 映射表 |

---

## 重要约定

1. **密码安全**：所有密码必须通过 `connection.Manager` 的加密方法处理后再存储
2. **连接管理**：使用 `connection.Manager` 管理所有数据库连接，不要直接创建连接
3. **适配器模式**：所有数据库操作都应通过适配器接口进行
4. **配置持久化**：连接配置保存在 `~/.dbm/connections.json`
5. **并发安全**：`connection.Manager` 已使用 `sync.RWMutex`，调用时无需额外加锁
6. **类型映射**：跨数据库导出时，使用 `configs/type_mapping.yaml` 配置类型转换规则

---

## AI 使用指引

### 添加新数据库支持

1. 在 `internal/model/connection.go` 添加新的数据库类型常量
   ```go
   const (
       // ...
       DatabaseNewDB DatabaseType = "newdb"
   )
   ```

2. 在 `internal/adapter/` 创建新的适配器文件，实现 `DatabaseAdapter` 接口
   ```go
   type NewDBAdapter struct {
       BaseAdapter
   }
   ```

3. 在 `internal/adapter/factory.go` 的 `CreateAdapter` 方法中添加新数据库的 case
   ```go
   case model.DatabaseNewDB:
       return &NewDBAdapter{BaseAdapter: *NewBaseAdapter()}, nil
   ```

4. 在 `SupportedTypes()` 方法中添加新类型

5. 如需本地驱动，使用 `replace` 指令（参考 KingBase）

### 添加数据库监控支持

1. 在 `internal/monitor/config/` 添加对应数据库的 YAML 配置文件
   ```yaml
   # 示例：newdb.yaml
   metrics:
     - context: uptime
       labels: ["version"]
       metricsdesc:
         seconds: "Gauge metric with uptime of database in seconds."
       metricstype:
         seconds: gauge
       request: "SELECT VERSION() AS version, UPTIME AS seconds"
   ```

2. 在 `internal/monitor/init.go` 的 `dbTypeConfigFile` 映射表中注册
   ```go
   var dbTypeConfigFile = map[string]string{
       // ...
       "newdb": "newdb.yaml",
   }
   ```

3. 使用默认采集器（自动解析 YAML 配置）
   ```go
   // 方式 1：初始化所有数据库监控
   monitor.InitAllScrapers(collector)

   // 方式 2：按需初始化
   monitor.InitMetricsForTypes(collector, []string{"newdb"})
   ```

4. 如需自定义采集逻辑，实现 `Scraper` 接口：
   ```go
   type CustomScraper struct {
       db *sql.DB
   }

   func (s *CustomScraper) Scrape(ctx context.Context) ([]Metric, error) {
       // 自定义采集逻辑
   }
   ```

### 添加新的导出格式

1. 在 `internal/export/` 创建新的导出器文件
2. 在 `DatabaseAdapter` 接口中添加新的导出方法
3. 在各适配器中实现新方法
4. 在 `server/handler.go` 添加对应的 API 路由

### 添加类型映射规则

1. 在 `configs/type_mapping.yaml` 中添加新的映射规则
2. 规则格式：`source_type_to_target_type`
3. 支持设置目标类型、安全降级、精度损失标记
4. 重启服务使配置生效

### 扩展 API

1. 在 `internal/server/handler.go` 中添加新的路由和处理函数
2. 使用统一的响应格式 `APIResponse`
3. 遵循 RESTful 设计原则

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
| gitee.com/chunanyong/dm            | 1.8.22  | 达梦数据库驱动   |
| go.mongodb.org/mongo-driver/v2     | 2.5.0   | MongoDB 驱动     |
| google/uuid                        | 1.6.0   | UUID 生成       |
| golang.org/x/crypto                | 0.47.0  | AES-256-GCM 加密 |
| gopkg.in/yaml.v3                    | 3.0.1   | YAML 配置解析   |
| prometheus/client_golang           | 1.23.2  | Prometheus 监控  |

### 前端主要依赖

| 库               | 版本  | 用途         |
|------------------|-------|------------|
| Vue.js           | 3.4+  | 前端框架    |
| Element Plus     | 2.5+  | UI 组件库   |
| Monaco Editor   | 0.45+ | SQL 编辑器  |
| Pinia            | 2.1+  | 状态管理    |
| Axios            | 1.6+  | HTTP 客户端 |
| ECharts          | 5.5+  | 图表组件    |
| sql-formatter    | 15.7+ | SQL 格式化  |

---

## 配置文件位置

| 平台 | 配置路径 |
|------|----------|
| Linux | `~/.dbm/` |
| macOS | `~/.dbm/` |
| Windows | `%APPDATA%/dbm/` |

### 存储文件

- `connections.json`：连接配置（密码已加密）
- `groups.json`：分组配置
- `.key`：密码加密密钥

---

## 相关文档

| 文档 | 路径 | 描述 |
|------|------|------|
| CLAUDE.md | [./CLAUDE.md](./CLAUDE.md) | AI 开发指南 |
| README.md | [./README.md](./README.md) | 项目说明文档 |
| DESIGN.md | [./docs/DESIGN.md](./docs/DESIGN.md) | 设计文档 |
| CHANGELOG.md | [./docs/CHANGELOG.md](./docs/CHANGELOG.md) | 变更日志 |

---

## 数据库特性差异

| 操作 | MySQL | PostgreSQL | SQLite | ClickHouse | KingBase | DM |
|------|-------|------------|--------|------------|----------|-----|
| 添加列 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 删除列 | ✅ | ✅ | ❌ 需重建表 | ✅ | ✅ | ✅ |
| 修改列 | ✅ | ✅ 需多条语句 | ❌ 需重建表 | ✅ | ✅ 需多条语句 | ✅ |
| 重命名列 | ✅ | ✅ | ✅ (3.25.0+) | ✅ | ✅ | ✅ |
| 添加索引 | ✅ | ✅ | ✅ | ❌ 使用 ORDER BY | ✅ | ✅ |
| 删除索引 | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ |
| 重命名表 | ✅ | ✅ | ✅ | ✅ (非复制表) | ✅ | ✅ |
| Schema 支持 | ❌ | ✅ | ❌ | ✅ | ✅ | ❌ |

---

**文档生成工具**: /project-index skill
**最后更新**: 2026-02-25