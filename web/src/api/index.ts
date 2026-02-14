import axios from 'axios'
import type { ApiResponse } from '@/types'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

request.interceptors.response.use(
  (response) => response.data,
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export default request

export const api = {
  // 连接管理
  getConnections: () => request.get<any, ApiResponse<ConnectionConfig[]>>('/connections'),
  createConnection: (data: any) => request.post<any, ApiResponse<ConnectionConfig>>('/connections', data),
  updateConnection: (id: string, data: any) => request.put<any, ApiResponse<ConnectionConfig>>(`/connections/${id}`, data),
  deleteConnection: (id: string) => request.delete<any, ApiResponse<null>>(`/connections/${id}`),
  testConnection: (id: string) => request.post<any, ApiResponse<any>>(`/connections/${id}/test`),
  connectConnection: (id: string) => request.post<any, ApiResponse<any>>(`/connections/${id}/connect`),
  closeConnection: (id: string) => request.post<any, ApiResponse<null>>(`/connections/${id}/close`),
  testConnectionConfig: (data: any) => request.post<any, ApiResponse<any>>('/connections/test', data),

  // 分组管理
  getGroups: () => request.get<any, ApiResponse<Group[]>>('/groups'),
  createGroup: (data: any) => request.post<any, ApiResponse<Group>>('/groups', data),
  updateGroup: (id: string, data: any) => request.put<any, ApiResponse<Group>>(`/groups/${id}`, data),
  deleteGroup: (id: string) => request.delete<any, ApiResponse<null>>(`/groups/${id}`),

  // 数据库元数据
  getDatabases: (id: string) => request.get<any, ApiResponse<string[]>>(`/connections/${id}/databases`),
  getSchemas: (id: string, database?: string) =>
    request.get<any, ApiResponse<string[]>>(`/connections/${id}/schemas`, { params: { database } }),
  getTables: (id: string, database?: string, schema?: string) =>
    request.get<any, ApiResponse<TableInfo[]>>(`/connections/${id}/tables`, { params: { database, schema } }),
  getTableSchema: (id: string, table: string, database?: string, schema?: string) =>
    request.get<any, ApiResponse<TableSchema>>(`/connections/${id}/tables/${table}/schema`, { params: { database, schema } }),
  getViews: (id: string, database?: string, schema?: string) =>
    request.get<any, ApiResponse<TableInfo[]>>(`/connections/${id}/views`, { params: { database, schema } }),

  // SQL 执行
  executeQuery: (id: string, query: string, opts?: QueryOptions) =>
    request.post<any, ApiResponse<QueryResult>>(`/connections/${id}/query`, { query, opts }),
  executeNonQuery: (id: string, query: string) =>
    request.post<any, ApiResponse<ExecuteResult>>(`/connections/${id}/execute`, { query }),

  // 导出
  exportCSV: (id: string, params: { query: string; opts: CSVOptions; database?: string }) =>
    request.post(`/connections/${id}/export/csv`, params, {
      params: { database: params.database },
      responseType: 'blob'
    }),
  exportSQL: (id: string, params: { tables: string[]; opts: SQLOptions; database?: string }) =>
    request.post(`/connections/${id}/export/sql`, params, {
      params: { database: params.database },
      responseType: 'blob'
    }),
  previewExportSQL: (id: string, params: { tables: string[]; targetDbType: DatabaseType }) =>
    request.post<any, ApiResponse<TypeMappingResult>>(`/connections/${id}/export/sql/preview`, params),

  // 数据编辑
  createRow: (id: string, table: string, database: string, data: any, schema?: string) =>
    request.post(`/connections/${id}/tables/${table}/data`, data, { params: { database, schema } }),
  updateRow: (id: string, table: string, database: string, data: any, where: string, schema?: string) =>
    request.put(`/connections/${id}/tables/${table}/data`, { data, where }, { params: { database, schema } }),
  deleteRow: (id: string, table: string, database: string, where: string, schema?: string) =>
    request.delete(`/connections/${id}/tables/${table}/data`, { params: { database, schema }, data: { where } }),

  // 表结构修改
  alterTable: (id: string, table: string, database: string, req: AlterTableRequest) =>
    request.post<any, ApiResponse<any>>(`/connections/${id}/tables/${table}/alter`, req, { params: { database } }),
  renameTable: (id: string, table: string, database: string, data: RenameTableRequest) =>
    request.post<any, ApiResponse<any>>(`/connections/${id}/tables/${table}/rename`, data, { params: { database } }),

  // 监控
  getMonitorStats: () => request.get<any, ApiResponse<any>>('/monitor/stats')
}

import type {
  ConnectionConfig,
  Group,
  TableInfo,
  TableSchema,
  QueryResult,
  ExecuteResult,
  QueryOptions,
  CSVOptions,
  SQLOptions,
  AlterTableRequest,
  RenameTableRequest,
  TypeMappingResult,
  DatabaseType
} from '@/types'
