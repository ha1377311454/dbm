# DBM 变更日志

本文档记录 DBM 项目的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [未发布]

### 新增
- KingBase 数据库支持（通过本地 gokb 驱动）
- ClickHouse 数据库支持
- 表结构修改功能（ALTER TABLE）
  - 添加/删除列
  - 修改列定义
  - 重命名列
  - 添加/删除索引
  - 重命名表
- SQL 导出类型映射功能
  - 跨数据库类型转换
  - 类型映射预览 API
  - 可配置的映射规则（`configs/type_mapping.yaml`）
- 表结构编辑器前端界面
- 分组管理功能（连接分组）

### 变更
- 密码加密从 AES-256 升级到 AES-256-GCM
- 更新 Go 版本要求至 1.24+
- 更新前端依赖版本（Vue 3.4+、Element Plus 2.5+、Monaco Editor 0.45+）

### 修复
- 修复 PostgreSQL schema 查询问题
- 修复 ClickHouse 连接配置问题
- 修复跨数据库 SQL 导出的类型兼容性

---

## [1.0.0] - 2026-02-13

### 新增
- 基础连接管理功能
  - 多数据库连接配置
  - 连接密码 AES-256-GCM 加密存储
  - 连接测试功能
- MySQL 支持（5.7+, 8.0+）
- PostgreSQL 支持（12+, 14+, 15+）
- SQLite 支持（3.x）
- SQL 编辑器功能
  - 语法高亮（Monaco Editor）
  - SQL 查询执行
  - 结果分页展示
- 数据浏览与编辑
  - 表结构查看
  - 数据分页浏览
  - 在线编辑单元格
  - INSERT/UPDATE/DELETE 操作
- 数据导出功能
  - CSV 导出（自定义分隔符、编码）
  - SQL 导出（INSERT 语句、包含建表语句）
- Web 界面
  - 基于 Vue 3 + Vite 构建
  - Element Plus UI 组件库
  - 响应式设计
- API 路由
  - RESTful API 设计
  - 统一响应格式
- 单文件部署
  - 前端资源嵌入 Go 二进制
  - 跨平台编译支持

### 技术栈
- 后端：Go 1.24、Gin 1.10、database/sql
- 前端：Vue.js 3.4、TypeScript 5.0、Vite 5.0
- UI 组件：Element Plus 2.5、Monaco Editor 0.45、ECharts 5.5
- 状态管理：Pinia 2.1
- 路由：Vue Router 4.2
- HTTP 客户端：Axios 1.6

---

## 版本说明

### 版本号规则
- **主版本号**：不兼容的 API 变更
- **次版本号**：向下兼容的功能新增
- **修订号**：向下兼容的问题修复

### 变更类型
- **新增**：新功能
- **变更**：现有功能的变更
- **弃用**：即将移除的功能
- **移除**：已移除的功能
- **修复**：问题修复
- **安全**：安全相关的修复
