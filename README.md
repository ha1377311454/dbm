# DBM - Database Manager

> 一个轻量、跨平台的通用数据库管理工具，支持多种主流数据库，单文件部署。

## 项目概述

DBM 是一个用 Go 语言开发的现代化数据库管理工具，旨在为开发者和 DBA 提供统一的数据库管理体验。

### 核心特性

- **多数据库支持**：MySQL、PostgreSQL、SQLite、SQL Server、Oracle
- **现代 Web 界面**：基于 Vue.js 的响应式 UI
- **单文件部署**：前端资源嵌入 Go 可执行文件，无需额外依赖
- **数据导出**：支持 CSV 和 SQL 格式导出
- **可视化查询**：拖拽式查询构建器
- **监控集成**：Prometheus 指标暴露
- **安全保障**：AES-256 密码加密存储

---

## 快速开始

### 下载

前往 [Releases](https://github.com/yourusername/dbm/releases) 页面下载对应平台的可执行文件。

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
```

启动后访问：http://localhost:8080

### 命令行参数

```bash
dbm [命令] [参数]

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

## 功能预览

### 连接管理

- 管理多个数据库连接
- 密码 AES-256 加密存储
- 一键测试连接

### SQL 编辑器

- 语法高亮
- 自动补全
- 多标签页支持

### 数据编辑

- 表格视图浏览数据
- 在线编辑单元格
- 支持 INSERT/UPDATE/DELETE

### 数据导出

- **CSV 导出**：自定义分隔符、编码
- **SQL 导出**：INSERT 语句，支持数据迁移

### 可视化查询

- 拖拽选择表和字段
- 图形化配置关联
- 实时 SQL 预览

### 监控功能

- Prometheus 指标暴露 (`/metrics`)
- 连接状态监控
- 查询性能统计

---

## 技术栈

### 后端

| 技术 | 用途 |
|-----|------|
| Go 1.21+ | 后端语言 |
| Gin | HTTP 框架 |
| database/sql | 统一数据库接口 |
| embed | 静态资源嵌入 |
| prometheus/client_golang | 监控指标 |

### 前端

| 技术 | 用途 |
|-----|------|
| Vue.js 3 | 前端框架 |
| TypeScript | 类型安全 |
| Vite | 构建工具 |
| Element Plus | UI 组件库 |
| Monaco Editor | 代码编辑器 |
| ECharts | 图表组件 |

---

## 项目结构

```
dbm/
├── cmd/dbm/           # 程序入口
├── internal/          # 内部包
│   ├── adapter/       # 数据库适配器
│   ├── connection/    # 连接管理
│   ├── engine/        # SQL 执行引擎
│   ├── export/        # 导出引擎
│   ├── monitor/       # 监控模块
│   └── server/        # HTTP 服务器
├── web/               # 前端项目 (Vue.js)
├── configs/           # 配置文件
├── scripts/           # 构建脚本
└── docs/              # 文档
```

---

## 从源码构建

### 环境要求

- Go 1.21+
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

---

## 配置说明

### 配置文件位置

| 平台 | 配置路径 |
|-----|---------|
| Linux | `~/.config/dbm/config.yaml` |
| macOS | `~/Library/Application Support/dbm/config.yaml` |
| Windows | `%APPDATA%/dbm/config.yaml` |

### 配置示例

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  path: "~/.local/share/dbm/dbm.db"

security:
  encryption_key: ""  # 留空则自动生成

logging:
  level: "info"
  file: ""  # 留空则输出到 stdout

monitoring:
  enabled: true
  slow_query_threshold: 3s
```

---

## API 文档

### RESTful API

```
BASE_URL: /api/v1
```

#### 连接管理

```
GET    /connections          # 获取连接列表
POST   /connections          # 创建连接
PUT    /connections/:id      # 更新连接
DELETE /connections/:id      # 删除连接
POST   /connections/:id/test # 测试连接
```

#### SQL 执行

```
POST   /connections/:id/query   # 执行查询
POST   /connections/:id/execute # 执行非查询 SQL
```

#### 数据导出

```
POST   /connections/:id/export/csv  # CSV 导出
POST   /connections/:id/export/sql  # SQL 导出
GET    /exports/:id/download        # 下载导出文件
```

完整 API 文档请参考 [API.md](./docs/API.md)

---

## 支持的数据库

| 数据库 | 版本 | 状态 |
|-------|------|-----|
| MySQL | 5.7+, 8.0+ | ✅ |
| PostgreSQL | 12+, 14+, 15+ | ✅ |
| SQLite | 3.x | ✅ |
| SQL Server | 2017+ | 🚧 |
| Oracle | 19c+ | 🚧 |

---

## 路线图

### V1.0 - MVP (当前)

- [x] 基础连接管理
- [x] SQL 编辑与执行
- [x] 数据浏览与编辑
- [x] CSV/SQL 导出
- [ ] MySQL/PostgreSQL/SQLite 支持

### V1.1 - 功能增强

- [ ] SQL Server/Oracle 支持
- [ ] 可视化查询构建器
- [ ] SQL 自动补全
- [ ] 格式化功能

### V1.2 - 监控与运维

- [ ] Prometheus 指标
- [ ] 前端监控面板
- [ ] SQL 历史记录

### V2.0 - 高级特性

- [ ] SSH 隧道支持
- [ ] 数据库备份还原
- [ ] ER 图展示
- [ ] 多用户权限控制

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

---

## 许可证

[MIT License](./LICENSE)

---

## 联系方式

- Issue: [GitHub Issues](https://github.com/yourusername/dbm/issues)
- Email: your.email@example.com

---

**DBM** - 让数据库管理更简单。
