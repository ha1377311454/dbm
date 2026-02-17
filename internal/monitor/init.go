package monitor

import (
	"embed"
	"fmt"
)

//go:embed config/*.yaml
var configFS embed.FS

// InitDefaultMetrics 初始化默认指标配置
func InitDefaultMetrics() (*MetricsConfig, *MetricsConfig, error) {
	// 读取 MySQL 配置
	mysqlYAML, err := configFS.ReadFile("config/mysql.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("read mysql config failed: %w", err)
	}
	mysqlCfg, err := LoadMetricsConfig("mysql", string(mysqlYAML))
	if err != nil {
		return nil, nil, fmt.Errorf("parse mysql config failed: %w", err)
	}

	// 读取 PostgreSQL 配置
	pgYAML, err := configFS.ReadFile("config/pg.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("read pg config failed: %w", err)
	}
	pgCfg, err := LoadMetricsConfig("postgresql", string(pgYAML))
	if err != nil {
		return nil, nil, fmt.Errorf("parse pg config failed: %w", err)
	}

	return mysqlCfg, pgCfg, nil
}