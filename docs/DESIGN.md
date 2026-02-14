# DBM 数据库管理工具 - 设计文档

## 一、系统架构

### 1.1 整体架构图

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
│  │  │   静态资源   │  │   API 路由   │  │   WebSocket Hub        │  │  │
│  │  │   (embed)    │  │   (gin/chi)  │  │                        │  │  │
│  │  └──────────────┘  └──────────────┘  └────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                   ↓                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        业务逻辑层                                   │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐      │  │
│  │  │连接管理器│  │ SQL引擎  │  │导出引擎  │  │ 监控收集器   │      │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────────┘      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                   ↓                                     │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        数据访问层                                   │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │              数据库适配器接口 (Database Adapter)              │  │  │
│  │  │  ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐          │  │  │
│  │  │  │ MySQL │ │ PG    │ │SQLite │ │ MSSQL │ │Oracle │  ...     │  │  │
│  │  │  └───────┘ └───────┘ └───────┘ └───────┘ └───────┘          │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                      ↓
┌─────────────────────────────────────────────────────────────────────────┐
│              MySQL │ PostgreSQL │ SQLite │ SQL Server │ Oracle          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 分层架构说明

| 层级 | 职责 | 主要组件 |
|-----|------|---------|
| **前端层** | 用户交互界面 | Vue.js + Vite 构建的 SPA |
| **接口层** | HTTP/WebSocket 通信 | Go HTTP 服务器 + 静态资源嵌入 |
| **业务层** | 核心业务逻辑 | 连接管理、SQL 执行、导出、监控 |
| **适配层** | 数据库抽象 | 统一的数据库适配器接口 |
| **驱动层** | 具体数据库驱动 | 各数据库官方驱动 |

---

## 二、技术选型

### 2.1 后端技术栈

#### 核心框架

| 技术 | 版本 | 用途 | 选择理由 |
|-----|------|-----|---------|
| **Go** | 1.21+ | 后端语言 | 高性能、跨平台编译、单文件部署 |
| **Gin** | 1.9+ | HTTP 框架 | 轻量、高性能、API 友好 |
| **GORM** | 1.25+ | 本地数据访问 | 用于管理 DBM 自身的配置数据 |

#### 数据库驱动

| 数据库 | Go 驱动 | 说明 |
|-------|--------|------|
| MySQL | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | 官方推荐 |
| PostgreSQL | [lib/pq](https://github.com/lib/pq) | 标准驱动 |
| SQLite | [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | CGO，需交叉编译支持 |
| SQL Server | [microsoft/go-mssqldb](https://github.com/microsoft/go-mssqldb) | 微软官方 |
| Oracle | [sijms/go-ora](https://github.com/sijms/go-ora) | 纯 Go 实现 |

#### 核心依赖库

| 库 | 用途 |
|-----|------|
| `embed` | 前端资源嵌入（Go 1.16+ 内置） |
| `crypto/aes` | 密码加密 |
| `database/sql` | 统一数据库接口 |
| `context` | 请求上下文与超时控制 |
| `prometheus/client_golang` | 指标暴露 |

### 2.2 前端技术栈

| 技术 | 版本 | 用途 |
|-----|------|-----|
| **Vue.js** | 3.3+ | 前端框架 |
| **TypeScript** | 5.0+ | 类型安全 |
| **Vite** | 5.0+ | 构建工具 |
| **Element Plus** | 2.4+ | UI 组件库 |
| **Monaco Editor** | 0.45+ | 代码编辑器 |
| **Pinia** | 2.1+ | 状态管理 |
| **Vue Router** | 4.2+ | 路由管理 |
| **Axios** | 1.6+ | HTTP 客户端 |
| **ECharts** | 5.4+ | 监控图表 |

### 2.3 技术选型对比表

#### Web 框架选择

| 方案 | 性能 | 开发成本 | 路由功能 | 中间件 | 推荐 |
|-----|------|---------|---------|-------|------|
| Gin | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ✅ 推荐 |
| Chi | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | |
| Fiber | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | |
| Echo | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | |

**推荐：Gin** - 生态成熟，文档完善，性能优秀

#### SQL 编辑器选择

| 方案 | 功能量 | 包体积 | 性能 | 推荐 |
|-----|-------|-------|------|------|
| Monaco Editor | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | ✅ 推荐 |
| CodeMirror | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | |
| Ace Editor | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | |

**推荐：Monaco Editor** - VS Code 同款，功能强大

---

## 三、核心模块设计

### 3.1 连接管理模块

#### 数据结构

```go
// 连接配置
type ConnectionConfig struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`        // 连接名称
    Type        string            `json:"type"`        // 数据库类型
    Host        string            `json:"host"`        // 主机地址
    Port        int               `json:"port"`        // 端口
    Username    string            `json:"username"`    // 用户名
    Password    string            `json:"-"`           // 加密密码
    Database    string            `json:"database"`    // 数据库名
    Params      map[string]string `json:"params"`      // 额外参数
    CreatedAt   time.Time         `json:"createdAt"`
    UpdatedAt   time.Time         `json:"updatedAt"`
}

// 连接池管理
type ConnectionPool struct {
    mu          sync.RWMutex
    connections map[string]*sql.DB  // key: connectionID
    configs     map[string]*ConnectionConfig
}
```

#### 核心接口

```go
type ConnectionManager interface {
    // 连接管理
    AddConnection(config *ConnectionConfig) error
    RemoveConnection(id string) error
    GetConnection(id string) (*sql.DB, error)
    TestConnection(config *ConnectionConfig) error

    // 配置管理
    SaveConfig(config *ConnectionConfig) error
    LoadConfig(id string) (*ConnectionConfig, error)
    ListConfigs() ([]*ConnectionConfig, error)

    // 密码加密
    EncryptPassword(password string) (string, error)
    DecryptPassword(encrypted string) (string, error)
}
```

### 3.2 数据库适配器模块

#### 适配器接口

```go
// 数据库类型
type DatabaseType string

const (
    DatabaseMySQL      DatabaseType = "mysql"
    DatabasePostgreSQL DatabaseType = "postgresql"
    DatabaseSQLite     DatabaseType = "sqlite"
    DatabaseMSSQL      DatabaseType = "mssql"
    DatabaseOracle     DatabaseType = "oracle"
)

// 数据库适配器接口
type DatabaseAdapter interface {
    // 连接
    Connect(config *ConnectionConfig) (*sql.DB, error)

    // 元数据查询
    GetDatabases() ([]string, error)
    GetTables(database string) ([]TableInfo, error)
    GetTableSchema(database, table string) (*TableSchema, error)
    GetViews(database string) ([]ViewInfo, error)
    GetIndexes(database, table string) ([]IndexInfo, error)

    // SQL 执行
    Execute(query string, args ...interface{}) (*ExecuteResult, error)
    Query(query string, args ...interface{}) (*QueryResult, error)

    // 导出
    ExportToCSV(writer io.Writer, query string, opts *CSVOptions) error
    ExportToSQL(writer io.Writer, tables []string, opts *SQLOptions) error

    // 建表语句生成
    GetCreateTableSQL(database, table string) (string, error)
}

// 适配器工厂
type AdapterFactory interface {
    CreateAdapter(dbType DatabaseType) (DatabaseAdapter, error)
    SupportedTypes() []DatabaseType
}
```

#### 元数据结构

```go
// 表信息
type TableInfo struct {
    Name        string       `json:"name"`
    Database    string       `json:"database"`
    Schema      string       `json:"schema"`
    TableType   string       `json:"tableType"`   // BASE TABLE, VIEW
    Rows        int64        `json:"rows"`
    Size        int64        `json:"size"`
    Comment     string       `json:"comment"`
}

// 表结构
type TableSchema struct {
    Database    string           `json:"database"`
    Table       string           `json:"table"`
    Columns     []ColumnInfo    `json:"columns"`
    Indexes     []IndexInfo      `json:"indexes"`
    Constraints []ConstraintInfo `json:"constraints"`
}

// 列信息
type ColumnInfo struct {
    Name         string `json:"name"`
    Type         string `json:"type"`
    Nullable     bool   `json:"nullable"`
    DefaultValue string `json:"defaultValue"`
    Key          string `json:"key"`          // PRI, UNI, MUL
    Extra        string `json:"extra"`        // auto_increment
    Comment      string `json:"comment"`
}
```

### 3.3 SQL 执行引擎

```go
// SQL 执行引擎
type SQLEngine struct {
    adapter DatabaseAdapter
    timeout time.Duration
}

// 执行结果
type ExecuteResult struct {
    RowsAffected int64         `json:"rowsAffected"`
    TimeCost     time.Duration `json:"timeCost"`
    Message      string        `json:"message"`
}

// 查询结果
type QueryResult struct {
    Columns []string                   `json:"columns"`
    Rows    []map[string]interface{}  `json:"rows"`
    Total   int64                      `json:"total"`
    TimeCost time.Duration             `json:"timeCost"`
}

// 分页查询
type QueryOptions struct {
    Page     int    `json:"page"`
    PageSize int    `json:"pageSize"`
    SortBy   string `json:"sortBy"`
    SortDesc bool   `json:"sortDesc"`
}
```

### 3.4 导出引擎

```go
// 导出引擎
type ExportEngine struct {
    adapter DatabaseAdapter
}

// CSV 导出选项
type CSVOptions struct {
    IncludeHeader    bool   `json:"includeHeader"`     // 包含表头
    Separator        string `json:"separator"`         // 分隔符
    Quote            string `json:"quote"`             // 引号字符
    Encoding         string `json:"encoding"`          // 编码
    NullValue        string `json:"nullValue"`         // NULL 值表示
    DateFormat       string `json:"dateFormat"`        // 日期格式
}

// SQL 导出选项
type SQLOptions struct {
    IncludeCreateTable bool   `json:"includeCreateTable"` // 包含建表语句
    IncludeDropTable   bool   `json:"includeDropTable"`   // 包含 DROP 语句
    BatchInsert        bool   `json:"batchInsert"`        // 批量 INSERT
    BatchSize          int    `json:"batchSize"`          // 批量大小
    StructureOnly      bool   `json:"structureOnly"`      // 仅结构
}
```

### 3.5 监控模块

```go
// 监控指标
type Metrics struct {
    // 连接指标
    ActiveConnections   prometheus.Gauge
    IdleConnections     prometheus.Gauge
    ConnectionErrors    prometheus.Counter

    // 查询指标
    QueryTotal          prometheus.Counter
    QueryDuration       prometheus.Histogram
    QueryErrors         prometheus.Counter
    SlowQueries         prometheus.Counter

    // 导出指标
    ExportTotal         prometheus.Counter
    ExportDuration      prometheus.Histogram
    ExportRows          prometheus.Counter

    // 系统指标
    MemoryUsage         prometheus.Gauge
    CPUUsage            prometheus.Gauge
}

// 慢查询阈值
const SlowQueryThreshold = 3 * time.Second
```

---

## 四、数据库设计

### 4.1 本地存储设计

DBM 使用 SQLite 存储自身的配置数据：

```sql
-- 连接配置表
CREATE TABLE connections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    host        TEXT,
    port        INTEGER,
    username    TEXT,
    password    TEXT NOT NULL,  -- AES 加密
    database    TEXT,
    params      TEXT,           -- JSON 格式
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 查询历史表
CREATE TABLE query_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    connection_id TEXT,
    query       TEXT NOT NULL,
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    duration    INTEGER,         -- 执行耗时(ms)
    rows        INTEGER,        -- 返回行数
    success     BOOLEAN,
    FOREIGN KEY (connection_id) REFERENCES connections(id)
);

-- 导出历史表
CREATE TABLE export_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    connection_id TEXT,
    export_type TEXT,           -- csv/sql
    tables      TEXT,           -- JSON 数组
    file_path   TEXT,
    rows        INTEGER,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (connection_id) REFERENCES connections(id)
);

-- 创建索引
CREATE INDEX idx_query_history_connection ON query_history(connection_id);
CREATE INDEX idx_query_history_time ON query_history(executed_at DESC);
CREATE INDEX idx_export_history_connection ON export_history(connection_id);
```

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
| GET | /connections/:id/tables | 获取表列表 |
| GET | /connections/:id/tables/:table/schema | 获取表结构 |
| GET | /connections/:id/views | 获取视图列表 |
| GET | /connections/:id/indexes/:table | 获取索引列表 |

#### SQL 执行

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/query | 执行查询 |
| POST | /connections/:id/execute | 执行非查询 SQL |

#### 数据导出

| 方法 | 路径 | 描述 |
|-----|------|-----|
| POST | /connections/:id/export/csv | CSV 导出 |
| POST | /connections/:id/export/sql | SQL 导出 |
| GET | /exports/:id/download | 下载导出文件 |

#### 监控

| 方法 | 路径 | 描述 |
|-----|------|-----|
| GET | /metrics | Prometheus 指标 |
| GET | /api/v1/monitor/connections | 连接状态 |
| GET | /api/v1/monitor/queries | 查询统计 |

### 5.2 API 响应格式

```go
// 统一响应格式
type APIResponse struct {
    Code    int         `json:"code"`    // 0=成功, 非0=错误
    Message string      `json:"message"` // 提示信息
    Data    interface{} `json:"data"`    // 数据
}

// 分页响应
type PaginatedResponse struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
    Total   int64       `json:"total"`
    Page    int         `json:"page"`
    PageSize int        `json:"pageSize"`
}
```

---

## 六、安全设计

### 6.1 密码加密

```go
// 使用 AES-256-GCM 加密密码
type PasswordEncryptor struct {
    key []byte  // 32 字节密钥
}

// 加密
func (e *PasswordEncryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

### 6.2 密钥管理

- 密钥存储在用户主目录配置文件中
- 支持环境变量 `DBM_ENCRYPTION_KEY` 覆盖
- 首次启动时自动生成密钥

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
    // 从嵌入的 FS 提供静态文件
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
dbm [命令]

命令:
  serve      启动 Web 服务 (默认)
  version    显示版本信息
  config     显示配置路径

参数:
  --host     监听地址 (默认: 0.0.0.0)
  --port     监听端口 (默认: 8080)
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
│   │   ├── mysql.go
│   │   ├── postgresql.go
│   │   ├── sqlite.go
│   │   ├── mssql.go
│   │   └── oracle.go
│   ├── connection/              # 连接管理
│   │   ├── manager.go
│   │   ├── pool.go
│   │   └── crypto.go
│   ├── engine/                  # SQL 执行引擎
│   │   ├── executor.go
│   │   └── query.go
│   ├── export/                  # 导出引擎
│   │   ├── csv.go
│   │   ├── sql.go
│   │   └── progress.go
│   ├── monitor/                 # 监控模块
│   │   ├── metrics.go
│   │   └── collector.go
│   ├── model/                   # 数据模型
│   │   ├── connection.go
│   │   └── database.go
│   ├── repository/              # 数据访问
│   │   └── sqlite.go
│   └── server/                  # HTTP 服务器
│       ├── router.go
│       ├── handler.go
│       └── middleware.go
├── web/                         # 前端项目
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   ├── stores/
│   │   ├── router/
│   │   ├── api/
│   │   └── main.ts
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
├── configs/
│   └── config.example.yaml
├── scripts/
│   └── build.sh                 # 构建脚本
├── docs/
│   ├── REQUIREMENTS.md
│   └── DESIGN.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 九、开发计划

### Phase 1: 基础框架 (Week 1-2)

- [ ] 项目初始化
- [ ] 目录结构搭建
- [ ] 前端 Vue 项目初始化
- [ ] HTTP 服务器框架搭建
- [ ] 静态资源嵌入

### Phase 2: 核心功能 (Week 3-4)

- [ ] 连接管理模块
- [ ] MySQL/PostgreSQL 适配器
- [ ] SQL 编辑与执行
- [ ] 结果展示与分页

### Phase 3: 数据编辑 (Week 5)

- [ ] 数据编辑功能
- [ ] 增删改操作
- [ ] 事务支持

### Phase 4: 导出功能 (Week 6)

- [ ] CSV 导出
- [ ] SQL 导出
- [ ] 批量导出

### Phase 5: 可视化构建器 (Week 7-8)

- [ ] 可视化查询构建器
- [ ] 表关联配置
- [ ] SQL 预览

### Phase 6: 监控功能 (Week 9)

- [ ] Prometheus 指标暴露
- [ ] 监控面板

### Phase 7: 扩展支持 (Week 10+)

- [ ] SQL Server 支持
- [ ] Oracle 支持
- [ ] 自动补全

---

*文档版本：v1.0*
*最后更新：2026-02-13*
