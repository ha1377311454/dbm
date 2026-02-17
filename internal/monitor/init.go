package monitor

import (
	"embed"
	"fmt"
	"sync"
)

//go:embed config/*.yaml
var configFS embed.FS

// dbTypeConfigFile 数据库类型到配置文件的映射
// 添加新数据库支持时，只需在此映射中添加配置文件名
var dbTypeConfigFile = map[string]string{
	"mysql":      "mysql.yaml",
	"postgresql": "pg.yaml",
	"dm":         "dm.yaml",
	"kingbase":   "kingbase.yaml",
}

// metricsConfigsCache 缓存已加载的指标配置
var (
	metricsConfigsCache = make(map[string]*MetricsConfig)
	metricsCacheMutex   sync.RWMutex
)

// SupportedDBTypes 返回所有支持监控的数据库类型
func SupportedDBTypes() []string {
	types := make([]string, 0, len(dbTypeConfigFile))
	for dbType := range dbTypeConfigFile {
		types = append(types, dbType)
	}
	return types
}

// LoadMetricsConfigForDB 加载指定数据库类型的指标配置
func LoadMetricsConfigForDB(dbType string) (*MetricsConfig, error) {
	// 检查缓存
	metricsCacheMutex.RLock()
	if cfg, ok := metricsConfigsCache[dbType]; ok {
		metricsCacheMutex.RUnlock()
		return cfg, nil
	}
	metricsCacheMutex.RUnlock()

	// 获取配置文件名
	configFile, ok := dbTypeConfigFile[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	// 读取配置文件
	yamlPath := "config/" + configFile
	yamlContent, err := configFS.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read config file %s failed: %w", yamlPath, err)
	}

	// 解析配置
	cfg, err := LoadMetricsConfig(dbType, string(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("parse config file %s failed: %w", yamlPath, err)
	}

	// 缓存配置
	metricsCacheMutex.Lock()
	metricsConfigsCache[dbType] = cfg
	metricsCacheMutex.Unlock()

	return cfg, nil
}

// InitDefaultMetrics 初始化默认指标配置
// 返回: 配置映射表, 错误
func InitDefaultMetrics() (map[string]*MetricsConfig, error) {
	configs := make(map[string]*MetricsConfig)

	// 加载所有支持数据库的配置
	for dbType := range dbTypeConfigFile {
		cfg, err := LoadMetricsConfigForDB(dbType)
		if err != nil {
			return nil, fmt.Errorf("load %s metrics config failed: %w", dbType, err)
		}
		configs[dbType] = cfg
	}

	return configs, nil
}

// RegisterDefaultScraper 为指定数据库类型注册默认采集器
// 此函数用于简化 handler.go 中的初始化代码
func RegisterDefaultScraper(collector *Collector, dbType string, cfg *MetricsConfig) {
	RegisterMetricsConfig(dbType, cfg)
	collector.RegisterScraper(dbType, NewDefaultScraper(dbType, cfg))
}

// InitAllScrapers 初始化所有数据库类型的采集器
// 便捷函数，用于一次性初始化所有支持的数据库监控
func InitAllScrapers(collector *Collector) error {
	configs, err := InitDefaultMetrics()
	if err != nil {
		return err
	}

	for dbType, cfg := range configs {
		RegisterDefaultScraper(collector, dbType, cfg)
	}

	return nil
}

// InitMetricsForTypes 初始化指定数据库类型的采集器
// 用于按需初始化特定数据库的监控
func InitMetricsForTypes(collector *Collector, dbTypes []string) error {
	for _, dbType := range dbTypes {
		cfg, err := LoadMetricsConfigForDB(dbType)
		if err != nil {
			return fmt.Errorf("load %s metrics config failed: %w", dbType, err)
		}
		RegisterDefaultScraper(collector, dbType, cfg)
	}
	return nil
}