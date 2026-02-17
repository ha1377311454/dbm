package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricDesc 指标描述
type MetricDesc struct {
	Name   string            // 指标名称
	Help   string            // 指标帮助信息
	Labels []string          // 标签名称
	Values map[string]string // 静态标签值（可选）
}

// RegisterMetric 注册 Prometheus 指标
// config: 采集器配置（包含标签）
// ch: Prometheus 指标 channel
// metricDesc: 指标描述
// value: 指标值
// valueType: 指标类型（Gauge/Counter）
// labelValues: 动态标签值（从 SQL 结果中提取）
func RegisterMetric(config ScraperConfig, ch chan<- prometheus.Metric, metricDesc MetricDesc, value float64, valueType prometheus.ValueType, labelValues ...string) {
	// 合并静态标签和动态标签
	allLabelKeys := make([]string, 0, len(config.Labels)+len(metricDesc.Labels))
	allLabelValues := make([]string, 0, len(config.Labels)+len(metricDesc.Labels))

	// 先添加静态标签
	for k, v := range config.Labels {
		allLabelKeys = append(allLabelKeys, k)
		allLabelValues = append(allLabelValues, v)
	}

	// 再添加动态标签
	allLabelKeys = append(allLabelKeys, metricDesc.Labels...)
	allLabelValues = append(allLabelValues, labelValues...)

	// 创建指标描述
	desc := prometheus.NewDesc(metricDesc.Name, metricDesc.Help, allLabelKeys, nil)

	// 创建指标并发送到 channel
	metric := prometheus.MustNewConstMetric(desc, valueType, value, allLabelValues...)
	if ch != nil {
		ch <- metric
	}
}

// GetMetricType 从配置中获取指标类型
func GetMetricType(metricName string, metricsType map[string]string) prometheus.ValueType {
	strToPromType := map[string]prometheus.ValueType{
		"gauge":   prometheus.GaugeValue,
		"counter": prometheus.CounterValue,
	}

	strType, ok := metricsType[strings.ToLower(metricName)]
	if !ok {
		return prometheus.GaugeValue
	}

	valueType, ok := strToPromType[strings.ToLower(strType)]
	if !ok {
		return prometheus.UntypedValue
	}
	return valueType
}

// QueryAndParse 执行 SQL 查询并解析结果
// ctx: 上下文
// db: 数据库连接
// parseFunc: 每行的解析函数
// sql: SQL 语句
func QueryAndParse(ctx context.Context, db *sql.DB, parseFunc func(map[string]string) error, sql string) error {
	rows, err := db.QueryContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("get columns failed: %w", err)
	}

	for rows.Next() {
		// 创建列值和列指针的切片
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// 扫描行数据
		if err := rows.Scan(columnPointers...); err != nil {
			return fmt.Errorf("scan row failed: %w", err)
		}

		// 将行数据转换为 map
		rowMap := make(map[string]string)
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			detected := *val

			// 转换为字符串
			colNameLower := strings.ToLower(colName)
			switch v := detected.(type) {
			case int32:
				rowMap[colNameLower] = strconv.Itoa(int(v))
			case int64:
				rowMap[colNameLower] = strconv.Itoa(int(v))
			case uint64:
				rowMap[colNameLower] = strconv.FormatUint(v, 10)
			case float64:
				rowMap[colNameLower] = strconv.FormatFloat(v, 'f', -1, 64)
			case string:
				rowMap[colNameLower] = v
			case time.Time:
				rowMap[colNameLower] = v.Format("2006-01-02 15:04:05")
			case []byte:
				rowMap[colNameLower] = string(v)
			case nil:
				rowMap[colNameLower] = ""
			default:
				rowMap[colNameLower] = fmt.Sprintf("%v", v)
			}
		}

		// 调用解析函数处理该行
		if err := parseFunc(rowMap); err != nil {
			return fmt.Errorf("parse row failed: %w", err)
		}
	}

	return rows.Err()
}