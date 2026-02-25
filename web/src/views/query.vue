<template>
  <div class="query-page">
    <el-container>
      <el-aside :width="asideWidth + 'px'">
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
          <el-input
            v-model="treeFilterText"
            placeholder="搜索节点..."
            size="small"
            style="margin-bottom: 10px"
            clearable
          />
        </div>
        <el-divider style="margin: 10px 0;" />
        <div class="tree-container">
          <el-tree
            v-if="currentConnectionId"
            :key="currentConnectionId"
            ref="dbTreeRef"
            :props="treeProps"
            :load="loadTreeNode"
            lazy
            :filter-node-method="filterNode"
            @node-click="handleNodeClick"
            highlight-current
            node-key="id"
          >
            <template #default="{ node, data }">
              <span class="custom-tree-node" style="display: flex; align-items: center; gap: 6px; overflow: hidden;">
                <el-icon v-if="data.type === 'database'" color="#409EFC"><Coin /></el-icon>
                <el-icon v-else-if="data.type === 'schema'" color="#E6A23C"><Folder /></el-icon>
                <el-icon v-else-if="data.type === 'folder'" color="#909399"><FolderOpened /></el-icon>
                <el-icon v-else-if="data.type === 'view'" color="#67C23A"><Reading /></el-icon>
                <el-icon v-else-if="data.type === 'procedure'" color="#F56C6C"><Setting /></el-icon>
                <el-icon v-else-if="data.type === 'function'" color="#E6A23C"><Operation /></el-icon>
                <el-icon v-else><Document /></el-icon>
                <span class="tree-node-label" :title="node.label" style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ node.label }}</span>
              </span>
            </template>
          </el-tree>
        </div>
      </el-aside>
      <div class="resizer" @mousedown="startResize"></div>

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
                <span v-if="queryStore.result.message" style="margin-right: 15px; color: #409EFF; font-weight: bold;">
                  {{ queryStore.result.message }}
                </span>
                耗时: {{ queryStore.result.timeCost }}ms |
                <template v-if="queryStore.result.columns && queryStore.result.columns.length > 0">
                  行数: {{ filteredResults.length }} / {{ queryStore.result.total }}
                </template>
                <template v-else>
                  影响行数: {{ queryStore.result.rowsAffected }}
                </template>
              </span>
            </div>
          </div>
          <el-table
            v-if="queryStore.result.columns && queryStore.result.columns.length > 0"
            :data="filteredResults"
            :default-sort="{ prop: 'id', order: 'ascending' }"
            stripe
            border
            height="100%"
          >
            <el-table-column
              v-for="col in queryStore.result.columns"
              :key="col"
              :prop="col"
              :label="col"
              min-width="120"
              show-overflow-tooltip
            >
              <template #default="{ row }">
                {{ formatCellValue(row[col]) }}
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="empty-result">
            <el-empty :description="queryStore.result.message || '执行成功，无返回数据'" />
          </div>
        </div>
      </el-main>
    </el-container>

    <!-- 新增数据对话框 -->
    <el-dialog
      v-model="addDataDialogVisible"
      title="新增数据"
      width="600px"
    >
      <el-form :model="addDataForm" label-width="100px">
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
import { VideoPlay, Delete, Download, Document, MagicStick, Search, Plus, Coin, Folder, FolderOpened, Reading, Setting, Operation } from '@element-plus/icons-vue'
import { ElMessage, ElNotification } from 'element-plus'
import type { ElTree } from 'element-plus'
import { api } from '@/api'

const router = useRouter()
const route = useRoute()
const connectionsStore = useConnectionsStore()
const queryStore = useQueryStore()

const currentConnectionId = ref(route.params.id as string || '')
const currentDatabase = ref('')
const currentSchema = ref('')
// 侧边栏宽度
const asideWidth = ref(250)
const isResizing = ref(false)
// tree 相关的 state
const dbTreeRef = ref<InstanceType<typeof ElTree>>()
const treeFilterText = ref('')
const treeProps = {
  label: 'label',
  isLeaf: 'isLeaf'
}

interface TreeNode {
  id: string
  label: string
  type: 'database' | 'schema' | 'table' | 'view' | 'folder' | 'procedure' | 'function'
  parentType?: 'database' | 'schema'
  database?: string
  schema?: string
  isLeaf?: boolean
}

watch(treeFilterText, (val) => {
  dbTreeRef.value?.filter(val)
})

const filterNode = (value: string, data: TreeNode) => {
  if (!value) return true
  return data.label.toLowerCase().includes(value.toLowerCase())
}

const resultSearch = ref('')
const selectedTable = ref('')

const connection = computed(() => 
  connectionsStore.connections.find(c => c.id === currentConnectionId.value)
)

const dbType = computed(() => connection.value?.type)

watch(dbType, (newType) => {
  if (editor) {
    const language = newType === 'mongodb' ? 'json' : 'sql'
    monaco.editor.setModelLanguage(editor.getModel()!, language)
  }
})

// 拖拽调整左侧栏宽度
function startResize(e: MouseEvent) {
  isResizing.value = true
  e.preventDefault()
}

function handleResize(e: MouseEvent) {
  if (!isResizing.value) return

  const container = document.querySelector('.query-page') as HTMLElement
  if (!container) return

  const containerRect = container.getBoundingClientRect()
  const newWidth = e.clientX - containerRect.left

  // 限制最小和最大宽度
  const minWidth = 180
  const maxWidth = 500
  const clampedWidth = Math.max(minWidth, Math.min(maxWidth, newWidth))

  asideWidth.value = clampedWidth
}

function stopResize() {
  isResizing.value = false
}
const editorContainer = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

// 新增数据相关状态
const addDataDialogVisible = ref(false)
const addDataForm = ref<Record<string, any>>({})
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

  // 添加全局事件监听用于拖拽
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
})

onBeforeUnmount(() => {
  editor?.dispose()
  // 移除全局事件监听
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
})

async function loadTreeNode(node: any, resolve: (data: TreeNode[]) => void) {
  if (!currentConnectionId.value) {
    return resolve([])
  }

  if (node.level === 0) {
    try {
      const res = await api.getDatabases(currentConnectionId.value)
      if (res.code === 0 && res.data) {
        queryStore.databases = res.data
        const nodes: TreeNode[] = res.data.map(db => ({
          id: `db_${db}`,
          label: db,
          type: 'database',
          database: db,
          isLeaf: false
        }))
        resolve(nodes)
      } else {
        resolve([])
      }
    } catch (e) {
      resolve([])
    }
    return
  }

  const data = node.data as TreeNode
  if (data.type === 'database') {
    try {
      const db = data.database!
      // 先尝试加载 schemas
      const schemaRes = await api.getSchemas(currentConnectionId.value, db)
      if (schemaRes.code === 0 && schemaRes.data && schemaRes.data.length > 0) {
        queryStore.schemas = schemaRes.data
        const nodes: TreeNode[] = schemaRes.data.map(schema => ({
          id: `schema_${db}_${schema}`,
          label: schema,
          type: 'schema',
          database: db,
          schema: schema,
          isLeaf: false
        }))
        resolve(nodes)
      } else {
        // 无 schema 则直接返回虚拟目录节点 (Tables, Views, Procedures, Functions)
        const folderNodes: TreeNode[] = [
          { id: `folder_tables_${db}`, label: 'Tables', type: 'folder', parentType: 'database', database: db, isLeaf: false },
          { id: `folder_views_${db}`, label: 'Views', type: 'folder', parentType: 'database', database: db, isLeaf: false },
          { id: `folder_procedures_${db}`, label: 'Procedures', type: 'folder', parentType: 'database', database: db, isLeaf: false },
          { id: `folder_functions_${db}`, label: 'Functions', type: 'folder', parentType: 'database', database: db, isLeaf: false }
        ]
        resolve(folderNodes)
      }
    } catch (e) {
      resolve([])
    }
    return
  }

  if (data.type === 'schema') {
    try {
      const db = data.database!
      const schema = data.schema!
      // 有 schema 下同样返回虚拟目录节点
      const folderNodes: TreeNode[] = [
        { id: `folder_tables_${db}_${schema}`, label: 'Tables', type: 'folder', parentType: 'schema', database: db, schema: schema, isLeaf: false },
        { id: `folder_views_${db}_${schema}`, label: 'Views', type: 'folder', parentType: 'schema', database: db, schema: schema, isLeaf: false },
        { id: `folder_procedures_${db}_${schema}`, label: 'Procedures', type: 'folder', parentType: 'schema', database: db, schema: schema, isLeaf: false },
        { id: `folder_functions_${db}_${schema}`, label: 'Functions', type: 'folder', parentType: 'schema', database: db, schema: schema, isLeaf: false }
      ]
      resolve(folderNodes)
    } catch (e) {
      resolve([])
    }
    return
  }

  if (data.type === 'folder') {
    try {
      const db = data.database!
      const schema = data.schema

      if (data.label === 'Tables') {
        const tableRes = await api.getTables(currentConnectionId.value, db, schema)
        if (tableRes.code === 0 && tableRes.data) {
          queryStore.tables = tableRes.data
          const nodes: TreeNode[] = tableRes.data.map(t => ({
            id: `table_${db}_${schema || ''}_${t.name}`,
            label: t.name,
            type: 'table',
            database: db,
            schema: schema,
            isLeaf: true
          }))
          resolve(nodes)
        } else {
          resolve([])
        }
      } else if (data.label === 'Views') {
        const viewRes = await api.getViews(currentConnectionId.value, db, schema)
        if (viewRes.code === 0 && viewRes.data) {
          const nodes: TreeNode[] = viewRes.data.map(v => ({
            id: `view_${db}_${schema || ''}_${v.name}`,
            label: v.name,
            type: 'view',
            database: db,
            schema: schema,
            isLeaf: true
          }))
          resolve(nodes)
        } else {
          resolve([])
        }
      } else if (data.label === 'Procedures') {
        const procRes = await api.getProcedures(currentConnectionId.value, db, schema)
        if (procRes.code === 0 && procRes.data) {
          const nodes: TreeNode[] = procRes.data.map(p => ({
            id: `proc_${db}_${schema || ''}_${p.name}`,
            label: p.name,
            type: 'procedure',
            database: db,
            schema: schema,
            isLeaf: true
          }))
          resolve(nodes)
        } else {
          resolve([])
        }
      } else if (data.label === 'Functions') {
        const funcRes = await api.getFunctions(currentConnectionId.value, db, schema)
        if (funcRes.code === 0 && funcRes.data) {
          const nodes: TreeNode[] = funcRes.data.map(f => ({
            id: `func_${db}_${schema || ''}_${f.name}`,
            label: f.name,
            type: 'function',
            database: db,
            schema: schema,
            isLeaf: true
          }))
          resolve(nodes)
        } else {
          resolve([])
        }
      } else {
        resolve([])
      }
    } catch (e) {
      resolve([])
    }
    return
  }

  resolve([])
}

function handleNodeClick(data: TreeNode) {
  if (data.type === 'table' || data.type === 'view' || data.type === 'procedure' || data.type === 'function') {
    currentDatabase.value = data.database || ''
    currentSchema.value = data.schema || ''
    queryStore.currentSchemaName = data.schema || ''
    
    if (data.type === 'table') {
      handleTableClick(data.label)
    } else if (data.type === 'view') {
      handleViewClick(data.label)
    } else if (data.type === 'procedure') {
      handleRoutineClick(data.label, 'PROCEDURE')
    } else if (data.type === 'function') {
      handleRoutineClick(data.label, 'FUNCTION')
    }
  }
}

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
    language: dbType.value === 'mongodb' ? 'json' : 'sql',
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
  // tree 会因为 :key="currentConnectionId" 自动重新 render 和 load
}

async function handleExecute() {
  if (!currentConnectionId.value) {
    ElMessage.warning('请先选择连接')
    return
  }

  const query = editor?.getValue()
  if (!query || query.trim() === '') {
    ElMessage.warning(dbType.value === 'mongodb' ? '请输入查询命令' : '请输入 SQL 语句')
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
    if (dbType.value === 'mongodb') {
      const obj = JSON.parse(sql)
      editor.setValue(JSON.stringify(obj, null, 2))
    } else {
      const formatted = format(sql, {
        language: 'sql',
        keywordCase: 'upper'
      })
      editor.setValue(formatted)
    }
  } catch (e: any) {
    ElMessage.error('格式化失败: ' + e.message)
  }
}

function handleTableClick(tableName: string) {
  selectedTable.value = tableName
  const tableRef = currentSchema.value ? `${currentSchema.value}.${tableName}` : tableName
  
  if (dbType.value === 'mongodb') {
    const template = {
      find: tableName,
      filter: {},
      limit: 100
    }
    editor?.setValue(JSON.stringify(template, null, 2))
  } else {
    const sql = `SELECT * FROM ${tableRef} LIMIT 100;`
    editor?.setValue(sql)
  }
}

async function handleViewClick(viewName: string) {
  selectedTable.value = viewName
  try {
    const res = await api.getViewDefinition(currentConnectionId.value, viewName, currentDatabase.value, currentSchema.value)
    if (res.code === 0 && res.data) {
      editor?.setValue(res.data)
      handleBeautify()
    } else {
      ElMessage.warning('未能获取到视图定义: ' + (res.message || '未知错误'))
    }
  } catch (e: any) {
    ElMessage.error(e.message || '获取视图定义失败')
  }
}

async function handleRoutineClick(routineName: string, routineType: 'PROCEDURE'|'FUNCTION') {
  selectedTable.value = routineName
  try {
    const res = await api.getRoutineDefinition(currentConnectionId.value, routineName, routineType, currentDatabase.value, currentSchema.value)
    if (res.code === 0 && res.data) {
      editor?.setValue(res.data)
      handleBeautify()
    } else {
      ElMessage.warning('未能获取到定义: ' + (res.message || '可能不支持或不存在该对象'))
    }
  } catch (e: any) {
    ElMessage.error(e.message || '获取定义失败')
  }
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

function formatCellValue(val: any) {
  if (val === null || val === undefined) return ''
  if (typeof val === 'object') {
    return JSON.stringify(val)
  }
  return val
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
  overflow: hidden;
  flex-shrink: 0;
}

.resizer {
  width: 4px;
  cursor: col-resize;
  background-color: #dcdfe6;
  transition: background-color 0.2s;
  flex-shrink: 0;
}

.resizer:hover,
.resizer:active {
  background-color: #409eff;
}

.connection-selector {
  margin-bottom: 10px;
}

.tree-container {
  height: calc(100vh - 120px);
  overflow-y: auto;
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  font-size: 14px;
  padding-right: 8px;
}

.el-main {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  flex: 1;
  min-width: 0;
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
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 200px;
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
