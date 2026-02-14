package export

import (
	"dbm/internal/model"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"
)

// CSVExporter CSV 导出器
type CSVExporter struct {
	opts *model.CSVOptions
}

// NewCSVExporter 创建 CSV 导出器
func NewCSVExporter(opts *model.CSVOptions) *CSVExporter {
	// 设置默认选项
	if opts == nil {
		opts = &model.CSVOptions{}
	}
	if opts.Separator == "" {
		opts.Separator = ","
	}
	if opts.Quote == "" {
		opts.Quote = `"`
	}
	if opts.Encoding == "" {
		opts.Encoding = "UTF-8"
	}
	if opts.NullValue == "" {
		opts.NullValue = "NULL"
	}
	if opts.DateFormat == "" {
		opts.DateFormat = "2006-01-02 15:04:05"
	}

	return &CSVExporter{opts: opts}
}

// Export 导出数据到 CSV
func (e *CSVExporter) Export(writer io.Writer, columns []string, rows []map[string]interface{}) error {
	// 如果是 UTF-8 编码，写入 BOM 头以解决中文乱码问题
	if e.opts.Encoding == "UTF-8" {
		if _, err := writer.Write([]byte("\xEF\xBB\xBF")); err != nil {
			return fmt.Errorf("failed to write BOM: %w", err)
		}
	}

	w := csv.NewWriter(writer)
	defer w.Flush()

	// 写入表头
	if e.opts.IncludeHeader {
		if err := w.Write(columns); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	// 写入数据行
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = e.formatValue(row[col])
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

// formatValue 格式化值
func (e *CSVExporter) formatValue(v interface{}) string {
	if v == nil {
		return e.opts.NullValue
	}

	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return strconv.FormatFloat(val.(float64), 'f', -1, 64)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case time.Time:
		return val.Format(e.opts.DateFormat)
	default:
		return fmt.Sprintf("%v", val)
	}
}
