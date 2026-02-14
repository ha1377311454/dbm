# DBM 数据库管理工具 - 设计文档

> 最后更新：2026-02-14

---

## 一、项目背景

### 1.1 为什么做这个项目？

在日常开发和运维工作中，开发者和 DBA 需要管理多种类型的数据库（MySQL、PostgreSQL、SQLite、ClickHouse、KingBase 等）。现有的数据库管理工具存在以下问题：

- **多工具切换**：不同数据库需要使用不同的管理工具，切换成本高
- **部署复杂**：许多工具需要复杂的安装配置，无法快速部署
- **跨平台限制**：某些工具只支持特定操作系统
- **团队协作**：团队缺乏统一的数据库管理工具

DBM 旨在提供一个**统一、轻量、跨平台**的数据库管理解决方案。

### 1.2 目标用户

| 用户类型 | 使用场景 | 核心诉求 |
|---------|---------|---------|
| **开发者** | 日常开发调试、数据查询修改 | 快速、易用、多数据库支持 |
| **DBA** | 数据库维护、性能监控、数据迁移 | 安全、可靠、监控能力 |

---

## 二、系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              用户浏览器                                  │
│                         ┌──────────────────┐                            │
│                         │   Vue.js 前端    │                            │
│                         │  (嵌入式静态资源) │                            │
│                         └──────────────────┘                            │
└─────────────────────────────────────────────────────────────────────────┘
                                      │ HTTP/WebSocket
                                      ↓
┌─────────────────────────────────────────────────────────────────────────┐
│                        Go 可执行文件 (单文件部署)                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                      嵌入式 HTTP 服务器                            │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────┐  │  │
│  │  │   静态资源   │  │   API 路由   │  │   CORS/Recovery      │  │  │
│  │  │   (embed)    │  │   (gin)      │  │   中间件             │  │  │
│  │  └──────────────┘  └──────────────┘  └────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                   ↓                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        业务逻辑层                                   │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐      │  │
│  │  │连接管理器│  │ SQL引擎  │  │导出引擎  │  │ 类型映射器   │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────┘      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                   ↓                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        数据访问层                                   │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │              数据库适配器接口 (Database Adapter)              │  │  │
│  │  │  ┌───────┐ ┌───────┐ ┌───────┐ ┌────────┐ ┌──────┐          │  │  │
│  │  │  │ MySQL │ │ PG    │ │SQLite │ │ClickHouse│ │KingBase│  ... │  │  │
│  │  │  └───────┘ └───────┘ └───────┘ └────────┘ └──────┘          │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                      ↓
┌─────────────────────────────────────────────────────────────────────────┐
│    MySQL │ PostgreSQL │ SQLite │ ClickHouse │ KingBase │ MSSQL │ Oracle │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 分层架构说明

| 层级 | 职责 | 主要组件 |
|-----|------|---------|
| **前端层** | 用户交互界面 | Vue.js + Vite 构建的 SPA |
| **接口层** | HTTP/WebSocket 通信 | Go HTTP 服务器 + 静态资源嵌入 |
| **业务层** | 核心业务逻辑 | 连接管理、SQL 执行、导出、类型映射 |
| **适配层** | 数据库抽象 | 统一的数据库适配器接口 |
| **驱动层** | 具体数据库驱动 | 各数据库官方驱动 |

---

## 三、技术选型

### 3.1 后端技术栈

#### 核心框架

| 技术 | 版本 | 用途 | 选择理由 |
|-----|------|-----|---------|
| **Go** | 1.24+ | 后端语言 | 高性能、跨平台编译、单文件部署 |
| **Gin** | 1.10.0 | HTTP 框架 | 轻量、高性能、API 友好 |
| **golang.org/x/crypto** | 0.47.0 | 密码加密 | AES-256-GCM 加密 |

#### 数据库驱动

| 数据库 | Go 驱动 | 版本 | 说明 |
|-------|--------|------|------|
| MySQL | go-sql-driver/mysql | 1.9.3 | 官方推荐 |
| PostgreSQL | lib/pq | 1.11.2 | 标准驱动 |
| SQLite | mattn/go-sqlite3 | 1.14.34 | CGO，需交叉编译支持 |
| ClickHouse | ClickHouse/clickhouse-go/v2 | 2.43.0 | 官方驱动 |
| KingBase | kingbase.com/gokb | 1.0.0 | 本地模块（replace） |

#### 核心依赖库

| 库 | 用途 |
|-----|------|
| `embed` | 前端资源嵌入（Go 1.16+ 内置） |
| `crypto/aes` | 密码加密 |
| `database/sql` | 统一数据库接口 |
| `context` | 请求上下文与超时控制 |
| `google/uuid` | UUID 生成 |

### 3.2 前端技术栈

| 技术 | 版本 | 用途 |
|-----|------|-----|
| **Vue.js** | 3.4+ | 前端框架 |
| **TypeScript** | 5.3+ | 类型安全 |
| **Vite** | 5.0+ | 构建工具 |
| **Element Plus** | 2.5+ | UI 组件库 |
| **Monaco Editor** | 0.45+ | 代码编辑器 |
| **Pinia** | 2.1+ | 状态管理 |
| **Vue Router** | 4.2+ | 路由管理 |
| **Axios** | 1.6+ | HTTP 客户端 |
| **ECharts** | 5.5+ | 监控图表 |

---

## 四、核心模块设计

### 4.1 连接管理模块

#### 数据结构

```go
// 连接配置
type ConnectionConfig struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Type        DatabaseType      `json:"type"`
    Host        string            `json:"host"`
    Port        int               `json:"port"`
    Username    string            `json:"username"`
    Password    string            `json:"-"`           // 加密密码
    Database    string            `json:"database"`
    Params      map[string]string `json:"params"`
    GroupID     string            `json:"groupId"`      // 分组 ID
    CreatedAt   time.Time         `json:"createdAt"`
    UpdatedAt   time.Time         `json:"updatedAt"`
}

// 分组配置
type Group struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Color       string    `json:"color"`
    SortOrder   int       `json:"sortOrder"`
    CreatedAt   time.Time `json:"createdAt"`
}
```

#### 核心接口

```go
type ConnectionManager interface {
    // 连接管理
    AddConnection(config *ConnectionConfig) error
    RemoveConnection(id string) error
    GetConnection(id string) (*sql.DB, error)
    TestConnection(config *ConnectionConfig) (*TestResult, error)

    // 配置管理
    SaveConfig(config *ConnectionConfig) error
    LoadConfig(id string) (*ConnectionConfig, error)
    ListConfigs() ([]*ConnectionConfig, error)

    // 密码加密
    EncryptPassword(password string) (string, error)
    DecryptPassword(encrypted string) (string, error)

    // 分组管理
    AddGroup(group *Group) error
    UpdateGroup(group *Group) error
    RemoveGroup(id string) error
    ListGroups() ([]*Group, error)
}
```

### 4.2 数据库适配器模块

#### 适配器接口

```go
// 数据库类型
type DatabaseType string

const (
    DatabaseMySQL      DatabaseType = "mysql"
    DatabasePostgreSQL DatabaseType = "postgresql"
    DatabaseSQLite     DatabaseType = "sqlite"
    DatabaseClickHouse DatabaseType = "clickhouse"
    DatabaseKingBase   DatabaseType = "kingbase"
)

// 数据库适配器接口
type DatabaseAdapter interface {
    // 连接
    Connect(config *ConnectionConfig) (*sql.DB, error)

    // 元数据查询
    GetDatabases(db *sql.DB) ([]string, error)
    GetTables(db *sql.DB, database string) ([]TableInfo, error)
    GetTableSchema(db *sql.DB, database, table string) (*TableSchema, error)
    GetViews(db *sql.DB, database string) ([]ViewInfo, error)

    // SQL 执行
    Execute(db *sql.DB, query string) (*ExecuteResult, error)
    Query(db *sql.DB, query string, opts *QueryOptions) (*QueryResult, error)

    // 数据编辑
    Insert(db *sql.DB, database, table string, data map[string]interface{}) error
    Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error
    Delete(db *sql.DB, database, table, where string) error

    // 表结构修改
    AlterTable(db *sql.DB, req *AlterTableRequest) error
    RenameTable(db *sql.DB, database, oldName, newName string) error

    // 导出
    ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *CSVOptions) error
    ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *SQLOptions) error

    // 建表语句生成
    GetCreateTableSQL(db *sql.DB, database, table string) (string, error)
}

// Schema 扩展接口（PostgreSQL、KingBase 等）
type SchemaAwareDatabase interface {
    GetSchemas(db *sql.DB, database string) ([]string, error)
    GetTablesWithSchema(db *sql.DB, database, schema string) ([]TableInfo, error)
    GetTableSchemaWithSchema(db *sql.DB, database, schema, table string) (*TableSchema, error)
    GetViewsWithSchema(db *sql.DB, database, schema string) ([]ViewInfo, error)
}
```

### 4.3 导出引擎

```go
// CSV 导出选项
type CSVOptions struct {
    IncludeHeader    bool   `json:"includeHeader"`
    Separator        string `json:"separator"`
    Quote            string `json:"quote"`
    Encoding         string `json:"encoding"`
    NullValue        string `json:"nullValue"`
    DateFormat       string `json:"dateFormat"`
}

// SQL 导出选项
type SQLOptions struct {
    IncludeCreateTable bool   `json:"includeCreateTable"`
    IncludeDropTable   bool   `json:"includeDropTable"`
    BatchInsert        bool   `json:"batchInsert"`
    BatchSize          int    `json:"batchSize"`
    StructureOnly      bool   `json:"structureOnly"`
    Query              string `json:"query"`              // 自定义查询导出
}
```

### 4.4 类型映射引擎

#### 核心功能

类型映射引擎用于跨数据库迁移时的类型转换，确保不同数据库之间的数据类型兼容性。

```go
// TypeMapper 类型映射器
type TypeMapper struct {
    mappings map[string]*TypeMapping  // key: "mysql_to_postgresql"
}

// TypeMappingResult 类型映射结果
type TypeMappingResult struct {
    Success        bool              `json:"success"`
    Mapped         map[string]string `json:"mapped"`
    Warnings       []string          `json:"warnings"`
    RequiresUser   map[string]TypeRule `json:"requiresUser"`
    Summary        TypeSummary       `json:"summary"`
}

// TypeSummary 映射摘要
type TypeSummary struct {
    Total       int `json:"total"`
    Direct      int `json:"direct"`       // 直接映射
    Fallback    int `json:"fallback"`     // 安全降级
    UserChoice  int `json:"userChoice"`   // 需要用户选择
    LossyCount  int `json:"lossyCount"`   // 有精度损失
}
```

#### 配置文件结构

类型映射规则存储在 `configs/type_mapping.yaml`：

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
    ENUM:
      target: "TEXT"
      requires_user: true
      user_options:
        - label: "转换为 TEXT"
          value: "TEXT"
        - label: "转换为 CHECK 约束"
          value: "CHECK_CONSTRAINT"
```

### 4.5 表结构修改引擎

```go
// AlterTableRequest 表结构修改请求
type AlterTableRequest struct {
    Database string         `json:"database"`
    Table    string         `json:"table"`
    Actions  []AlterAction  `json:"actions"`
}

// AlterActionType 操作类型
type AlterActionType string

const (
    AddColumn    AlterActionType = "ADD_COLUMN"
    DropColumn   AlterActionType = "DROP_COLUMN"
    ModifyColumn AlterActionType = "MODIFY_COLUMN"
    RenameColumn AlterActionType = "RENAME_COLUMN"
    AddIndex     AlterActionType = "ADD_INDEX"
    DropIndex    AlterActionType = "DROP_INDEX"
)

// AlterAction 修改操作
type AlterAction struct {
    Type    AlterActionType `json:"type"`
    Column  *ColumnInfo     `json:"column,omitempty"`
    Index   *IndexInfo      `json:"index,omitempty"`
    OldName string          `json:"oldName,omitempty"`
    NewName string          `json:"newName,omitempty"`
}
```

#### 数据库特性差异

| 操作 | MySQL | PostgreSQL | SQLite | ClickHouse | KingBase |
|------|-------|------------|--------|------------|----------|
| 添加列 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 删除列 | ✅ | ✅ | ❌ 需重建表 | ✅ | ✅ |
| 修改列 | ✅ | ✅ 需多条语句 | ❌ 需重建表 | ✅ | ✅ 需多条语句 |
| 重命名列 | ✅ | ✅ | ✅ (3.25.0+) | ✅ | ✅ |
| 添加索引 | ✅ | ✅ | ✅ | ❌ 使用 ORDER BY | ✅ |
| 删除索引 | ✅ | ✅ | ✅ | ❌ | ✅ |
| 重命名表 | ✅ | ✅ | ✅ | ✅ (非复制表) | ✅ |

---

## 五、API 设计

### 5.1 RESTful API 规范

```
BASE_URL: /api/v1
```

#### 连接管理

| 方法 | 路径 | 描述 |
|-----|------|-----|
| GET | /connections | 获取连接列表 |
| POST | /connections | 创建连接 |
| PUT | /connections/:id | 更新连接 |
| DELETE | /connections/:id | 删除连接 |
| POST | /connections/:id/test | 测试连接 |

#### 数据库元数据

| 方法 | 路径 | 描述 |
|-----|------|-----|
| GET | /connections/:id/databases | 获取数据库列表 |
| GET | /connections/:id/schemas | 获取 schema 列表 |
| GET | /connections/:id/tables | 获取表列表 |
| GET | /connections/:id/tables/:table/schema | 获取表结构 |
| GET | /connections/:id/views | 获取视图列表 |

#### SQL 执行

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/query | 执行查询 |
| POST | /connections/:id/execute | 执行非查询 SQL |

#### 数据编辑

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/tables/:table/data | 创建数据 |
| PUT | /connections/:id/tables/:table/data | 更新数据 |
| DELETE | /connections/:id/tables/:table/data | 删除数据 |

#### 表结构修改

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/tables/:table/alter | 修改表结构 |
| POST | /connections/:id/tables/:table/rename | 重命名表 |

#### 数据导出

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/export/csv | CSV 导出 |
| POST | /connections/:id/export/sql | SQL 导出 |
| POST | /connections/:id/export/sql/preview | SQL 导出类型映射预览 |

#### 分组管理

| 方法 | 路径 | 描述 |
|-----|------|-----|
| GET | /groups | 获取分组列表 |
| POST | /groups | 创建分组 |
| PUT | /groups/:id | 更新分组 |
| DELETE | /groups/:id | 删除分组 |

#### 监控

| 方法 | 路径 | 描述 |
|-----|------|-----|
| GET | /metrics | Prometheus 指标 |
| GET | /api/v1/monitor/stats | 监控统计 |

### 5.2 API 响应格式

```go
// 统一响应格式
type APIResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

---

## 六、安全设计

### 6.1 密码加密

使用 AES-256-GCM 加密算法存储数据库密码：

- 密钥存储位置：`~/.dbm/.key`
- 加密模式：GCM（带认证的加密）
- 密钥长度：256 位

### 6.2 密钥管理

- 首次启动时自动生成密钥
- 支持环境变量 `DBM_ENCRYPTION_KEY` 覆盖
- 密钥文件权限设置为仅用户可读写

---

## 七、部署设计

### 7.1 单文件打包

```
dbm
├── 嵌入的前端资源 (embed)
└── Go 可执行代码
```

### 7.2 前端嵌入

```go
//go:embed all:dist
var frontendFS embed.FS

func main() {
    httpFS, _ := fs.Sub(frontendFS, "dist")
    http.FileServer(http.FS(httpFS))
}
```

### 7.3 跨平台编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o dbm-linux-amd64

# macOS
GOOS=darwin GOARCH=amd64 go build -o dbm-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o dbm-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o dbm-windows-amd64.exe
```

### 7.4 启动参数

```bash
dbm [命令] [参数]

命令:
  serve      启动 Web 服务 (默认)
  version    显示版本信息
  config     显示配置路径

参数:
  --host     监听地址 (默认: 0.0.0.0)
  --port     监听端口 (默认: 2048)
  --config   配置文件路径
  --data     数据目录路径
```

---

## 八、目录结构

```
dbm/
├── cmd/
│   └── dbm/
│       └── main.go              # 程序入口
├── internal/
│   ├── adapter/                 # 数据库适配器
│   │   ├── adapter.go          # 接口定义
│   │   ├── factory.go          # 适配器工厂
│   │   ├── base.go             # 基础适配器
│   │   ├── mysql.go            # MySQL 适配器
│   │   ├── postgresql.go       # PostgreSQL 适配器
│   │   ├── sqlite.go           # SQLite 适配器
│   │   ├── clickhouse.go       # ClickHouse 适配器
│   │   ├── kingbase.go         # KingBase 适配器
│   │   └── gokb/               # KingBase 驱动（本地模块）
│   ├── connection/              # 连接管理
│   │   ├── manager.go          # 连接管理器
│   │   ├── crypto.go           # 密码加密
│   │   └── errors.go           # 错误定义
│   ├── export/                  # 导出引擎
│   │   ├── csv.go              # CSV 导出
│   │   ├── sql.go              # SQL 导出
│   │   └── type_mapper.go      # 类型映射
│   ├── model/                   # 数据模型
│   │   ├── connection.go       # 连接配置
│   │   ├── database.go         # 数据库元数据
│   │   ├── export.go           # 导出模型
│   │   └── group.go            # 分组模型
│   ├── server/                  # HTTP 服务器
│   │   └── handler.go          # 路由与处理器
│   ├── service/                 # 业务服务层
│   │   ├── connection.go       # 连接服务
│   │   └── database.go         # 数据库服务
│   └── assets/                  # 嵌入的前端资源
│       └── assets.go           # embed 文件系统
├── web/                         # 前端项目
│   ├── src/
│   │   ├── components/         # 组件
│   │   ├── views/              # 页面
│   │   ├── stores/             # Pinia stores
│   │   ├── router/             # Vue Router
│   │   ├── api/                # API 客户端
│   │   └── main.ts             # 入口文件
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
├── configs/
│   ├── config.example.yaml     # 配置示例
│   └── type_mapping.yaml       # 类型映射配置
├── scripts/
│   └── build.sh                # 构建脚本
├── docs/
│   ├── DESIGN.md               # 设计文档（本文件）
│   └── CHANGELOG.md            # 变更日志
├── Makefile                     # 构建命令
├── CLAUDE.md                    # AI 开发指南
├── go.mod
├── go.sum
└── README.md                    # 项目主页
```

---

## 九、已实现功能清单

### V1.0 - 基础版本（已完成）

#### 连接管理
- [x] 多数据库连接配置
- [x] 密码 AES-256-GCM 加密存储
- [x] 连接测试功能
- [x] 连接分组管理

#### 数据库支持
- [x] MySQL 支持（5.7+, 8.0+）
- [x] PostgreSQL 支持（12+, 14+, 15+）
- [x] SQLite 支持（3.x）
- [x] ClickHouse 支持（22.3+）
- [x] KingBase 支持（ES V8）

#### SQL 功能
- [x] SQL 编辑器（Monaco Editor）
- [x] 语法高亮
- [x] SQL 格式化
- [x] 自动补全
- [x] 查询执行
- [x] 结果分页展示
- [x] 非查询 SQL 执行

#### 数据编辑
- [x] 表结构查看
- [x] 数据浏览
- [x] 在线编辑
- [x] INSERT/UPDATE/DELETE 操作

#### 表结构管理
- [x] 添加列
- [x] 删除列
- [x] 修改列
- [x] 重命名列
- [x] 管理索引
- [x] 重命名表

#### 数据导出
- [x] CSV 导出
- [x] SQL 导出
- [x] 类型映射功能
- [x] SQL 导出预览 API

#### Web 界面
- [x] Vue 3 + Vite 构建
- [x] Element Plus UI 组件
- [x] 响应式设计
- [x] 单文件部署

---

## 十、未实现功能清单

### V1.1 - 功能增强（计划中）

- [ ] SQL Server 支持
- [ ] Oracle 支持
- [ ] 查询历史记录

### V1.2 - 监控与运维（计划中）

- [ ] 完整的 Prometheus 指标
- [ ] Grafana 监控面板模板
- [ ] 慢查询分析

---

## 十一、技术约束

| 约束项 | 说明 |
|-------|------|
| 后端语言 | Go 1.24+ |
| 前端框架 | Vue.js 3.4+ |
| 部署方式 | 单文件可执行程序（前端嵌入） |
| 监控协议 | Prometheus |
| 支持系统 | Linux、macOS、Windows |
| SQLite 交叉编译 | 需要 CGO，构建较复杂 |

---

## 十二、性能要求

| 指标 | 要求 |
|-----|------|
| 查询响应时间 | 简单查询 < 1s，复杂查询 < 5s |
| 并发连接数 | 支持同时管理 10+ 个连接 |
| 结果集支持 | 支持万行级数据流畅展示 |
| 导出性能 | 万行数据导出 < 10s |
| 内存占用 | 空载 < 100MB，正常使用 < 500MB |
| 启动时间 | < 3 秒 |

---

*文档版本：v2.0*
*最后更新：2026-02-14*
