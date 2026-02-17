package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"dbm/internal/adapter"
	"dbm/internal/connection"
	"dbm/internal/model"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector Prometheus 指标采集器
type Collector struct {
	connMgr  *connection.Manager
	factory  adapter.AdapterFactory
	scrapers map[model.DatabaseType][]Scraper
	timeout  time.Duration

	// 指标描述
	scrapeDurationDesc *prometheus.Desc
	scrapeFailuresDesc *prometheus.Desc
	upDesc             *prometheus.Desc
}

// NewCollector 创建采集器
func NewCollector(connMgr *connection.Manager, factory adapter.AdapterFactory) *Collector {
	// 创建默认的采集器（稍后会在 InitMetrics 中加载配置后注册）
	scrapers := make(map[model.DatabaseType][]Scraper)

	return &Collector{
		connMgr:  connMgr,
		factory:  factory,
		scrapers: scrapers,
		timeout:  15 * time.Second, // 默认 15 秒超时
		scrapeDurationDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "collector_duration_seconds"),
			"Collector time duration.",
			nil, nil,
		),
		scrapeFailuresDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "scrape_failures_total"),
			"Number of errors while scraping database.",
			[]string{"connection_id", "connection_name", "db_type", "host", "port", "error"}, nil,
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "", "up"),
			"Whether the database connection is up.",
			[]string{"connection_id", "connection_name", "db_type", "host", "port"}, nil,
		),
	}
}

// SetTimeout 设置查询超时
func (c *Collector) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// RegisterScraper 注册采集器
func (c *Collector) RegisterScraper(dbType model.DatabaseType, scraper Scraper) {
	c.scrapers[dbType] = append(c.scrapers[dbType], scraper)
}

// Describe 实现 prometheus.Collector 接口
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.scrapeDurationDesc
	ch <- c.scrapeFailuresDesc
	ch <- c.upDesc
}

// Collect 实现 prometheus.Collector 接口
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 获取所有连接配置
	configs, err := c.connMgr.ListConfigs()
	if err != nil {
		// 记录获取配置列表失败
		ch <- prometheus.MustNewConstMetric(
			c.scrapeFailuresDesc,
			prometheus.CounterValue,
			1,
			"", "", "", "", "", fmt.Sprintf("Failed to list configs: %v", err),
		)
		return
	}

	// 并发采集所有启用了监控的连接
	var wg sync.WaitGroup
	for _, cfg := range configs {
		// 跳过未启用监控的连接
		if !cfg.MonitoringEnabled {
			continue
		}

		wg.Add(1)
		go func(cfg *model.ConnectionConfig) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					ch <- prometheus.MustNewConstMetric(
						c.scrapeFailuresDesc,
						prometheus.CounterValue,
						1,
						cfg.ID, cfg.Name, string(cfg.Type), cfg.Host, fmt.Sprint(cfg.Port),
						fmt.Sprintf("Panic: %v", r),
					)
				}
			}()

			// 使用 GetConfig 获取解密后的配置
			config, err := c.connMgr.GetConfig(cfg.ID)
			if err != nil {
				ch <- prometheus.MustNewConstMetric(
					c.scrapeFailuresDesc,
					prometheus.CounterValue,
					1,
					cfg.ID, cfg.Name, string(cfg.Type), cfg.Host, fmt.Sprint(cfg.Port),
					fmt.Sprintf("Failed to get config: %v", err),
				)
				return
			}
			c.scrapeConnection(ctx, ch, config)
		}(cfg)
	}
	wg.Wait()

	// 发送采集耗时
	ch <- prometheus.MustNewConstMetric(
		c.scrapeDurationDesc,
		prometheus.GaugeValue,
		time.Since(startTime).Seconds(),
	)
}

// scrapeConnection 采集单个连接的指标
func (c *Collector) scrapeConnection(ctx context.Context, ch chan<- prometheus.Metric, config *model.ConnectionConfig) {
	// 创建新的数据库连接（不使用连接池）
	db, err := c.createConnection(config)
	if err != nil {
		// 连接失败，记录 up=0
		ch <- prometheus.MustNewConstMetric(
			c.upDesc,
			prometheus.GaugeValue,
			0,
			config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
		)
		// 记录失败
		ch <- prometheus.MustNewConstMetric(
			c.scrapeFailuresDesc,
			prometheus.CounterValue,
			1,
			config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
			fmt.Sprintf("Connection failed: %v", err),
		)
		return
	}
	defer db.Close()

	// 测试连接
	if err := db.PingContext(ctx); err != nil {
		// Ping 失败，记录 up=0
		ch <- prometheus.MustNewConstMetric(
			c.upDesc,
			prometheus.GaugeValue,
			0,
			config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
		)
		// 记录失败
		ch <- prometheus.MustNewConstMetric(
			c.scrapeFailuresDesc,
			prometheus.CounterValue,
			1,
			config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
			fmt.Sprintf("Ping failed: %v", err),
		)
		return
	}

	// 连接成功，记录 up=1
	ch <- prometheus.MustNewConstMetric(
		c.upDesc,
		prometheus.GaugeValue,
		1,
		config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
	)

	// 获取该数据库类型的采集器
	scrapers, ok := c.scrapers[config.Type]
	if !ok || len(scrapers) == 0 {
		// 没有注册采集器，跳过
		return
	}

	// 构建采集器配置
	scraperConfig := ScraperConfig{
		Labels: map[string]string{
			"connection_id":   config.ID,
			"connection_name": config.Name,
			"db_type":         string(config.Type),
			"host":            config.Host,
			"port":            fmt.Sprint(config.Port),
		},
		DBType: string(config.Type),
	}

	// 执行所有采集器
	for _, scraper := range scrapers {
		if err := scraper.Scrape(ctx, db, ch, scraperConfig); err != nil {
			// 记录采集失败
			ch <- prometheus.MustNewConstMetric(
				c.scrapeFailuresDesc,
				prometheus.CounterValue,
				1,
				config.ID, config.Name, string(config.Type), config.Host, fmt.Sprint(config.Port),
				fmt.Sprintf("Scraper[%s] failed: %v", scraper.Name(), err),
			)
		}
	}
}

// createConnection 创建新的数据库连接
func (c *Collector) createConnection(config *model.ConnectionConfig) (*sql.DB, error) {
	// 获取适配器
	dbAdapter, err := c.factory.CreateAdapter(config.Type)
	if err != nil {
		return nil, fmt.Errorf("create adapter failed: %w", err)
	}

	// 使用适配器的 Connect 方法创建连接
	db, err := dbAdapter.Connect(config)
	if err != nil {
		return nil, fmt.Errorf("connect database failed: %w", err)
	}

	// 设置连接参数（用于采集，限制连接数）
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// 确保 Collector 实现了 prometheus.Collector 接口
var _ prometheus.Collector = (*Collector)(nil)
