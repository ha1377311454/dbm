package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dbm/internal/adapter"
	"dbm/internal/model"

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

// QueryAndParse 执行查询并解析结果
// ctx: 上下文
// adp: 数据库适配器
// db: 数据库连接
// parseFunc: 每行的解析函数
// query: 查询语句
// config: 采集器配置
func QueryAndParse(ctx context.Context, adp adapter.DatabaseAdapter, db any, parseFunc func(map[string]string) error, query string, config ScraperConfig) error {
	opts := &model.QueryOptions{
		Database: config.Labels["database"], // 从标签或配置中获取？
	}
	// 如果标签里没有，尝试从 connection_config 获取？
	// 实际上 DefaultScraper 的 Scrape 方法会传入 ScraperConfig

	result, err := adp.Query(db, query, opts)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	for _, row := range result.Rows {
		// 将 map[string]interface{} 转换为 map[string]string
		rowMap := make(map[string]string)
		for k, v := range row {
			kLower := strings.ToLower(k)
			if v == nil {
				rowMap[kLower] = ""
				continue
			}

			switch val := v.(type) {
			case string:
				rowMap[kLower] = val
			case int32:
				rowMap[kLower] = strconv.Itoa(int(val))
			case int64:
				rowMap[kLower] = strconv.FormatInt(val, 10)
			case float64:
				rowMap[kLower] = strconv.FormatFloat(val, 'f', -1, 64)
			case time.Time:
				rowMap[kLower] = val.Format("2006-01-02 15:04:05")
			case []byte:
				rowMap[kLower] = string(val)
			default:
				rowMap[kLower] = fmt.Sprintf("%v", val)
			}
		}

		if err := parseFunc(rowMap); err != nil {
			return fmt.Errorf("parse row failed: %w", err)
		}
	}

	return nil
}
