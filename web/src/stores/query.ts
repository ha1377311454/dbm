import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'
import type { QueryResult, TableInfo, TableSchema } from '@/types'

export const useQueryStore = defineStore('query', () => {
  const result = ref<QueryResult | null>(null)
  const loading = ref(false)
  const tables = ref<TableInfo[]>([])
  const databases = ref<string[]>([])
  const schemas = ref<string[]>([])
  const currentSchema = ref<TableSchema | null>(null)
  const currentSchemaName = ref('')

  async function executeQuery(connectionId: string, query: string, opts?: any) {
    loading.value = true
    try {
      const res = await api.executeQuery(connectionId, query, opts)
      if (res.code === 0) {
        result.value = res.data
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchDatabases(connectionId: string) {
    const res = await api.getDatabases(connectionId)
    if (res.code === 0) {
      databases.value = res.data
    }
  }

  async function fetchSchemas(connectionId: string, database?: string) {
    const res = await api.getSchemas(connectionId, database)
    if (res.code === 0) {
      schemas.value = res.data
    }
  }

  async function fetchTables(connectionId: string, database?: string, schema?: string) {
    const res = await api.getTables(connectionId, database, schema)
    if (res.code === 0) {
      tables.value = res.data
    }
  }

  async function fetchTableSchema(connectionId: string, table: string, database?: string, schema?: string) {
    const res = await api.getTableSchema(connectionId, table, database, schema)
    console.log('fetchTableSchema API 响应:', res)
    if (res.code === 0) {
      console.log('API 返回的 data:', res.data)
      console.log('data.columns:', res.data?.columns)
      currentSchema.value = res.data
    } else {
      console.error('API 返回错误:', res.message)
      throw new Error(res.message || '获取表结构失败')
    }
  }

  async function createRow(connectionId: string, table: string, database: string, data: any) {
    const res = await api.createRow(connectionId, table, database, data, currentSchemaName.value)
    if (res.code !== 0) {
      throw new Error(res.message || '添加数据失败')
    }
    return res
  }

  async function updateRow(connectionId: string, table: string, database: string, data: any, where: string) {
    const res = await api.updateRow(connectionId, table, database, data, where, currentSchemaName.value)
    if (res.code !== 0) {
      throw new Error(res.message || '更新数据失败')
    }
    return res
  }

  async function deleteRow(connectionId: string, table: string, database: string, where: string) {
    const res = await api.deleteRow(connectionId, table, database, where, currentSchemaName.value)
    if (res.code !== 0) {
      throw new Error(res.message || '删除数据失败')
    }
    return res
  }

  function clearResult() {
    result.value = null
  }

  return {
    result,
    loading,
    tables,
    databases,
    schemas,
    currentSchema,
    currentSchemaName,
    executeQuery,
    fetchDatabases,
    fetchSchemas,
    fetchTables,
    fetchTableSchema,
    createRow,
    updateRow,
    deleteRow,
    clearResult
  }
})
