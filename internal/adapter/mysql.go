package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"fmt"
	"io"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLAdapter MySQL 数据库适配器
type MySQLAdapter struct {
	*BaseAdapter
}

// NewMySQLAdapter 创建 MySQL 适配器
func NewMySQLAdapter() *MySQLAdapter {
	return &MySQLAdapter{
		BaseAdapter: NewBaseAdapter(),
	}
}

// Connect 连接 MySQL 数据库
func (a *MySQLAdapter) Connect(config *model.ConnectionConfig) (any, error) {
	dsn := a.buildDSN(config)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// buildDSN 构建 MySQL DSN
func (a *MySQLAdapter) buildDSN(config *model.ConnectionConfig) string {
	var params []string

	// 基础参数
	params = append(params, fmt.Sprintf("%s:%s@tcp(%s:%d)",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
	))

	if config.Database != "" {
		params[0] += "/" + config.Database
	} else {
		params[0] += "/"
	}

	// 额外参数
	if len(config.Params) > 0 {
		var paramStrs []string
		for k, v := range config.Params {
			paramStrs = append(paramStrs, fmt.Sprintf("%s=%s", k, v))
		}
		if len(paramStrs) > 0 {
			params[0] += "?" + strings.Join(paramStrs, "&")
		}
	}

	return params[0]
}

// Close 关闭连接
func (a *MySQLAdapter) Close(db any) error {
	return db.(*sql.DB).Close()
}

// Ping 测试连接
func (a *MySQLAdapter) Ping(db any) error {
	return db.(*sql.DB).Ping()
}

// GetDatabases 获取数据库列表
func (a *MySQLAdapter) GetDatabases(db any) ([]string, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT SCHEMA_NAME
		FROM INFORMATION_SCHEMA.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
	`

	rows, err := dbSQL.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}

	return databases, nil
}

// GetTables 获取表列表
func (a *MySQLAdapter) GetTables(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			TABLE_ROWS,
			DATA_LENGTH,
			TABLE_COMMENT
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?
			AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')
		ORDER BY TABLE_NAME
	`

	rows, err := dbSQL.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []model.TableInfo
	for rows.Next() {
		var t model.TableInfo
		var tableType, comment sql.NullString
		if err := rows.Scan(&t.Name, &tableType, &t.Rows, &t.Size, &comment); err != nil {
			return nil, err
		}
		t.Database = database
		t.TableType = tableType.String
		t.Comment = comment.String
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *MySQLAdapter) GetTableSchema(db any, database, table string) (*model.TableSchema, error) {
	dbSQL := db.(*sql.DB)
	schema := &model.TableSchema{
		Database: database,
		Table:    table,
	}

	// 获取列信息
	colsQuery := `
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_DEFAULT,
			COLUMN_KEY,
			EXTRA,
			COLUMN_COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	colsRows, err := dbSQL.Query(colsQuery, database, table)
	if err != nil {
		return nil, err
	}
	defer colsRows.Close()

	for colsRows.Next() {
		var col model.ColumnInfo
		var nullable, key, extra, def, comment sql.NullString
		if err := colsRows.Scan(&col.Name, &col.Type, &nullable, &def, &key, &extra, &comment); err != nil {
			return nil, err
		}
		col.Nullable = nullable.String == "YES"
		col.DefaultValue = def.String
		col.Key = key.String
		col.Extra = extra.String
		col.Comment = comment.String
		schema.Columns = append(schema.Columns, col)
	}

	// 获取索引信息
	idxQuery := `
		SELECT
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			INDEX_TYPE
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	idxRows, err := dbSQL.Query(idxQuery, database, table)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	indexMap := make(map[string]*model.IndexInfo)
	for idxRows.Next() {
		var indexName, column, indexType string
		var nonUnique int
		if err := idxRows.Scan(&indexName, &column, &nonUnique, &indexType); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &model.IndexInfo{
				Name:    indexName,
				Unique:  nonUnique == 0,
				Primary: indexName == "PRIMARY",
			}
		}
		indexMap[indexName].Columns = append(indexMap[indexName].Columns, column)
	}

	for _, idx := range indexMap {
		schema.Indexes = append(schema.Indexes, *idx)
	}

	return schema, nil
}

// GetViews 获取视图列表
func (a *MySQLAdapter) GetViews(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			TABLE_NAME,
			TABLE_ROWS,
			DATA_LENGTH
		FROM INFORMATION_SCHEMA.VIEWS
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`

	rows, err := dbSQL.Query(query, database)
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

// GetProcedures 获取存储过程列表
func (a *MySQLAdapter) GetProcedures(db any, database string) ([]model.RoutineInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			ROUTINE_NAME,
			ROUTINE_COMMENT
		FROM INFORMATION_SCHEMA.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'PROCEDURE'
		ORDER BY ROUTINE_NAME
	`

	rows, err := dbSQL.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var procedures []model.RoutineInfo
	for rows.Next() {
		var p model.RoutineInfo
		var comment sql.NullString
		if err := rows.Scan(&p.Name, &comment); err != nil {
			return nil, err
		}
		p.Database = database
		p.Type = "PROCEDURE"
		p.Comment = comment.String
		procedures = append(procedures, p)
	}

	return procedures, nil
}

// GetViewDefinition 获取视图定义
func (a *MySQLAdapter) GetViewDefinition(db any, database, viewName string) (string, error) {
	dbSQL := db.(*sql.DB)
	var definition string
	query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", database, viewName)
	row := dbSQL.QueryRow(query)

	var viewNameResult, createViewSQL, characterSet, collationConnection sql.NullString
	if err := row.Scan(&viewNameResult, &createViewSQL, &characterSet, &collationConnection); err != nil {
		return "", err
	}

	definition = createViewSQL.String
	if definition == "" {
		definition = "/* 无法获取视图定义，可能由于权限不足 */"
	}
	return definition, nil
}

// GetRoutineDefinition 获取存储过程或函数定义
func (a *MySQLAdapter) GetRoutineDefinition(db any, database, routineName, routineType string) (string, error) {
	dbSQL := db.(*sql.DB)
	var definition string
	var query string
	if strings.ToUpper(routineType) == "FUNCTION" {
		query = fmt.Sprintf("SHOW CREATE FUNCTION `%s`.`%s`", database, routineName)
	} else {
		query = fmt.Sprintf("SHOW CREATE PROCEDURE `%s`.`%s`", database, routineName)
	}

	row := dbSQL.QueryRow(query)
	var routineNameResult, sqlMode, createRoutineSQL, characterSet, collationConnection, databaseCollation sql.NullString
	if err := row.Scan(&routineNameResult, &sqlMode, &createRoutineSQL, &characterSet, &collationConnection, &databaseCollation); err != nil {
		return "", err
	}

	definition = createRoutineSQL.String
	if definition == "" {
		definition = "/* 无法获取定义，可能由于权限不足 */"
	}
	return definition, nil
}

// GetFunctions 获取函数列表
func (a *MySQLAdapter) GetFunctions(db any, database string) ([]model.RoutineInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			ROUTINE_NAME,
			ROUTINE_COMMENT
		FROM INFORMATION_SCHEMA.ROUTINES
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_TYPE = 'FUNCTION'
		ORDER BY ROUTINE_NAME
	`

	rows, err := dbSQL.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []model.RoutineInfo
	for rows.Next() {
		var f model.RoutineInfo
		var comment sql.NullString
		if err := rows.Scan(&f.Name, &comment); err != nil {
			return nil, err
		}
		f.Database = database
		f.Type = "FUNCTION"
		f.Comment = comment.String
		functions = append(functions, f)
	}

	return functions, nil
}

// GetIndexes 获取索引列表
func (a *MySQLAdapter) GetIndexes(db any, database, table string) ([]model.IndexInfo, error) {
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}
	return schema.Indexes, nil
}

// Execute 执行非查询 SQL
func (a *MySQLAdapter) Execute(db any, query string, args ...interface{}) (*model.ExecuteResult, error) {
	dbSQL := db.(*sql.DB)
	start := time.Now()

	result, err := dbSQL.Exec(query, args...)
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
func (a *MySQLAdapter) Query(db any, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	dbSQL := db.(*sql.DB)
	start := time.Now()

	trimQuery := strings.TrimSpace(strings.ToUpper(query))
	isQuery := strings.HasPrefix(trimQuery, "SELECT") ||
		strings.HasPrefix(trimQuery, "SHOW") ||
		strings.HasPrefix(trimQuery, "DESC") ||
		strings.HasPrefix(trimQuery, "EXPLAIN") ||
		strings.HasPrefix(trimQuery, "WITH") ||
		strings.HasPrefix(trimQuery, "PRAGMA")

	if !isQuery {
		result, err := dbSQL.Exec(query)
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

	rows, err := dbSQL.Query(query)
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
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// 转换为 map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理 NULL 值
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
func (a *MySQLAdapter) Insert(db any, database, table string, data map[string]interface{}) error {
	dbSQL := db.(*sql.DB)
	columns := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		database, table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	_, err := dbSQL.Exec(query, values...)
	return err
}

// Update 更新数据
func (a *MySQLAdapter) Update(db any, database, table string, data map[string]interface{}, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, fmt.Sprintf("`%s` = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s",
		database, table,
		strings.Join(sets, ", "),
		where)

	_, err := dbSQL.Exec(query, values...)
	return err
}

// Delete 删除数据
func (a *MySQLAdapter) Delete(db any, database, table, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	query := fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s", database, table, where)
	_, err := dbSQL.Exec(query)
	return err
}

// ExportToCSV 导出为 CSV
func (a *MySQLAdapter) ExportToCSV(db any, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewCSVExporter(opts)

	rows, err := dbSQL.Query(query)
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
func (a *MySQLAdapter) ExportToSQL(db any, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewSQLExporter(opts, model.DatabaseMySQL)

	// 如果提供了自定义查询，则按查询导出
	if opts.Query != "" {
		rows, err := dbSQL.Query(opts.Query)
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
			var createSQL string
			// 尝试获取原生建表语句
			if rawSQL, err := a.GetCreateTableSQL(db, database, table); err == nil {
				createSQL = rawSQL + ";\n\n"
				if opts.IncludeDropTable {
					if database != "" {
						createSQL = fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`;\n\n", database, table) + createSQL
					} else {
						createSQL = fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n\n", table) + createSQL
					}
				}
				if _, err := writer.Write([]byte(createSQL)); err != nil {
					return err
				}
			} else {
				// 降级：使用通用 Exporter 重建（如果获取原生失败）
				schema, err := a.GetTableSchema(db, database, table)
				if err != nil {
					return err
				}
				if err := exporter.ExportSchema(writer, schema); err != nil {
					return err
				}
			}
		}

		// 导出数据
		if !opts.StructureOnly {
			query := fmt.Sprintf("SELECT * FROM `%s`", table)
			if opts.MaxRows > 0 {
				query = fmt.Sprintf("%s LIMIT %d", query, opts.MaxRows)
			}
			rows, err := dbSQL.Query(query)
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
func (a *MySQLAdapter) GetCreateTableSQL(db any, database, table string) (string, error) {
	dbSQL := db.(*sql.DB)
	var createSQL string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, table)
	row := dbSQL.QueryRow(query)

	var tableName, createTableSQL sql.NullString
	if err := row.Scan(&tableName, &createTableSQL); err != nil {
		return "", err
	}

	createSQL = createTableSQL.String
	return createSQL, nil
}

// AlterTable 修改表结构
func (a *MySQLAdapter) AlterTable(db any, request *model.AlterTableRequest) error {
	dbSQL := db.(*sql.DB)
	if len(request.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	// 构建 ALTER TABLE 语句
	var alterClauses []string
	for _, action := range request.Actions {
		clause, err := a.buildAlterClause(action)
		if err != nil {
			return fmt.Errorf("build alter clause failed: %w", err)
		}
		alterClauses = append(alterClauses, clause)
	}

	// 执行 ALTER TABLE
	sql := fmt.Sprintf("ALTER TABLE `%s`.`%s` %s",
		request.Database,
		request.Table,
		strings.Join(alterClauses, ", "))

	_, err := dbSQL.Exec(sql)
	return err
}

// buildAlterClause 构建单个 ALTER 子句
func (a *MySQLAdapter) buildAlterClause(action model.AlterTableAction) (string, error) {
	switch action.Type {
	case model.AlterActionAddColumn:
		return a.buildAddColumnClause(action.Column)
	case model.AlterActionDropColumn:
		return fmt.Sprintf("DROP COLUMN `%s`", action.OldName), nil
	case model.AlterActionModifyColumn:
		return a.buildModifyColumnClause(action.Column)
	case model.AlterActionRenameColumn:
		if action.Column == nil {
			return "", fmt.Errorf("column definition required for rename")
		}
		return fmt.Sprintf("CHANGE COLUMN `%s` `%s` %s",
			action.OldName,
			action.NewName,
			a.buildColumnType(action.Column)), nil
	case model.AlterActionAddIndex:
		return a.buildAddIndexClause(action.Index)
	case model.AlterActionDropIndex:
		return fmt.Sprintf("DROP INDEX `%s`", action.OldName), nil
	default:
		return "", fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// buildAddColumnClause 构建添加列子句
func (a *MySQLAdapter) buildAddColumnClause(col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	clause := fmt.Sprintf("ADD COLUMN `%s` %s", col.Name, a.buildColumnType(col))

	if col.After != "" {
		clause += fmt.Sprintf(" AFTER `%s`", col.After)
	}

	return clause, nil
}

// buildModifyColumnClause 构建修改列子句
func (a *MySQLAdapter) buildModifyColumnClause(col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	return fmt.Sprintf("MODIFY COLUMN `%s` %s", col.Name, a.buildColumnType(col)), nil
}

// buildColumnType 构建列类型定义
func (a *MySQLAdapter) buildColumnType(col *model.ColumnDef) string {
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
	} else {
		parts = append(parts, "NULL")
	}

	// 默认值
	if col.DefaultValue != "" {
		if strings.ToUpper(col.DefaultValue) == "NULL" {
			parts = append(parts, "DEFAULT NULL")
		} else if strings.ToUpper(col.DefaultValue) == "CURRENT_TIMESTAMP" {
			parts = append(parts, "DEFAULT CURRENT_TIMESTAMP")
		} else {
			parts = append(parts, fmt.Sprintf("DEFAULT '%s'", col.DefaultValue))
		}
	}

	// 自增
	if col.AutoIncrement {
		parts = append(parts, "AUTO_INCREMENT")
	}

	// 注释
	if col.Comment != "" {
		parts = append(parts, fmt.Sprintf("COMMENT '%s'", strings.ReplaceAll(col.Comment, "'", "''")))
	}

	return strings.Join(parts, " ")
}

// buildAddIndexClause 构建添加索引子句
func (a *MySQLAdapter) buildAddIndexClause(idx *model.IndexDef) (string, error) {
	if idx == nil {
		return "", fmt.Errorf("index definition is required")
	}

	if len(idx.Columns) == 0 {
		return "", fmt.Errorf("index columns are required")
	}

	var indexType string
	if idx.Unique {
		indexType = "UNIQUE INDEX"
	} else {
		indexType = "INDEX"
	}

	columns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		columns[i] = fmt.Sprintf("`%s`", col)
	}

	clause := fmt.Sprintf("ADD %s `%s` (%s)",
		indexType,
		idx.Name,
		strings.Join(columns, ", "))

	if idx.Type != "" {
		clause += fmt.Sprintf(" USING %s", strings.ToUpper(idx.Type))
	}

	if idx.Comment != "" {
		clause += fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(idx.Comment, "'", "''"))
	}

	return clause, nil
}

// RenameTable 重命名表
func (a *MySQLAdapter) RenameTable(db any, database, oldName, newName string) error {
	dbSQL := db.(*sql.DB)
	sql := fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`",
		database, oldName, database, newName)
	_, err := dbSQL.Exec(sql)
	return err
}
