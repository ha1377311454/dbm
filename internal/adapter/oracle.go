package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v2"
)

// OracleAdapter Oracle 数据库适配器
type OracleAdapter struct {
	*BaseAdapter
}

// NewOracleAdapter 创建 Oracle 数据库适配器
func NewOracleAdapter() *OracleAdapter {
	return &OracleAdapter{
		BaseAdapter: NewBaseAdapter(),
	}
}

// Connect 连接 Oracle 数据库
func (a *OracleAdapter) Connect(config *model.ConnectionConfig) (any, error) {
	dsn := a.buildDSN(config)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// buildDSN 构建 Oracle 数据库 DSN
func (a *OracleAdapter) buildDSN(config *model.ConnectionConfig) string {
	// Oracle 中连接目标通常是 Service Name 或 SID，而非传统的 Database
	// 优先从 Params 中获取，如果未指定则使用 Database 字段
	service := config.Database
	if s, ok := config.Params["service_name"]; ok && s != "" {
		service = s
	} else if s, ok := config.Params["service"]; ok && s != "" {
		service = s
	} else if s, ok := config.Params["sid"]; ok && s != "" {
		service = s
	}

	// 使用 go-ora 构建 URL
	// 移除内部使用的参数，避免驱动报错 unknown URL option
	options := make(map[string]string)
	internalKeys := map[string]bool{"connectType": true, "service": true, "service_name": true, "sid": true}
	for k, v := range config.Params {
		if !internalKeys[k] {
			options[k] = v
		}
	}

	// BuildUrl(server, port, service, user, password, options)
	return go_ora.BuildUrl(config.Host, config.Port, service, config.Username, config.Password, options)
}

// Close 关闭数据库连接
func (a *OracleAdapter) Close(db any) error {
	return db.(*sql.DB).Close()
}

// Ping 测试数据库连接
func (a *OracleAdapter) Ping(db any) error {
	return db.(*sql.DB).Ping()
}

// GetDatabases 获取数据库列表（Oracle 中通常对应 User/Schema）
func (a *OracleAdapter) GetDatabases(db any) ([]string, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT username 
		FROM all_users 
		WHERE username NOT IN ('SYS', 'SYSTEM', 'SYSAUX', 'DBSNMP', 'OUTLN', 'APPQOSSYS')
		ORDER BY username
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
func (a *OracleAdapter) GetTables(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT 
			TABLE_NAME, 
			NUM_ROWS, 
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
		var numRows sql.NullInt64
		if err := rows.Scan(&t.Name, &numRows, &t.Size); err != nil {
			return nil, err
		}
		t.Rows = numRows.Int64
		t.Database = database
		t.Schema = database
		t.TableType = "BASE TABLE"
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *OracleAdapter) GetTableSchema(db any, database, table string) (*model.TableSchema, error) {
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
			DATA_LENGTH,
			DATA_PRECISION,
			DATA_SCALE
		FROM ALL_TAB_COLUMNS 
		WHERE OWNER = :1 AND TABLE_NAME = :2
		ORDER BY COLUMN_ID
	`

	rows, err := dbSQL.Query(colsQuery, strings.ToUpper(database), strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col model.ColumnInfo
		var colType, nullable, def sql.NullString
		var length, precision, scale sql.NullInt64

		if err := rows.Scan(&col.Name, &colType, &nullable, &def, &length, &precision, &scale); err != nil {
			return nil, err
		}

		col.Type = a.buildTypeString(colType.String, length, precision, scale)
		col.Nullable = nullable.String == "Y"
		col.DefaultValue = strings.TrimSpace(def.String)
		tableSchema.Columns = append(tableSchema.Columns, col)
	}

	// 获取索引信息
	idxQuery := `
		SELECT 
			idx.INDEX_NAME, 
			cols.COLUMN_NAME, 
			idx.UNIQUENESS
		FROM ALL_INDEXES idx
		JOIN ALL_IND_COLUMNS cols ON idx.INDEX_NAME = cols.INDEX_NAME AND idx.OWNER = cols.INDEX_OWNER
		WHERE idx.TABLE_OWNER = :1 AND idx.TABLE_NAME = :2
		ORDER BY idx.INDEX_NAME, cols.COLUMN_POSITION
	`

	idxRows, err := dbSQL.Query(idxQuery, strings.ToUpper(database), strings.ToUpper(table))
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	indexMap := make(map[string]*model.IndexInfo)
	for idxRows.Next() {
		var indexName, column, uniqueness string
		if err := idxRows.Scan(&indexName, &column, &uniqueness); err != nil {
			return nil, err
		}

		if _, exists := indexMap[indexName]; !exists {
			indexMap[indexName] = &model.IndexInfo{
				Name:   indexName,
				Unique: uniqueness == "UNIQUE",
			}
		}
		indexMap[indexName].Columns = append(indexMap[indexName].Columns, column)
	}

	for _, idx := range indexMap {
		tableSchema.Indexes = append(tableSchema.Indexes, *idx)
	}

	return tableSchema, nil
}

func (a *OracleAdapter) buildTypeString(dataType string, length, precision, scale sql.NullInt64) string {
	dt := strings.ToUpper(dataType)
	switch dt {
	case "VARCHAR2", "CHAR", "RAW":
		if length.Valid {
			return fmt.Sprintf("%s(%d)", dt, length.Int64)
		}
	case "NUMBER":
		if precision.Valid && precision.Int64 > 0 {
			if scale.Valid && scale.Int64 > 0 {
				return fmt.Sprintf("NUMBER(%d,%d)", precision.Int64, scale.Int64)
			}
			return fmt.Sprintf("NUMBER(%d)", precision.Int64)
		}
	}
	return dt
}

// GetViews 获取视图列表
func (a *OracleAdapter) GetViews(db any, database string) ([]model.TableInfo, error) {
	dbSQL := db.(*sql.DB)
	query := `
		SELECT VIEW_NAME 
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
		if err := rows.Scan(&v.Name); err != nil {
			return nil, err
		}
		v.Database = database
		v.Schema = database
		v.TableType = "VIEW"
		views = append(views, v)
	}

	return views, nil
}

// GetIndexes 获取索引列表
func (a *OracleAdapter) GetIndexes(db any, database, table string) ([]model.IndexInfo, error) {
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}
	return schema.Indexes, nil
}

// Execute 执行非查询 SQL
func (a *OracleAdapter) Execute(db any, query string, args ...interface{}) (*model.ExecuteResult, error) {
	dbSQL := db.(*sql.DB)
	start := time.Now()

	query = a.rewriteQuery(query)
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
func (a *OracleAdapter) Query(db any, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	dbSQL := db.(*sql.DB)
	start := time.Now()

	query = a.rewriteQuery(query)
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
func (a *OracleAdapter) Insert(db any, database, table string, data map[string]interface{}) error {
	dbSQL := db.(*sql.DB)
	cols := make([]string, 0, len(data))
	placeholders := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	argNum := 1

	for col, val := range data {
		cols = append(cols, fmt.Sprintf(`"%s"`, strings.ToUpper(col)))
		placeholders = append(placeholders, fmt.Sprintf(":%d", argNum))
		argNum++
		values = append(values, val)
	}

	query := fmt.Sprintf(`INSERT INTO "%s"."%s" (%s) VALUES (%s)`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	_, err := dbSQL.Exec(query, values...)
	return err
}

// Update 更新数据
func (a *OracleAdapter) Update(db any, database, table string, data map[string]interface{}, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	argNum := 1

	for col, val := range data {
		sets = append(sets, fmt.Sprintf(`"%s" = :%d`, strings.ToUpper(col), argNum))
		argNum++
		values = append(values, val)
	}

	query := fmt.Sprintf(`UPDATE "%s"."%s" SET %s WHERE %s`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		strings.Join(sets, ", "),
		where)

	_, err := dbSQL.Exec(query, values...)
	return err
}

// Delete 删除数据
func (a *OracleAdapter) Delete(db any, database, table, where string) error {
	dbSQL := db.(*sql.DB)
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	query := fmt.Sprintf(`DELETE FROM "%s"."%s" WHERE %s`,
		strings.ToUpper(database),
		strings.ToUpper(table),
		where)

	_, err := dbSQL.Exec(query)
	return err
}

// ExportToCSV 导出为 CSV
func (a *OracleAdapter) ExportToCSV(db any, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewCSVExporter(opts)

	query = a.rewriteQuery(query)
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
func (a *OracleAdapter) ExportToSQL(db any, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	dbSQL := db.(*sql.DB)
	exporter := export.NewSQLExporter(opts, model.DatabaseOracle)

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
			query := fmt.Sprintf(`SELECT * FROM "%s"."%s"`, strings.ToUpper(database), strings.ToUpper(table))
			rows, err := dbSQL.Query(query)
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

			if err := exporter.ExportData(writer, database, table, colNames, rowData); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCreateTableSQL 获取建表语句
func (a *OracleAdapter) GetCreateTableSQL(db any, database, table string) (string, error) {
	dbSQL := db.(*sql.DB)
	var createSQL string
	query := `SELECT DBMS_METADATA.GET_DDL('TABLE', :1, :2) FROM DUAL`
	row := dbSQL.QueryRow(query, strings.ToUpper(table), strings.ToUpper(database))
	if err := row.Scan(&createSQL); err != nil {
		return "", err
	}
	return createSQL, nil
}

// AlterTable 修改表结构
func (a *OracleAdapter) AlterTable(db any, request *model.AlterTableRequest) error {
	// 简单实现，可根据需求扩展
	return fmt.Errorf("Oracle AlterTable 尚未实现")
}

// RenameTable 重命名表
func (a *OracleAdapter) RenameTable(db any, database, oldName, newName string) error {
	dbSQL := db.(*sql.DB)
	query := fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME TO "%s"`,
		strings.ToUpper(database), strings.ToUpper(oldName), strings.ToUpper(newName))
	_, err := dbSQL.Exec(query)
	return err
}

var limitRegex = regexp.MustCompile(`(?i)\bLIMIT\s+(\d+)\b`)

func (a *OracleAdapter) rewriteQuery(query string) string {
	original := query
	query = strings.TrimSpace(query)
	// 移除结尾的分号及空白字符，Oracle 驱动通常不需要且会报错
	query = strings.TrimRight(query, "; \t\n\r")

	// 处理 LIMIT n 语法，转换为 Oracle 兼容的 ROWNUM 包装语法
	// Oracle 12c 以前不支持 FETCH NEXT，使用 ROWNUM 更加通用
	if strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		matches := limitRegex.FindStringSubmatch(query)
		if len(matches) > 1 {
			limit := matches[1]
			// 移除 LIMIT 子句
			innerQuery := limitRegex.ReplaceAllString(query, "")
			// 使用 ROWNUM 包装以支持分页且不破坏 ORDER BY
			query = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %s", innerQuery, limit)
		}
	}

	if query != original {
		log.Printf("[Oracle] SQL Rewritten:\n  Original: %s\n  Final:    %s", original, query)
	}

	return query
}
