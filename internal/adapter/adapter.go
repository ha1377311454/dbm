package adapter

import (
	"dbm/internal/model"
	"io"
)

// DatabaseAdapter 数据库适配器接口
type DatabaseAdapter interface {
	// 连接管理
	Connect(config *model.ConnectionConfig) (any, error)
	Close(db any) error
	Ping(db any) error

	// 元数据查询
	GetDatabases(db any) ([]string, error)
	GetTables(db any, database string) ([]model.TableInfo, error)
	GetTableSchema(db any, database, table string) (*model.TableSchema, error)
	GetViews(db any, database string) ([]model.TableInfo, error)
	GetIndexes(db any, database, table string) ([]model.IndexInfo, error)
	GetProcedures(db any, database string) ([]model.RoutineInfo, error)
	GetFunctions(db any, database string) ([]model.RoutineInfo, error)
	GetViewDefinition(db any, database, viewName string) (string, error)
	GetRoutineDefinition(db any, database, routineName, routineType string) (string, error)

	// SQL 执行
	Execute(db any, query string, args ...interface{}) (*model.ExecuteResult, error)
	Query(db any, query string, opts *model.QueryOptions) (*model.QueryResult, error)

	// 数据编辑
	Insert(db any, database, table string, data map[string]interface{}) error
	Update(db any, database, table string, data map[string]interface{}, where string) error
	Delete(db any, database, table, where string) error

	// 导出
	ExportToCSV(db any, writer io.Writer, database, query string, opts *model.CSVOptions) error
	ExportToSQL(db any, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error

	// 建表语句
	GetCreateTableSQL(db any, database, table string) (string, error)

	// 表结构修改
	AlterTable(db any, request *model.AlterTableRequest) error
	RenameTable(db any, database, oldName, newName string) error
}

// SchemaAwareDatabase 支持 schema 的数据库接口（PostgreSQL）
type SchemaAwareDatabase interface {
	// GetSchemas 获取 schema 列表
	GetSchemas(db any, database string) ([]string, error)

	// GetTablesWithSchema 获取指定 schema 下的表列表
	GetTablesWithSchema(db any, database, schema string) ([]model.TableInfo, error)

	// GetTableSchemaWithSchema 获取指定 schema 下表结构
	GetTableSchemaWithSchema(db any, database, schema, table string) (*model.TableSchema, error)

	// GetViewsWithSchema 获取指定 schema 下的视图列表
	GetViewsWithSchema(db any, database, schema string) ([]model.TableInfo, error)

	// GetProceduresWithSchema 获取指定 schema 下的存储过程
	GetProceduresWithSchema(db any, database, schema string) ([]model.RoutineInfo, error)

	// GetFunctionsWithSchema 获取指定 schema 下的函数
	GetFunctionsWithSchema(db any, database, schema string) ([]model.RoutineInfo, error)

	// GetViewDefinitionWithSchema 获取指定 schema 下的视图定义
	GetViewDefinitionWithSchema(db any, database, schema, viewName string) (string, error)

	// GetRoutineDefinitionWithSchema 获取指定 schema 下的存储过程或函数定义
	GetRoutineDefinitionWithSchema(db any, database, schema, routineName, routineType string) (string, error)
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

// GetProcedures 默认不实现
func (a *BaseAdapter) GetProcedures(db any, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetFunctions 默认不实现
func (a *BaseAdapter) GetFunctions(db any, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetViewDefinition 默认不实现
func (a *BaseAdapter) GetViewDefinition(db any, database, viewName string) (string, error) {
	return "", nil
}

// GetRoutineDefinition 默认不实现
func (a *BaseAdapter) GetRoutineDefinition(db any, database, routineName, routineType string) (string, error) {
	return "", nil
}
