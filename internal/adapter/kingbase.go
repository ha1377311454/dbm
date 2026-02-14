package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"fmt"
	"io"
	"strings"
	"time"

	_ "kingbase.com/gokb"
)

// KingBaseAdapter KingBase（人大金仓）数据库适配器
type KingBaseAdapter struct {
	*PostgreSQLAdapter
}

// NewKingBaseAdapter 创建 KingBase 适配器
func NewKingBaseAdapter() *KingBaseAdapter {
	return &KingBaseAdapter{
		PostgreSQLAdapter: NewPostgreSQLAdapter(),
	}
}

// Connect 连接 KingBase 数据库
func (a *KingBaseAdapter) Connect(config *model.ConnectionConfig) (*sql.DB, error) {
	dsn := a.buildDSN(config)
	db, err := sql.Open("kingbase", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// buildDSN 构建 KingBase DSN
func (a *KingBaseAdapter) buildDSN(config *model.ConnectionConfig) string {
	var params []string

	// KingBase 连接格式: host=localhost port=54321 user=system password= dbname= sslmode=disable
	params = append(params, fmt.Sprintf("host=%s", config.Host))
	params = append(params, fmt.Sprintf("port=%d", config.Port))
	params = append(params, fmt.Sprintf("user=%s", config.Username))
	if config.Password != "" {
		params = append(params, fmt.Sprintf("password=%s", config.Password))
	}
	if config.Database != "" {
		params = append(params, fmt.Sprintf("dbname=%s", config.Database))
	}

	// 默认参数
	params = append(params, "sslmode=disable")

	// 额外参数
	for k, v := range config.Params {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(params, " ")
}

// GetDatabases 获取数据库列表
func (a *KingBaseAdapter) GetDatabases(db *sql.DB) ([]string, error) {
	query := `
		SELECT datname
		FROM pg_database
		WHERE datistemplate = false
			AND datname != 'postgres'
		ORDER BY datname
	`

	rows, err := db.Query(query)
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

// GetSchemas 获取 schema 列表
func (a *KingBaseAdapter) GetSchemas(db *sql.DB, database string) ([]string, error) {
	query := `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}

	return schemas, nil
}

// GetTables 获取表列表
func (a *KingBaseAdapter) GetTables(db *sql.DB, database string) ([]model.TableInfo, error) {
	return a.GetTablesWithSchema(db, database, "public")
}

// GetTablesWithSchema 获取指定 schema 下的表列表
func (a *KingBaseAdapter) GetTablesWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error) {
	query := `
		SELECT
			t.table_name,
			COALESCE(s.n_tup_ins + s.n_tup_upd + s.n_tup_del, 0) as row_count,
			pg_total_relation_size(quote_ident(t.table_schema)||'.'||quote_ident(t.table_name)) as table_size
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s ON s.schemaname = t.table_schema AND s.relname = t.table_name
		WHERE t.table_schema = $1
			AND t.table_type = 'BASE TABLE'
		ORDER BY t.table_name
	`

	rows, err := db.Query(query, schema)
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
		t.Schema = schema
		t.TableType = "BASE TABLE"
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *KingBaseAdapter) GetTableSchema(db *sql.DB, database, table string) (*model.TableSchema, error) {
	return a.GetTableSchemaWithSchema(db, database, "public", table)
}

// GetTableSchemaWithSchema 获取指定 schema 下表结构
func (a *KingBaseAdapter) GetTableSchemaWithSchema(db *sql.DB, database, schema, table string) (*model.TableSchema, error) {
	tableSchema := &model.TableSchema{
		Database: database,
		Table:    table,
	}

	// 获取列信息
	colsQuery := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			'' as column_key,
			'' as extra,
			'' as column_comment
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	colsRows, err := db.Query(colsQuery, schema, table)
	if err != nil {
		return nil, err
	}
	defer colsRows.Close()

	for colsRows.Next() {
		var col model.ColumnInfo
		var colType, nullable, def, key, extra, comment sql.NullString
		if err := colsRows.Scan(&col.Name, &colType, &nullable, &def, &key, &extra, &comment); err != nil {
			return nil, err
		}
		col.Type = colType.String
		col.Nullable = nullable.String == "YES"
		col.DefaultValue = def.String
		col.Key = key.String
		col.Extra = extra.String
		col.Comment = comment.String
		tableSchema.Columns = append(tableSchema.Columns, col)
	}

	// 获取索引信息
	idxQuery := `
		SELECT
			i.indexrelid::regclass as indexname,
			a.attname,
			i.indisunique,
			i.indisprimary
		FROM pg_index i
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(i.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.indexrelid::regclass, a.attnum
	`

	idxRows, err := db.Query(idxQuery, schema, table)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	indexMap := make(map[string]*model.IndexInfo)
	for idxRows.Next() {
		var indexName, column string
		var isUnique, isPrimary bool
		if err := idxRows.Scan(&indexName, &column, &isUnique, &isPrimary); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &model.IndexInfo{
				Name:    indexName,
				Unique:  isUnique,
				Primary: isPrimary,
			}
		}
		indexMap[indexName].Columns = append(indexMap[indexName].Columns, column)
	}

	for _, idx := range indexMap {
		tableSchema.Indexes = append(tableSchema.Indexes, *idx)
	}

	return tableSchema, nil
}

// GetViews 获取视图列表
func (a *KingBaseAdapter) GetViews(db *sql.DB, database string) ([]model.TableInfo, error) {
	return a.GetViewsWithSchema(db, database, "public")
}

// GetViewsWithSchema 获取指定 schema 下的视图列表
func (a *KingBaseAdapter) GetViewsWithSchema(db *sql.DB, database, schema string) ([]model.TableInfo, error) {
	query := `
		SELECT
			table_name,
			0 as row_count,
			0 as table_size
		FROM information_schema.views
		WHERE table_schema = $1
		ORDER BY table_name
	`

	rows, err := db.Query(query, schema)
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
		v.Schema = schema
		v.TableType = "VIEW"
		views = append(views, v)
	}

	return views, nil
}

// GetIndexes 获取索引列表
func (a *KingBaseAdapter) GetIndexes(db *sql.DB, database, table string) ([]model.IndexInfo, error) {
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}
	return schema.Indexes, nil
}

// Execute 执行非查询 SQL
func (a *KingBaseAdapter) Execute(db *sql.DB, query string, args ...interface{}) (*model.ExecuteResult, error) {
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
func (a *KingBaseAdapter) Query(db *sql.DB, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	start := time.Now()

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
		TimeCost: time.Since(start),
	}, nil
}

// Insert 插入数据
func (a *KingBaseAdapter) Insert(db *sql.DB, database, table string, data map[string]interface{}) error {
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	argNum := 1

	for col, val := range data {
		cols = append(cols, fmt.Sprintf(`"%s"`, col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
		argNum++
		values = append(values, val)
	}

	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
		table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	_, err := db.Exec(query, values...)
	return err
}

// Update 更新数据
func (a *KingBaseAdapter) Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error {
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	argNum := 1

	for col, val := range data {
		sets = append(sets, fmt.Sprintf(`"%s" = $%d`, col, argNum))
		argNum++
		values = append(values, val)
	}

	query := fmt.Sprintf(`UPDATE "%s" SET %s WHERE %s`,
		table,
		strings.Join(sets, ", "),
		where)

	_, err := db.Exec(query, values...)
	return err
}

// Delete 删除数据
func (a *KingBaseAdapter) Delete(db *sql.DB, database, table, where string) error {
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	query := fmt.Sprintf(`DELETE FROM "%s" WHERE %s`, table, where)
	_, err := db.Exec(query)
	return err
}

// ExportToCSV 导出为 CSV
func (a *KingBaseAdapter) ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	exporter := export.NewCSVExporter(opts)

	rows, err := db.Query(query)
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
func (a *KingBaseAdapter) ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	exporter := export.NewSQLExporter(opts, model.DatabaseKingBase)

	// 如果提供了自定义查询，则按查询导出
	if opts.Query != "" {
		rows, err := db.Query(opts.Query)
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
			query := fmt.Sprintf(`SELECT * FROM "%s"`, table)
			if opts.MaxRows > 0 {
				query = fmt.Sprintf("%s LIMIT %d", query, opts.MaxRows)
			}
			rows, err := db.Query(query)
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
func (a *KingBaseAdapter) GetCreateTableSQL(db *sql.DB, database, table string) (string, error) {
	var createSQL string
	query := `
		SELECT
			'CREATE TABLE ' || c.relname || ' (' ||
			string_agg(
				a.attname || ' ' || pg_catalog.format_type(a.atttypid, a.atttypmod) ||
				CASE WHEN a.attnotnull THEN ' NOT NULL' ELSE '' END ||
				CASE WHEN a.atthasdef THEN ' DEFAULT ' || (SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid) FROM pg_catalog.pg_attrdef d WHERE d.adrelid = a.attrelid AND d.adnum = a.attnum) ELSE '' END,
				', '
			) ||
			');' AS create_statement
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		WHERE c.relname = $1
			AND a.attnum > 0
			AND NOT a.attisdropped
		GROUP BY c.relname
	`

	row := db.QueryRow(query, table)
	if err := row.Scan(&createSQL); err != nil {
		return "", err
	}

	return createSQL, nil
}

// AlterTable 修改表结构
func (a *KingBaseAdapter) AlterTable(db *sql.DB, request *model.AlterTableRequest) error {
	if len(request.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	// KingBase 需要分别执行每个 ALTER 语句
	for _, action := range request.Actions {
		var sql string
		var err error

		switch action.Type {
		case model.AlterActionAddColumn:
			sql, err = a.buildAddColumnSQL(request.Database, request.Table, action.Column)
		case model.AlterActionDropColumn:
			sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" DROP COLUMN "%s"`,
				request.Database, request.Table, action.OldName)
		case model.AlterActionModifyColumn:
			// KingBase 需要多个语句来修改列
			if err := a.modifyColumn(db, request.Database, request.Table, action.Column); err != nil {
				return err
			}
			continue
		case model.AlterActionRenameColumn:
			sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME COLUMN "%s" TO "%s"`,
				request.Database, request.Table, action.OldName, action.NewName)
		case model.AlterActionAddIndex:
			sql, err = a.buildAddIndexSQL(request.Database, request.Table, action.Index)
		case model.AlterActionDropIndex:
			sql = fmt.Sprintf(`DROP INDEX "%s"."%s"`, request.Database, action.OldName)
		default:
			return fmt.Errorf("unsupported action type: %s", action.Type)
		}

		if err != nil {
			return fmt.Errorf("build SQL failed: %w", err)
		}

		if sql != "" {
			if _, err := db.Exec(sql); err != nil {
				return fmt.Errorf("execute SQL failed: %w", err)
			}
		}
	}

	return nil
}

// buildAddColumnSQL 构建添加列 SQL
func (a *KingBaseAdapter) buildAddColumnSQL(database, table string, col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	sql := fmt.Sprintf(`ALTER TABLE "%s"."%s" ADD COLUMN "%s" %s`,
		database, table, col.Name, a.buildColumnType(col))

	return sql, nil
}

// modifyColumn 修改列（KingBase 需要多个语句）
func (a *KingBaseAdapter) modifyColumn(db *sql.DB, database, table string, col *model.ColumnDef) error {
	if col == nil {
		return fmt.Errorf("column definition is required")
	}

	// 修改类型
	sql := fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER COLUMN "%s" TYPE %s`,
		database, table, col.Name, a.getBaseType(col))
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("alter column type failed: %w", err)
	}

	// 修改可空性
	if col.Nullable {
		sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER COLUMN "%s" DROP NOT NULL`,
			database, table, col.Name)
	} else {
		sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER COLUMN "%s" SET NOT NULL`,
			database, table, col.Name)
	}
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("alter column nullable failed: %w", err)
	}

	// 修改默认值
	if col.DefaultValue != "" {
		if strings.ToUpper(col.DefaultValue) == "NULL" {
			sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER COLUMN "%s" DROP DEFAULT`,
				database, table, col.Name)
		} else {
			sql = fmt.Sprintf(`ALTER TABLE "%s"."%s" ALTER COLUMN "%s" SET DEFAULT %s`,
				database, table, col.Name, a.formatDefaultValue(col.DefaultValue))
		}
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("alter column default failed: %w", err)
		}
	}

	return nil
}

// buildColumnType 构建列类型定义
func (a *KingBaseAdapter) buildColumnType(col *model.ColumnDef) string {
	var parts []string

	// 类型
	parts = append(parts, a.getBaseType(col))

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

// getBaseType 获取基础类型
func (a *KingBaseAdapter) getBaseType(col *model.ColumnDef) string {
	colType := strings.ToUpper(col.Type)

	// 处理自增
	if col.AutoIncrement {
		if strings.Contains(colType, "INT") {
			if strings.Contains(colType, "BIGINT") {
				return "BIGSERIAL"
			}
			return "SERIAL"
		}
	}

	if col.Length > 0 {
		return fmt.Sprintf("%s(%d)", colType, col.Length)
	} else if col.Precision > 0 {
		if col.Scale > 0 {
			return fmt.Sprintf("%s(%d,%d)", colType, col.Precision, col.Scale)
		}
		return fmt.Sprintf("%s(%d)", colType, col.Precision)
	}

	return colType
}

// formatDefaultValue 格式化默认值
func (a *KingBaseAdapter) formatDefaultValue(value string) string {
	upper := strings.ToUpper(value)
	if upper == "NULL" || upper == "CURRENT_TIMESTAMP" || upper == "NOW()" {
		return upper
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "''"))
}

// buildAddIndexSQL 构建添加索引 SQL
func (a *KingBaseAdapter) buildAddIndexSQL(database, table string, idx *model.IndexDef) (string, error) {
	if idx == nil {
		return "", fmt.Errorf("index definition is required")
	}

	if len(idx.Columns) == 0 {
		return "", fmt.Errorf("index columns are required")
	}

	columns := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		columns[i] = fmt.Sprintf(`"%s"`, col)
	}

	var sql string
	if idx.Unique {
		sql = fmt.Sprintf(`CREATE UNIQUE INDEX "%s" ON "%s"."%s" (%s)`,
			idx.Name, database, table, strings.Join(columns, ", "))
	} else {
		sql = fmt.Sprintf(`CREATE INDEX "%s" ON "%s"."%s" (%s)`,
			idx.Name, database, table, strings.Join(columns, ", "))
	}

	if idx.Type != "" {
		sql += fmt.Sprintf(" USING %s", strings.ToUpper(idx.Type))
	}

	return sql, nil
}

// RenameTable 重命名表
func (a *KingBaseAdapter) RenameTable(db *sql.DB, database, oldName, newName string) error {
	sql := fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME TO "%s"`,
		database, oldName, newName)
	_, err := db.Exec(sql)
	return err
}
