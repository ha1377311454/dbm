package monitor

import (
	"context"
	"dbm/internal/adapter"

	"github.com/prometheus/client_golang/prometheus"
)

// Scraper 监控指标采集器接口
type Scraper interface {
	// Name 返回采集器名称，应该是唯一的
	Name() string

	// Help 返回采集器的描述信息
	Help() string

	// Scrape 从数据库连接采集指标并发送到 channel
	// ctx: 上下文，用于超时控制
	// adp: 数据库适配器
	// db: 数据库连接
	// ch: Prometheus 指标 channel
	// config: 连接配置信息
	Scrape(ctx context.Context, adp adapter.DatabaseAdapter, db any, ch chan<- prometheus.Metric, config ScraperConfig) error
}

// ScraperConfig 采集器配置
type ScraperConfig struct {
	// 连接配置标签
	Labels map[string]string
	// 数据库类型
	DBType string
}
