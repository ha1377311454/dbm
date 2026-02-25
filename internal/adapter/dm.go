package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"fmt"
	"io"
	"strings"
	"time"

	_ "gitee.com/chunanyong/dm"
)

// DMAdapter 达梦数据库适配器
type DMAdapter struct {
	*BaseAdapter
}

// NewDMAdapter 创建达梦数据库适配器
func NewDMAdapter() *DMAdapter {
	return &DMAdapter{
		BaseAdapter: NewBaseAdapter(),
	}
}

// Connect 连接达梦数据库
func (a *DMAdapter) Connect(config *model.ConnectionConfig) (any, error) {
	dsn := a.buildDSN(config)
	db, err := sql.Open("dm", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// buildDSN 构建达梦数据库 DSN
// 达梦连接格式: dm://用户名:密码@主机:端口?schema=数据库名
func (a *DMAdapter) buildDSN(config *model.ConnectionConfig) string {
	// 构建 URL 格式: dm://用户名:密码@主机:端口
	var dsn string

	// 构建用户名:密码部分
	userPass := ""
	if config.Username != "" {
		userPass = config.Username
		if config.Password != "" {
			userPass = fmt.Sprintf("%s:%s", config.Username, config.Password)
		}
	}

	// 构建主机:端口部分
	hostPort := config.Host
	if config.Port > 0 {
		hostPort = fmt.Sprintf("%s:%d", config.Host, config.Port)
	}

	// 组合基础 DSN
	if userPass != "" {
		dsn = fmt.Sprintf("dm://%s@%s", userPass, hostPort)
	} else {
		dsn = fmt.Sprintf("dm://%s", hostPort)
	}

	// 添加额外参数（如 schema）
	var params []string
	for k, v := range config.Params {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	// 如果没有指定 schema 但有 database 参数，则添加 schema
	if len(params) == 0 && config.Database != "" {
		params = append(params, fmt.Sprintf("schema=%s", config.Database))
	}

	if len(params) > 0 {
		dsn = fmt.Sprintf("%s?%s", dsn, strings.Join(params, "&"))
	}

	return dsn
}

// Close 关闭数据库连接
func (a *DMAdapter) Close(db any) error {
	return db.(*sql.DB).Close()
}

// Ping 测试数据库连接
func (a *DMAdapter) Ping(db any) error {
	return db.(*sql.DB).Ping()
}

// GetDatabases 获取数据库列表
// 达梦中使用模式(Schema)概念
func (a *DMAdapter) GetDatabases(db any) ([]string, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT DISTINCT OWNER
		FROM ALL_OBJECTS
		WHERE OBJECT_TYPE IN ('TABLE', 'VIEW')
			AND OWNER NOT IN ('SYS', 'SYSTEM', 'SYSAUX', 'SYSDBA')
		ORDER BY OWNER
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
func (a *DMAdapter) GetTables(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			TABLE_NAME,
			0 as ROW_COUNT,
			0 as TABLE_SIZE
		FROM ALL_TABLES
		WHERE OWNER = :1
		ORDER BY TABLE_NAME
	`

	rows, err := dbSQL.Query(query, strings.ToUpper(database))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []model.TableInfo
	for rows.Next() {
		var t model.TableInfo
		if err := rows.Scan(&t.Name, &t.Rows, &t.Size); err != nil {
			return nil, err
		}
		t.Database = database
		t.Schema = database
		t.TableType = "BASE TABLE"
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *DMAdapter) GetTableSchema(db any, database, table string) (*model.TableSchema, error) {
	dbSQL := db.(*sql.DB)
	tableSchema := &model.TableSchema{
		Database: database,
		Table:    table,
	}

	// 获取列信息
	colsQuery := `
		SELECT
			COLUMN_NAME,
			DATA_TYPE,
			NULLABLE,
			DATA_DEFAULT,
			'' as COLUMN_KEY,
			'' as EXTRA
		FROM ALL_TAB_COLUMNS
		WHERE OWNER = :1 AND TABLE_NAME = :2
		ORDER BY COLUMN_ID
	`

	colsRows, err := dbSQL.Query(colsQuery, strings.ToUpper(database), strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer colsRows.Close()

	// 构建列名列表，用于后续查询注释
	var colNames []string

	for colsRows.Next() {
		var col model.ColumnInfo
		var colType, nullable, def, key, extra sql.NullString

		if err := colsRows.Scan(&col.Name, &colType, &nullable, &def, &key, &extra); err != nil {
			return nil, err
		}

		// 构建类型字符串
		col.Type = a.buildTypeString(colType.String, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{})
		col.Nullable = nullable.String == "Y"
		col.DefaultValue = def.String
		col.Key = key.String
		col.Extra = extra.String
		col.Comment = "" // 初始为空，后面从 ALL_COL_COMMENTS 获取
		tableSchema.Columns = append(tableSchema.Columns, col)
		colNames = append(colNames, col.Name)
	}

	// 获取列注释（从 ALL_COL_COMMENTS）
	if len(colNames) > 0 {
		commentQuery := `
			SELECT COLUMN_NAME, COMMENTS
			FROM ALL_COL_COMMENTS
			WHERE OWNER = :1 AND TABLE_NAME = :2
		`
		commentRows, err := dbSQL.Query(commentQuery, strings.ToUpper(database), strings.ToUpper(table))
		if err == nil {
			defer commentRows.Close()
			commentMap := make(map[string]string)
			for commentRows.Next() {
				var colName, comments sql.NullString
				if err := commentRows.Scan(&colName, &comments); err != nil {
					continue
				}
				commentMap[colName.String] = comments.String
			}
			// 更新列注释
			for i := range tableSchema.Columns {
				if comment, ok := commentMap[tableSchema.Columns[i].Name]; ok {
					tableSchema.Columns[i].Comment = comment
				}
			}
		}
	}

	// 获取索引信息（从 ALL_INDEXES 获取唯一性，从 ALL_IND_COLUMNS 获取列）
	idxQuery := `
		SELECT
			ic.INDEX_NAME,
			ic.COLUMN_NAME,
			i.UNIQUENESS
		FROM ALL_IND_COLUMNS ic
		LEFT JOIN ALL_INDEXES i ON ic.INDEX_NAME = i.INDEX_NAME AND ic.TABLE_OWNER = i.TABLE_OWNER
		WHERE ic.TABLE_OWNER = :1 AND ic.TABLE_NAME = :2
		ORDER BY ic.INDEX_NAME, ic.COLUMN_POSITION
	`

	idxRows, err := dbSQL.Query(idxQuery, strings.ToUpper(database), strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	indexMap := make(map[string]*model.IndexInfo)
	for idxRows.Next() {
		var indexName, column, uniqueness sql.NullString
		if err := idxRows.Scan(&indexName, &column, &uniqueness); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName.String]; !exists {
			indexMap[indexName.String] = &model.IndexInfo{
				Name:   indexName.String,
				Unique: uniqueness.String == "UNIQUE",
			}
		}
		indexMap[indexName.String].Columns = append(indexMap[indexName.String].Columns, column.String)
	}

	for _, idx := range indexMap {
		tableSchema.Indexes = append(tableSchema.Indexes, *idx)
	}

	return tableSchema, nil
}

// buildTypeString 构建类型字符串
func (a *DMAdapter) buildTypeString(dataType string, length, precision, scale sql.NullInt64) string {
	dt := strings.ToUpper(dataType)

	switch dt {
	case "VARCHAR", "VARCHAR2", "CHAR":
		if length.Valid && length.Int64 > 0 {
			return fmt.Sprintf("%s(%d)", dt, length.Int64)
		}
	case "NUMBER":
		if precision.Valid {
			if scale.Valid && scale.Int64 > 0 {
				return fmt.Sprintf("NUMBER(%d,%d)", precision.Int64, scale.Int64)
			}
			return fmt.Sprintf("NUMBER(%d)", precision.Int64)
		}
	case "DECIMAL", "NUMERIC":
		if precision.Valid {
			if scale.Valid && scale.Int64 > 0 {
				return fmt.Sprintf("%s(%d,%d)", dt, precision.Int64, scale.Int64)
			}
			return fmt.Sprintf("%s(%d)", dt, precision.Int64)
		}
	case "TIMESTAMP":
		return "TIMESTAMP"
	}

	return dt
}

// GetViews 获取视图列表
func (a *DMAdapter) GetViews(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT
			VIEW_NAME,
			0 as ROW_COUNT,
			0 as TABLE_SIZE
		FROM ALL_VIEWS
		WHERE OWNER = :1
		ORDER BY VIEW_NAME
	`

	rows, err := dbSQL.Query(query, strings.ToUpper(database))
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
		v.Schema = database
		v.TableType = "VIEW"
		views = append(views, v)
	}

	return views, nil
}

// GetProcedures 获取存储过程列表
func (a *DMAdapter) GetProcedures(db any, database string) ([]model.RoutineInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT OBJECT_NAME
		FROM ALL_OBJECTS
		WHERE OWNER = :1 AND OBJECT_TYPE = 'PROCEDURE'
		ORDER BY OBJECT_NAME
	`

	rows, err := dbSQL.Query(query, strings.ToUpper(database))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var procedures []model.RoutineInfo
	for rows.Next() {
		var p model.RoutineInfo
		if err := rows.Scan(&p.Name); err != nil {
			return nil, err
		}
		p.Database = database
		p.Schema = database
		p.Type = "PROCEDURE"
		procedures = append(procedures, p)
	}

	return procedures, nil
}

// GetFunctions 获取函数列表
func (a *DMAdapter) GetFunctions(db any, database string) ([]model.RoutineInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT OBJECT_NAME
		FROM ALL_OBJECTS
		WHERE OWNER = :1 AND OBJECT_TYPE = 'FUNCTION'
		ORDER BY OBJECT_NAME
	`

	rows, err := dbSQL.Query(query, strings.ToUpper(database))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []model.RoutineInfo
	for rows.Next() {
		var f model.RoutineInfo
		if err := rows.Scan(&f.Name); err != nil {
			return nil, err
		}
		f.Database = database
		f.Schema = database
		f.Type = "FUNCTION"
		functions = append(functions, f)
	}

	return functions, nil
}

// GetViewDefinition 获取视图定义
func (a *DMAdapter) GetViewDefinition(db any, database, viewName string) (string, error) {
	dbSQL := db.(*sql.DB)
	var definition string
	query := `SELECT DBMS_METADATA.GET_DDL('VIEW', :1, :2) FROM DUAL`
	row := dbSQL.QueryRow(query, strings.ToUpper(viewName), strings.ToUpper(database))
	if err := row.Scan(&definition); err != nil {
		return "", err
	}
	return definition, nil
}

// GetRoutineDefinition 获取存储过程或函数定义
func (a *DMAdapter) GetRoutineDefinition(db any, database, routineName, routineType string) (string, error) {
	dbSQL := db.(*sql.DB)
	var definition string
	query := `SELECT DBMS_METADATA.GET_DDL(:1, :2, :3) FROM DUAL`
	row := dbSQL.QueryRow(query, strings.ToUpper(routineType), strings.ToUpper(routineName), strings.ToUpper(database))
	if err := row.Scan(&definition); err != nil {
		return "", err
	}
	return definition, nil
}

// GetIndexes 获取索引列表
func (a *DMAdapter) GetIndexes(db any, database, table string) ([]model.IndexInfo, error) {
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}
	return schema.Indexes, nil
}

// Execute 执行非查询 SQL
func (a *DMAdapter) Execute(db any, query string, args ...interface{}) (*model.ExecuteResult, error) {
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
func (a *DMAdapter) Query(db any, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	dbSQL := db.(*sql.DB)
	start := time.Now()

	trimQuery := strings.TrimSpace(strings.ToUpper(query))
	isQuery := strings.HasPrefix(trimQuery, "SELECT") ||
		strings.HasPrefix(trimQuery, "DESC") ||
		strings.HasPrefix(trimQuery, "EXPLAIN") ||
		strings.HasPrefix(trimQuery, "WITH")

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
func (a *DMAdapter) Insert(db any, database, table string, data map[string]interface{}) error {
	dbSQL := db.(*sql.DB)
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		cols = append(cols, fmt.Sprintf(`"%s"`, strings.ToUpper(col)))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	insertSql := fmt.Sprintf(`INSERT INTO "%s"."%s" (%s) VALUES (%s)`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	_, err := dbSQL.Exec(insertSql, values...)
	return err
}

// Update 更新数据
func (a *DMAdapter) Update(db any, database, table string, data map[string]interface{}, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, fmt.Sprintf(`"%s" = ?`, strings.ToUpper(col)))
		values = append(values, val)
	}

	updateSql := fmt.Sprintf(`UPDATE "%s"."%s" SET %s WHERE %s`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		strings.Join(sets, ", "),
		where)

	_, err := dbSQL.Exec(updateSql, values...)
	return err
}

// Delete 删除数据
func (a *DMAdapter) Delete(db any, database, table, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	deleteSql := fmt.Sprintf(`DELETE FROM "%s"."%s" WHERE %s`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		where)

	_, err := dbSQL.Exec(deleteSql)
	return err
}

// ExportToCSV 导出为 CSV
func (a *DMAdapter) ExportToCSV(db any, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewCSVExporter(opts)

	rows, err := dbSQL.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return err
	}

	var rowData []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		row := make(map[string]interface{})
		for i, colName := range colNames {
			val := values[i]
			if val == nil {
				row[colName] = nil
			} else if b, ok := val.([]byte); ok {
				row[colName] = string(b)
			} else {
				row[colName] = val
			}
		}
		rowData = append(rowData, row)
	}

	return exporter.Export(writer, colNames, rowData)
}

// ExportToSQL 导出为 SQL
func (a *DMAdapter) ExportToSQL(db any, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewSQLExporter(opts, model.DatabaseDM)

	// 如果提供了自定义查询，则按查询导出
	if opts.Query != "" {
		rows, err := dbSQL.Query(opts.Query)
		if err != nil {
			return err
		}
		defer rows.Close()

		colNames, err := rows.Columns()
		if err != nil {
			return err
		}

		var rowData []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(colNames))
			valuePtrs := make([]interface{}, len(colNames))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return err
			}

			row := make(map[string]interface{})
			for i, colName := range colNames {
				val := values[i]
				if val == nil {
					row[colName] = nil
				} else if b, ok := val.([]byte); ok {
					row[colName] = string(b)
				} else {
					row[colName] = val
				}
			}
			rowData = append(rowData, row)
		}

		tableName := opts.TableName
		if tableName == "" {
			tableName = "query_result"
		}
		return exporter.ExportData(writer, "", tableName, colNames, rowData)
	}

	for _, table := range tables {
		// 导出表结构
		if opts.IncludeCreateTable || opts.StructureOnly {
			schema, err := a.GetTableSchema(db, database, table)
			if err != nil {
				return err
			}
			if err := exporter.ExportSchema(writer, schema); err != nil {
				return err
			}
		}

		// 导出数据
		if !opts.StructureOnly {
			querySQL := fmt.Sprintf(`SELECT * FROM "%s"."%s"`, strings.ToUpper(database), strings.ToUpper(table))
			if opts.MaxRows > 0 {
				querySQL = fmt.Sprintf("%s LIMIT %d", querySQL, opts.MaxRows)
			}
			rows, err := dbSQL.Query(querySQL)
			if err != nil {
				return err
			}

			colNames, err := rows.Columns()
			if err != nil {
				rows.Close()
				return err
			}

			var rowData []map[string]interface{}
			for rows.Next() {
				values := make([]interface{}, len(colNames))
				valuePtrs := make([]interface{}, len(colNames))
				for i := range values {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					rows.Close()
					return err
				}

				row := make(map[string]interface{})
				for i, colName := range colNames {
					val := values[i]
					if val == nil {
						row[colName] = nil
					} else if b, ok := val.([]byte); ok {
						row[colName] = string(b)
					} else {
						row[colName] = val
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
func (a *DMAdapter) GetCreateTableSQL(db any, database, table string) (string, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT DBMS_METADATA.GET_DDL('TABLE', :1, :2) AS CREATE_SQL
		FROM DUAL
	`

	var createSQL sql.NullString
	err := dbSQL.QueryRow(query, strings.ToUpper(table), strings.ToUpper(database)).Scan(&createSQL)
	if err != nil {
		return "", err
	}

	if !createSQL.Valid {
		return "", fmt.Errorf("table not found: %s", table)
	}

	return createSQL.String, nil
}

// AlterTable 修改表结构
func (a *DMAdapter) AlterTable(db any, request *model.AlterTableRequest) error {
	dbSQL := db.(*sql.DB)
	if len(request.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	schemaName := strings.ToUpper(request.Database)
	tableName := strings.ToUpper(request.Table)

	for _, action := range request.Actions {
		var alterSql string
		var err error

		switch action.Type {
		case model.AlterActionAddColumn:
			alterSql, err = a.buildAddColumnSQL(schemaName, tableName, action.Column)
		case model.AlterActionDropColumn:
			alterSql = fmt.Sprintf(`ALTER TABLE "%s"."%s" DROP COLUMN "%s"`,
				schemaName, tableName, strings.ToUpper(action.OldName))
		case model.AlterActionModifyColumn:
			alterSql, err = a.buildModifyColumnSQL(schemaName, tableName, action.Column)
		case model.AlterActionRenameColumn:
			alterSql = fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME COLUMN "%s" TO "%s"`,
				schemaName, tableName, strings.ToUpper(action.OldName), strings.ToUpper(action.NewName))
		case model.AlterActionAddIndex:
			alterSql, err = a.buildAddIndexSQL(schemaName, tableName, action.Index)
		case model.AlterActionDropIndex:
			alterSql = fmt.Sprintf(`DROP INDEX "%s"."%s"`, schemaName, strings.ToUpper(action.OldName))
		default:
			return fmt.Errorf("unsupported action type: %s", action.Type)
		}

		if err != nil {
			return fmt.Errorf("build SQL failed: %w", err)
		}

		if alterSql != "" {
			if _, err := dbSQL.Exec(alterSql); err != nil {
				return fmt.Errorf("execute SQL failed: %w", err)
			}
		}
	}

	return nil
}

// buildAddColumnSQL 构建添加列 SQL
func (a *DMAdapter) buildAddColumnSQL(schema, table string, col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	sql := fmt.Sprintf(`ALTER TABLE "%s"."%s" ADD "%s" %s`,
		schema, table, strings.ToUpper(col.Name), a.buildColumnType(col))

	return sql, nil
}

// buildModifyColumnSQL 构建修改列 SQL
func (a *DMAdapter) buildModifyColumnSQL(schema, table string, col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	// 达梦修改列需要分别处理类型、可空性和默认值
	var sqls []string

	// 修改类型
	sqls = append(sqls, fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" %s`,
		schema, table, strings.ToUpper(col.Name), a.buildColumnType(col)))

	return strings.Join(sqls, "; "), nil
}

// buildColumnType 构建列类型定义
func (a *DMAdapter) buildColumnType(col *model.ColumnDef) string {
	var parts []string
	colType := strings.ToUpper(col.Type)

	// 处理类型和长度
	if col.Length > 0 {
		switch colType {
		case "VARCHAR", "VARCHAR2", "CHAR":
			colType = fmt.Sprintf("%s(%d)", colType, col.Length)
		}
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
		parts = append(parts, fmt.Sprintf("DEFAULT %s", a.formatDefaultValue(col.DefaultValue)))
	}

	return strings.Join(parts, " ")
}

// formatDefaultValue 格式化默认值
func (a *DMAdapter) formatDefaultValue(value string) string {
	upper := strings.ToUpper(value)
	if upper == "NULL" || upper == "CURRENT_TIMESTAMP" ||
		upper == "SYSDATE" || upper == "NOW()" {
		return upper
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "''"))
}

// buildAddIndexSQL 构建添加索引 SQL
func (a *DMAdapter) buildAddIndexSQL(schema, table string, idx *model.IndexDef) (string, error) {
	if idx == nil {
		return "", fmt.Errorf("index definition is required")
	}

	if len(idx.Columns) == 0 {
		return "", fmt.Errorf("index columns are required")
	}

	columns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		columns[i] = fmt.Sprintf(`"%s"`, strings.ToUpper(col))
	}

	var sql string
	if idx.Unique {
		sql = fmt.Sprintf(`CREATE UNIQUE INDEX "%s" ON "%s"."%s" (%s)`,
			strings.ToUpper(idx.Name), schema, table, strings.Join(columns, ", "))
	} else {
		sql = fmt.Sprintf(`CREATE INDEX "%s" ON "%s"."%s" (%s)`,
			strings.ToUpper(idx.Name), schema, table, strings.Join(columns, ", "))
	}

	return sql, nil
}

// RenameTable 重命名表
func (a *DMAdapter) RenameTable(db any, database, oldName, newName string) error {
	dbSQL := db.(*sql.DB)
	renameSql := fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME TO "%s"`,
		strings.ToUpper(database),
		strings.ToUpper(oldName),
		strings.ToUpper(newName))
	_, err := dbSQL.Exec(renameSql)
	return err
}
