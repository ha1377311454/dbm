package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v3"
)

const (
	// Namespace 指标命名空间
	Namespace = "dbm"
)

// MetricsConfig 指标配置
type MetricsConfig struct {
	Metrics []Metric `yaml:"metrics"`
}

// Metric 单个指标配置
type Metric struct {
	Context     string            `yaml:"context"`     // 上下文（用作指标名前缀）
	Labels      []string          `yaml:"labels"`      // 从 SQL 结果中提取的标签列
	MetricsDesc map[string]string `yaml:"metricsdesc"` // 指标描述 {指标名: 描述}
	MetricsType map[string]string `yaml:"metricstype"` // 指标类型 {指标名: gauge|counter}
	Request     string            `yaml:"request"`     // SQL 查询语句
}

// metricsConfigs 缓存各数据库类型的指标配置
var metricsConfigs = make(map[string]*MetricsConfig)

// LoadMetricsConfig 从 YAML 内容加载指标配置
func LoadMetricsConfig(dbType string, yamlContent string) (*MetricsConfig, error) {
	var cfg MetricsConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		return nil, fmt.Errorf("parse metrics config failed: %w", err)
	}

	// 验证配置
	for i, m := range cfg.Metrics {
		if m.Context == "" {
			return nil, fmt.Errorf("metric[%d]: context is required", i)
		}
		if len(m.MetricsDesc) == 0 {
			return nil, fmt.Errorf("metric[%d]: metricsdesc is required", i)
		}
		if m.Request == "" {
			return nil, fmt.Errorf("metric[%d]: request is required", i)
		}
	}

	return &cfg, nil
}

// RegisterMetricsConfig 注册指标配置
func RegisterMetricsConfig(dbType string, cfg *MetricsConfig) {
	metricsConfigs[dbType] = cfg
}

// GetMetricsConfig 获取指标配置
func GetMetricsConfig(dbType string) (*MetricsConfig, bool) {
	cfg, ok := metricsConfigs[dbType]
	return cfg, ok
}

// DefaultScraper 默认采集器，基于 YAML 配置
type DefaultScraper struct {
	dbType string
	config *MetricsConfig
}

// NewDefaultScraper 创建默认采集器
func NewDefaultScraper(dbType string, config *MetricsConfig) *DefaultScraper {
	return &DefaultScraper{
		dbType: dbType,
		config: config,
	}
}

// Name 返回采集器名称
func (s *DefaultScraper) Name() string {
	return fmt.Sprintf("%s_default_scraper", s.dbType)
}

// Help 返回采集器帮助信息
func (s *DefaultScraper) Help() string {
	return "Collect metrics based on YAML configuration"
}

// Scrape 执行采集
func (s *DefaultScraper) Scrape(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric, config ScraperConfig) error {
	for _, mc := range s.config.Metrics {
		if err := s.scrapeMetric(ctx, db, ch, config, mc); err != nil {
			return fmt.Errorf("scrape metric[%s] failed: %w", mc.Context, err)
		}
	}
	return nil
}

// scrapeMetric 采集单个指标
func (s *DefaultScraper) scrapeMetric(ctx context.Context, db *sql.DB, ch chan<- prometheus.Metric, scraperConfig ScraperConfig, metric Metric) error {
	// 定义行解析函数
	parseRow := func(row map[string]string) error {
		// 提取标签值
		labelValues := make([]string, 0, len(metric.Labels))
		for _, label := range metric.Labels {
			labelValues = append(labelValues, row[strings.ToLower(label)])
		}

		// 遍历所有指标字段
		for metricName, metricHelp := range metric.MetricsDesc {
			// 从行数据中获取指标值
			valueStr, ok := row[strings.ToLower(metricName)]
			if !ok {
				continue
			}

			// 转换为 float64
			value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
			if err != nil {
				// 跳过无法解析的值
				continue
			}

			// 构建完整的指标名称: dbm_<context>_<metricName>
			fullMetricName := prometheus.BuildFQName(Namespace, metric.Context, metricName)

			// 获取指标类型
			valueType := GetMetricType(metricName, metric.MetricsType)

			// 注册指标
			desc := MetricDesc{
				Name:   fullMetricName,
				Help:   metricHelp,
				Labels: metric.Labels,
			}
			RegisterMetric(scraperConfig, ch, desc, value, valueType, labelValues...)
		}
		return nil
	}

	// 执行查询并解析
	if err := QueryAndParse(ctx, db, parseRow, metric.Request); err != nil {
		return err
	}

	return nil
}

// 确保 DefaultScraper 实现了 Scraper 接口
var _ Scraper = (*DefaultScraper)(nil)