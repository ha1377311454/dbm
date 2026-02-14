// 数据库类型
export enum DatabaseType {
  MySQL = 'mysql',
  PostgreSQL = 'postgresql',
  SQLite = 'sqlite',
  MSSQL = 'mssql',
  Oracle = 'oracle',
  ClickHouse = 'clickhouse',
  KingBase = 'kingbase'
}

// 连接配置
export interface ConnectionConfig {
  id: string
  name: string
  type: DatabaseType
  host: string
  port: number
  username: string
  password: string
  database: string
  params: Record<string, string>
  createdAt: string
  updatedAt: string
  groupId: string // 所属分组 ID
  connected: boolean
}

// 分组信息
export interface Group {
  id: string
  name: string
  parentId: string
}

// 表信息
export interface TableInfo {
  name: string
  database: string
  schema: string
  tableType: string
  rows: number
  size: number
  comment: string
}

// 列信息
export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  defaultValue: string
  key: string
  extra: string
  comment: string
}

// 索引信息
export interface IndexInfo {
  name: string
  columns: string[]
  unique: boolean
  primary: boolean
  comment: string
}

// 表结构
export interface TableSchema {
  database: string
  table: string
  columns: ColumnInfo[]
  indexes: IndexInfo[]
  constraints: any[]
}

// 查询结果
export interface QueryResult {
  columns: string[]
  rows: Record<string, any>[]
  total: number
  timeCost: number
}

// 执行结果
export interface ExecuteResult {
  rowsAffected: number
  timeCost: number
  message: string
}

// 查询选项
export interface QueryOptions {
  database?: string
  schema?: string
  page?: number
  pageSize?: number
  sortBy?: string
  sortDesc?: boolean
}

// CSV 导出选项
export interface CSVOptions {
  includeHeader: boolean
  separator: string
  quote: string
  encoding: string
  nullValue: string
  dateFormat: string
  maxRows?: number
}

// SQL 导出选项
export interface SQLOptions {
  includeCreateTable: boolean
  includeDropTable: boolean
  batchInsert: boolean
  batchSize: number
  structureOnly: boolean
  maxRows?: number
  query?: string
  tableName?: string  // 自定义查询时的表名（用于 INSERT 语句）
}

// API 响应
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 表结构修改相关类型
export enum AlterActionType {
  ADD_COLUMN = 'ADD_COLUMN',
  DROP_COLUMN = 'DROP_COLUMN',
  MODIFY_COLUMN = 'MODIFY_COLUMN',
  RENAME_COLUMN = 'RENAME_COLUMN',
  ADD_INDEX = 'ADD_INDEX',
  DROP_INDEX = 'DROP_INDEX',
  RENAME_TABLE = 'RENAME_TABLE'
}

export interface ColumnDef {
  name: string
  type: string
  length?: number
  precision?: number
  scale?: number
  nullable: boolean
  defaultValue?: string
  autoIncrement?: boolean
  comment?: string
  after?: string
}

export interface IndexDef {
  name: string
  columns: string[]
  unique: boolean
  type?: string
  comment?: string
}

export interface AlterTableAction {
  type: AlterActionType
  column?: ColumnDef
  oldName?: string
  newName?: string
  index?: IndexDef
}

export interface AlterTableRequest {
  database: string
  table: string
  actions: AlterTableAction[]
}

export interface RenameTableRequest {
  newName: string
}

// 类型映射相关类型
export interface TypeOption {
  label: string
  value: string
}

export interface TypeRule {
  targetType: string
  safeFallback: string
  precisionLoss: boolean
  requiresUser: boolean
  userOptions: TypeOption[]
  note: string
}

export interface TypeSummary {
  total: number
  direct: number
  fallback: number
  userChoice: number
  lossyCount: number
}

export interface TypeMappingResult {
  success: boolean
  mapped: Record<string, string>
  warnings: string[]
  requiresUser: Record<string, TypeRule>
  summary: TypeSummary
}
