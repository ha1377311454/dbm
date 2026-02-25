<template>
  <div class="schema-editor-page">
    <el-page-header title="表结构编辑器" @back="() => $router.push('/tables')">
      <template #content>
        <el-breadcrumb separator="/">
          <el-breadcrumb-item>{{ connectionName }}</el-breadcrumb-item>
          <el-breadcrumb-item>{{ currentDatabase }}</el-breadcrumb-item>
          <el-breadcrumb-item>{{ currentTable }}</el-breadcrumb-item>
        </el-breadcrumb>
      </template>
    </el-page-header>

    <div class="content" v-loading="loading">
      <!-- 表信息卡片 -->
      <el-card class="table-info-card">
        <template #header>
          <div class="card-header">
            <span>表信息</span>
            <el-button type="primary" size="small" @click="showRenameDialog = true">
              <el-icon><Edit /></el-icon>
              重命名表
            </el-button>
          </div>
        </template>
        <el-descriptions :column="3" border>
          <el-descriptions-item label="表名">{{ currentTable }}</el-descriptions-item>
          <el-descriptions-item label="数据库">{{ currentDatabase }}</el-descriptions-item>
          <el-descriptions-item label="数据库类型">{{ dbType }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 列管理 -->
      <el-card class="columns-card">
        <template #header>
          <div class="card-header">
            <span>列管理</span>
            <el-button type="primary" size="small" @click="handleAddColumn">
              <el-icon><Plus /></el-icon>
              添加列
            </el-button>
          </div>
        </template>
        <el-table :data="columns" border stripe>
          <el-table-column prop="name" label="列名" width="150" />
          <el-table-column prop="type" label="类型" width="150" />
          <el-table-column label="可空" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.nullable ? 'success' : 'danger'" size="small">
                {{ row.nullable ? 'YES' : 'NO' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="defaultValue" label="默认值" width="120" />
          <el-table-column prop="key" label="键" width="80" />
          <el-table-column prop="extra" label="额外" width="120" />
          <el-table-column prop="comment" label="注释" min-width="150" />
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row, $index }">
              <el-button size="small" @click="handleEditColumn(row, $index)">编辑</el-button>
              <el-button size="small" @click="handleRenameColumn(row, $index)">重命名</el-button>
              <el-button size="small" type="danger" @click="handleDropColumn(row, $index)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 待执行操作 -->
      <el-card class="actions-card" v-if="pendingActions.length > 0">
        <template #header>
          <div class="card-header">
            <span>待执行操作 ({{ pendingActions.length }})</span>
            <div>
              <el-button size="small" @click="handleClearActions">清空</el-button>
              <el-button type="primary" size="small" @click="handleExecuteActions">
                <el-icon><Check /></el-icon>
                执行变更
              </el-button>
            </div>
          </div>
        </template>
        <el-timeline>
          <el-timeline-item
            v-for="(action, index) in pendingActions"
            :key="index"
            :timestamp="getActionTypeLabel(action.type)"
            placement="top"
          >
            <el-card>
              <div class="action-item">
                <div class="action-content">
                  {{ getActionDescription(action) }}
                </div>
                <el-button size="small" type="danger" @click="handleRemoveAction(index)">
                  移除
                </el-button>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-card>
    </div>

    <!-- 添加/编辑列对话框 -->
    <el-dialog
      v-model="columnDialogVisible"
      :title="columnDialogMode === 'add' ? '添加列' : '编辑列'"
      width="600px"
    >
      <el-form :model="columnForm" label-width="100px" ref="columnFormRef" :rules="columnRules">
        <el-form-item label="列名" prop="name">
          <el-input v-model="columnForm.name" placeholder="请输入列名" />
        </el-form-item>
        <el-form-item label="数据类型" prop="type">
          <el-select
            v-model="columnForm.type"
            placeholder="选择数据类型"
            style="width: 100%"
            filterable
          >
            <el-option-group
              v-for="group in dataTypes"
              :key="group.label"
              :label="group.label"
            >
              <el-option
                v-for="type in group.options"
                :key="type"
                :label="type"
                :value="type"
              />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="长度" v-if="needsLength">
          <el-input-number v-model="columnForm.length" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item label="精度" v-if="needsPrecision">
          <el-input-number v-model="columnForm.precision" :min="1" :max="65" style="width: 120px" />
          <span style="margin: 0 10px">小数位</span>
          <el-input-number v-model="columnForm.scale" :min="0" :max="30" style="width: 120px" />
        </el-form-item>
        <el-form-item label="可空">
          <el-switch v-model="columnForm.nullable" />
        </el-form-item>
        <el-form-item label="默认值">
          <el-input v-model="columnForm.defaultValue" placeholder="留空表示无默认值" />
        </el-form-item>
        <el-form-item label="自增" v-if="supportsAutoIncrement">
          <el-switch v-model="columnForm.autoIncrement" />
        </el-form-item>
        <el-form-item label="注释">
          <el-input v-model="columnForm.comment" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="columnDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleColumnSubmit">确定</el-button>
      </template>
    </el-dialog>
    <!-- 重命名列对话框 -->
    <el-dialog v-model="renameColumnDialogVisible" title="重命名列" width="500px">
      <el-form :model="renameColumnForm" label-width="100px">
        <el-form-item label="原列名">
          <el-input v-model="renameColumnForm.oldName" disabled />
        </el-form-item>
        <el-form-item label="新列名">
          <el-input v-model="renameColumnForm.newName" placeholder="请输入新列名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="renameColumnDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleRenameColumnSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 重命名表对话框 -->
    <el-dialog v-model="showRenameDialog" title="重命名表" width="500px">
      <el-form :model="renameTableForm" label-width="100px">
        <el-form-item label="原表名">
          <el-input v-model="currentTable" disabled />
        </el-form-item>
        <el-form-item label="新表名">
          <el-input v-model="renameTableForm.newName" placeholder="请输入新表名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRenameDialog = false">取消</el-button>
        <el-button type="primary" @click="handleRenameTableSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Edit, Plus, Check } from '@element-plus/icons-vue'
import { api } from '@/api'
import { useConnectionsStore } from '@/stores/connections'
import type {
  ColumnInfo,
  AlterTableAction,
  AlterActionType,
  ColumnDef
} from '@/types'

const route = useRoute()
const connectionsStore = useConnectionsStore()

// 路由参数
const connectionId = ref(route.query.connectionId as string)
const currentDatabase = ref(route.query.database as string)
const currentTable = ref(route.query.table as string)

// 数据
const loading = ref(false)
const columns = ref<ColumnInfo[]>([])
const pendingActions = ref<AlterTableAction[]>([])

// 连接信息
const connection = computed(() => 
  connectionsStore.connections.find(c => c.id === connectionId.value)
)
const connectionName = computed(() => connection.value?.name || '')
const dbType = computed(() => connection.value?.type || '')

// 数据库特性支持
const supportsAutoIncrement = computed(() => {
  return ['mysql', 'sqlite'].includes(dbType.value)
})

// 列对话框
const columnDialogVisible = ref(false)
const columnDialogMode = ref<'add' | 'edit'>('add')
const columnFormRef = ref<FormInstance>()
const editingColumnIndex = ref(-1)
const columnForm = reactive<ColumnDef>({
  name: '',
  type: '',
  length: undefined,
  precision: undefined,
  scale: undefined,
  nullable: true,
  defaultValue: '',
  autoIncrement: false,
  comment: ''
})

const columnRules: FormRules = {
  name: [{ required: true, message: '请输入列名', trigger: 'blur' }],
  type: [{ required: true, message: '请选择数据类型', trigger: 'change' }]
}

// 数据类型选项
const dataTypes = computed(() => {
  const types: { label: string; options: string[] }[] = []
  
  if (dbType.value === 'mysql') {
    types.push(
      { label: '数值类型', options: ['TINYINT', 'SMALLINT', 'MEDIUMINT', 'INT', 'BIGINT', 'FLOAT', 'DOUBLE', 'DECIMAL'] },
      { label: '字符串类型', options: ['CHAR', 'VARCHAR', 'TEXT', 'MEDIUMTEXT', 'LONGTEXT'] },
      { label: '日期时间', options: ['DATE', 'TIME', 'DATETIME', 'TIMESTAMP', 'YEAR'] },
      { label: '其他', options: ['BLOB', 'JSON', 'ENUM', 'SET'] }
    )
  } else if (dbType.value === 'postgresql') {
    types.push(
      { label: '数值类型', options: ['SMALLINT', 'INTEGER', 'BIGINT', 'DECIMAL', 'NUMERIC', 'REAL', 'DOUBLE PRECISION'] },
      { label: '字符串类型', options: ['CHAR', 'VARCHAR', 'TEXT'] },
      { label: '日期时间', options: ['DATE', 'TIME', 'TIMESTAMP', 'INTERVAL'] },
      { label: '其他', options: ['BOOLEAN', 'JSON', 'JSONB', 'UUID', 'BYTEA'] }
    )
  } else if (dbType.value === 'sqlite') {
    types.push(
      { label: '基本类型', options: ['INTEGER', 'REAL', 'TEXT', 'BLOB', 'NUMERIC'] }
    )
  } else if (dbType.value === 'clickhouse') {
    types.push(
      { label: '数值类型', options: ['UInt8', 'UInt16', 'UInt32', 'UInt64', 'Int8', 'Int16', 'Int32', 'Int64', 'Float32', 'Float64'] },
      { label: '字符串类型', options: ['String', 'FixedString'] },
      { label: '日期时间', options: ['Date', 'DateTime', 'DateTime64'] },
      { label: '其他', options: ['UUID', 'IPv4', 'IPv6'] }
    )
  } else if (dbType.value === 'mongodb') {
    types.push(
      { label: '基本类型', options: ['String', 'Number', 'Double', 'Int32', 'Int64', 'Decimal128', 'Boolean', 'Date', 'ObjectId', 'Null', 'Regex', 'JavaScript', 'Symbol', 'Timestamp'] },
      { label: '复杂类型', options: ['Object', 'Array', 'Binary', 'MinKey', 'MaxKey'] }
    )
  }
  
  return types
})

const needsLength = computed(() => {
  const type = columnForm.type.toUpperCase()
  return ['VARCHAR', 'CHAR', 'FIXEDSTRING'].includes(type)
})

const needsPrecision = computed(() => {
  const type = columnForm.type.toUpperCase()
  return ['DECIMAL', 'NUMERIC'].includes(type)
})

// 重命名列对话框
const renameColumnDialogVisible = ref(false)
const renameColumnForm = reactive({
  oldName: '',
  newName: '',
  columnDef: null as ColumnInfo | null
})

// 重命名表对话框
const showRenameDialog = ref(false)
const renameTableForm = reactive({
  newName: ''
})

// 加载表结构
const loadTableSchema = async () => {
  loading.value = true
  try {
    const res = await api.getTableSchema(connectionId.value, currentTable.value, currentDatabase.value)
    if (res.code === 200) {
      columns.value = res.data.columns
    }
  } catch (error: any) {
    ElMessage.error('加载表结构失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

// 添加列
const handleAddColumn = () => {
  columnDialogMode.value = 'add'
  Object.assign(columnForm, {
    name: '',
    type: '',
    length: undefined,
    precision: undefined,
    scale: undefined,
    nullable: true,
    defaultValue: '',
    autoIncrement: false,
    comment: ''
  })
  columnDialogVisible.value = true
}

// 编辑列
const handleEditColumn = (row: ColumnInfo, index: number) => {
  columnDialogMode.value = 'edit'
  editingColumnIndex.value = index
  Object.assign(columnForm, {
    name: row.name,
    type: row.type,
    nullable: row.nullable,
    defaultValue: row.defaultValue,
    comment: row.comment,
    autoIncrement: row.extra?.includes('auto_increment') || false
  })
  columnDialogVisible.value = true
}

// 重命名列
const handleRenameColumn = (row: ColumnInfo, index: number) => {
  renameColumnForm.oldName = row.name
  renameColumnForm.newName = row.name
  renameColumnForm.columnDef = row
  renameColumnDialogVisible.value = true
}

// 删除列
const handleDropColumn = async (row: ColumnInfo) => {
  // SQLite 不支持删除列
  try {
    await ElMessageBox.confirm(`确定要删除列 "${row.name}" 吗？此操作不可恢复！`, '警告', {
      type: 'warning'
    })

    pendingActions.value.push({
      type: 'DROP_COLUMN' as AlterActionType,
      oldName: row.name
    })

    ElMessage.success('已添加到待执行操作')
  } catch {
    // 用户取消
  }
}

// 提交列表单
const handleColumnSubmit = async () => {
  if (!columnFormRef.value) return

  await columnFormRef.value.validate((valid) => {
    if (valid) {
      const action: AlterTableAction = {
        type: columnDialogMode.value === 'add' ? 'ADD_COLUMN' as AlterActionType : 'MODIFY_COLUMN' as AlterActionType,
        column: { ...columnForm }
      }

      pendingActions.value.push(action)
      columnDialogVisible.value = false
      ElMessage.success('已添加到待执行操作')
    }
  })
}

// 提交重命名列
const handleRenameColumnSubmit = () => {
  if (!renameColumnForm.newName) {
    ElMessage.warning('请输入新列名')
    return
  }

  if (renameColumnForm.newName === renameColumnForm.oldName) {
    ElMessage.warning('新列名与原列名相同')
    return
  }

  const action: AlterTableAction = {
    type: 'RENAME_COLUMN' as AlterActionType,
    oldName: renameColumnForm.oldName,
    newName: renameColumnForm.newName,
    column: renameColumnForm.columnDef ? {
      name: renameColumnForm.newName,
      type: renameColumnForm.columnDef.type,
      nullable: renameColumnForm.columnDef.nullable
    } : undefined
  }

  pendingActions.value.push(action)
  renameColumnDialogVisible.value = false
  ElMessage.success('已添加到待执行操作')
}

// 提交重命名表
const handleRenameTableSubmit = async () => {
  if (!renameTableForm.newName) {
    ElMessage.warning('请输入新表名')
    return
  }

  if (renameTableForm.newName === currentTable.value) {
    ElMessage.warning('新表名与原表名相同')
    return
  }

  try {
    await ElMessageBox.confirm(`确定要将表 "${currentTable.value}" 重命名为 "${renameTableForm.newName}" 吗？`, '确认', {
      type: 'warning'
    })

    loading.value = true
    const res = await api.renameTable(
      connectionId.value,
      currentTable.value,
      currentDatabase.value,
      { newName: renameTableForm.newName }
    )

    if (res.code === 200) {
      ElMessage.success('表重命名成功')
      currentTable.value = renameTableForm.newName
      showRenameDialog.value = false
    } else {
      ElMessage.error(res.message)
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('重命名失败: ' + error.message)
    }
  } finally {
    loading.value = false
  }
}

// 执行所有待执行操作
const handleExecuteActions = async () => {
  if (pendingActions.value.length === 0) {
    ElMessage.warning('没有待执行的操作')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要执行 ${pendingActions.value.length} 个表结构变更操作吗？此操作不可恢复！`,
      '警告',
      { type: 'warning' }
    )

    loading.value = true
    const res = await api.alterTable(
      connectionId.value,
      currentTable.value,
      currentDatabase.value,
      {
        database: currentDatabase.value,
        table: currentTable.value,
        actions: pendingActions.value
      }
    )

    if (res.code === 200) {
      ElMessage.success('表结构修改成功')
      pendingActions.value = []
      await loadTableSchema()
    } else {
      ElMessage.error(res.message)
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('执行失败: ' + error.message)
    }
  } finally {
    loading.value = false
  }
}

// 清空待执行操作
const handleClearActions = () => {
  pendingActions.value = []
  ElMessage.success('已清空待执行操作')
}

// 移除单个操作
const handleRemoveAction = (index: number) => {
  pendingActions.value.splice(index, 1)
  ElMessage.success('已移除操作')
}

// 获取操作类型标签
const getActionTypeLabel = (type: AlterActionType): string => {
  const labels: Record<string, string> = {
    ADD_COLUMN: '添加列',
    DROP_COLUMN: '删除列',
    MODIFY_COLUMN: '修改列',
    RENAME_COLUMN: '重命名列',
    ADD_INDEX: '添加索引',
    DROP_INDEX: '删除索引',
    RENAME_TABLE: '重命名表'
  }
  return labels[type] || type
}

// 获取操作描述
const getActionDescription = (action: AlterTableAction): string => {
  switch (action.type) {
    case 'ADD_COLUMN':
      return `添加列: ${action.column?.name} (${action.column?.type})`
    case 'DROP_COLUMN':
      return `删除列: ${action.oldName}`
    case 'MODIFY_COLUMN':
      return `修改列: ${action.column?.name} (${action.column?.type})`
    case 'RENAME_COLUMN':
      return `重命名列: ${action.oldName} → ${action.newName}`
    case 'ADD_INDEX':
      return `添加索引: ${action.index?.name} (${action.index?.columns.join(', ')})`
    case 'DROP_INDEX':
      return `删除索引: ${action.oldName}`
    default:
      return '未知操作'
  }
}

onMounted(() => {
  loadTableSchema()
})
</script>

<style scoped>
.schema-editor-page {
  padding: 20px;
}

.content {
  margin-top: 20px;
}

.table-info-card,
.columns-card,
.actions-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.action-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.action-content {
  flex: 1;
  font-size: 14px;
}

:deep(.el-timeline-item__timestamp) {
  font-weight: bold;
  color: #409eff;
}
</style>
