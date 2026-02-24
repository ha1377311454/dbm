package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"fmt"
	"io"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteAdapter SQLite 数据库适配器
type SQLiteAdapter struct {
	*BaseAdapter
}

// NewSQLiteAdapter 创建 SQLite 适配器
func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{
		BaseAdapter: NewBaseAdapter(),
	}
}

// Connect 连接 SQLite 数据库
func (a *SQLiteAdapter) Connect(config *model.ConnectionConfig) (*sql.DB, error) {
	// SQLite 使用 host 字段作为文件路径
	dbPath := config.Host
	if dbPath == "" {
		dbPath = config.Database
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Close 关闭连接
func (a *SQLiteAdapter) Close(db *sql.DB) error {
	return db.Close()
}

// Ping 测试连接
func (a *SQLiteAdapter) Ping(db *sql.DB) error {
	return db.Ping()
}

// GetDatabases 获取数据库列表（SQLite 只有一个数据库文件）
func (a *SQLiteAdapter) GetDatabases(db *sql.DB) ([]string, error) {
	// SQLite 只有一个主数据库
	return []string{"main"}, nil
}

// GetTables 获取表列表
func (a *SQLiteAdapter) GetTables(db *sql.DB, database string) ([]model.TableInfo, error) {
	query := `
		SELECT
			name,
			'BASE TABLE' as type,
			0 as row_count,
			0 as table_size
		FROM sqlite_master
		WHERE type IN ('table', 'view')
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []model.TableInfo
	for rows.Next() {
		var t model.TableInfo
		var tableType string
		if err := rows.Scan(&t.Name, &tableType, &t.Rows, &t.Size); err != nil {
			return nil, err
		}
		t.Database = database
		t.TableType = tableType
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *SQLiteAdapter) GetTableSchema(db *sql.DB, database, table string) (*model.TableSchema, error) {
	schema := &model.TableSchema{
		Database: database,
		Table:    table,
	}

	// 获取列信息（使用 PRAGMA）
	colsQuery := fmt.Sprintf("PRAGMA table_info(`%s`)", table)
	colsRows, err := db.Query(colsQuery)
	if err != nil {
		return nil, err
	}
	defer colsRows.Close()

	for colsRows.Next() {
		var col model.ColumnInfo
		var cid, pkColumn int
		var notNull int
		var defaultValue sql.NullString
		var extra sql.NullString

		if err := colsRows.Scan(&cid, &col.Name, &col.Type, &notNull, &defaultValue, &pkColumn, &extra); err != nil {
			return nil, err
		}

		col.Nullable = notNull == 0
		col.DefaultValue = defaultValue.String
		col.Extra = extra.String

		// 判断是否是主键
		if pkColumn > 0 {
			col.Key = "PRI"
		}

		schema.Columns = append(schema.Columns, col)
	}

	// 获取索引信息
	idxQuery := fmt.Sprintf("PRAGMA index_list(`%s`)", table)
	idxRows, err := db.Query(idxQuery)
	if err != nil {
		return schema, nil
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var indexName string
		var isUnique int
		var origin string
		var partial int
		if err := idxRows.Scan(&indexName, &isUnique, &origin, &partial); err != nil {
			continue
		}

		// 获取索引列
		colQuery := fmt.Sprintf("PRAGMA index_info(`%s`)", indexName)
		colRows, _ := db.Query(colQuery)
		if colRows == nil {
			continue
		}

		var columns []string
		for colRows.Next() {
			var rank, table, colName string
			var cid2 int
			if colRows.Scan(&cid2, &rank, &table, &colName) == nil {
				columns = append(columns, colName)
			}
		}
		colRows.Close()

		schema.Indexes = append(schema.Indexes, model.IndexInfo{
			Name:    indexName,
			Columns: columns,
			Unique:  isUnique == 1,
			Primary: strings.HasPrefix(indexName, "sqlite_autoindex_"),
		})
	}

	return schema, nil
}

// GetViews 获取视图列表
func (a *SQLiteAdapter) GetViews(db *sql.DB, database string) ([]model.TableInfo, error) {
	query := `
		SELECT
			name,
			0 as row_count,
			0 as table_size
		FROM sqlite_master
		WHERE type = 'view'
			AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []model.TableInfo
	for rows.Next() {
		var v model.TableInfo
		if err := rows.Scan(&v.Name, &v.Rows, &v.Size); err != nil {
			return nil, err
		}
		v.Database = database
		v.TableType = "VIEW"
		views = append(views, v)
	}

	return views, nil
}

// GetProcedures 获取存储过程列表（SQLite 不支持）
func (a *SQLiteAdapter) GetProcedures(db *sql.DB, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetViewDefinition 获取视图定义
func (a *SQLiteAdapter) GetViewDefinition(db *sql.DB, database, viewName string) (string, error) {
	var definition string
	query := `SELECT sql FROM sqlite_master WHERE type = 'view' AND name = ?`
	row := db.QueryRow(query, viewName)
	if err := row.Scan(&definition); err != nil {
		return "", err
	}
	return definition, nil
}

// GetRoutineDefinition 获取存储过程或函数定义（SQLite 不支持）
func (a *SQLiteAdapter) GetRoutineDefinition(db *sql.DB, database, routineName, routineType string) (string, error) {
	return "", fmt.Errorf("SQLite doesn't support viewing routine definition natively")
}

// GetFunctions 获取函数列表（SQLite 不支持）
func (a *SQLiteAdapter) GetFunctions(db *sql.DB, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetIndexes 获取索引列表
func (a *SQLiteAdapter) GetIndexes(db *sql.DB, database, table string) ([]model.IndexInfo, error) {
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}
	return schema.Indexes, nil
}

// Execute 执行非查询 SQL
func (a *SQLiteAdapter) Execute(db *sql.DB, query string, args ...interface{}) (*model.ExecuteResult, error) {
	start := time.Now()

	result, err := db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()

	return &model.ExecuteResult{
		RowsAffected: rowsAffected,
		TimeCost:     time.Since(start),
		Message:      "执行成功",
	}, nil
}

// Query 执行查询
func (a *SQLiteAdapter) Query(db *sql.DB, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	start := time.Now()

	trimQuery := strings.TrimSpace(strings.ToUpper(query))
	isQuery := strings.HasPrefix(trimQuery, "SELECT") ||
		strings.HasPrefix(trimQuery, "EXPLAIN") ||
		strings.HasPrefix(trimQuery, "PRAGMA") ||
		strings.HasPrefix(trimQuery, "WITH")

	if !isQuery {
		result, err := db.Exec(query)
		if err != nil {
			return nil, err
		}
		rowsAffected, _ := result.RowsAffected()
		return &model.QueryResult{
			RowsAffected: rowsAffected,
			Message:      "执行成功",
			TimeCost:     time.Since(start),
		}, nil
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var rowData []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col] = nil
			} else if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		rowData = append(rowData, row)
	}

	return &model.QueryResult{
		Columns:  columns,
		Rows:     rowData,
		Total:    int64(len(rowData)),
		Message:  "查询成功",
		TimeCost: time.Since(start),
	}, nil
}

// Insert 插入数据
func (a *SQLiteAdapter) Insert(db *sql.DB, database, table string, data map[string]interface{}) error {
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	_, err := db.Exec(query, values...)
	return err
}

// Update 更新数据
func (a *SQLiteAdapter) Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error {
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, fmt.Sprintf("`%s` = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE %s",
		table,
		strings.Join(sets, ", "),
		where)

	_, err := db.Exec(query, values...)
	return err
}

// Delete 删除数据
func (a *SQLiteAdapter) Delete(db *sql.DB, database, table, where string) error {
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", table, where)
	_, err := db.Exec(query)
	return err
}

// ExportToCSV 导出为 CSV
func (a *SQLiteAdapter) ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	exporter := export.NewCSVExporter(opts)

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	colNames := make([]string, len(columns))
	for i, col := range columns {
		colNames[i] = col.Name()
	}

	var rowData []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val == nil {
				row[col.Name()] = nil
			} else if b, ok := val.([]byte); ok {
				row[col.Name()] = string(b)
			} else {
				row[col.Name()] = val
			}
		}
		rowData = append(rowData, row)
	}

	return exporter.Export(writer, colNames, rowData)
}

// ExportToSQL 导出为 SQL
func (a *SQLiteAdapter) ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	exporter := export.NewSQLExporter(opts, model.DatabaseSQLite)

	// 如果提供了自定义查询，则按查询导出
	if opts.Query != "" {
		rows, err := db.Query(opts.Query)
		if err != nil {
			return err
		}
		defer rows.Close()

		columns, err := rows.ColumnTypes()
		if err != nil {
			return err
		}

		colNames := make([]string, len(columns))
		for i, col := range columns {
			colNames[i] = col.Name()
		}

		var rowData []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return err
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				if val == nil {
					row[col.Name()] = nil
				} else if b, ok := val.([]byte); ok {
					row[col.Name()] = string(b)
				} else {
					row[col.Name()] = val
				}
			}
			rowData = append(rowData, row)
		}

		// 使用自定义表名，如果没有指定则使用 "query_result" 作为默认值
		tableName := opts.TableName
		if tableName == "" {
			tableName = "query_result"
		}
		return exporter.ExportData(writer, "", tableName, colNames, rowData)
	}

	for _, table := range tables {
		// 导出表结构
		if opts.IncludeCreateTable || opts.StructureOnly {
			schema, err := a.GetTableSchema(db, "", table)
			if err != nil {
				return err
			}
			if err := exporter.ExportSchema(writer, schema); err != nil {
				return err
			}
		}

		// 导出数据
		if !opts.StructureOnly {
			query := fmt.Sprintf("SELECT * FROM `%s`", table)
			if opts.MaxRows > 0 {
				query = fmt.Sprintf("%s LIMIT %d", query, opts.MaxRows)
			}
			rows, err := db.Query(query)
			if err != nil {
				return err
			}

			columns, err := rows.ColumnTypes()
			if err != nil {
				rows.Close()
				return err
			}

			colNames := make([]string, len(columns))
			for i, col := range columns {
				colNames[i] = col.Name()
			}

			var rowData []map[string]interface{}
			for rows.Next() {
				values := make([]interface{}, len(columns))
				valuePtrs := make([]interface{}, len(columns))
				for i := range values {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					rows.Close()
					return err
				}

				row := make(map[string]interface{})
				for i, col := range columns {
					val := values[i]
					if val == nil {
						row[col.Name()] = nil
					} else if b, ok := val.([]byte); ok {
						row[col.Name()] = string(b)
					} else {
						row[col.Name()] = val
					}
				}
				rowData = append(rowData, row)
			}
			rows.Close()

			if err := exporter.ExportData(writer, "", table, colNames, rowData); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCreateTableSQL 获取建表语句
func (a *SQLiteAdapter) GetCreateTableSQL(db *sql.DB, database, table string) (string, error) {
	var createSQL string
	query := fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%s'", table)
	row := db.QueryRow(query)

	if err := row.Scan(&createSQL); err != nil {
		return "", err
	}

	return createSQL, nil
}

// AlterTable 修改表结构
func (a *SQLiteAdapter) AlterTable(db *sql.DB, request *model.AlterTableRequest) error {
	if len(request.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	// SQLite 对 ALTER TABLE 支持有限，需要根据操作类型选择策略
	for _, action := range request.Actions {
		var err error

		switch action.Type {
		case model.AlterActionAddColumn:
			// SQLite 支持 ADD COLUMN
			err = a.addColumn(db, request.Table, action.Column)
		case model.AlterActionRenameColumn:
			// SQLite 3.25.0+ 支持 RENAME COLUMN
			err = a.renameColumn(db, request.Table, action.OldName, action.NewName)
		case model.AlterActionDropColumn, model.AlterActionModifyColumn:
			// SQLite 不支持 DROP COLUMN 和 MODIFY COLUMN，需要重建表
			return fmt.Errorf("SQLite does not support DROP/MODIFY COLUMN directly, table rebuild required")
		case model.AlterActionAddIndex:
			err = a.addIndex(db, request.Table, action.Index)
		case model.AlterActionDropIndex:
			err = a.dropIndex(db, action.OldName)
		default:
			return fmt.Errorf("unsupported action type: %s", action.Type)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// addColumn 添加列
func (a *SQLiteAdapter) addColumn(db *sql.DB, table string, col *model.ColumnDef) error {
	if col == nil {
		return fmt.Errorf("column definition is required")
	}

	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s",
		table, col.Name, a.buildColumnType(col))

	_, err := db.Exec(sql)
	return err
}

// renameColumn 重命名列
func (a *SQLiteAdapter) renameColumn(db *sql.DB, table, oldName, newName string) error {
	sql := fmt.Sprintf("ALTER TABLE `%s` RENAME COLUMN `%s` TO `%s`",
		table, oldName, newName)
	_, err := db.Exec(sql)
	return err
}

// buildColumnType 构建列类型定义
func (a *SQLiteAdapter) buildColumnType(col *model.ColumnDef) string {
	var parts []string

	// 类型
	colType := strings.ToUpper(col.Type)
	if col.Length > 0 {
		colType = fmt.Sprintf("%s(%d)", colType, col.Length)
	} else if col.Precision > 0 {
		if col.Scale > 0 {
			colType = fmt.Sprintf("%s(%d,%d)", colType, col.Precision, col.Scale)
		} else {
			colType = fmt.Sprintf("%s(%d)", colType, col.Precision)
		}
	}
	parts = append(parts, colType)

	// 可空性
	if !col.Nullable {
		parts = append(parts, "NOT NULL")
	}

	// 默认值
	if col.DefaultValue != "" {
		if strings.ToUpper(col.DefaultValue) == "NULL" {
			parts = append(parts, "DEFAULT NULL")
		} else if strings.ToUpper(col.DefaultValue) == "CURRENT_TIMESTAMP" {
			parts = append(parts, "DEFAULT CURRENT_TIMESTAMP")
		} else {
			parts = append(parts, fmt.Sprintf("DEFAULT '%s'", strings.ReplaceAll(col.DefaultValue, "'", "''")))
		}
	}

	// 自增
	if col.AutoIncrement {
		parts = append(parts, "AUTOINCREMENT")
	}

	return strings.Join(parts, " ")
}

// addIndex 添加索引
func (a *SQLiteAdapter) addIndex(db *sql.DB, table string, idx *model.IndexDef) error {
	if idx == nil {
		return fmt.Errorf("index definition is required")
	}

	if len(idx.Columns) == 0 {
		return fmt.Errorf("index columns are required")
	}

	columns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		columns[i] = fmt.Sprintf("`%s`", col)
	}

	var sql string
	if idx.Unique {
		sql = fmt.Sprintf("CREATE UNIQUE INDEX `%s` ON `%s` (%s)",
			idx.Name, table, strings.Join(columns, ", "))
	} else {
		sql = fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)",
			idx.Name, table, strings.Join(columns, ", "))
	}

	_, err := db.Exec(sql)
	return err
}

// dropIndex 删除索引
func (a *SQLiteAdapter) dropIndex(db *sql.DB, indexName string) error {
	sql := fmt.Sprintf("DROP INDEX `%s`", indexName)
	_, err := db.Exec(sql)
	return err
}

// RenameTable 重命名表
func (a *SQLiteAdapter) RenameTable(db *sql.DB, database, oldName, newName string) error {
	sql := fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", oldName, newName)
	_, err := db.Exec(sql)
	return err
}

// rebuildTable 重建表（用于不支持的 ALTER 操作）
// 注意：这是一个复杂操作，需要谨慎使用
func (a *SQLiteAdapter) rebuildTable(db *sql.DB, table string, newSchema *model.TableSchema) error {
	// 1. 开启事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}
	defer tx.Rollback()

	// 2. 创建新表
	tempTable := table + "_new"
	createSQL := a.buildCreateTableSQL(tempTable, newSchema)
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("create temp table failed: %w", err)
	}

	// 3. 复制数据
	columns := make([]string, len(newSchema.Columns))
	for i, col := range newSchema.Columns {
		columns[i] = fmt.Sprintf("`%s`", col.Name)
	}
	copySQL := fmt.Sprintf("INSERT INTO `%s` (%s) SELECT %s FROM `%s`",
		tempTable,
		strings.Join(columns, ", "),
		strings.Join(columns, ", "),
		table)
	if _, err := tx.Exec(copySQL); err != nil {
		return fmt.Errorf("copy data failed: %w", err)
	}

	// 4. 删除旧表
	dropSQL := fmt.Sprintf("DROP TABLE `%s`", table)
	if _, err := tx.Exec(dropSQL); err != nil {
		return fmt.Errorf("drop old table failed: %w", err)
	}

	// 5. 重命名新表
	renameSQL := fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", tempTable, table)
	if _, err := tx.Exec(renameSQL); err != nil {
		return fmt.Errorf("rename table failed: %w", err)
	}

	// 6. 提交事务
	return tx.Commit()
}

// buildCreateTableSQL 构建建表 SQL
func (a *SQLiteAdapter) buildCreateTableSQL(table string, schema *model.TableSchema) string {
	var columns []string
	for _, col := range schema.Columns {
		colDef := &model.ColumnDef{
			Name:          col.Name,
			Type:          col.Type,
			Nullable:      col.Nullable,
			DefaultValue:  col.DefaultValue,
			AutoIncrement: strings.Contains(col.Extra, "auto_increment"),
		}
		columns = append(columns, fmt.Sprintf("`%s` %s", col.Name, a.buildColumnType(colDef)))
	}

	return fmt.Sprintf("CREATE TABLE `%s` (%s)", table, strings.Join(columns, ", "))
}
