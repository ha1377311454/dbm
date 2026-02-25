<template>
  <div class="tables-page">
    <el-page-header title="数据浏览" @back="() => $router.push('/connections')">
      <template #content>
        <el-select
          v-model="currentConnectionId"
          placeholder="选择连接"
          filterable
          @change="handleConnectionChange"
          style="width: 200px; margin-right: 10px"
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
      </template>
    </el-page-header>

    <div v-if="currentConnectionId" class="content">
      <el-row :gutter="20">
        <el-col :span="8">
          <el-card header="表列表">
            <el-table
              :data="queryStore.tables"
              @row-click="handleTableClick"
              highlight-current-row
              border
              max-height="500"
            >
              <el-table-column prop="name" label="表名" />
              <el-table-column prop="rows" label="行数" width="100" />
              <el-table-column prop="tableType" label="类型" width="100" />
            </el-table>
          </el-card>
        </el-col>

        <el-col :span="16">
          <el-card v-if="selectedTable">
            <template #header>
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>表结构</span>
                <el-button type="primary" size="small" @click="handleEditSchema">
                  <el-icon><Edit /></el-icon>
                  编辑表结构
                </el-button>
              </div>
            </template>
            <el-table :data="queryStore.currentSchema?.columns" border max-height="300">
              <el-table-column prop="name" label="列名" />
              <el-table-column prop="type" label="类型" width="150" />
              <el-table-column label="可空" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.nullable ? 'info' : 'danger'" size="small">
                    {{ row.nullable ? 'YES' : 'NO' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="key" label="键" width="80" />
              <el-table-column prop="comment" label="注释" />
            </el-table>
          </el-card>

          <el-card v-if="selectedTable && previewData" header="数据预览" style="margin-top: 20px">
            <template #header>
              <div class="card-header" style="display: flex; justify-content: space-between; align-items: center;">
                <div class="search-bar" style="display: flex; gap: 10px;">
                  <el-select v-model="searchCol" placeholder="选择列" size="small" style="width: 120px" clearable>
                    <el-option
                      v-for="col in columns"
                      :key="col.name"
                      :label="col.name"
                      :value="col.name"
                    />
                  </el-select>
                  <el-input
                    v-model="searchVal"
                    placeholder="搜索值..."
                    size="small"
                    style="width: 150px"
                    @keyup.enter="handleSearch"
                    clearable
                  />
                  <el-button type="primary" size="small" :icon="Search" @click="handleSearch">搜索</el-button>
                  <el-button size="small" @click="handleReset">重置</el-button>
                  <el-button size="small" @click="handleQuickExport">导出</el-button>
                </div>
              </div>
            </template>
            <el-table :data="previewData" stripe border max-height="400" v-loading="loading">
              <el-table-column
                v-for="col in columns"
                :key="col.name"
                :prop="col.name"
                :label="col.name"
                min-width="120"
                show-overflow-tooltip
              />
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="handleDelete(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 编辑数据对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="编辑数据"
      width="500px"
    >
      <el-form :model="editForm" label-width="100px" ref="formRef">
        <el-form-item
          v-for="col in columns"
          :key="col.name"
          :label="col.name"
          :prop="col.name"
        >
          <el-input v-model="editForm[col.name]" :placeholder="col.type" />
          <div v-if="col.comment" style="font-size: 12px; color: #999">{{ col.comment }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, reactive, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useConnectionsStore } from '@/stores/connections'
import { useQueryStore } from '@/stores/query'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { Search, Edit } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const connectionsStore = useConnectionsStore()
const queryStore = useQueryStore()

const currentConnectionId = ref(route.params.id as string || '')
const currentDatabase = ref('')
const selectedTable = ref('')
const previewData = ref<Record<string, any>[]>()
const loading = ref(false)

// Data Editing State
const dialogVisible = ref(false)
const currentRow = ref<Record<string, any>>({})
const editForm = ref<Record<string, any>>({})

// Search State
const searchCol = ref('')
const searchVal = ref('')

const columns = computed(() => {
  return queryStore.currentSchema?.columns || []
})

const connection = computed(() => 
  connectionsStore.connections.find(c => c.id === currentConnectionId.value)
)

const dbType = computed(() => connection.value?.type)

const primaryKeys = computed(() => {
  if (!queryStore.currentSchema) return []
  const pkIndex = queryStore.currentSchema.indexes.find(i => i.primary)
  return pkIndex ? pkIndex.columns : []
})

onMounted(async () => {
  await connectionsStore.fetchConnections()
  if (currentConnectionId.value) {
    await handleConnectionChange(currentConnectionId.value)
  }
})

async function handleConnectionChange(id: string) {
  currentConnectionId.value = id
  currentDatabase.value = ''
  selectedTable.value = ''
  previewData.value = []
  queryStore.currentSchema = null
  if (id) {
    try {
      await queryStore.fetchDatabases(id)
      // If there's a default DB in config or only one DB, pick it? 
      // For now let user pick.
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
  selectedTable.value = ''
  previewData.value = []
  queryStore.currentSchema = null
  if (db) {
    await loadTables(currentConnectionId.value, db)
  }
}

async function loadTables(id: string, database: string) {
  try {
    await queryStore.fetchTables(id, database)
  } catch (e: any) {
    ElNotification.error({
      title: '加载失败',
      message: e.response?.data?.message || '加载表列表失败',
      position: 'top-right'
    })
  }
}

async function handleTableClick(row: any) {
  selectedTable.value = row.name
  previewData.value = []
  await queryStore.fetchTableSchema(currentConnectionId.value, row.name, currentDatabase.value)
  await loadPreview(row.name)
}

async function loadPreview(tableName: string) {
  loading.value = true
  try {
    let query = ''
    if (dbType.value === 'mongodb') {
      // For MongoDB, we can just send the collection name
      // The backend will wrap it in a find command
      query = tableName
    } else {
      query = `SELECT * FROM ${tableName}`
      if (searchCol.value && searchVal.value) {
        query += ` WHERE \`${searchCol.value}\` LIKE '%${searchVal.value}%'`
      }
      query += ' LIMIT 100'
    }

    await queryStore.executeQuery(currentConnectionId.value, query, {
      database: currentDatabase.value
    })
    if (queryStore.result) {
      previewData.value = queryStore.result.rows
    }
  } catch (e: any) {
    ElNotification.error({
      title: '加载失败',
      message: e.response?.data?.message || '加载数据失败',
      position: 'top-right'
    })
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  if (selectedTable.value) {
    loadPreview(selectedTable.value)
  }
}

function handleReset() {
  searchCol.value = ''
  searchVal.value = ''
  if (selectedTable.value) {
    loadPreview(selectedTable.value)
  }
}

function handleQuickExport() {
  router.push(`/export/${currentConnectionId.value}`)
}

// Data Editing Functions
function getWhereClause(row: Record<string, any>) {
  if (primaryKeys.value.length > 0) {
    // Use Primary Key
    return primaryKeys.value.map(pk => `\`${pk}\` = '${row[pk]}'`).join(' AND ')
  } else {
    // Fallback: Use all columns (unsafe but better than nothing)
    return Object.keys(row).map(key => `\`${key}\` = '${row[key]}'`).join(' AND ')
  }
}

function handleEdit(row: any) {
  currentRow.value = { ...row }
  editForm.value = { ...row }
  dialogVisible.value = true
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm('确定要删除这条数据吗？', '提示', {
      type: 'warning'
    })

    const where = getWhereClause(row)
    await queryStore.deleteRow(currentConnectionId.value, selectedTable.value, currentDatabase.value, where)
    ElMessage.success('删除成功')
    loadPreview(selectedTable.value)
  } catch (e: any) {
    if (e !== 'cancel') {
      ElNotification.error({
        title: '删除失败',
        message: e.response?.data?.message || e.message || '未知错误',
        position: 'top-right'
      })
    }
  }
}

async function handleSubmit() {
  try {
    const data = { ...editForm.value }
    const where = getWhereClause(currentRow.value)

    await queryStore.updateRow(currentConnectionId.value, selectedTable.value, currentDatabase.value, data, where)
    ElMessage.success('更新成功')
    dialogVisible.value = false
    loadPreview(selectedTable.value)
  } catch (e: any) {
    ElNotification.error({
      title: '更新失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  }
}

// 跳转到表结构编辑器
function handleEditSchema() {
  router.push({
    path: '/schema-editor',
    query: {
      connectionId: currentConnectionId.value,
      database: currentDatabase.value,
      table: selectedTable.value
    }
  })
}
</script>

<style scoped>
.tables-page {
  padding: 20px;
}

.el-page-header {
  margin-bottom: 20px;
}

.content {
  margin-top: 20px;
}
</style>
