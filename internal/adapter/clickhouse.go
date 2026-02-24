package adapter

import (
	"database/sql"
	"dbm/internal/export"
	"dbm/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// ClickHouseAdapter ClickHouse 数据库适配器
type ClickHouseAdapter struct {
	*BaseAdapter
}

// NewClickHouseAdapter 创建 ClickHouse 适配器
func NewClickHouseAdapter() *ClickHouseAdapter {
	return &ClickHouseAdapter{
		BaseAdapter: NewBaseAdapter(),
	}
}

// Connect 连接 ClickHouse 数据库
func (a *ClickHouseAdapter) Connect(config *model.ConnectionConfig) (*sql.DB, error) {
	dsn := a.buildDSN(config)
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// buildDSN 构建 ClickHouse DSN
// Native: clickhouse://username:password@host:port/database?param1=val1
// HTTP: http://username:password@host:port/database?param1=val1
func (a *ClickHouseAdapter) buildDSN(config *model.ConnectionConfig) string {
	protocol := "clickhouse" // 默认是 native 协议

	// 如果没有显式指定协议且端口是常用的 HTTP/HTTPS 端口，则自动切换
	if config.Params == nil {
		if config.Port == 8123 {
			protocol = "http"
		} else if config.Port == 8443 {
			protocol = "https"
		}
	} else {
		if p, ok := config.Params["protocol"]; ok && (p == "http" || p == "https" || p == "clickhouse") {
			protocol = p
		} else if config.Port == 8123 {
			protocol = "http"
		} else if config.Port == 8443 {
			protocol = "https"
		}
	}

	var auth string
	if config.Username != "" {
		auth = config.Username
		if config.Password != "" {
			auth += ":" + config.Password
		}
		auth += "@"
	}

	dsn := fmt.Sprintf("%s://%s%s:%d", protocol, auth, config.Host, config.Port)
	if config.Database != "" {
		dsn += "/" + config.Database
	}

	// 过滤掉内部使用的参数
	var queryParams []string
	if config.Params != nil {
		for k, v := range config.Params {
			if k == "protocol" {
				continue
			}
			queryParams = append(queryParams, fmt.Sprintf("%s=%s", k, v))
		}
	}

	if len(queryParams) > 0 {
		dsn += "?" + strings.Join(queryParams, "&")
	}

	return dsn
}

// Close 关闭连接
func (a *ClickHouseAdapter) Close(db *sql.DB) error {
	return db.Close()
}

// Ping 测试连接
func (a *ClickHouseAdapter) Ping(db *sql.DB) error {
	return db.Ping()
}

// GetDatabases 获取数据库列表
func (a *ClickHouseAdapter) GetDatabases(db *sql.DB) ([]string, error) {
	query := `
		SELECT name
		FROM system.databases
		WHERE name NOT IN ('system', 'INFORMATION_SCHEMA', 'information_schema')
		ORDER BY name
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

// GetTables 获取表列表
func (a *ClickHouseAdapter) GetTables(db *sql.DB, database string) ([]model.TableInfo, error) {
	query := `
		SELECT
			name,
			engine,
			total_rows,
			total_bytes,
			comment
		FROM system.tables
		WHERE database = ?
		ORDER BY name
	`

	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []model.TableInfo
	for rows.Next() {
		var t model.TableInfo
		var tableType, comment sql.NullString
		var totalRows, totalBytes sql.NullInt64
		if err := rows.Scan(&t.Name, &tableType, &totalRows, &totalBytes, &comment); err != nil {
			return nil, err
		}
		t.Database = database
		t.TableType = tableType.String
		t.Comment = comment.String
		if totalRows.Valid {
			t.Rows = totalRows.Int64
		}
		if totalBytes.Valid {
			t.Size = totalBytes.Int64
		}
		tables = append(tables, t)
	}

	return tables, nil
}

// GetTableSchema 获取表结构
func (a *ClickHouseAdapter) GetTableSchema(db *sql.DB, database, table string) (*model.TableSchema, error) {
	schema := &model.TableSchema{
		Database: database,
		Table:    table,
	}

	// 获取列信息
	colsQuery := `
		SELECT
			name,
			type,
			is_in_primary_key,
			default_expression,
			comment
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position
	`

	colsRows, err := db.Query(colsQuery, database, table)
	if err != nil {
		return nil, err
	}
	defer colsRows.Close()

	for colsRows.Next() {
		var col model.ColumnInfo
		var isPK uint8
		var typeStr, def, comment sql.NullString
		if err := colsRows.Scan(&col.Name, &typeStr, &isPK, &def, &comment); err != nil {
			return nil, err
		}
		col.Type = typeStr.String
		col.Nullable = strings.HasPrefix(col.Type, "Nullable")
		col.DefaultValue = def.String
		if isPK == 1 {
			col.Key = "PRI"
		}
		col.Comment = comment.String
		schema.Columns = append(schema.Columns, col)
	}

	return schema, nil
}

// GetViews 获取视图列表
func (a *ClickHouseAdapter) GetViews(db *sql.DB, database string) ([]model.TableInfo, error) {
	query := `
		SELECT
			name,
			engine,
			total_rows,
			total_bytes
		FROM system.tables
		WHERE database = ? AND engine LIKE '%View'
		ORDER BY name
	`

	rows, err := db.Query(query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var views []model.TableInfo
	for rows.Next() {
		var v model.TableInfo
		var engine string
		if err := rows.Scan(&v.Name, &engine, &v.Rows, &v.Size); err != nil {
			return nil, err
		}
		v.Database = database
		v.TableType = "VIEW"
		views = append(views, v)
	}

	return views, nil
}

// GetViewDefinition 获取视图定义
func (a *ClickHouseAdapter) GetViewDefinition(db *sql.DB, database, viewName string) (string, error) {
	var definition string
	query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", database, viewName)
	row := db.QueryRow(query)

	if err := row.Scan(&definition); err != nil {
		return "", err
	}

	return definition, nil
}

// GetProcedures 获取存储过程列表（ClickHouse 不支持传统的存储过程）
func (a *ClickHouseAdapter) GetProcedures(db *sql.DB, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetFunctions 获取函数列表
// ClickHouse 有 UDF 但系统表里通常不在这里展示为 schema 层面的 routine，暂不实现
func (a *ClickHouseAdapter) GetFunctions(db *sql.DB, database string) ([]model.RoutineInfo, error) {
	return []model.RoutineInfo{}, nil
}

// GetRoutineDefinition 获取存储过程或函数定义（ClickHouse 不支持）
func (a *ClickHouseAdapter) GetRoutineDefinition(db *sql.DB, database, routineName, routineType string) (string, error) {
	return "", fmt.Errorf("ClickHouse doesn't support viewing routine definition natively")
}

// GetIndexes 获取索引列表
func (a *ClickHouseAdapter) GetIndexes(db *sql.DB, database, table string) ([]model.IndexInfo, error) {
	// ClickHouse 索引概念不同，通常指 Skipping Indexes 或 Primary Key
	// 这里简单返回主键作为索引
	schema, err := a.GetTableSchema(db, database, table)
	if err != nil {
		return nil, err
	}

	var pkColumns []string
	for _, col := range schema.Columns {
		if col.Key == "PRI" {
			pkColumns = append(pkColumns, col.Name)
		}
	}

	if len(pkColumns) > 0 {
		return []model.IndexInfo{
			{
				Name:    "PRIMARY",
				Columns: pkColumns,
				Unique:  false, // ClickHouse PK 不一定是唯一的
				Primary: true,
			},
		}, nil
	}

	return nil, nil
}

// Execute 执行非查询 SQL
func (a *ClickHouseAdapter) Execute(db *sql.DB, query string, args ...interface{}) (*model.ExecuteResult, error) {
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
func (a *ClickHouseAdapter) Query(db *sql.DB, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
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
				// 处理 Map、Array、Tuple 等复杂类型
				// 使用反射检测所有 map 和 slice 类型
				rv := reflect.ValueOf(val)
				kind := rv.Kind()

				if kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array {
					// 将复杂类型序列化为 JSON 字符串以便前端显示
					if jsonBytes, err := json.Marshal(val); err == nil {
						row[col] = string(jsonBytes)
					} else {
						row[col] = fmt.Sprintf("%v", val)
					}
				} else {
					row[col] = val
				}
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

// Insert 插入数据（ClickHouse 通常不建议单条插入，但在管控工具中支持）
func (a *ClickHouseAdapter) Insert(db *sql.DB, database, table string, data map[string]interface{}) error {
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

	_, err := db.Exec(query, values...)
	return err
}

// Update 更新数据 (ClickHouse 使用 ALTER TABLE ... UPDATE)
func (a *ClickHouseAdapter) Update(db *sql.DB, database, table string, data map[string]interface{}, where string) error {
	if where == "" {
		return fmt.Errorf("更新操作必须指定 WHERE 条件")
	}

	sets := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for col, val := range data {
		sets = append(sets, fmt.Sprintf("`%s` = ?", col))
		values = append(values, val)
	}

	query := fmt.Sprintf("ALTER TABLE `%s`.`%s` UPDATE %s WHERE %s",
		database, table,
		strings.Join(sets, ", "),
		where)

	_, err := db.Exec(query, values...)
	return err
}

// Delete 删除数据 (ClickHouse 使用 ALTER TABLE ... DELETE)
func (a *ClickHouseAdapter) Delete(db *sql.DB, database, table, where string) error {
	if where == "" {
		return fmt.Errorf("删除操作必须指定 WHERE 条件")
	}

	query := fmt.Sprintf("ALTER TABLE `%s`.`%s` DELETE WHERE %s", database, table, where)
	_, err := db.Exec(query)
	return err
}

// ExportToCSV 导出为 CSV
func (a *ClickHouseAdapter) ExportToCSV(db *sql.DB, writer io.Writer, database, query string, opts *model.CSVOptions) error {
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
func (a *ClickHouseAdapter) ExportToSQL(db *sql.DB, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	exporter := export.NewSQLExporter(opts, model.DatabaseClickHouse)

	for _, table := range tables {
		if opts.IncludeCreateTable || opts.StructureOnly {
			createSQL, err := a.GetCreateTableSQL(db, database, table)
			if err != nil {
				return err
			}
			if _, err := writer.Write([]byte(createSQL + ";\n\n")); err != nil {
				return err
			}
		}

		if !opts.StructureOnly {
			query := fmt.Sprintf("SELECT * FROM `%s`.`%s` ", database, table)
			if opts.MaxRows > 0 {
				query += fmt.Sprintf(" LIMIT %d", opts.MaxRows)
			}

			// ClickHouse 数据导出逻辑同 MySQL
			res, err := a.Query(db, query, nil)
			if err != nil {
				return err
			}

			if err := exporter.ExportData(writer, database, table, res.Columns, res.Rows); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetCreateTableSQL 获取建表语句
func (a *ClickHouseAdapter) GetCreateTableSQL(db *sql.DB, database, table string) (string, error) {
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s` ", database, table)
	var createSQL string
	err := db.QueryRow(query).Scan(&createSQL)
	return createSQL, err
}

// AlterTable 修改表结构
// 注意：对于 ReplicatedMergeTree 表，ALTER 操作会自动通过 ZooKeeper 同步到所有副本
func (a *ClickHouseAdapter) AlterTable(db *sql.DB, request *model.AlterTableRequest) error {
	if len(request.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	// 检查表引擎类型
	engineType, err := a.getTableEngine(db, request.Database, request.Table)
	if err != nil {
		return fmt.Errorf("get table engine failed: %w", err)
	}

	// 对于复制表，给出警告信息（通过日志或返回信息）
	isReplicated := strings.Contains(engineType, "Replicated")

	// ClickHouse 支持在一个 ALTER TABLE 语句中执行多个操作
	var alterClauses []string
	for _, action := range request.Actions {
		clause, err := a.buildAlterClause(action)
		if err != nil {
			return fmt.Errorf("build alter clause failed: %w", err)
		}
		if clause != "" {
			alterClauses = append(alterClauses, clause)
		}
	}

	if len(alterClauses) == 0 {
		return nil
	}

	// 执行 ALTER TABLE
	// 对于 ReplicatedMergeTree，这个操作会：
	// 1. 在 ZooKeeper 中创建一个任务
	// 2. 所有副本会自动执行这个 ALTER 操作
	// 3. 操作是异步的，可能需要一些时间完成
	sql := fmt.Sprintf("ALTER TABLE `%s`.`%s` %s",
		request.Database,
		request.Table,
		strings.Join(alterClauses, ", "))

	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	// 如果是复制表，返回提示信息
	if isReplicated {
		// 注意：这里可以考虑添加一个等待机制，确保所有副本都完成了 ALTER
		// 可以查询 system.mutations 表来检查进度
		return fmt.Errorf("ALTER executed on replicated table, changes will be synchronized via ZooKeeper. Check system.mutations for progress")
	}

	return nil
}

// getTableEngine 获取表引擎类型
func (a *ClickHouseAdapter) getTableEngine(db *sql.DB, database, table string) (string, error) {
	query := `SELECT engine FROM system.tables WHERE database = ? AND name = ?`
	var engine string
	err := db.QueryRow(query, database, table).Scan(&engine)
	return engine, err
}

// buildAlterClause 构建单个 ALTER 子句
func (a *ClickHouseAdapter) buildAlterClause(action model.AlterTableAction) (string, error) {
	switch action.Type {
	case model.AlterActionAddColumn:
		return a.buildAddColumnClause(action.Column)
	case model.AlterActionDropColumn:
		return fmt.Sprintf("DROP COLUMN `%s`", action.OldName), nil
	case model.AlterActionModifyColumn:
		return a.buildModifyColumnClause(action.Column)
	case model.AlterActionRenameColumn:
		return fmt.Sprintf("RENAME COLUMN `%s` TO `%s`", action.OldName, action.NewName), nil
	case model.AlterActionAddIndex:
		// ClickHouse 不使用传统索引，使用 ORDER BY 和 PRIMARY KEY
		return "", fmt.Errorf("ClickHouse does not support traditional indexes, use ORDER BY or PRIMARY KEY")
	case model.AlterActionDropIndex:
		return "", fmt.Errorf("ClickHouse does not support traditional indexes")
	default:
		return "", fmt.Errorf("unsupported action type: %s", action.Type)
	}
}

// buildAddColumnClause 构建添加列子句
func (a *ClickHouseAdapter) buildAddColumnClause(col *model.ColumnDef) (string, error) {
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
func (a *ClickHouseAdapter) buildModifyColumnClause(col *model.ColumnDef) (string, error) {
	if col == nil {
		return "", fmt.Errorf("column definition is required")
	}

	return fmt.Sprintf("MODIFY COLUMN `%s` %s", col.Name, a.buildColumnType(col)), nil
}

// buildColumnType 构建列类型定义
func (a *ClickHouseAdapter) buildColumnType(col *model.ColumnDef) string {
	var parts []string

	// 类型
	colType := col.Type
	if col.Length > 0 {
		colType = fmt.Sprintf("%s(%d)", colType, col.Length)
	} else if col.Precision > 0 {
		if col.Scale > 0 {
			colType = fmt.Sprintf("%s(%d,%d)", colType, col.Precision, col.Scale)
		} else {
			colType = fmt.Sprintf("%s(%d)", colType, col.Precision)
		}
	}

	// ClickHouse 使用 Nullable() 包装类型来表示可空
	if col.Nullable {
		colType = fmt.Sprintf("Nullable(%s)", colType)
	}

	parts = append(parts, colType)

	// 默认值
	if col.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", a.formatDefaultValue(col.DefaultValue)))
	}

	// 注释
	if col.Comment != "" {
		parts = append(parts, fmt.Sprintf("COMMENT '%s'", strings.ReplaceAll(col.Comment, "'", "\\'")))
	}

	return strings.Join(parts, " ")
}

// formatDefaultValue 格式化默认值
func (a *ClickHouseAdapter) formatDefaultValue(value string) string {
	upper := strings.ToUpper(value)
	if upper == "NULL" || upper == "NOW()" {
		return upper
	}
	// ClickHouse 的数字和字符串默认值
	if _, err := fmt.Sscanf(value, "%f", new(float64)); err == nil {
		return value
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "\\'"))
}

// RenameTable 重命名表
// 注意：对于 ReplicatedMergeTree 表，RENAME 操作不会通过 ZooKeeper 同步
// 需要在所有副本上分别执行
func (a *ClickHouseAdapter) RenameTable(db *sql.DB, database, oldName, newName string) error {
	// 检查是否为复制表
	engineType, err := a.getTableEngine(db, database, oldName)
	if err != nil {
		return fmt.Errorf("get table engine failed: %w", err)
	}

	if strings.Contains(engineType, "Replicated") {
		return fmt.Errorf("RENAME TABLE is not supported for replicated tables via ZooKeeper. You must rename on each replica separately")
	}

	sql := fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`",
		database, oldName, database, newName)
	_, err = db.Exec(sql)
	return err
}

// CheckMutationStatus 检查 ALTER 操作的执行状态（用于复制表）
func (a *ClickHouseAdapter) CheckMutationStatus(db *sql.DB, database, table string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			mutation_id,
			command,
			create_time,
			is_done,
			latest_fail_reason
		FROM system.mutations
		WHERE database = ? AND table = ?
		ORDER BY create_time DESC
		LIMIT 10
	`

	rows, err := db.Query(query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var mutationID, command, latestFailReason string
		var createTime time.Time
		var isDone uint8

		if err := rows.Scan(&mutationID, &command, &createTime, &isDone, &latestFailReason); err != nil {
			return nil, err
		}

		results = append(results, map[string]interface{}{
			"mutation_id":        mutationID,
			"command":            command,
			"create_time":        createTime,
			"is_done":            isDone == 1,
			"latest_fail_reason": latestFailReason,
		})
	}

	return results, nil
}
