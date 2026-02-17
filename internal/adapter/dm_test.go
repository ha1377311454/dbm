package adapter

import (
	"testing"

	"dbm/internal/model"

	_ "gitee.com/chunanyong/dm"
)

const (
	dmHost     = "192.168.1.11"
	dmPort     = 5236
	dmUser     = "SYSDBA"
	dmPassword = "SYSDBA"
	dmDatabase = "SYSDBA"
)

// TestDMAdapterBuildDSN 测试 DSN 构建
func TestDMAdapterBuildDSN(t *testing.T) {
	adapter := NewDMAdapter()

	tests := []struct {
		name     string
		config   *model.ConnectionConfig
		expected string
	}{
		{
			name: "基础连接",
			config: &model.ConnectionConfig{
				Host:     "localhost",
				Port:     5236,
				Username: "SYSDBA",
				Password: "SYSDBA",
			},
			expected: "dm://SYSDBA:SYSDBA@localhost:5236",
		},
		{
			name: "带额外参数",
			config: &model.ConnectionConfig{
				Host:     "192.168.1.11",
				Port:     5236,
				Username: "SYSDBA",
				Password: "SYSDBA",
				Params: map[string]string{
					"schema": "SYSDBA",
				},
			},
			expected: "dm://SYSDBA:SYSDBA@192.168.1.11:5236?schema=SYSDBA",
		},
		{
			name: "仅主机无端口",
			config: &model.ConnectionConfig{
				Host:     "localhost",
				Username: "SYSDBA",
				Password: "SYSDBA",
			},
			expected: "dm://SYSDBA:SYSDBA@localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := adapter.buildDSN(tt.config)
			if dsn != tt.expected {
				t.Errorf("buildDSN() = %v, want %v", dsn, tt.expected)
			}
			// 验证 DSN 以 dm:// 开头
			if len(dsn) < 5 || dsn[:5] != "dm://" {
				t.Errorf("DSN 必须以 dm:// 开头, got: %s", dsn)
			}
		})
	}
}

// TestDMAdapterFactory 测试工厂方法
func TestDMAdapterFactory(t *testing.T) {
	factory := NewFactory()

	adapter, err := factory.CreateAdapter(model.DatabaseDM)
	if err != nil {
		t.Fatalf("创建达梦适配器失败: %v", err)
	}

	if _, ok := adapter.(*DMAdapter); !ok {
		t.Error("返回的不是达梦适配器")
	}

	supportedTypes := factory.SupportedTypes()
	var found bool
	for _, ttype := range supportedTypes {
		if ttype == model.DatabaseDM {
			found = true
			break
		}
	}

	if !found {
		t.Error("SupportedTypes 中未包含达梦数据库")
	}
}

// TestDMAdapterConnect 测试达梦数据库连接
func TestDMAdapterConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	if err := adapter.Ping(db); err != nil {
		t.Errorf("Ping 失败: %v", err)
	} else {
		t.Log("达梦数据库连接成功")
	}
}

// TestDMAdapterGetDatabases 测试获取数据库列表
func TestDMAdapterGetDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	databases, err := adapter.GetDatabases(db)
	if err != nil {
		t.Fatalf("获取数据库列表失败: %v", err)
	}

	t.Logf("获取到 %d 个数据库:", len(databases))
	for _, db := range databases {
		t.Logf("  - %s", db)
	}

	if len(databases) == 0 {
		t.Error("未获取到任何数据库")
	}
}

// TestDMAdapterGetTables 测试获取表列表
func TestDMAdapterGetTables(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	tables, err := adapter.GetTables(db, dmDatabase)
	if err != nil {
		t.Fatalf("获取表列表失败: %v", err)
	}

	t.Logf("获取到 %d 个表:", len(tables))
	for i, tbl := range tables {
		if i < 10 { // 只打印前10个
			t.Logf("  - %s (%s)", tbl.Name, tbl.TableType)
		}
	}

	if len(tables) == 0 {
		t.Error("未获取到任何表")
	}
}

// TestDMAdapterExecute 测试 SQL 执行
func TestDMAdapterExecute(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	// 测试简单查询
	result, err := adapter.Execute(db, "SELECT COUNT(*) FROM USER_TABLES")
	if err != nil {
		t.Errorf("执行查询失败: %v", err)
	} else {
		t.Logf("查询成功, 耗时: %v, 消息: %s", result.TimeCost, result.Message)
	}
}

// TestDMAdapterQuery 测试查询功能
func TestDMAdapterQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	// 查询系统表
	result, err := adapter.Query(db, "SELECT * FROM USER_TABLES WHERE ROWNUM <= 5", nil)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	t.Logf("查询成功, 耗时: %v, 行数: %d", result.TimeCost, result.Total)
	t.Logf("列: %v", result.Columns)

	if len(result.Rows) == 0 {
		t.Error("未查询到任何数据")
	}
}

// TestDMAdapterGetViews 测试获取视图列表
func TestDMAdapterGetViews(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	views, err := adapter.GetViews(db, dmDatabase)
	if err != nil {
		t.Fatalf("获取视图列表失败: %v", err)
	}

	t.Logf("获取到 %d 个视图:", len(views))
	for i, v := range views {
		if i < 5 { // 只打印前5个
			t.Logf("  - %s", v.Name)
		}
	}
}

// TestDMAdapterTableSchema 测试获取表结构
func TestDMAdapterTableSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()

	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
		Database: dmDatabase,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	// 先获取一个表名
	tables, err := adapter.GetTables(db, dmDatabase)
	if err != nil || len(tables) == 0 {
		t.Skip("需要至少一个表来测试表结构查询")
	}

	tableName := tables[0].Name
	schema, err := adapter.GetTableSchema(db, dmDatabase, tableName)
	if err != nil {
		t.Fatalf("获取表结构失败: %v", err)
	}

	t.Logf("表 %s 的结构:", tableName)
	t.Logf("  列数: %d", len(schema.Columns))
	t.Logf("  索引数: %d", len(schema.Indexes))

	for i, col := range schema.Columns {
		if i < 5 { // 只打印前5列
			t.Logf("  - %s %s NULL: %v", col.Name, col.Type, col.Nullable)
		}
	}
}
