package monitor

import (
	"dbm/internal/adapter"
	"dbm/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSupportedDBTypes 测试支持的数据库类型
func TestSupportedDBTypes(t *testing.T) {
	types := SupportedDBTypes()
	assert.NotEmpty(t, types, "应该有支持的数据库类型")

	// 验证包含所有预期类型
	expectedTypes := []string{"mysql", "postgresql", "dm", "kingbase"}
	for _, expected := range expectedTypes {
		assert.Contains(t, types, expected, "应该包含 %s", expected)
	}
}

// TestLoadAllMetricsConfigs 测试加载所有数据库的监控配置
func TestLoadAllMetricsConfigs(t *testing.T) {
	types := SupportedDBTypes()

	for _, dbType := range types {
		t.Run(dbType, func(t *testing.T) {
			cfg, err := LoadMetricsConfigForDB(dbType)
			require.NoError(t, err, "加载 %s 配置不应出错", dbType)
			require.NotNil(t, cfg, "%s 配置不应为空", dbType)
			assert.NotEmpty(t, cfg.Metrics, "%s 应该有指标配置", dbType)

			// 验证每个指标配置的完整性
			for i, metric := range cfg.Metrics {
				assert.NotEmpty(t, metric.Context, "%s[%d]: context 不应为空", dbType, i)
				assert.NotEmpty(t, metric.MetricsDesc, "%s[%d]: metricsdesc 不应为空", dbType, i)
				assert.NotEmpty(t, metric.Request, "%s[%d]: request 不应为空", dbType, i)
			}

			t.Logf("%s: 成功加载 %d 个指标", dbType, len(cfg.Metrics))
		})
	}
}

// TestInitAllScrapers 测试初始化所有采集器
func TestInitAllScrapers(t *testing.T) {
	// 创建一个真实的 collector
	collector := NewCollector(nil, adapter.DefaultFactory)

	err := InitAllScrapers(collector)
	require.NoError(t, err, "初始化所有采集器不应出错")

	// 验证每个数据库类型都有采集器
	types := SupportedDBTypes()
	for _, dbTypeStr := range types {
		dbType := model.DatabaseType(dbTypeStr)
		scrapers, ok := collector.scrapers[dbType]
		assert.True(t, ok, "%s 应该有采集器", dbType)
		assert.NotEmpty(t, scrapers, "%s 应该有至少一个采集器", dbType)
		t.Logf("%s: %d 个采集器", dbType, len(scrapers))
	}
}

// TestLoadMetricsConfigForDB_Cache 测试配置缓存功能
func TestLoadMetricsConfigForDB_Cache(t *testing.T) {
	// 第一次加载
	cfg1, err1 := LoadMetricsConfigForDB("mysql")
	require.NoError(t, err1)
	require.NotNil(t, cfg1)

	// 第二次加载（应该从缓存读取）
	cfg2, err2 := LoadMetricsConfigForDB("mysql")
	require.NoError(t, err2)
	require.NotNil(t, cfg2)

	// 验证两次获取的是同一个实例（缓存生效）
	assert.Same(t, cfg1, cfg2, "应该返回缓存的同一实例")
}