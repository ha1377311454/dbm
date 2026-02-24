package model

import "time"

// TableInfo 表信息
type TableInfo struct {
	Name      string `json:"name"`
	Database  string `json:"database"`
	Schema    string `json:"schema"`
	TableType string `json:"tableType"` // BASE TABLE, VIEW
	Rows      int64  `json:"rows"`
	Size      int64  `json:"size"`
	Comment   string `json:"comment"`
}

// RoutineInfo 存储过程与函数信息
type RoutineInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // PROCEDURE or FUNCTION
	Database string `json:"database"`
	Schema   string `json:"schema"`
	Comment  string `json:"comment"`
}

// TableSchema 表结构
type TableSchema struct {
	Database    string           `json:"database"`
	Table       string           `json:"table"`
	Columns     []ColumnInfo     `json:"columns"`
	Indexes     []IndexInfo      `json:"indexes"`
	Constraints []ConstraintInfo `json:"constraints"`
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"defaultValue"`
	Key          string `json:"key"`   // PRI, UNI, MUL
	Extra        string `json:"extra"` // auto_increment
	Comment      string `json:"comment"`
}

// IndexInfo 索引信息
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Primary bool     `json:"primary"`
	Comment string   `json:"comment"`
}

// ConstraintInfo 约束信息
type ConstraintInfo struct {
	Name            string `json:"name"`
	Type            string `json:"type"` // PRIMARY KEY, FOREIGN KEY, UNIQUE
	ColumnName      string `json:"columnName"`
	ReferenceTable  string `json:"referenceTable"`
	ReferenceColumn string `json:"referenceColumn"`
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	RowsAffected int64         `json:"rowsAffected"`
	TimeCost     time.Duration `json:"timeCost"`
	Message      string        `json:"message"`
}

// QueryResult 查询结果
type QueryResult struct {
	Columns      []string                 `json:"columns"`
	Rows         []map[string]interface{} `json:"rows"`
	Total        int64                    `json:"total"`
	RowsAffected int64                    `json:"rowsAffected"`
	Message      string                   `json:"message"`
	TimeCost     time.Duration            `json:"timeCost"`
}

// QueryOptions 分页查询选项
type QueryOptions struct {
	Database string `json:"database"` // 目标数据库
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	SortBy   string `json:"sortBy"`
	SortDesc bool   `json:"sortDesc"`
}

// CSVOptions CSV 导出选项
type CSVOptions struct {
	IncludeHeader bool   `json:"includeHeader"` // 包含表头
	Separator     string `json:"separator"`     // 分隔符
	Quote         string `json:"quote"`         // 引号字符
	Encoding      string `json:"encoding"`      // 编码
	NullValue     string `json:"nullValue"`     // NULL 值表示
	DateFormat    string `json:"dateFormat"`    // 日期格式
	MaxRows       int    `json:"maxRows"`       // 最大行数 (0表示无限制)
}

// SQLOptions SQL 导出选项
type SQLOptions struct {
	IncludeCreateTable bool   `json:"includeCreateTable"` // 包含建表语句
	IncludeDropTable   bool   `json:"includeDropTable"`   // 包含 DROP 语句
	BatchInsert        bool   `json:"batchInsert"`        // 批量 INSERT
	BatchSize          int    `json:"batchSize"`          // 批量大小
	StructureOnly      bool   `json:"structureOnly"`      // 仅结构
	MaxRows            int    `json:"maxRows"`            // 最大行数 (0表示无限制)
	Query              string `json:"query"`              // 自定义查询 SQL
	TableName          string `json:"tableName"`          // 自定义查询时的表名（用于 INSERT 语句）
}

// AlterTableRequest 修改表结构请求
type AlterTableRequest struct {
	Database string             `json:"database"`
	Table    string             `json:"table"`
	Actions  []AlterTableAction `json:"actions"`
}

// AlterTableAction 表结构修改操作
type AlterTableAction struct {
	Type    AlterActionType `json:"type"`              // 操作类型
	Column  *ColumnDef      `json:"column,omitempty"`  // 列定义（用于添加/修改列）
	OldName string          `json:"oldName,omitempty"` // 旧名称（用于重命名）
	NewName string          `json:"newName,omitempty"` // 新名称（用于重命名）
	Index   *IndexDef       `json:"index,omitempty"`   // 索引定义
}

// AlterActionType 修改操作类型
type AlterActionType string

const (
	AlterActionAddColumn    AlterActionType = "ADD_COLUMN"
	AlterActionDropColumn   AlterActionType = "DROP_COLUMN"
	AlterActionModifyColumn AlterActionType = "MODIFY_COLUMN"
	AlterActionRenameColumn AlterActionType = "RENAME_COLUMN"
	AlterActionAddIndex     AlterActionType = "ADD_INDEX"
	AlterActionDropIndex    AlterActionType = "DROP_INDEX"
	AlterActionRenameTable  AlterActionType = "RENAME_TABLE"
)

// ColumnDef 列定义
type ColumnDef struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Length        int    `json:"length,omitempty"`    // 类型长度，如 VARCHAR(255)
	Precision     int    `json:"precision,omitempty"` // 精度，如 DECIMAL(10,2)
	Scale         int    `json:"scale,omitempty"`     // 小数位数
	Nullable      bool   `json:"nullable"`
	DefaultValue  string `json:"defaultValue,omitempty"`
	AutoIncrement bool   `json:"autoIncrement,omitempty"`
	Comment       string `json:"comment,omitempty"`
	After         string `json:"after,omitempty"` // 在哪个列之后（MySQL）
}

// IndexDef 索引定义
type IndexDef struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"` // BTREE, HASH, etc.
	Comment string   `json:"comment,omitempty"`
}
