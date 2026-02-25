# DBM - Database Manager

> 一个轻量、跨平台的通用数据库管理工具，支持多种主流数据库，单文件部署。

## 项目概述

DBM 是一个用 Go 语言开发的现代化数据库管理工具，旨在为开发者提供统一的数据库管理体验，支持多平台单文件部署，欢迎自用和二开，禁止用于商业用途。

### 核心特性

- **多数据库支持**：MySQL、PostgreSQL、SQLite、ClickHouse、KingBase
- **现代 Web 界面**：基于 Vue.js 的响应式 UI
- **单文件部署**：前端资源嵌入 Go 可执行文件，无需额外依赖
- **数据导出**：支持 CSV 和 SQL 格式导出，含类型映射功能
- **表结构管理**：可视化表结构编辑，支持 ALTER TABLE 操作
- **安全保障**：AES-256-GCM 密码加密存储
- **连接分组**：支持连接配置分组管理

---

## 快速开始

### 下载

前往 [Releases](https://github.com/ha1377311454/dbm/releases/tag/v1.0.0) 页面下载对应平台的可执行文件。

| 平台 | 文件名 |
|-----|-------|
| Linux (amd64) | `dbm-linux-amd64` |
| macOS (Intel) | `dbm-darwin-amd64` |
| macOS (Apple Silicon) | `dbm-darwin-arm64` |
| Windows (amd64) | `dbm-windows-amd64.exe` |

### 运行

```bash
# 赋予执行权限 (Linux/macOS)
chmod +x dbm-linux-amd64

# 启动服务
./dbm-linux-amd64

# 自定义端口
./dbm-linux-amd64 --port 9000

# 本地编译调试
lsof -t -i:2048 | xargs kill || true && make build && ./dist/dbm > server.log 2>&1 &
```

启动后访问：http://localhost:2048

### 命令行参数

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

## 功能预览

### 连接管理

- 管理多个数据库连接
- 密码 AES-256-GCM 加密存储
- 一键测试连接
- 连接分组管理

### SQL 编辑器

- 语法高亮（Monaco Editor）
- 多标签页支持
- 查询结果分页展示
- 执行非查询 SQL

### 数据编辑

- 表格视图浏览数据
- 在线编辑单元格
- 支持 INSERT/UPDATE/DELETE

### 表结构管理

- 可视化表结构查看
- 添加/删除/修改列
- 管理索引
- 重命名表
- 跨数据库类型兼容处理

### 数据导出

- **CSV 导出**：自定义分隔符、编码
- **SQL 导出**：
  - INSERT 语句导出
  - 跨数据库类型映射
  - 类型映射预览
  - 支持数据迁移

---

## 技术栈

### 后端

| 技术 | 版本 | 用途 |
|-----|------|-----|
| Go | 1.24+ | 后端语言 |
| Gin | 1.10.0 | HTTP 框架 |
| go-sql-driver/mysql | 1.9.3 | MySQL 驱动 |
| lib/pq | 1.11.2 | PostgreSQL 驱动 |
| mattn/go-sqlite3 | 1.14.34 | SQLite 驱动 |
| ClickHouse/clickhouse-go/v2 | 2.43.0 | ClickHouse 驱动 |
| kingbase.com/gokb | 1.0.0 | KingBase 驱动（本地模块） |
| google/uuid | 1.6.0 | UUID 生成 |
| golang.org/x/crypto | 0.47.0 | 密码加密 |

### 前端

| 技术 | 版本 | 用途 |
|-----|------|-----|
| Vue.js | 3.4+ | 前端框架 |
| TypeScript | 5.3+ | 类型安全 |
| Vite | 5.0+ | 构建工具 |
| Element Plus | 2.5+ | UI 组件库 |
| Monaco Editor | 0.45+ | 代码编辑器 |
| Pinia | 2.1+ | 状态管理 |
| Vue Router | 4.2+ | 路由管理 |
| Axios | 1.6+ | HTTP 客户端 |
| ECharts | 5.5+ | 图表组件 |

---

## 项目结构

```
dbm/
├── cmd/dbm/           # 程序入口
├── internal/          # 内部包
│   ├── adapter/       # 数据库适配器
│   │   └── gokb/      # KingBase 驱动（本地模块）
│   ├── connection/    # 连接管理
│   ├── export/        # 导出引擎
│   ├── model/         # 数据模型
│   ├── server/        # HTTP 服务器
│   └── service/       # 业务服务层
├── web/               # 前端项目 (Vue.js)
├── configs/           # 配置文件
│   └── type_mapping.yaml  # 类型映射配置
├── docs/              # 文档
│   ├── DESIGN.md      # 设计文档
│   └── CHANGELOG.md   # 变更日志
├── Makefile           # 构建命令
├── CLAUDE.md          # AI 开发指南
└── README.md          # 本文件
```

---

## 从源码构建

### 环境要求

- Go 1.24+
- Node.js 18+
- Make (可选)

### 构建步骤

```bash
# 克隆仓库
git clone https://github.com/yourusername/dbm.git
cd dbm

# 安装依赖
go mod download
cd web && npm install && cd ..

# 构建
make build

# 或使用脚本
./scripts/build.sh
```

### 跨平台编译

```bash
# Linux
GOOS=linux GOARCH=amd64 make build

# macOS
GOOS=darwin GOARCH=amd64 make build
GOOS=darwin GOARCH=arm64 make build

# Windows
GOOS=windows GOARCH=amd64 make build
```

### 开发模式

```bash
# 后端热重载
make dev

# 前端开发服务器
make dev-web
```

---

## 配置说明

### 配置文件位置

| 平台 | 配置路径 |
|-----|---------|
| Linux | `~/.config/dbm/` |
| macOS | `~/Library/Application Support/dbm/` |
| Windows | `%APPDATA%/dbm/` |

### 存储文件

- `connections.json`：连接配置（密码已加密）
- `groups.json`：分组配置
- `.key`：密码加密密钥

---

## API 文档

### RESTful API

```
BASE_URL: /api/v1
```

#### 连接管理

```
GET    /connections              # 获取连接列表
POST   /connections              # 创建连接
PUT    /connections/:id          # 更新连接
DELETE /connections/:id          # 删除连接
POST   /connections/:id/connect  # 建立连接
POST   /connections/:id/close    # 关闭连接
POST   /connections/:id/test     # 测试连接
POST   /connections/test         # 测试连接配置（未保存）
```

#### 数据库元数据

```
GET    /connections/:id/databases           # 获取数据库列表
GET    /connections/:id/schemas             # 获取 schema 列表
GET    /connections/:id/tables              # 获取表列表
GET    /connections/:id/tables/:table/schema # 获取表结构
GET    /connections/:id/views               # 获取视图列表
```

#### SQL 执行

```
POST   /connections/:id/query   # 执行查询
POST   /connections/:id/execute # 执行非查询 SQL
```

#### 数据编辑

```
POST   /connections/:id/tables/:table/data  # 创建数据
PUT    /connections/:id/tables/:table/data  # 更新数据
DELETE /connections/:id/tables/:table/data  # 删除数据
```

#### 表结构修改

```
POST   /connections/:id/tables/:table/alter  # 修改表结构
POST   /connections/:id/tables/:table/rename # 重命名表
```

#### 导出

```
POST   /connections/:id/export/csv          # CSV 导出
POST   /connections/:id/export/sql          # SQL 导出
POST   /connections/:id/export/sql/preview  # SQL 导出类型映射预览
```

#### 分组管理

```
GET    /groups         # 获取分组列表
POST   /groups         # 创建分组
PUT    /groups/:id     # 更新分组
DELETE /groups/:id     # 删除分组
```

#### 监控

```
GET    /metrics           # Prometheus 指标
GET    /api/v1/monitor/stats  # 监控统计
```

---

## 支持的数据库

| 数据库 | 版本 | 状态 |
|-------|------|-----|
| MySQL | 5.7+, 8.0+ | ✅ 已实现 |
| PostgreSQL | 12+, 14+, 15+ | ✅ 已实现 |
| SQLite | 3.x | ✅ 已实现 |
| ClickHouse | 22.3+ | ✅ 已实现 |
| KingBase | ES V8 | ✅ 已实现 |

### 数据库特性差异

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

## 路线图

### V1.0 - 基础版本（已完成）

- [x] 基础连接管理
- [x] MySQL/PostgreSQL/SQLite 支持
- [x] SQL 编辑与执行
- [x] 数据浏览与编辑
- [x] CSV/SQL 导出
- [x] 表结构修改功能
- [x] ClickHouse/KingBase 支持
- [x] 类型映射功能

### V1.1 - 功能增强（计划中）

- [ ] SQL Server 支持
- [ ] Oracle 支持
- [ ] 查询历史记录

### V1.2 - 监控与运维（计划中）

- [ ] 完整的 Prometheus 指标
- [ ] 前端监控面板
- [ ] 慢查询分析

---

## 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 提交 Pull Request

### 代码规范

- Go 代码遵循 [Effective Go](https://go.dev/doc/effective_go)
- 前端代码遵循 [Vue 风格指南](https://vuejs.org/style-guide/)

更多开发细节请参考 [CLAUDE.md](./CLAUDE.md)。

---

## 文档

- [CLAUDE.md](./CLAUDE.md) - AI 开发指南
- [docs/DESIGN.md](./docs/DESIGN.md) - 设计文档
- [docs/CHANGELOG.md](./docs/CHANGELOG.md) - 变更日志

---

## 许可证

[MIT License](./LICENSE)

---

## 联系方式

- Email: 1617802907@qq.com

---

**DBM** - 让数据库管理更简单。
