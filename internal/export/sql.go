package export

import (
	"dbm/internal/model"
	"fmt"
	"io"
	"strings"
)

// SQLExporter SQL 导出器
type SQLExporter struct {
	opts   *model.SQLOptions
	dbType model.DatabaseType
}

// NewSQLExporter 创建 SQL 导出器
func NewSQLExporter(opts *model.SQLOptions, dbType model.DatabaseType) *SQLExporter {
	// 设置默认选项
	if opts == nil {
		opts = &model.SQLOptions{}
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}

	return &SQLExporter{
		opts:   opts,
		dbType: dbType,
	}
}

// ExportSchema 导出表结构
func (e *SQLExporter) ExportSchema(writer io.Writer, schema *model.TableSchema) error {
	// DROP TABLE
	if e.opts.IncludeDropTable {
		dropSQL := e.generateDropTable(schema.Database, schema.Table)
		if _, err := writer.Write([]byte(dropSQL)); err != nil {
			return err
		}
	}

	// CREATE TABLE
	if e.opts.IncludeCreateTable {
		createSQL := e.generateCreateTable(schema)
		if _, err := writer.Write([]byte(createSQL)); err != nil {
			return err
		}
	}

	return nil
}

// ExportData 导出表数据
func (e *SQLExporter) ExportData(writer io.Writer, database, table string, columns []string, rows []map[string]interface{}) error {
	if e.opts.StructureOnly {
		return nil
	}

	if e.opts.BatchInsert && e.opts.BatchSize > 1 {
		return e.exportBatchInsert(writer, database, table, columns, rows)
	}

	return e.exportSingleInsert(writer, database, table, columns, rows)
}

// exportSingleInsert 单行 INSERT
func (e *SQLExporter) exportSingleInsert(writer io.Writer, database, table string, columns []string, rows []map[string]interface{}) error {
	for _, row := range rows {
		insertSQL := e.generateInsert(database, table, columns, row)
		if _, err := writer.Write([]byte(insertSQL)); err != nil {
			return err
		}
	}
	return nil
}

// exportBatchInsert 批量 INSERT
func (e *SQLExporter) exportBatchInsert(writer io.Writer, database, table string, columns []string, rows []map[string]interface{}) error {
	for i := 0; i < len(rows); i += e.opts.BatchSize {
		end := i + e.opts.BatchSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := rows[i:end]
		batchSQL := e.generateBatchInsert(database, table, columns, batch)
		if _, err := writer.Write([]byte(batchSQL)); err != nil {
			return err
		}
	}

	return nil
}

// generateDropTable 生成 DROP TABLE 语句
func (e *SQLExporter) generateDropTable(database, table string) string {
	if database != "" {
		return fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`;\n\n", database, table)
	}
	return fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n\n", table)
}

// generateCreateTable 生成 CREATE TABLE 语句
func (e *SQLExporter) generateCreateTable(schema *model.TableSchema) string {
	var sb strings.Builder

	qualifiedName := schema.Table
	if schema.Database != "" {
		qualifiedName = fmt.Sprintf("`%s`.`%s`", schema.Database, schema.Table)
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", qualifiedName))

	// 列定义
	for i, col := range schema.Columns {
		colDef := fmt.Sprintf("  `%s` %s", col.Name, col.Type)

		if !col.Nullable {
			colDef += " NOT NULL"
		}

		if col.DefaultValue != "" {
			colDef += fmt.Sprintf(" DEFAULT %s", col.DefaultValue)
		}

		if col.Extra != "" {
			colDef += fmt.Sprintf(" %s", col.Extra)
		}

		if i < len(schema.Columns)-1 {
			colDef += ","
		}

		sb.WriteString(colDef + "\n")
	}

	// 主键
	for _, idx := range schema.Indexes {
		if idx.Primary && len(idx.Columns) > 0 {
			sb.WriteString(fmt.Sprintf("  PRIMARY KEY (`%s`)", strings.Join(idx.Columns, "`, `")))
			break
		}
	}

	sb.WriteString("\n)")

	// 表注释
	if schema.Columns != nil && len(schema.Columns) > 0 {
		// 获取表注释（从第一个列的注释或其他方式）
	}

	sb.WriteString(";\n\n")

	return sb.String()
}

// generateInsert 生成 INSERT 语句
func (e *SQLExporter) generateInsert(database, table string, columns []string, row map[string]interface{}) string {
	var sb strings.Builder

	qualifiedTable := table
	if database != "" {
		qualifiedTable = fmt.Sprintf("`%s`.`%s`", database, table)
	}

	sb.WriteString(fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES (", qualifiedTable, strings.Join(columns, "`, `")))

	values := make([]string, len(columns))
	for i, col := range columns {
		values[i] = e.formatValue(row[col])
	}

	sb.WriteString(strings.Join(values, ", "))
	sb.WriteString(");\n")

	return sb.String()
}

// generateBatchInsert 生成批量 INSERT 语句
func (e *SQLExporter) generateBatchInsert(database, table string, columns []string, rows []map[string]interface{}) string {
	var sb strings.Builder

	qualifiedTable := table
	if database != "" {
		qualifiedTable = fmt.Sprintf("`%s`.`%s`", database, table)
	}

	sb.WriteString(fmt.Sprintf("INSERT INTO `%s` (`%s`) VALUES\n", qualifiedTable, strings.Join(columns, "`, `")))

	for i, row := range rows {
		sb.WriteString("(")

		values := make([]string, len(columns))
		for j, col := range columns {
			values[j] = e.formatValue(row[col])
		}

		sb.WriteString(strings.Join(values, ", "))
		sb.WriteString(")")

		if i < len(rows)-1 {
			sb.WriteString(",\n")
		} else {
			sb.WriteString(";\n")
		}
	}

	return sb.String()
}

// formatValue 格式化值
func (e *SQLExporter) formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}

	switch val := v.(type) {
	case string:
		// 转义单引号
		escaped := strings.ReplaceAll(val, "'", "''")
		return "'" + escaped + "'"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%f", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return "NULL"
	}
}
