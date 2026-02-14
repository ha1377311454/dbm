<template>
  <div class="query-page">
    <el-container>
      <el-aside width="250px">
        <div class="connection-selector">
          <el-select
            v-model="currentConnectionId"
            placeholder="选择连接"
            filterable
            @change="handleConnectionChange"
            style="width: 100%; margin-bottom: 10px"
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
            filterable
            @change="handleDatabaseChange"
            style="width: 100%; margin-bottom: 10px"
            v-if="currentConnectionId"
          >
            <el-option
              v-for="db in queryStore.databases"
              :key="db"
              :label="db"
              :value="db"
            />
          </el-select>
          <el-select
            v-model="currentSchema"
            placeholder="选择 Schema"
            filterable
            @change="handleSchemaChange"
            style="width: 100%; margin-bottom: 10px"
            v-if="currentDatabase && queryStore.schemas.length > 0"
            clearable
          >
            <el-option
              v-for="schema in queryStore.schemas"
              :key="schema"
              :label="schema"
              :value="schema"
            />
          </el-select>
          <el-input
            v-model="tableFilter"
            placeholder="搜索表..."
            size="small"
            style="margin-top: 10px"
            clearable
          />
        </div>
        <el-divider />
        <div class="tables-list">
          <div
            v-for="table in filteredTables"
            :key="table.name"
            class="table-item"
            @click="handleTableClick(table.name)"
          >
            <el-icon><Document /></el-icon>
            <span>{{ table.name }}</span>
          </div>
        </div>
      </el-aside>

      <el-main>
        <div class="editor-container">
          <div class="editor-toolbar">
            <el-button type="primary" :icon="VideoPlay" @click="handleExecute" :loading="queryStore.loading">
              执行 (F5)
            </el-button>
            <el-button :icon="Delete" @click="handleClear">清空</el-button>
            <el-button :icon="MagicStick" @click="handleBeautify">美化</el-button>
            <el-button :icon="Download" @click="handleExport">导出</el-button>
            <el-button type="success" :icon="Plus" @click="handleAddData" :disabled="!selectedTable">
              新增数据
            </el-button>
          </div>
          <div ref="editorContainer" class="monaco-editor"></div>
        </div>

        <div v-if="queryStore.result" class="result-container">
          <div class="result-header">
            <span>查询结果</span>
            <div style="display: flex; align-items: center; gap: 15px;">
              <el-input
                v-model="resultSearch"
                placeholder="结果过滤..."
                size="small"
                style="width: 200px"
                :prefix-icon="Search"
                clearable
              />
              <span class="result-info">
                耗时: {{ queryStore.result.timeCost }}ms |
                行数: {{ filteredResults.length }} / {{ queryStore.result.total }}
              </span>
            </div>
          </div>
          <el-table
            :data="filteredResults"
            :default-sort="{ prop: 'id', order: 'ascending' }"
            stripe
            border
            height="400"
          >
            <el-table-column
              v-for="col in queryStore.result.columns"
              :key="col"
              :prop="col"
              :label="col"
              min-width="120"
              show-overflow-tooltip
            />
          </el-table>
        </div>
      </el-main>
    </el-container>

    <!-- 新增数据对话框 -->
    <el-dialog
      v-model="addDataDialogVisible"
      title="新增数据"
      width="600px"
    >
      <el-form :model="addDataForm" label-width="100px" ref="addDataFormRef">
        <template v-for="col in tableColumns" :key="col?.name">
          <el-form-item
            v-if="col && col.name"
            :label="col.name"
            :prop="col.name"
          >
            <el-input v-model="addDataForm[col.name]" :placeholder="col.type" />
            <div v-if="col.comment" style="font-size: 12px; color: #999; margin-top: 4px">{{ col.comment }}</div>
          </el-form-item>
        </template>
        <el-empty v-if="!tableColumns || tableColumns.length === 0" description="暂无表结构信息" />
      </el-form>
      <template #footer>
        <el-button @click="addDataDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleAddDataSubmit" :loading="submittingData">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useConnectionsStore } from '@/stores/connections'
import { useQueryStore } from '@/stores/query'
import * as monaco from 'monaco-editor'
import { format } from 'sql-formatter'
import { VideoPlay, Delete, Download, Document, MagicStick, Search, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'

const router = useRouter()
const route = useRoute()
const connectionsStore = useConnectionsStore()
const queryStore = useQueryStore()

const filteredTables = computed(() => {
  // 确保 tables 是有效数组
  if (!Array.isArray(queryStore.tables)) return []

  if (!tableFilter.value) return queryStore.tables
  return queryStore.tables.filter(t => t.name && t.name.toLowerCase().includes(tableFilter.value.toLowerCase()))
})

const currentConnectionId = ref(route.params.id as string || '')
const currentDatabase = ref('')
const currentSchema = ref('')
const tableFilter = ref('')
const resultSearch = ref('')
const selectedTable = ref('')
const editorContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

// 新增数据相关状态
const addDataDialogVisible = ref(false)
const addDataForm = ref<Record<string, any>>({})
const addDataFormRef = ref()
const submittingData = ref(false)
const tableColumns = ref<any[]>([])

// 监听 selectedTable 变化，加载表结构
watch(selectedTable, async (newVal) => {
  if (newVal && currentConnectionId.value) {
    // 先清空列，避免使用旧数据
    tableColumns.value = []
    await loadTableSchema()
  }
})

// 加载表结构
async function loadTableSchema() {
  if (!selectedTable.value || !currentConnectionId.value) return

  try {
    await queryStore.fetchTableSchema(currentConnectionId.value, selectedTable.value, currentDatabase.value, currentSchema.value)
    console.log('获取到的表结构:', queryStore.currentSchema)

    // 确保 columns 是有效数组，否则使用空数组
    const columns = queryStore.currentSchema?.columns
    console.log('列信息:', columns)

    if (Array.isArray(columns) && columns.length > 0) {
      tableColumns.value = columns
      console.log('设置 tableColumns 为:', columns.length, '列')
    } else {
      console.warn('未能获取到表结构列信息或列为空, currentSchema:', queryStore.currentSchema)
      tableColumns.value = []
    }
  } catch (e) {
    console.error('加载表结构失败:', e)
    ElNotification.error({
      title: '加载失败',
      message: '无法获取表结构',
      position: 'top-right'
    })
    tableColumns.value = []
  }
}

const filteredResults = computed(() => {
  if (!queryStore.result || !queryStore.result.rows) return []
  if (!resultSearch.value) return queryStore.result.rows
  if (!Array.isArray(queryStore.result.rows)) return []

  const search = resultSearch.value.toLowerCase()
  return queryStore.result.rows.filter(row => {
    if (!row) return false
    return Object.values(row).some(val =>
      String(val).toLowerCase().includes(search)
    )
  })
})

onMounted(async () => {
  // 清空历史的查询结果
  queryStore.clearResult()

  await connectionsStore.fetchConnections()

  if (currentConnectionId.value) {
    await handleConnectionChange(currentConnectionId.value)
  }

  initEditor()
})

onBeforeUnmount(() => {
  editor?.dispose()
})

const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'LIMIT', 'ORDER BY', 'GROUP BY',
  'INSERT INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE', 'CREATE TABLE',
  'DROP TABLE', 'ALTER TABLE', 'JOIN', 'LEFT JOIN', 'RIGHT JOIN',
  'INNER JOIN', 'ON', 'AS', 'DISTINCT', 'COUNT', 'SUM', 'AVG', 'MIN', 'MAX'
]

function initEditor() {
  if (!editorContainer.value) return

  // Register SQL Autocomplete
  monaco.languages.registerCompletionItemProvider('sql', {
    provideCompletionItems: (model, position) => {
      const suggestions: monaco.languages.CompletionItem[] = []
      const word = model.getWordUntilPosition(position)
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn
      }

      // Add SQL Keywords
      if (Array.isArray(SQL_KEYWORDS)) {
        SQL_KEYWORDS.forEach(keyword => {
          suggestions.push({
            label: keyword,
            kind: monaco.languages.CompletionItemKind.Keyword,
            insertText: keyword,
            range
          })
        })
      }

      // Add Table Names
      if (Array.isArray(queryStore.tables) && queryStore.tables.length > 0) {
        queryStore.tables.forEach(table => {
          if (table && table.name) {
            suggestions.push({
              label: table.name,
              kind: monaco.languages.CompletionItemKind.Struct,
              insertText: table.name,
              detail: 'Table',
              range
            })
          }
        })
      }

      return { suggestions }
    }
  })

  editor = monaco.editor.create(editorContainer.value, {
    value: '',
    language: 'sql',
    theme: 'vs-dark',
    minimap: { enabled: false },
    fontSize: 14,
    lineHeight: 24,
    padding: { top: 10 },
    scrollBeyondLastLine: false,
    automaticLayout: true
  })

  editor.addCommand(monaco.KeyCode.F5, () => {
    handleExecute()
  })
}

async function handleConnectionChange(id: string) {
  currentConnectionId.value = id
  currentDatabase.value = ''
  currentSchema.value = ''
  queryStore.currentSchemaName = ''
  queryStore.tables = [] // 清空表列表
  queryStore.currentSchema = null // 清空当前表结构
  selectedTable.value = '' // 清空选中的表
  if (id) {
    try {
      await queryStore.fetchDatabases(id)
      // 不再自动选择第一个数据库,让用户手动选择
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
  // 重置 schema
  currentSchema.value = ''
  queryStore.currentSchemaName = ''
  queryStore.schemas = []
  queryStore.tables = [] // 清空表列表
  queryStore.currentSchema = null // 清空当前表结构
  selectedTable.value = '' // 清空选中的表

  if (db) {
    try {
      // 先尝试加载 schemas（PostgreSQL 支持）
      await queryStore.fetchSchemas(currentConnectionId.value, db)

      // 如果有 schemas，等待用户选择；否则直接加载表
      if (queryStore.schemas.length === 0) {
        await loadTables(currentConnectionId.value, db)
        if (Array.isArray(queryStore.tables) && queryStore.tables.length > 0) {
          handleTableClick(queryStore.tables[0].name)
        }
      }
    } catch (e: any) {
      ElNotification.error({
        title: '加载失败',
        message: e.response?.data?.message || '加载 schemas 失败',
        position: 'top-right'
      })
    }
  }
}

async function handleSchemaChange(schema: string) {
  // 更新 store 中的 schema 名称
  queryStore.currentSchemaName = schema
  queryStore.tables = [] // 清空表列表
  queryStore.currentSchema = null // 清空当前表结构
  selectedTable.value = '' // 清空选中的表

  if (schema) {
    await loadTables(currentConnectionId.value, currentDatabase.value, schema)
    if (Array.isArray(queryStore.tables) && queryStore.tables.length > 0) {
      handleTableClick(queryStore.tables[0].name)
    }
  } else {
    // 如果清空了 schema，重新加载默认表
    await loadTables(currentConnectionId.value, currentDatabase.value)
    if (Array.isArray(queryStore.tables) && queryStore.tables.length > 0) {
      handleTableClick(queryStore.tables[0].name)
    }
  }
}

async function loadTables(id: string, database?: string, schema?: string) {
  try {
    await queryStore.fetchTables(id, database, schema)
  } catch (e: any) {
    ElNotification.error({
      title: '加载失败',
      message: e.response?.data?.message || '加载表列表失败',
      position: 'top-right'
    })
  }
}

async function handleExecute() {
  if (!currentConnectionId.value) {
    ElMessage.warning('请先选择连接')
    return
  }

  const query = editor?.getValue()
  if (!query || query.trim() === '') {
    ElMessage.warning('请输入 SQL 语句')
    return
  }

  try {
    await queryStore.executeQuery(currentConnectionId.value, query, {
      database: currentDatabase.value,
      schema: currentSchema.value
    })
  } catch (e: any) {
    ElNotification.error({
      title: '执行失败',
      message: e.response?.data?.message || e.message,
      position: 'top-right'
    })
  }
}

function handleClear() {
  editor?.setValue('')
  queryStore.clearResult()
}

function handleBeautify() {
  if (!editor) return
  const sql = editor.getValue()
  if (!sql) return

  try {
    const formatted = format(sql, {
      language: 'sql',
      keywordCase: 'upper'
    })
    editor.setValue(formatted)
  } catch (e: any) {
    ElMessage.error('格式化失败: ' + e.message)
  }
}

function handleTableClick(tableName: string) {
  selectedTable.value = tableName
  // 如果有 schema，在表名前添加 schema 前缀
  const tableRef = currentSchema.value ? `${currentSchema.value}.${tableName}` : tableName
  const sql = `SELECT * FROM ${tableRef} LIMIT 100;`
  editor?.setValue(sql)
}

function handleExport() {
  if (!currentConnectionId.value) {
    ElMessage.warning('请先选择连接')
    return
  }
  const query = editor?.getValue()
  router.push({
    path: `/export/${currentConnectionId.value}`,
    query: {
      sql: query,
      db: currentDatabase.value,
      table: selectedTable.value
    }
  })
}

async function handleAddData() {
  if (!currentConnectionId.value) {
    ElMessage.warning('请先选择连接')
    return
  }
  if (!selectedTable.value) {
    ElMessage.warning('请先选择要添加数据的表')
    return
  }

  console.log('=== handleAddData 开始 ===')
  console.log('当前连接:', currentConnectionId.value)
  console.log('当前数据库:', currentDatabase.value)
  console.log('当前 schema:', currentSchema.value)
  console.log('选择的表:', selectedTable.value)

  // 清空表单
  addDataForm.value = {}

  // 加载表结构（如果尚未加载）
  try {
    if (!queryStore.currentSchema || queryStore.currentSchema.table !== selectedTable.value) {
      console.log('需要加载表结构...')
      await loadTableSchema()
    } else {
      console.log('使用已缓存的表结构')
      // 使用缓存的表结构
      const columns = queryStore.currentSchema?.columns
      if (Array.isArray(columns)) {
        tableColumns.value = columns
        console.log('从缓存恢复列信息:', columns.length, '列')
      }
    }

    console.log('加载后的 tableColumns:', tableColumns.value)

    // 确保 tableColumns 是有效的数组
    if (Array.isArray(tableColumns.value) && tableColumns.value.length > 0) {
      tableColumns.value.forEach(col => {
        if (col && col.name) {
          addDataForm.value[col.name] = ''
        }
      })
      console.log('初始化表单数据:', addDataForm.value)
      console.log('准备打开对话框')
      addDataDialogVisible.value = true
    } else {
      console.error('tableColumns 无效:', tableColumns.value)
      ElMessage.warning('无法获取表结构信息，请稍后重试')
    }
  } catch (e) {
    console.error('加载表结构失败:', e)
    ElMessage.error('加载表结构失败')
  }
}

async function handleAddDataSubmit() {
  submittingData.value = true
  try {
    // 过滤空值
    const data: Record<string, any> = {}
    const keys = addDataForm.value ? Object.keys(addDataForm.value) : []
    if (keys.length === 0) {
      ElMessage.warning('表单数据为空')
      return
    }

    keys.forEach(key => {
      const val = addDataForm.value[key]
      if (val !== null && val !== undefined && val !== '') {
        data[key] = val
      }
    })

    if (Object.keys(data).length === 0) {
      ElMessage.warning('请填写至少一个字段')
      return
    }

    await queryStore.createRow(currentConnectionId.value, selectedTable.value, currentDatabase.value, data)
    ElMessage.success('添加成功')
    addDataDialogVisible.value = false

    // 重新执行查询
    handleExecute()
  } catch (e: any) {
    console.error('添加数据失败:', e)
    ElNotification.error({
      title: '添加失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    submittingData.value = false
  }
}
</script>

<style scoped>
.query-page {
  height: 100%;
}

.el-container {
  height: 100%;
}

.el-aside {
  border-right: 1px solid #dcdfe6;
  padding: 10px;
}

.connection-selector {
  margin-bottom: 10px;
}

.tables-list {
  max-height: calc(100vh - 150px);
  overflow-y: auto;
}

.table-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
}

.table-item:hover {
  background-color: #f5f7fa;
}

.el-main {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.editor-container {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
}

.editor-toolbar {
  display: flex;
  gap: 10px;
  padding: 10px;
  background-color: #f5f7fa;
  border-bottom: 1px solid #dcdfe6;
}

.monaco-editor {
  height: 300px;
}

.result-container {
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  overflow: hidden;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 15px;
  background-color: #f5f7fa;
  border-bottom: 1px solid #dcdfe6;
  font-weight: 500;
}

.result-info {
  font-size: 12px;
  color: #909399;
}
</style>
