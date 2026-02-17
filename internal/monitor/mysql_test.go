package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testMySQLHost     = "192.168.5.229"
	testMySQLPort     = 10606
	testMySQLUser     = "root"
	testMySQLPassword = "root"
)

// getTestDB 获取测试数据库连接
func getTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := getTestDSN()
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err, "Failed to open database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to ping database")

	return db
}

// getTestDSN 获取测试数据库 DSN
func getTestDSN() string {
	if dsn := os.Getenv("TEST_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/", testMySQLUser, testMySQLPassword, testMySQLHost, testMySQLPort)
}

// getMySQLConfig 获取 MySQL 配置
func getMySQLConfig() *MetricsConfig {
	return &MetricsConfig{
		Metrics: []Metric{
			{
				Context: "uptime",
				Labels:  []string{"version"},
				MetricsDesc: map[string]string{
					"seconds": "Gauge metric with uptime of database in seconds.",
				},
				MetricsType: map[string]string{
					"seconds": "gauge",
				},
				Request: "SELECT VERSION() AS version, VARIABLE_VALUE AS seconds FROM performance_schema.global_status WHERE VARIABLE_NAME = 'UPTIME'",
			},
			{
				Context: "sessions",
				Labels:  []string{},
				MetricsDesc: map[string]string{
					"active": "Gauge metric with count of active sessions.",
					"idle":   "Gauge metric with count of idle sessions.",
					"total":  "Gauge metric with total number of sessions.",
				},
				MetricsType: map[string]string{
					"active": "gauge",
					"idle":   "gauge",
					"total":  "gauge",
				},
				Request: "SELECT SUM(CASE WHEN COMMAND != 'Sleep' THEN 1 ELSE 0 END) AS active, SUM(CASE WHEN COMMAND = 'Sleep' THEN 1 ELSE 0 END) AS idle, COUNT(*) AS total FROM information_schema.PROCESSLIST",
			},
			{
				Context: "connections",
				Labels:  []string{},
				MetricsDesc: map[string]string{
					"max_connections":       "Gauge metric with maximum allowed connections.",
					"current_connections":   "Gauge metric with current number of connections.",
					"available_connections": "Gauge metric with available connections.",
				},
				MetricsType: map[string]string{
					"max_connections":       "gauge",
					"current_connections":   "gauge",
					"available_connections": "gauge",
				},
				Request: "SELECT (SELECT VARIABLE_VALUE FROM performance_schema.global_variables WHERE VARIABLE_NAME = 'MAX_CONNECTIONS') AS max_connections, (SELECT COUNT(*) FROM information_schema.PROCESSLIST) AS current_connections, (SELECT VARIABLE_VALUE FROM performance_schema.global_variables WHERE VARIABLE_NAME = 'MAX_CONNECTIONS') - (SELECT COUNT(*) FROM information_schema.PROCESSLIST) AS available_connections",
			},
		},
	}
}

// TestMySQLScraper 测试 MySQL 采集器
func TestMySQLScraper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	config := getMySQLConfig()
	scraper := NewDefaultScraper("mysql", config)

	ctx := context.Background()
	ch := make(chan prometheus.Metric, 100)

	scraperConfig := ScraperConfig{
		Labels: map[string]string{
			"connection_id":   "test-connection-id",
			"connection_name": "test-connection",
			"db_type":         "mysql",
			"host":            testMySQLHost,
			"port":            "10606",
		},
		DBType: "mysql",
	}

	err := scraper.Scrape(ctx, db, ch, scraperConfig)
	require.NoError(t, err, "Scrape should not return error")

	close(ch)

	// 收集所有指标
	metrics := make([]prometheus.Metric, 0, cap(ch))
	for m := range ch {
		metrics = append(metrics, m)
	}

	// 验证至少采集到一些指标
	assert.Greater(t, len(metrics), 0, "Should have collected at least one metric")

	// 打印所有指标用于调试
	t.Logf("Collected %d metrics:", len(metrics))
	for i, m := range metrics {
		desc := m.Desc()
		t.Logf("Metric[%d]: %s", i, desc.String())
	}
}

// TestMySQLScraperUptime 测试运行时长指标采集
func TestMySQLScraperUptime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	config := &MetricsConfig{
		Metrics: []Metric{
			{
				Context: "uptime",
				Labels:  []string{"version"},
				MetricsDesc: map[string]string{
					"seconds": "Gauge metric with uptime of database in seconds.",
				},
				MetricsType: map[string]string{
					"seconds": "gauge",
				},
				Request: "SELECT VERSION() AS version, VARIABLE_VALUE AS seconds FROM performance_schema.global_status WHERE VARIABLE_NAME = 'UPTIME'",
			},
		},
	}

	scraper := NewDefaultScraper("mysql", config)
	ctx := context.Background()
	ch := make(chan prometheus.Metric, 10)

	scraperConfig := ScraperConfig{
		Labels: map[string]string{
			"connection_id":   "test-connection-id",
			"connection_name": "test-connection",
			"db_type":         "mysql",
			"host":            testMySQLHost,
			"port":            "10606",
		},
		DBType: "mysql",
	}

	err := scraper.Scrape(ctx, db, ch, scraperConfig)
	require.NoError(t, err)

	close(ch)

	// 验证采集到至少一个指标
	metrics := make([]prometheus.Metric, 0)
	for m := range ch {
		metrics = append(metrics, m)
	}

	assert.Greater(t, len(metrics), 0, "Should have collected uptime metric")
	t.Logf("Collected %d uptime metrics", len(metrics))
}

// TestMySQLScraperSessions 测试会话指标采集
func TestMySQLScraperSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	config := &MetricsConfig{
		Metrics: []Metric{
			{
				Context: "sessions",
				Labels:  []string{},
				MetricsDesc: map[string]string{
					"active": "Gauge metric with count of active sessions.",
					"idle":   "Gauge metric with count of idle sessions.",
					"total":  "Gauge metric with total number of sessions.",
				},
				MetricsType: map[string]string{
					"active": "gauge",
					"idle":   "gauge",
					"total":  "gauge",
				},
				Request: "SELECT SUM(CASE WHEN COMMAND != 'Sleep' THEN 1 ELSE 0 END) AS active, SUM(CASE WHEN COMMAND = 'Sleep' THEN 1 ELSE 0 END) AS idle, COUNT(*) AS total FROM information_schema.PROCESSLIST",
			},
		},
	}

	scraper := NewDefaultScraper("mysql", config)
	ctx := context.Background()
	ch := make(chan prometheus.Metric, 10)

	scraperConfig := ScraperConfig{
		Labels: map[string]string{
			"connection_id":   "test-connection-id",
			"connection_name": "test-connection",
			"db_type":         "mysql",
			"host":            testMySQLHost,
			"port":            "10606",
		},
		DBType: "mysql",
	}

	err := scraper.Scrape(ctx, db, ch, scraperConfig)
	require.NoError(t, err)

	close(ch)

	// 验证采集到指标
	metrics := make([]prometheus.Metric, 0)
	for m := range ch {
		metrics = append(metrics, m)
	}

	assert.Greater(t, len(metrics), 0, "Should have collected sessions metrics")

	t.Logf("Collected %d session metrics", len(metrics))
	for i, m := range metrics {
		t.Logf("Session metric[%d]: %s", i, m.Desc().String())
	}
}

// TestLoadMySQLConfig 测试 MySQL 配置文件加载
func TestLoadMySQLConfig(t *testing.T) {
	mysqlCfg, pgCfg, err := InitDefaultMetrics()
	require.NoError(t, err, "InitDefaultMetrics should not return error")
	assert.NotNil(t, mysqlCfg, "MySQL config should not be nil")
	assert.NotNil(t, pgCfg, "PostgreSQL config should not be nil")

	// 验证 MySQL 配置
	assert.NotEmpty(t, mysqlCfg.Metrics, "MySQL config should have metrics")

	// 验证每个指标配置
	for i, m := range mysqlCfg.Metrics {
		assert.NotEmpty(t, m.Context, "Metric[%d]: context should not be empty", i)
		assert.NotEmpty(t, m.MetricsDesc, "Metric[%d]: metricsdesc should not be empty", i)
		assert.NotEmpty(t, m.Request, "Metric[%d]: request should not be empty", i)

		t.Logf("Metric[%d]: context=%s, metrics=%d",
			i, m.Context, len(m.MetricsDesc))
	}
}

// TestQueryAndParse 测试通用查询解析函数
func TestQueryAndParse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	callCount := 0

	parseFunc := func(row map[string]string) error {
		callCount++
		// 验证返回的数据
		assert.NotEmpty(t, row, "Row should not be empty")
		t.Logf("Row %d: %+v", callCount, row)
		return nil
	}

	query := "SELECT VERSION() AS version, NOW() AS now"
	err := QueryAndParse(ctx, db, parseFunc, query)
	require.NoError(t, err, "QueryAndParse should not return error")
	assert.Equal(t, 1, callCount, "Should have called parseFunc once")
}

// TestRegisterMetric 测试指标注册函数
func TestRegisterMetric(t *testing.T) {
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)

	config := ScraperConfig{
		Labels: map[string]string{
			"connection_id":   "test-id",
			"connection_name": "test-name",
			"db_type":         "mysql",
			"host":            "localhost",
			"port":            "3306",
		},
	}

	desc := MetricDesc{
		Name:   "dbm_test_metric",
		Help:   "Test metric help",
		Labels: []string{"label1"},
	}

	// 测试注册指标
	RegisterMetric(config, ch, desc, 42.0, prometheus.GaugeValue, "value1")

	// 从 channel 中读取指标
	metric, ok := <-ch
	assert.True(t, ok, "Should be able to read metric from channel")
	assert.NotNil(t, metric, "Metric should not be nil")
	assert.NotNil(t, metric.Desc(), "Metric description should not be nil")

	t.Logf("Metric descriptor: %s", metric.Desc().String())
}
