<template>
  <div class="export-page">
    <el-page-header title="数据导出" @back="() => $router.push('/connections')">
      <template #content>
        <div style="display: flex; gap: 10px;">
          <el-select
            v-model="currentConnectionId"
            placeholder="选择连接"
            @change="handleConnectionChange"
            style="width: 200px"
          >
            <el-option
              v-for="conn in connectionsStore.connections"
              :key="conn.id"
              :label="conn.name"
              :value="conn.id"
            />
          </el-select>
          <el-select
            v-model="currentDatabase"
            placeholder="选择数据库"
            @change="handleDatabaseChange"
            style="width: 200px"
            v-if="currentConnectionId"
          >
            <el-option
              v-for="db in queryStore.databases"
              :key="db"
              :label="db"
              :value="db"
            />
          </el-select>
        </div>
      </template>
    </el-page-header>

    <el-row :gutter="20" v-if="currentConnectionId">
      <el-col :span="8">
        <el-card header="导出配置">
          <el-form :model="exportConfig" label-width="100px">
            <el-form-item label="导出模式">
              <el-radio-group v-model="exportConfig.mode">
                <el-radio value="table">按表导出</el-radio>
                <el-radio value="query">按 SQL 导出</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="导出格式">
              <el-radio-group v-model="exportConfig.format">
                <el-radio value="csv">CSV</el-radio>
                <el-radio value="sql">SQL</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item v-if="exportConfig.mode === 'table'" label="选择表">
              <el-select
                v-model="exportConfig.selectedTables"
                multiple
                placeholder="选择要导出的表"
                style="width: 100%"
              >
                <el-option
                  v-for="table in queryStore.tables"
                  :key="table.name"
                  :label="table.name"
                  :value="table.name"
                />
              </el-select>
            </el-form-item>

            <el-form-item v-else label="查询 SQL">
              <div ref="queryEditorContainer" class="query-editor-small"></div>
            </el-form-item>

            <template v-if="exportConfig.format === 'csv'">
              <el-form-item label="包含表头">
                <el-switch v-model="csvOptions.includeHeader" />
              </el-form-item>
              <el-form-item label="分隔符">
                <el-input v-model="csvOptions.separator" style="width: 100px" />
              </el-form-item>
              <el-form-item label="编码">
                <el-select v-model="csvOptions.encoding">
                  <el-option label="UTF-8" value="UTF-8" />
                  <el-option label="GBK" value="GBK" />
                </el-select>
              </el-form-item>
            </template>

            <template v-if="exportConfig.format === 'sql'">
              <el-form-item label="目标数据库类型">
                <el-select v-model="exportConfig.targetDbType" placeholder="选择目标数据库类型">
                  <el-option label="同源数据库" value="" />
                  <el-option label="MySQL" value="mysql" />
                  <el-option label="PostgreSQL" value="postgresql" />
                  <el-option label="SQLite" value="sqlite" />
                  <el-option label="MSSQL" value="mssql" />
                  <el-option label="ClickHouse" value="clickhouse" />
                  <el-option label="KingBase" value="kingbase" />
                </el-select>
              </el-form-item>
              <el-form-item label="包含建表语句">
                <el-switch v-model="sqlOptions.includeCreateTable" />
              </el-form-item>
              <el-form-item label="包含 DROP">
                <el-switch v-model="sqlOptions.includeDropTable" />
              </el-form-item>
              <el-form-item label="批量插入">
                <el-switch v-model="sqlOptions.batchInsert" />
              </el-form-item>
              <el-form-item label="批量大小">
                <el-input-number v-model="sqlOptions.batchSize" :min="1" :max="1000" />
              </el-form-item>
            </template>

            <el-form-item>
              <el-button type="primary" @click="handleExport" :loading="exporting">
                开始导出
              </el-button>
              <el-button v-if="exportConfig.format === 'sql' && exportConfig.targetDbType && exportConfig.mode === 'table'" @click="handleTypePreview" :loading="typePreviewing">
                类型映射预览
              </el-button>
              <el-button @click="handlePreview" :loading="previewing">
                预览
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card header="导出预览" class="preview-card">
          <div v-show="previewContent" ref="previewEditorContainer" class="preview-editor"></div>
          <el-empty v-if="!previewContent" description="选择表后点击预览查看内容" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 类型映射预览对话框 -->
    <el-dialog
      v-model="typeMappingDialogVisible"
      title="SQL 导出 - 类型映射预览"
      width="800px"
    >
      <div v-if="typeMappingResult" class="type-mapping-content">
        <!-- 映射摘要 -->
        <div class="mapping-summary">
          <el-alert type="success" :closable="false">
            <template #title>
              <div>
                ✅ 成功映射 {{ typeMappingResult.summary.direct }} 个类型
              </div>
            </template>
          </el-alert>

          <el-alert v-if="typeMappingResult.summary.fallback > 0" type="warning" :closable="false" style="margin-top: 10px;">
            <template #title>
              <div>
                ⚠️ 有 {{ typeMappingResult.summary.fallback }} 个类型使用了安全降级
              </div>
            </template>
          </el-alert>

          <el-alert v-if="typeMappingResult.summary.userChoice > 0" type="info" :closable="false" style="margin-top: 10px;">
            <template #title>
              <div>
                ℹ️ 有 {{ typeMappingResult.summary.userChoice }} 个类型需要您选择
              </div>
            </template>
          </el-alert>

          <el-alert v-if="typeMappingResult.summary.lossyCount > 0" type="warning" :closable="false" style="margin-top: 10px;">
            <template #title>
              <div>
                ⚠️ 有 {{ typeMappingResult.summary.lossyCount }} 个类型存在精度损失
              </div>
            </template>
          </el-alert>

          <!-- 类型总数 -->
          <div class="mapping-stats">
            <span>总类型数: {{ typeMappingResult.summary.total }}</span>
            <el-divider direction="vertical" />
          </div>
        </div>

        <!-- 需要用户选择的类型 -->
        <div v-if="hasUserChoices" class="user-choices">
          <h3>需要您选择的类型（{{ Object.keys(typeMappingResult.requiresUser).length }} 个）</h3>
          <p class="tip">请为以下类型选择目标数据库类型：</p>

          <div v-for="(rule, sourceType) in typeMappingResult.requiresUser" :key="sourceType" class="choice-item">
            <div class="choice-header">
              <span class="source-type">{{ sourceType }}</span>
              <el-icon><Document /></el-icon>
              <el-tooltip v-if="rule.note" :content="rule.note">
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>

            <el-select v-model="userChoices[sourceType]" placeholder="请选择目标类型">
              <el-option
                v-for="opt in rule.userOptions"
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
          </div>
        </div>

        <!-- 转换警告信息 -->
        <div v-if="typeMappingResult.warnings.length > 0" class="warnings-section">
          <h4>⚠️ 转换警告</h4>
          <ul>
            <li v-for="warning in typeMappingResult.warnings" :key="warning">
              {{ warning }}
            </li>
          </ul>
        </div>

        <!-- 类型映射详情 -->
        <div v-if="showMappingDetails" class="mapping-details">
          <h4>📋 类型映射详情</h4>
          <el-table :data="mappingDetailsData" style="width: 100%">
            <el-table-column prop="sourceType" label="源类型" width="200" />
            <el-table-column prop="targetType" label="目标类型" width="200" />
            <el-table-column prop="status" label="状态" width="120">
              <template #default="scope">
                <el-tag v-if="scope.row.status === 'direct'" type="success">直接映射</el-tag>
                <el-tag v-else-if="scope.row.status === 'fallback'" type="warning">安全降级</el-tag>
                <el-tag v-else-if="scope.row.status === 'user'" type="info">用户选择</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <template #footer>
        <el-button @click="typeMappingDialogVisible = false">取消</el-button>
        <el-button @click="showMappingDetails = !showMappingDetails">
          {{ showMappingDetails ? '隐藏' : '显示' }}映射详情
        </el-button>
        <el-button type="primary" @click="handleConfirmTypeMapping" :disabled="!canConfirmTypeMapping">
          确认并导出
          <el-icon><CircleCheck /></el-icon>
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, onBeforeUnmount, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useConnectionsStore } from '@/stores/connections'
import { useQueryStore } from '@/stores/query'
import { ElMessage, ElNotification } from 'element-plus'
import { Document, QuestionFilled, CircleCheck } from '@element-plus/icons-vue'
import * as monaco from 'monaco-editor'
import { api } from '@/api'
import type { CSVOptions, SQLOptions, TypeMappingResult } from '@/types'

const route = useRoute()
const router = useRouter()
const connectionsStore = useConnectionsStore()
const queryStore = useQueryStore()
const currentConnectionId = ref(route.params.id as string || '')
const currentDatabase = ref('')
const exporting = ref(false)
const previewing = ref(false)
const typePreviewing = ref(false)
const previewContent = ref('')
const previewEditorContainer = ref<HTMLElement>()
let previewEditor: monaco.editor.IStandaloneCodeEditor | null = null

// 从路由参数获取表名（用于 SQL 导出时的表名）
const tableFromQuery = ref((route.query.table as string) || '')

const exportConfig = reactive({
  format: 'csv',
  mode: (route.query.sql ? 'query' : 'table') as 'table' | 'query',
  selectedTables: [] as string[],
  targetDbType: '' as string | undefined
})

const queryEditorContainer = ref<HTMLElement>()
let queryEditor: monaco.editor.IStandaloneCodeEditor | null = null

// 类型映射相关状态
const typeMappingDialogVisible = ref(false)
const typeMappingResult = ref<TypeMappingResult | null>(null)
const userChoices = ref<Record<string, string>>({})
const showMappingDetails = ref(false)

const hasUserChoices = computed(() => {
  return typeMappingResult.value ? Object.keys(typeMappingResult.value.requiresUser).length > 0 : false
})

const canConfirmTypeMapping = computed(() => {
  if (!typeMappingResult.value) return false
  // 检查所有需要用户选择的类型是否已选择
  for (const sourceType of Object.keys(typeMappingResult.value.requiresUser)) {
    if (!userChoices.value[sourceType]) {
      return false
    }
  }
  return true
})

const mappingDetailsData = computed(() => {
  if (!typeMappingResult.value) return []
  const result: any[] = []
  for (const [sourceType, targetType] of Object.entries(typeMappingResult.value.mapped)) {
    let status = 'direct'
    if (typeMappingResult.value!.requiresUser[sourceType]) {
      status = 'user'
    } else if (typeMappingResult.value!.summary.fallback > 0) {
      status = 'fallback'
    }
    result.push({
      sourceType,
      targetType,
      status
    })
  }
  return result
})

const isLossy = (targetType: string) => {
  const lossyTypes = ['TINYINT', 'SMALLINT', 'TINYINT_UNSIGNED', 'FLOAT']
  return lossyTypes.some(type => targetType.includes(type))
}

const initQueryEditor = () => {
  if (!queryEditorContainer.value) return
  queryEditor = monaco.editor.create(queryEditorContainer.value, {
    value: (route.query.sql as string) || '',
    language: 'sql',
    theme: 'vs-dark',
    minimap: { enabled: false },
    fontSize: 12,
    automaticLayout: true,
    scrollBeyondLastLine: false
  })
}

watch(() => exportConfig.mode, (newMode) => {
  if (newMode === 'query' && !queryEditor) {
    nextTick(() => initQueryEditor())
  }
})

const csvOptions = reactive<CSVOptions>({
  includeHeader: true,
  separator: ',',
  quote: '"',
  encoding: 'UTF-8',
  nullValue: 'NULL',
  dateFormat: '2006-01-02 15:04:05'
})

const sqlOptions = reactive<SQLOptions>({
  includeCreateTable: true,
  includeDropTable: false,
  batchInsert: true,
  batchSize: 100,
  structureOnly: false
})

watch(currentConnectionId, (newId) => {
  if (newId) {
    queryStore.fetchDatabases(newId)
  } else {
    queryStore.databases = []
    currentDatabase.value = ''
  }
})

onBeforeUnmount(() => {
  previewEditor?.dispose()
  queryEditor?.dispose()
})

const initPreviewEditor = (value: string = '') => {
  if (!previewEditorContainer.value) return

  previewEditor = monaco.editor.create(previewEditorContainer.value, {
    value: value,
    language: exportConfig.format === 'sql' ? 'sql' : 'text',
    theme: 'vs-dark',
    readOnly: true,
    minimap: { enabled: false },
    fontSize: 12,
    automaticLayout: true
  })
}

watch(() => exportConfig.format, (newFormat) => {
  if (previewEditor) {
    monaco.editor.setModelLanguage(previewEditor.getModel()!, newFormat === 'sql' ? 'sql' : 'text')
  }
})

onMounted(async () => {
  await connectionsStore.fetchConnections()
  if (route.query.db) {
    currentDatabase.value = route.query.db as string
  }
  if (currentConnectionId.value) {
    try {
      await queryStore.fetchDatabases(currentConnectionId.value)
      if (currentDatabase.value) {
        await queryStore.fetchTables(currentConnectionId.value, currentDatabase.value)
      } else {
        await queryStore.fetchTables(currentConnectionId.value)
      }
    } catch (e: any) {
      ElNotification.error({
        title: '加载失败',
        message: e.response?.data?.message || '加载元数据失败',
        position: 'top-right'
      })
    }
  }
  if (exportConfig.mode === 'query') {
    nextTick(() => initQueryEditor())
  }
})

async function handleConnectionChange(id: string) {
  currentConnectionId.value = id
  currentDatabase.value = ''
  if (id) {
    try {
      await queryStore.fetchDatabases(id)
    } catch (e: any) {
      ElNotification.error({
        title: '加载失败',
        message: e.response?.data?.message || '加载数据库列表失败',
        position: 'top-right'
      })
    }
  }
}

async function handleDatabaseChange(db: string) {
  if (db) {
    try {
      await queryStore.fetchTables(currentConnectionId.value, db)
    } catch (e: any) {
      ElNotification.error({
        title: '加载失败',
        message: e.response?.data?.message || '加载表列表失败',
        position: 'top-right'
      })
    }
  }
}

async function handlePreview() {
  if (exportConfig.mode === 'table' && exportConfig.selectedTables.length === 0) {
    ElMessage.warning('请选择要预览的表')
    return
  }
  if (exportConfig.mode === 'query' && !queryEditor?.getValue()) {
    ElMessage.warning('请输入要预览的 SQL')
    return
  }

  previewing.value = true
  try {
    let res
    const sql = exportConfig.mode === 'query' ? queryEditor?.getValue() : ''

    if (exportConfig.format === 'csv') {
      const query = exportConfig.mode === 'query' ? sql : `SELECT * FROM ${exportConfig.selectedTables[0]} LIMIT 10`
      res = await api.exportCSV(currentConnectionId.value, {
        query: query as string,
        opts: { ...csvOptions, maxRows: 10 },
        database: currentDatabase.value
      })
    } else {
      res = await api.exportSQL(currentConnectionId.value, {
        tables: exportConfig.mode === 'table' ? exportConfig.selectedTables : [],
        opts: {
          ...sqlOptions,
          structureOnly: false,
          maxRows: 10,
          query: exportConfig.mode === 'query' ? sql : ''
        },
        database: currentDatabase.value
      })
    }

    const text = typeof res === 'string' ? res : await (res as unknown as Blob).text()
    previewContent.value = text

    if (!previewEditor) {
      nextTick(() => initPreviewEditor(text))
    } else {
      previewEditor.setValue(text)
    }
  } catch (e: any) {
    ElNotification.error({
      title: '预览失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    previewing.value = false
  }
}

// 类型映射预览
async function handleTypePreview() {
  if (exportConfig.mode === 'table' && exportConfig.selectedTables.length === 0) {
    ElMessage.warning('请选择要导出的表')
    return
  }
  if (!exportConfig.targetDbType) {
    ElMessage.warning('请选择目标数据库类型')
    return
  }

  typePreviewing.value = true
  try {
    const result = await api.previewExportSQL(currentConnectionId.value, {
      tables: exportConfig.mode === 'table' ? exportConfig.selectedTables : [],
      targetDbType: exportConfig.targetDbType as any
    })

    typeMappingResult.value = result
    userChoices.value = {}
    showMappingDetails.value = false
    typeMappingDialogVisible.value = true
  } catch (e: any) {
    ElNotification.error({
      title: '类型映射预览失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    typePreviewing.value = false
  }
}

// 确认类型映射并导出
async function handleConfirmTypeMapping() {
  if (!typeMappingResult.value) return

  typeMappingDialogVisible.value = false

  // TODO: 应用用户选择的类型映射到导出
  ElMessage.success('类型映射已确认，正在导出...')
  await handleExport()
}

async function handleExport() {
  if (exportConfig.mode === 'table' && exportConfig.selectedTables.length === 0) {
    ElMessage.warning('请选择要导出的表')
    return
  }
  if (exportConfig.mode === 'query' && !queryEditor?.getValue()) {
    ElMessage.warning('请输入要导出的 SQL')
    return
  }

  exporting.value = true
  try {
    let res
    const sql = exportConfig.mode === 'query' ? queryEditor?.getValue() : ''

    if (exportConfig.format === 'csv') {
      const query = exportConfig.mode === 'query' ? sql : `SELECT * FROM ${exportConfig.selectedTables[0]}`
      res = await api.exportCSV(currentConnectionId.value, {
        query: query as string,
        opts: csvOptions,
        database: currentDatabase.value
      })
    } else {
      res = await api.exportSQL(currentConnectionId.value, {
        tables: exportConfig.mode === 'table' ? exportConfig.selectedTables : [],
        opts: {
          ...sqlOptions,
          query: exportConfig.mode === 'query' ? sql : '',
          tableName: exportConfig.mode === 'query' ? (tableFromQuery.value || 'query_result') : ''
        },
        database: currentDatabase.value
      })
    }

    // Trigger download
    const blob = new Blob([res], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `export_${new Date().getTime()}.${exportConfig.format}`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)

    ElMessage.success('导出成功')
  } catch (e: any) {
    ElNotification.error({
      title: '导出失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.export-page {
  padding: 20px;
}

.preview-card :deep(.el-card__body) {
  height: 500px;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.preview-editor {
  flex: 1;
  width: 100%;
}

.query-editor-small {
  height: 150px;
  width: 100%;
  border: 1px solid #dcdfe6;
}

/* 类型映射预览样式 */
.type-mapping-content {
  padding: 10px 0;
}

.mapping-summary {
  margin-bottom: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
}

.mapping-stats {
  display: flex;
  align-items: center;
  gap: 15px;
  font-size: 13px;
  color: #909399;
  margin-top: 10px;
}

.user-choices {
  margin-bottom: 20px;
}

.user-choices h3 {
  margin: 0 0 10px 0;
  font-size: 16px;
  color: #303133;
}

.user-choices .tip {
  margin: 0 0 15px 0;
  font-size: 13px;
  color: #606266;
}

.choice-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 15px;
}

.choice-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 200px;
}

.source-type {
  font-family: 'Courier New', monospace;
  background: #e6f7ff;
  padding: 4px 8px;
  border-radius: 3px;
  font-weight: 500;
  color: #0050b3;
}

.option-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.warnings-section {
  margin-bottom: 20px;
  padding: 15px;
  background: #fff3cd;
  border-radius: 4px;
  border-left: 4px solid #ffc107;
}

.warnings-section h4 {
  margin: 0 0 10px 0;
  color: #e6a23c;
  font-size: 14px;
}

.warnings-section ul {
  margin: 0;
  padding-left: 20px;
  color: #e6a23c;
}

.mapping-details {
  margin-top: 20px;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 4px;
}

.mapping-details h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #303133;
}
</style>