<template>
  <div class="connections-page">
    <el-page-header title="连接管理">
      <template #content>
        <div class="header-actions">
          <el-input
            v-model="filterText"
            placeholder="搜索连接或分组..."
            :prefix-icon="Search"
            style="width: 200px"
            clearable
          />
          <el-divider direction="vertical" />
          <el-button-group>
            <el-button :icon="Expand" @click="handleExpandAll" title="全部展开" />
            <el-button :icon="Fold" @click="handleCollapseAll" title="全部收起" />
          </el-button-group>
          <el-divider direction="vertical" />
          <el-button 
            v-if="selectedConnections.length > 0" 
            type="success" 
            :icon="Download" 
            @click="handleExportConnections"
          >
            导出选中 ({{ selectedConnections.length }})
          </el-button>
          <el-button :icon="Upload" @click="triggerFileInput">
            导入连接
          </el-button>
          <input 
            ref="fileInput" 
            type="file" 
            accept=".json" 
            @change="handleFileImport" 
            style="display: none"
          />
          <el-divider direction="vertical" />
          <el-button type="primary" :icon="Plus" @click="handleCreateConnection">
            新建连接
          </el-button>
          <el-button :icon="FolderAdd" @click="handleCreateGroup">
            新建分组
          </el-button>
        </div>
      </template>
    </el-page-header>

    <div class="tree-container" v-loading="connectionsStore.loading">
      <el-tree
        ref="treeRef"
        :data="treeData"
        :props="defaultProps"
        node-key="id"
        :filter-node-method="filterNode"
        default-expand-all
        draggable
        :allow-drop="allowDrop"
        @node-drop="handleDrop"
        class="connection-tree"
      >
        <template #default="{ node, data }">
          <span class="custom-tree-node" @dblclick="handleNodeDblClick(data)">
            <span class="node-label">
              <el-checkbox 
                v-if="data.type === 'connection'" 
                :model-value="isConnectionSelected(data.data.id)"
                @change="handleConnectionSelect(data.data.id, $event)"
                @click.stop
                style="margin-right: 8px"
              />
              <el-icon v-if="data.type === 'group'"><Folder /></el-icon>
              <component v-else :is="getDatabaseIcon(data.data.type)" class="db-icon" />
              {{ node.label }}
              <el-tag 
                v-if="data.type === 'connection'" 
                size="small" 
                :type="getDatabaseTypeColor(data.data.type)"
                class="db-type-tag"
              >
                {{ getDatabaseTypeName(data.data.type) }}
              </el-tag>
              <el-tag
                v-if="data.type === 'connection' && data.data.connected"
                size="small"
                type="success"
                effect="dark"
                class="status-tag"
              >
                在线
              </el-tag>
              <el-tag
                v-if="data.type === 'connection' && data.data.monitoringEnabled"
                size="small"
                type="warning"
                effect="plain"
                class="status-tag"
              >
                监控中
              </el-tag>
            </span>
            <span class="node-actions">
              <template v-if="data.type === 'group'">
                <el-button link type="primary" :icon="Edit" @click.stop="handleEditGroup(data)"></el-button>
                <el-button link type="danger" :icon="Delete" @click.stop="handleDeleteGroup(data)"></el-button>
              </template>
              <template v-else>
                <el-button link type="primary" @click.stop="handleManage(data.data)">管理</el-button>
                <el-button link type="primary" @click.stop="handleQuery(data.data)">查询</el-button>
                <el-dropdown trigger="click" @click.stop>
                  <el-button link type="primary" :icon="MoreFilled"></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="handleTest(data.data)">测试连接</el-dropdown-item>
                      <el-dropdown-item @click="handleEdit(data.data)">编辑配置</el-dropdown-item>
                      <el-dropdown-item @click="handleToggleMonitoring(data.data)">
                        {{ data.data.monitoringEnabled ? '关闭监控' : '开启监控' }}
                      </el-dropdown-item>
                      <el-dropdown-item @click="handleConnectionToggle(data.data, !data.data.connected)">
                        {{ data.data.connected ? '断开连接' : '建立连接' }}
                      </el-dropdown-item>
                      <el-dropdown-item divided type="danger" @click="handleDelete(data.data)">
                        删除连接
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </span>
          </span>
        </template>
      </el-tree>
    </div>

    <!-- 连接表单对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      :title="editingConnection ? '编辑连接' : '新建连接'"
      width="600px"
    >
      <el-form :model="formData" :rules="formRules" label-width="100px">
        <el-form-item label="所属分组">
          <el-select v-model="formData.groupId" placeholder="根目录" clearable>
            <el-option
              v-for="group in connectionsStore.groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="连接名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入连接名称" />
        </el-form-item>
        <el-form-item label="数据库类型" prop="type">
          <el-select v-model="formData.type" placeholder="选择数据库类型">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="SQLite" value="sqlite" />
            <el-option label="ClickHouse" value="clickhouse" />
            <el-option label="KingBase" value="kingbase" />
            <el-option label="达梦数据库" value="dm" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="formData.type === 'clickhouse'" label="协议">
          <el-select v-model="formData.params.protocol" @change="handleProtocolChange">
            <el-option label="Native (TCP)" value="clickhouse" />
            <el-option label="HTTP" value="http" />
            <el-option label="HTTPS" value="https" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="formData.type !== 'sqlite'" label="主机" prop="host">
          <el-input v-model="formData.host" placeholder="localhost" />
        </el-form-item>
        <el-form-item v-if="formData.type !== 'sqlite'" label="端口" prop="port">
          <el-input-number v-model="formData.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item v-if="formData.type !== 'sqlite'" label="用户名" prop="username">
          <el-input v-model="formData.username" />
        </el-form-item>
        <el-form-item v-if="formData.type !== 'sqlite'" label="密码" prop="password">
          <el-input v-model="formData.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="数据库" prop="database">
          <el-input
            v-if="formData.type === 'sqlite'"
            v-model="formData.database"
            placeholder="/path/to/database.db"
          />
          <el-input v-else v-model="formData.database" />
        </el-form-item>
        <el-form-item label="启用监控">
          <el-switch
            v-model="formData.monitoringEnabled"
            active-text="开启"
            inactive-text="关闭"
          />
          <span class="form-item-tip">开启后将通过 Prometheus 采集数据库运行指标</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button @click="handleTestConfig" :loading="testing">
          测试连接
        </el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 分组表单对话框 -->
    <el-dialog
      v-model="showGroupDialog"
      :title="editingGroup ? '编辑分组' : '新建分组'"
      width="400px"
    >
      <el-form :model="groupData" label-width="80px">
        <el-form-item label="父级分组">
          <el-select v-model="groupData.parentId" placeholder="根目录" clearable>
            <el-option
              v-for="group in connectionsStore.groups"
              :key="group.id"
              :label="group.name"
              :value="group.id"
              :disabled="editingGroup && group.id === editingGroup.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="分组名称">
          <el-input v-model="groupData.name" placeholder="请输入分组名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGroupDialog = false">取消</el-button>
        <el-button type="primary" @click="handleGroupSubmit" :loading="submittingGroup">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed, watch, h } from 'vue'
import { useRouter } from 'vue-router'
import { useConnectionsStore } from '@/stores/connections'
import { ElMessage, ElMessageBox, ElNotification, ElTree } from 'element-plus'
import { 
  Plus, Edit, Delete, Connection as ConnectionIcon, 
  DataLine, Search, Folder, FolderAdd, Monitor, MoreFilled,
  Expand, Fold, Download, Upload
} from '@element-plus/icons-vue'
import { api } from '@/api'
import type { ConnectionConfig, DatabaseType, Group } from '@/types'

const router = useRouter()
const connectionsStore = useConnectionsStore()
const treeRef = ref<InstanceType<typeof ElTree>>()

const filterText = ref('')
const showCreateDialog = ref(false)
const showGroupDialog = ref(false)
const submitting = ref(false)
const submittingGroup = ref(false)
const testing = ref(false)
const editingConnection = ref<ConnectionConfig | null>(null)
const editingGroup = ref<Group | null>(null)
const selectedConnections = ref<string[]>([])
const fileInput = ref<HTMLInputElement | null>(null)

const formData = reactive({
  name: '',
  type: 'mysql' as DatabaseType,
  host: 'localhost',
  port: 3306,
  username: '',
  password: '',
  database: '',
  groupId: '',
  monitoringEnabled: false,
  params: {} as Record<string, string>
})

const groupData = reactive({
  name: '',
  parentId: ''
})

const formRules = {
  name: [{ required: true, message: '请输入连接名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择数据库类型', trigger: 'change' }],
  host: [{ required: (form: any) => form.type !== 'sqlite', message: '请输入主机地址', trigger: 'blur' }],
  username: [{ required: (form: any) => form.type !== 'sqlite', message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: (form: any) => form.type !== 'sqlite', message: '请输入密码', trigger: 'blur' }],
  database: [{ required: false, validator: (_rule: any, value: any, callback: any) => {
    if (formData.type !== 'clickhouse' && formData.type !== 'sqlite' && !value) {
      callback(new Error('请输入数据库名称'))
    } else {
      callback()
    }
  }, trigger: 'blur' }]
}

const defaultProps = {
  children: 'children',
  label: 'label'
}

onMounted(() => {
  connectionsStore.fetchConnections()
  connectionsStore.fetchGroups()
})

watch(filterText, (val) => {
  treeRef.value?.filter(val)
})

watch(() => formData.type, (val) => {
  handleDbTypeChange(val)
})

const treeData = computed(() => {
  const groups = connectionsStore.groups
  const connections = connectionsStore.connections

  const groupMap = new Map<string, any>()
  
  // 创建分组节点
  groups.forEach(group => {
    groupMap.set(group.id, {
      id: group.id,
      label: group.name,
      type: 'group',
      children: []
    })
  })

  // 添加连接到对应分组
  connections.forEach(conn => {
    const node = {
      id: conn.id,
      label: conn.name,
      type: 'connection',
      data: conn
    }

    if (conn.groupId && groupMap.has(conn.groupId)) {
      groupMap.get(conn.groupId).children.push(node)
    } else {
      // 未分组的连接
      if (!groupMap.has('ungrouped')) {
        groupMap.set('ungrouped', {
          id: 'ungrouped',
          label: '未分组',
          type: 'group',
          children: []
        })
      }
      groupMap.get('ungrouped').children.push(node)
    }
  })

  return Array.from(groupMap.values()).filter(group => group.children.length > 0)
})

function filterNode(value: string, data: any) {
  if (!value) return true
  return data.label.toLowerCase().includes(value.toLowerCase())
}

const MysqlIcon = {
  render() {
    return h('svg', { viewBox: '0 0 128 128', width: '16', height: '16', style: 'margin-right: 8px;' }, [
      h('path', { fill: '#00618A', d: 'M117.688 98.242c-6.973-.191-12.297.461-16.852 2.379-1.293.547-3.355.559-3.566 2.18.711.746.82 1.859 1.387 2.777 1.086 1.754 2.922 4.113 4.559 5.352 1.789 1.348 3.633 2.793 5.551 3.961 3.414 2.082 7.223 3.27 10.504 5.352 1.938 1.23 3.859 2.777 5.75 4.164.934.684 1.563 1.75 2.773 2.18v-.195c-.637-.812-.801-1.93-1.387-2.777l-2.578-2.578c-2.52-3.344-5.719-6.281-9.117-8.719-2.711-1.949-8.781-4.578-9.91-7.73l-.199-.199c1.922-.219 4.172-.914 5.949-1.391 2.98-.797 5.645-.59 8.719-1.387l4.164-1.187v-.793c-1.555-1.594-2.664-3.707-4.359-5.152-4.441-3.781-9.285-7.555-14.273-10.703-2.766-1.746-6.184-2.883-9.117-4.363-.988-.496-2.719-.758-3.371-1.586-1.539-1.961-2.379-4.449-3.566-6.738-2.488-4.793-4.93-10.023-7.137-15.066-1.504-3.437-2.484-6.828-4.359-9.91-9-14.797-18.687-23.73-33.695-32.508-3.195-1.867-7.039-2.605-11.102-3.57l-6.543-.395c-1.332-.555-2.715-2.184-3.965-2.977C16.977 3.52 4.223-3.312.539 5.672-1.785 11.34 4.016 16.871 6.09 19.746c1.457 2.012 3.32 4.273 4.359 6.539.688 1.492.805 2.984 1.391 4.559 1.438 3.883 2.695 8.109 4.559 11.695.941 1.816 1.98 3.727 3.172 5.352.727.996 1.98 1.438 2.18 2.973-1.227 1.715-1.297 4.375-1.984 6.543-3.098 9.77-1.926 21.91 2.578 29.137 1.383 2.223 4.641 6.98 9.117 5.156 3.918-1.598 3.043-6.539 4.164-10.902.254-.988.098-1.715.594-2.379v.199l3.57 7.133c2.641 4.254 7.324 8.699 11.297 11.699 2.059 1.555 3.68 4.242 6.344 5.152v-.199h-.199c-.516-.805-1.324-1.137-1.98-1.781-1.551-1.523-3.277-3.414-4.559-5.156-3.613-4.902-6.805-10.27-9.711-15.855-1.391-2.668-2.598-5.609-3.77-8.324-.453-1.047-.445-2.633-1.387-3.172-1.281 1.988-3.172 3.598-4.164 5.945-1.582 3.754-1.789 8.336-2.375 13.082-.348.125-.195.039-.398.199-2.762-.668-3.73-3.508-4.758-5.949-2.594-6.164-3.078-16.09-.793-23.191.59-1.836 3.262-7.617 2.18-9.316-.516-1.691-2.219-2.672-3.172-3.965-1.18-1.598-2.355-3.703-3.172-5.551-2.125-4.805-3.113-10.203-5.352-15.062-1.07-2.324-2.875-4.676-4.359-6.738-1.645-2.289-3.484-3.977-4.758-6.742-.453-.984-1.066-2.559-.398-3.566.215-.684.516-.969 1.191-1.191 1.148-.887 4.352.297 5.547.793 3.18 1.32 5.832 2.578 8.527 4.363 1.289.855 2.598 2.512 4.16 2.973h1.785c2.789.641 5.914.195 8.523.988 4.609 1.402 8.738 3.582 12.488 5.949 11.422 7.215 20.766 17.48 27.156 29.734 1.027 1.973 1.473 3.852 2.379 5.945 1.824 4.219 4.125 8.559 5.941 12.688 1.816 4.113 3.582 8.27 6.148 11.695 1.348 1.801 6.551 2.766 8.918 3.766 1.66.699 4.379 1.43 5.949 2.379 3 1.809 5.906 3.965 8.723 5.945 1.402.992 5.73 3.168 5.945 4.957zm-88.605-75.52c-1.453-.027-2.48.156-3.566.395v.199h.195c.695 1.422 1.918 2.34 2.777 3.566l1.98 4.164.199-.195c1.227-.867 1.789-2.25 1.781-4.363-.492-.52-.562-1.164-.992-1.785-.562-.824-1.66-1.289-2.375-1.98zm0 0' })
    ])
  }
}

const PostgresIcon = {
  render() {
    return h('svg', { viewBox: '0 0 128 128', width: '16', height: '16', style: 'margin-right: 8px;' }, [
      h('path', { fill: '#336791', d: 'M93.809 92.112c.785-6.533.55-7.492 5.416-6.433l1.235.108c3.742.17 8.637-.602 11.513-1.938 6.191-2.873 9.861-7.668 3.758-6.409-13.924 2.873-14.881-1.842-14.881-1.842 14.703-21.815 20.849-49.508 15.543-56.287-14.47-18.489-39.517-9.746-39.936-9.52l-.134.025c-2.751-.571-5.83-.912-9.289-.968-6.301-.104-11.082 1.652-14.709 4.402 0 0-44.683-18.409-42.604 23.151.442 8.841 12.672 66.898 27.26 49.362 5.332-6.412 10.484-11.834 10.484-11.834 2.558 1.699 5.622 2.567 8.834 2.255l.249-.212c-.078.796-.044 1.575.099 2.497-3.757 4.199-2.653 4.936-10.166 6.482-7.602 1.566-3.136 4.355-.221 5.084 3.535.884 11.712 2.136 17.238-5.598l-.22.882c1.474 1.18 1.375 8.477 1.583 13.69.209 5.214.558 10.079 1.621 12.948 1.063 2.868 2.317 10.256 12.191 8.14 8.252-1.764 14.561-4.309 15.136-27.985' })
    ])
  }
}

const SqliteIcon = {
  render() {
    return h('svg', { viewBox: '0 0 128 128', width: '16', height: '16', style: 'margin-right: 8px;' }, [
      h('path', { fill: '#003B57', d: 'M115.6 98.4c4.8-5.6 7.2-12.8 7.2-20V40.8c0-7.2-2.4-14.4-7.2-20c-5.6-5.6-12.8-8-20-8H32.4c-7.2 0-14.4 2.4-20 7.2-4.8 5.6-7.2 12.8-7.2 20v37.6c0 7.2 2.4 14.4 7.2 20 4.8 5.6 12.8 7.2 20 7.2h63.2c7.2 0 14.4-2.4 20-7.2z' })
    ])
  }
}

const ClickHouseIcon = {
  render() {
    return h('svg', { viewBox: '0 0 128 128', width: '16', height: '16', style: 'margin-right: 8px;' }, [
      h('path', { fill: '#FFCC00', d: 'M0 0h20v128H0zm36 0h20v128H36zm36 0h20v128H72zm36 0h20v128h-20z' })
    ])
  }
}

const DMIcon = {
  render() {
    return h('svg', { viewBox: '0 0 128 128', width: '16', height: '16', style: 'margin-right: 8px;' }, [
      h('path', { fill: '#E63946', d: 'M64 4C30.9 4 4 30.9 4 64s26.9 60 60 60 60-26.9 60-60S97.1 4 64 4zm0 110c-27.6 0-50-22.4-50-50S36.4 14 64 14s50 22.4 50 50-22.4 50-50 50z' }),
      h('text', { x: '64', y: '75', 'text-anchor': 'middle', 'font-size': '32', 'font-weight': 'bold', fill: '#E63946' }, 'DM')
    ])
  }
}

function getDatabaseIcon(type: string) {
  const icons: Record<string, any> = {
    mysql: MysqlIcon,
    postgresql: PostgresIcon,
    sqlite: SqliteIcon,
    clickhouse: ClickHouseIcon,
    kingbase: Monitor,
    dm: DMIcon
  }
  return icons[type] || Monitor
}

function handleExpandAll() {
  const nodes = (treeRef.value as any)?.store.nodesMap
  for (const i in nodes) {
    nodes[i].expanded = true
  }
}

function handleCollapseAll() {
  const nodes = (treeRef.value as any)?.store.nodesMap
  for (const i in nodes) {
    nodes[i].expanded = false
  }
}

function allowDrop(draggingNode: any, dropNode: any, type: string) {
  if (dropNode.data.type === 'connection' && type === 'inner') {
    return false
  }
  return true
}

async function handleDrop(draggingNode: any, dropNode: any, dropType: string) {
  const data = draggingNode.data
  const targetData = dropNode.data
  
  let newParentId = ''
  if (dropType === 'inner') {
    newParentId = targetData.data.id
  } else {
    // prev or next
    newParentId = targetData.type === 'group' ? targetData.data.parentId : targetData.data.groupId
  }

  try {
    if (data.type === 'group') {
      await api.updateGroup(data.data.id, { ...data.data, parentId: newParentId })
      ElMessage.success('操作成功')
    } else {
      await connectionsStore.updateConnection(data.data.id, { ...data.data, groupId: newParentId })
      ElMessage.success('操作成功')
    }
  } catch (e: any) {
    ElNotification.error({
      title: '操作失败',
      message: e.response?.data?.message || e.message || '未知错误'
    })
  } finally {
    await connectionsStore.fetchConnections()
  }
}

function getDatabaseTypeName(type: string) {
  const names: Record<string, string> = {
    mysql: 'MySQL',
    postgresql: 'PostgreSQL',
    sqlite: 'SQLite',
    clickhouse: 'ClickHouse',
    kingbase: 'KingBase',
    dm: '达梦数据库'
  }
  return names[type] || type
}

function getDatabaseTypeColor(type: string) {
  const colors: Record<string, string> = {
    mysql: 'success',
    postgresql: 'primary',
    sqlite: 'info',
    clickhouse: 'warning',
    kingbase: 'danger',
    dm: 'danger'
  }
  return colors[type] || ''
}

function handleManage(row: ConnectionConfig) {
  router.push(`/tables/${row.id}`)
}

function handleQuery(row: ConnectionConfig) {
  router.push(`/query/${row.id}`)
}

function handleNodeDblClick(data: any) {
  console.log('Double clicked node:', data)
  if (data.type === 'connection' && data.data?.id) {
    handleQuery(data.data)
  }
}

async function handleTest(row: ConnectionConfig) {
  const loading = ElMessage.info('测试连接中...')
  try {
    await connectionsStore.testConnection(row.id)
    loading.close()
    ElMessage.success('连接成功')
  } catch (e: any) {
    loading.close()
    ElNotification.error({
      title: '连接失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  }
}

async function handleConnectionToggle(row: ConnectionConfig, val: boolean) {
  try {
    if (!val) {
      await api.closeConnection(row.id)
      ElMessage.success('连接已断开')
    } else {
      await api.connectConnection(row.id)
      ElMessage.success('连接已建立')
    }
    await connectionsStore.fetchConnections()
  } catch (e: any) {
    ElNotification.error({
      title: '操作失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  }
}

async function handleToggleMonitoring(row: ConnectionConfig) {
  const newStatus = !row.monitoringEnabled
  const action = newStatus ? '开启' : '关闭'
  try {
    await connectionsStore.updateConnection(row.id, {
      ...row,
      monitoringEnabled: newStatus
    })
    ElMessage.success(`监控已${action}`)
  } catch (e: any) {
    ElNotification.error({
      title: '操作失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  }
}

async function handleTestConfig() {
  testing.value = true
  try {
    const res = await api.testConnectionConfig(formData)
    testing.value = false
    if (res.data.connected) {
      ElMessage.success('连接成功')
    } else {
      ElNotification.error({
        title: '连接失败',
        message: res.data.error || '连接失败',
        position: 'top-right'
      })
    }
  } catch (e: any) {
    testing.value = false
    ElNotification.error({
      title: '连接失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  }
}

function handleCreateConnection() {
  editingConnection.value = null
  resetForm()
  showCreateDialog.value = true
}

function handleCreateGroup() {
  editingGroup.value = null
  groupData.name = ''
  groupData.parentId = ''
  showGroupDialog.value = true
}

function handleEdit(row: ConnectionConfig) {
  editingConnection.value = row
  Object.assign(formData, row)
  showCreateDialog.value = true
}

function handleEditGroup(node: any) {
  const g = node.data
  editingGroup.value = g
  groupData.name = g.name
  groupData.parentId = g.parentId
  showGroupDialog.value = true
}

async function handleDelete(row: ConnectionConfig) {
  await ElMessageBox.confirm(`确定删除连接 "${row.name}" 吗？`, '提示', {
    type: 'warning'
  })
  await connectionsStore.deleteConnection(row.id)
  ElMessage.success('删除成功')
}

async function handleDeleteGroup(node: any) {
  const g = node.data
  await ElMessageBox.confirm(`确定删除分组 "${g.name}" 吗？其下连接及子分组将变为未分类状态。`, '提示', {
    type: 'warning'
  })
  try {
    await connectionsStore.deleteGroup(g.id)
    ElMessage.success('删除成功')
    await connectionsStore.fetchConnections()
    await connectionsStore.fetchGroups()
  } catch (e: any) {
    ElNotification.error({
      title: '删除失败',
      message: e.message || '未知错误',
      position: 'top-right'
    })
  }
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editingConnection.value) {
      await connectionsStore.updateConnection(editingConnection.value.id, formData)
      ElMessage.success('更新成功')
    } else {
      await connectionsStore.createConnection(formData)
      ElMessage.success('创建成功')
    }
    showCreateDialog.value = false
    resetForm()
  } catch (e: any) {
    ElNotification.error({
      title: '提交失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    submitting.value = false
  }
}

async function handleGroupSubmit() {
  if (!groupData.name) {
    ElMessage.warning('请输入分组名称')
    return
  }
  submittingGroup.value = true
  try {
    if (editingGroup.value) {
      await api.updateGroup(editingGroup.value.id, groupData)
      ElMessage.success('更新成功')
    } else {
      await connectionsStore.createGroup(groupData)
      ElMessage.success('创建成功')
    }
    await connectionsStore.fetchGroups()
    showGroupDialog.value = false
  } catch (e: any) {
    ElNotification.error({
      title: '提交失败',
      message: e.response?.data?.message || e.message || '未知错误',
      position: 'top-right'
    })
  } finally {
    submittingGroup.value = false
  }
}

function resetForm() {
  editingConnection.value = null
  Object.assign(formData, {
    name: '',
    type: 'mysql' as DatabaseType,
    host: 'localhost',
    port: 3306,
    username: '',
    password: '',
    database: '',
    groupId: '',
    monitoringEnabled: false,
    params: {}
  })
}

function handleDbTypeChange(val: DatabaseType) {
  if (val === 'mysql') {
    formData.port = 3306
  } else if (val === 'postgresql') {
    formData.port = 5432
  } else if (val === 'kingbase') {
    formData.port = 54321
  } else if (val === 'dm') {
    formData.port = 5236
  } else if (val === 'clickhouse') {
    formData.port = 9000
    if (!formData.params) formData.params = {}
    formData.params.protocol = 'clickhouse'
  }
}

function handleProtocolChange(val: string) {
  if (val === 'clickhouse') {
    formData.port = 9000
  } else if (val === 'http') {
    formData.port = 8123
  } else if (val === 'https') {
    formData.port = 8443
  }
}

function isConnectionSelected(id: string): boolean {
  return selectedConnections.value.includes(id)
}

function handleConnectionSelect(id: string, checked: boolean) {
  if (checked) {
    if (!selectedConnections.value.includes(id)) {
      selectedConnections.value.push(id)
    }
  } else {
    const index = selectedConnections.value.indexOf(id)
    if (index > -1) {
      selectedConnections.value.splice(index, 1)
    }
  }
}

function handleExportConnections() {
  if (selectedConnections.value.length === 0) {
    ElMessage.warning('请选择要导出的连接')
    return
  }

  const connectionsToExport = connectionsStore.connections.filter(conn => 
    selectedConnections.value.includes(conn.id)
  )

  // 移除敏感信息（密码）
  const exportData = connectionsToExport.map(conn => ({
    ...conn,
    password: '******' // 隐藏密码
  }))

  const dataStr = JSON.stringify(exportData, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `connections_export_${new Date().getTime()}.json`
  link.click()
  URL.revokeObjectURL(url)

  ElMessage.success(`已导出 ${selectedConnections.value.length} 个连接配置`)
  selectedConnections.value = []
}

function triggerFileInput() {
  fileInput.value?.click()
}

async function handleFileImport(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  
  if (!file) return
  
  try {
    const text = await file.text()
    const connections = JSON.parse(text) as ConnectionConfig[]
    
    if (!Array.isArray(connections)) {
      ElMessage.error('文件格式错误：应为连接配置数组')
      return
    }
    
    let successCount = 0
    let errorCount = 0
    
    for (const conn of connections) {
      try {
        // 生成新的 ID 避免冲突
        const newConn = {
          ...conn,
          id: undefined, // 让后端生成新 ID
          password: '' // 需要用户重新输入密码
        }
        
        await connectionsStore.createConnection(newConn)
        successCount++
      } catch (e: any) {
        console.error('导入连接失败:', conn.name, e)
        errorCount++
      }
    }
    
    if (successCount > 0) {
      ElMessage.success(`成功导入 ${successCount} 个连接${errorCount > 0 ? `，失败 ${errorCount} 个` : ''}`)
      await connectionsStore.fetchConnections()
    } else {
      ElMessage.error('导入失败，请检查文件格式')
    }
  } catch (e: any) {
    ElMessage.error('文件解析失败: ' + e.message)
  } finally {
    // 清空文件输入，允许重复导入同一文件
    if (target) target.value = ''
  }
}
</script>

<style scoped>
.connections-page {
  padding: 20px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-actions .el-divider--vertical {
  height: 1.5em;
  margin: 0;
}

.tree-container {
  margin-top: 20px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 10px;
  background: #fff;
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  padding-right: 8px;
}

.node-label {
  display: flex;
  align-items: center;
  gap: 8px;
}

.node-actions {
  display: flex;
  align-items: center;
}

.db-type-tag, .status-tag {
  margin-left: 8px;
}

.el-tree {
  --el-tree-node-content-height: 40px;
}

.db-type-tag {
  text-transform: capitalize;
}

.tree-container {
  margin-top: 20px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  padding: 10px;
  background: #fff;
}

.form-item-tip {
  margin-left: 12px;
  font-size: 12px;
  color: #909399;
}
</style>
