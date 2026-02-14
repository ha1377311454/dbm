package adapter

import (
	"database/sql"
	"dbm/internal/model"
	"io"
)

// DatabaseAdapter 数据库适配器接口
type DatabaseAdapter interface {
	// 连接管理
	Connect(config *model.ConnectionConfig) (*sql.DB, error)
	Close(db *sql.DB) error
	Ping(db *sql.DB) error

	// 元数据查询
	GetDatabases(db *sql.DB) ([]string, error)
	GetTables(db *sql.DB, database string) ([]model.TableInfo, error)
	GetTableSchema(db *sql.DB, database, table string) (*model.TableSchema, error)
	GetViews(db *sql.DB, database string) ([]model.TableInfo, error)
	GetIndexes(db *sql.DB, database, table string) ([]model.IndexInfo, error)

	// SQL 执行
	Execute(db *sql.DB, query string, args ...interface{}) (*model.ExecuteResult, error)
	Query(db *sql.DB, query string, opts *model.QueryOptions) (*model.QueryResult, error)

	// 数据编辑
	Insert(db *sql.DB, database, table string, data map[string]interface{}) error
	Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error
	Delete(db *sql.DB, database, table, where string) error

	// 导出
	ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *model.CSVOptions) error
	ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error

	// 建表语句
	GetCreateTableSQL(db *sql.DB, database, table string) (string, error)

	// 表结构修改
	AlterTable(db *sql.DB, request *model.AlterTableRequest) error
	RenameTable(db *sql.DB, database, oldName, newName string) error
}

// SchemaAwareDatabase 支持 schema 的数据库接口（PostgreSQL）
type SchemaAwareDatabase interface {
	// GetSchemas 获取 schema 列表
	GetSchemas(db *sql.DB, database string) ([]string, error)

	// GetTablesWithSchema 获取指定 schema 下的表列表
	GetTablesWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error)

	// GetTableSchemaWithSchema 获取指定 schema 下表结构
	GetTableSchemaWithSchema(db *sql.DB, database, schema, table string) (*model.TableSchema, error)

	// GetViewsWithSchema 获取指定 schema 下的视图列表
	GetViewsWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error)
}

// AdapterFactory 适配器工厂接口
type AdapterFactory interface {
	CreateAdapter(dbType model.DatabaseType) (DatabaseAdapter, error)
	SupportedTypes() []model.DatabaseType
}

// BaseAdapter 基础适配器，提供通用功能
type BaseAdapter struct{}

// NewBaseAdapter 创建基础适配器
func NewBaseAdapter() *BaseAdapter {
	return &BaseAdapter{}
}
