# SQL 导出类型映射功能设计文档

> 创建时间：2026-02-14
> 状态：待实现

---

## 一、需求背景

### 1.1 问题分析

当前项目的 CSV 导出和 SQL 导出功能存在以下问题：

1. **CSV 导出**：相对通用，主要涉及数据格式化
2. **SQL 导出**：存在问题
   - 同种数据库导出可用（MySQL → MySQL）
   - 跨数据库导出**未处理类型映射**（MySQL → PostgreSQL）
   - 各适配器中存在大量重复代码

### 1.2 核心问题

**不同数据库的数据类型不一致**，导出 SQL 时需要考虑对不同类型的适配。

**示例**：
- MySQL 的 `TINYINT` → PostgreSQL 无对应类型
- MySQL 的 `AUTO_INCREMENT` → PostgreSQL 的 `SERIAL`
- MySQL 的 `DATETIME` → PostgreSQL 的 `TIMESTAMP`

---

## 二、需求定义

### 2.1 场景划分

#### CSV 导出（给人看）
- **目的**：可读性
- **处理**：数据格式化（日期、数字、NULL 等）
- **状态**：问题不大，当前实现基本可用

#### SQL 导出（数据迁移）

**场景 A：同种数据库迁移**
```
MySQL → MySQL
PostgreSQL → PostgreSQL
```
- 直接导出原始 DDL/DML 即可
- 类型、约束、语法都兼容
- **当前状态**：已支持

**场景 B：跨数据库迁移**
```
MySQL → PostgreSQL
MySQL → SQLite
PostgreSQL → ClickHouse
```
- 需要**类型映射**
- 需要**语法转换**
- 需要**约束转换**
- **当前状态**：**完全未处理**

### 2.2 功能需求

#### 需求 1：UI 交互流程

**情况 A：类型可以直接映射**
```
MySQL VARCHAR(255) → PostgreSQL VARCHAR(255) ✅
```
- 显示"类型映射完成"摘要

**情况 B：类型无法精确映射**
```
MySQL TINYINT UNSIGNED → PostgreSQL ？(无对应类型)
```
- 弹出对话框让用户选择

#### 需求 2：可配置映射规则

使用**配置文件**方式存储类型映射规则

#### 需求 3：降级策略

当类型无法精确映射时：
- **直接选择最安全的类型**（如 INTEGER 而非 SMALLINT）
- 不使用最接近的（避免精度损失）

---

## 三、技术设计

### 3.1 配置文件结构

#### 文件路径
```
configs/type_mapping.yaml
```

#### 配置格式

```yaml
# configs/type_mapping.yaml

type_mapping:
  # MySQL → PostgreSQL
  mysql_to_postgresql:
    TINYINT:
      target: "SMALLINT"
      safe_fallback: "INTEGER"
      precision_loss: true
    SMALLINT:
      target: "SMALLINT"
      precision_loss: false
    INT:
      target: "INTEGER"
      precision_loss: false
    BIGINT:
      target: "BIGINT"
      precision_loss: false
    TINYINT_UNSIGNED:
      target: "INTEGER"
      safe_fallback: "BIGINT"
      precision_loss: true
    FLOAT:
      target: "REAL"
      precision_loss: true
    DOUBLE:
      target: "DOUBLE PRECISION"
      precision_loss: false
    DATETIME:
      target: "TIMESTAMP"
      precision_loss: false
    TIMESTAMP:
      target: "TIMESTAMP"
      precision_loss: false
    ENUM:
      target: "TEXT"
      safe_fallback: "TEXT"
      requires_user: true
      user_options:
        - label: "转换为 TEXT"
          value: "TEXT"
        - label: "转换为 VARCHAR(255)"
          value: "VARCHAR(255)"
        - label: "转换为 CHECK 约束"
          value: "CHECK_CONSTRAINT"
    AUTO_INCREMENT:
      target: "SERIAL"
      precision_loss: false

  # MySQL → SQLite
  mysql_to_sqlite:
    TINYINT:
      target: "INTEGER"
      precision_loss: true
    DATETIME:
      target: "TEXT"
      safe_fallback: "TEXT"
      note: "SQLite 没有原生 DATETIME"
    AUTO_INCREMENT:
      target: "AUTOINCREMENT"
      precision_loss: false

  # PostgreSQL → MySQL
  postgresql_to_mysql:
    SMALLINT:
      target: "TINYINT"
      safe_fallback: "SMALLINT"
      precision_loss: true
    SERIAL:
      target: "INT AUTO_INCREMENT"
      precision_loss: false
    TIMESTAMP:
      target: "DATETIME"
      precision_loss: false
    JSONB:
      target: "JSON"
      safe_fallback: "TEXT"
      note: "MySQL 5.7+ 支持 JSON"

  # 通用安全降级规则
  fallback_rules:
    integer:
      priority: ["TINYINT", "SMALLINT", "INTEGER", "BIGINT"]
      safest: "INTEGER"
    string:
      priority: ["CHAR", "VARCHAR", "TEXT"]
      safest: "TEXT"
    decimal:
      priority: ["FLOAT", "DOUBLE", "DECIMAL"]
      safest: "DECIMAL"
```

### 3.2 核心数据结构

```go
// internal/model/export.go

// TypeMapping 类型映射配置
type TypeMapping struct {
    Source      string            `yaml:"source"`
    Target      string            `yaml:"target"`
    Mappings    map[string]*TypeRule `yaml:"mappings"`
}

// TypeRule 类型转换规则
type TypeRule struct {
    TargetType    string   `yaml:"target"`
    SafeFallback  string   `yaml:"safe_fallback"`
    PrecisionLoss bool     `yaml:"precision_loss"`
    RequiresUser  bool     `yaml:"requires_user"`
    UserOptions   []TypeOption `yaml:"user_options,omitempty"`
    Note         string   `yaml:"note,omitempty"`
}

// TypeOption 用户可选择的类型选项
type TypeOption struct {
    Label string `yaml:"label"`
    Value string `yaml:"value"`
}

// TypeMappingResult 类型映射结果
type TypeMappingResult struct {
    Success        bool              `json:"success"`
    Mapped         map[string]string `json:"mapped"`        // 源类型 → 目标类型
    Warnings       []string          `json:"warnings"`
    RequiresUser   map[string]TypeRule `json:"requiresUser"`  // 需要用户选择的类型
    Summary        TypeSummary       `json:"summary"`
}

// TypeSummary 映射摘要
type TypeSummary struct {
    Total       int `json:"total"`
    Direct      int `json:"direct"`       // 直接映射
    Fallback    int `json:"fallback"`     // 安全降级
    UserChoice  int `json:"userChoice"`   // 需要用户选择
    LossyCount  int `json:"lossyCount"`  // 有精度损失
}

// ExportSQLRequest SQL 导出请求（扩展）
type ExportSQLRequest struct {
    ConnectionID    string          `json:"connectionId"`
    Tables          []string         `json:"tables"`
    TargetDBType    DatabaseType     `json:"targetDbType"`     // 新增：目标数据库类型
    TypeMapping     map[string]string `json:"typeMapping,omitempty"`  // 新增：用户选择的映射
    Options         SQLOptions       `json:"options"`
}
```

### 3.3 映射引擎设计

```go
// internal/export/type_mapper.go

package export

import (
    "dbm/internal/model"
    "os"
    "gopkg.in/yaml.v3"
)

// TypeMapper 类型映射器
type TypeMapper struct {
    mappings   map[string]*TypeMapping  // key: "mysql_to_postgresql"
    fallback   map[string][]string
}

// NewTypeMapper 创建类型映射器
func NewTypeMapper(configPath string) (*TypeMapper, error) {
    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, err
    }

    var config struct {
        TypeMappings map[string]*TypeMapping `yaml:"type_mapping"`
    }

    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    m := &TypeMapper{
        mappings: config.TypeMappings,
    }

    return m, nil
}

// MapTypes 映射类型
// sourceDB: 源数据库类型
// targetDB: 目标数据库类型
// columns: 源表列信息
func (m *TypeMapper) MapTypes(sourceDB, targetDB model.DatabaseType, columns []model.ColumnInfo) (*model.TypeMappingResult, error) {
    result := &model.TypeMappingResult{
        Success:      true,
        Mapped:       make(map[string]string),
        RequiresUser: make(map[string]*model.TypeRule),
        Warnings:     []string{},
    }

    mappingKey := fmt.Sprintf("%s_to_%s", sourceDB, targetDB)
    mapping, exists := m.mappings[mappingKey]

    if !exists {
        // 无映射配置，返回原始类型
        for _, col := range columns {
            result.Mapped[col.Type] = col.Type
        }
        return result, nil
    }

    result.Summary.Total = len(columns)

    for _, col := range columns {
        rule, hasRule := mapping.Mappings[col.Type]

        if !hasRule {
            // 无规则，保持原类型
            result.Mapped[col.Type] = col.Type
            continue
        }

        if rule.RequiresUser {
            // 需要用户选择
            result.RequiresUser[col.Type] = rule
            result.Summary.UserChoice++
            // 暂时使用默认值
            result.Mapped[col.Type] = rule.TargetType
        } else if rule.PrecisionLoss {
            // 有精度损失，使用安全降级
            result.Mapped[col.Type] = rule.SafeFallback
            result.Summary.Fallback++
            result.Summary.LossyCount++
            result.Warnings = append(result.Warnings,
                fmt.Sprintf("%s → %s (有精度损失)", col.Type, rule.SafeFallback))
        } else {
            // 直接映射
            result.Mapped[col.Type] = rule.TargetType
            result.Summary.Direct++
        }
    }

    return result, nil
}

// ApplyUserChoices 应用用户选择的类型映射
func (m *TypeMapper) ApplyUserChoices(result *model.TypeMappingResult, choices map[string]string) error {
    for sourceType, targetType := range choices {
        result.Mapped[sourceType] = targetType
        result.Summary.UserChoice--
    }
    return nil
}
```

### 3.4 API 扩展

```go
// internal/server/handler.go

// ExportSQLPreviewRequest 预览 SQL 导出
type ExportSQLPreviewRequest struct {
    Tables       []string     `json:"tables"`
    TargetDBType model.DatabaseType `json:"targetDbType"`
}

// previewExportSQL 预览类型映射（不执行导出）
func (s *Server) previewExportSQL(c *gin.Context) {
    id := c.Param("id")

    var req ExportSQLPreviewRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
        return
    }

    // 获取连接
    db, config, err := s.connectionSvc.GetDB(id, "")
    if err != nil {
        c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
        return
    }

    // 获取表结构
    dbAdapter, _ := s.databaseSvc.GetAdapter(config.Type)

    var allColumns []model.ColumnInfo
    for _, table := range req.Tables {
        schema, _ := dbAdapter.GetTableSchema(db, "", table)
        allColumns = append(allColumns, schema.Columns...)
    }

    // 执行类型映射
    mapper, _ := export.NewTypeMapper("configs/type_mapping.yaml")
    result, _ := mapper.MapTypes(config.Type, req.TargetDBType, allColumns)

    c.JSON(http.StatusOK, successResponse(result))
}

// exportSQL 执行 SQL 导出（扩展）
func (s *Server) exportSQL(c *gin.Context) {
    // ... 原有代码 ...

    var req model.ExportSQLRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
        return
    }

    // 新增：处理类型映射
    if req.TargetDBType != "" && req.TypeMapping != nil {
        // 应用用户选择的映射
        mapper, _ := export.NewTypeMapper("configs/type_mapping.yaml")
        // ... 应用映射逻辑
    }

    // ... 原有导出逻辑 ...
}
```

**新增路由**：
```go
// 预览类型映射
POST /api/v1/connections/:id/export/sql/preview

// 执行导出
POST /api/v1/connections/:id/export/sql
```

### 3.5 UI 交互设计

```vue
<!-- web/src/views/export/TypeMappingDialog.vue -->
<template>
  <el-dialog
    v-model="visible"
    title="SQL 导出 - 类型映射预览"
    width="800px"
  >
    <!-- 映射摘要 -->
    <el-alert type="success" :closable="false">
      <template #title>
        <div>成功映射 {{ mappingResult.summary.direct }} 个类型</div>
      </template>
    </el-alert>

    <el-alert v-if="mappingResult.summary.fallback > 0" type="warning" :closable="false">
      <template #title>
        <div>有 {{ mappingResult.summary.fallback }} 个类型使用了安全降级</div>
      </template>
      <div>为避免精度损失，系统选择了最安全的类型</div>
    </el-alert>

    <!-- 需要用户选择的类型 -->
    <div v-if="hasUserChoices" class="user-choices">
      <h3>需要您选择的类型（{{ Object.keys(mappingResult.requiresUser).length }} 个）</h3>

      <div v-for="(rule, sourceType) in mappingResult.requiresUser" :key="sourceType" class="choice-item">
        <div class="choice-label">
          <span class="source-type">{{ sourceType }}</span>
          <el-icon><Right /></el-icon>
        </div>

        <el-select v-model="userChoices[sourceType]" placeholder="请选择目标类型">
          <el-option
            v-for="opt in rule.user_options"
            :key="opt.value"
            :value="opt.value"
            :label="opt.label"
          >
            <div class="option-item">
              <span>{{ opt.label }}</span>
              <el-tag v-if="isLossy(opt.value)" type="warning" size="small">
                可能有损失
              </el-tag>
            </div>
          </el-option>
        </el-select>

        <el-tooltip v-if="rule.note" :content="rule.note" placement="top">
          <el-icon><QuestionFilled /></el-icon>
        </el-tooltip>
      </div>
    </div>

    <!-- 警告信息 -->
    <div v-if="mappingResult.warnings.length > 0" class="warnings">
      <h4>转换警告：</h4>
      <ul>
        <li v-for="warning in mappingResult.warnings" :key="warning">
          {{ warning }}
        </li>
      </ul>
    </div>

    <!-- 操作按钮 -->
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button @click="handlePreview">重新预览</el-button>
      <el-button type="primary" @click="handleConfirm" :disabled="!canConfirm">
        确认并导出
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Right, QuestionFilled } from '@element-plus/icons-vue'

const props = defineProps<{
  mappingResult: TypeMappingResult
  sourceDB: string
  targetDB: string
}>()

const emit = defineEmits<{
  confirm: [choices: Map<string, string>]
  preview: []
}>()

const userChoices = ref<Map<string, string>>({})

const hasUserChoices = computed(() => {
  return Object.keys(props.mappingResult.requiresUser).length > 0
})

const canConfirm = computed(() => {
  // 检查所有需要用户选择的类型是否已选择
  for (const sourceType in Object.keys(props.mappingResult.requiresUser)) {
    if (!userChoices.value[sourceType]) {
      return false
    }
  }
  return true
})

const isLossy = (targetType: string) => {
  // 判断是否有精度损失
  return false // 实现判断逻辑
}

const handlePreview = () => {
  emit('preview')
}

const handleConfirm = () => {
  emit('confirm', userChoices.value)
}
</script>

<style scoped>
.user-choices {
  margin: 20px 0;
}

.choice-item {
  display: flex;
  align-items: center;
  gap: 15px;
  margin-bottom: 15px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
}

.choice-label {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 150px;
  font-weight: 500;
}

.source-type {
  font-family: monospace;
  background: #e6f7ff;
  padding: 4px 8px;
  border-radius: 3px;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.warnings {
  margin: 20px 0;
  padding: 15px;
  background: #fff3cd;
  border-radius: 4px;
}

.warnings h4 {
  margin: 0 0 10px 0;
  color: #e6a23c;
}

.warnings ul {
  margin: 0;
  padding-left: 20px;
}
</style>
```

---

## 四、实现步骤

### 第一阶段：配置文件和加载器
- [ ] 创建 `configs/type_mapping.yaml` 示例文件
- [ ] 实现 `export/type_mapper.go`
- [ ] 单元测试：映射规则加载

### 第二阶段：映射引擎
- [ ] 实现 `MapTypes()` 方法
- [ ] 实现降级策略（选择最安全类型）
- [ ] 实现 `ApplyUserChoices()` 方法
- [ ] 单元测试：各种映射场景

### 第三阶段：API 扩展
- [ ] 添加 `previewExportSQL` 端点
- [ ] 扩展 `exportSQL` 支持目标数据库类型
- [ ] 集成测试：API 端到端点

### 第四阶段：UI 实现
- [ ] 创建 `TypeMappingDialog.vue` 组件
- [ ] 集成到导出流程
- [ ] E2E 测试：用户交互流程

### 第五阶段：默认映射规则
- [ ] 完善 MySQL → PostgreSQL 映射
- [ ] 完善 MySQL → SQLite 映射
- [ ] 添加 PostgreSQL → MySQL 映射
- [ ] 添加常用数据库组合

---

## 五、测试用例

### 5.1 单元测试

```go
func TestTypeMapper_MapTypes(t *testing.T) {
    mapper := loadTestMapper()

    columns := []ColumnInfo{
        {Name: "id", Type: "TINYINT"},
        {Name: "name", Type: "VARCHAR(255)"},
    }

    result := mapper.MapTypes("mysql", "postgresql", columns)

    assert.Equal(t, "SMALLINT", result.Mapped["TINYINT"])
    assert.Equal(t, "VARCHAR(255)", result.Mapped["VARCHAR(255)"])
}

func TestTypeMapper_RequiresUserChoice(t *testing.T) {
    // 测试需要用户选择的场景（ENUM 等）
}
```

### 5.2 集成测试

```bash
# 测试 API
curl -X POST http://localhost:2048/api/v1/connections/{id}/export/sql/preview \
  -H "Content-Type: application/json" \
  -d '{"tables": ["users"], "targetDbType": "postgresql"}'
```

---

## 六、后续优化

- [ ] 支持自定义映射规则（UI 配置界面）
- [ ] 导出时生成映射报告文件
- [ ] 支持更多数据库类型（Oracle、MSSQL）
- [ ] 智能推荐最佳映射策略