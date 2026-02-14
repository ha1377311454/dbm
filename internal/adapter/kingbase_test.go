package adapter

import (
	"testing"

	"dbm/internal/model"

	_ "kingbase.com/gokb"
)

const (
	host     = "192.168.5.229"
	port     = 30432
	user     = "system"
	password = "12345678ab"
	dbname   = "test"
)

// TestKingBaseAdapterPing 测试 KingBase 连接
func TestKingBaseAdapterPing(t *testing.T) {
	adapter := NewKingBaseAdapter()

	config := &model.ConnectionConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: password,
		Database: dbname,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	if err := adapter.Ping(db); err != nil {
		t.Errorf("Ping 失败: %v", err)
	} else {
		t.Log("KingBase 连接成功")
	}
}

// TestKingBaseAdapterGetDatabases 测试获取数据库列表
func TestKingBaseAdapterGetDatabases(t *testing.T) {
	adapter := NewKingBaseAdapter()

	config := &model.ConnectionConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: password,
		Database: dbname,
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
}

// TestKingBaseAdapterFactory 测试工厂方法
func TestKingBaseAdapterFactory(t *testing.T) {
	factory := NewFactory()

	adapter, err := factory.CreateAdapter(model.DatabaseKingBase)
	if err != nil {
		t.Fatalf("创建 KingBase 适配器失败: %v", err)
	}

	if _, ok := adapter.(*KingBaseAdapter); !ok {
		t.Error("返回的不是 KingBase 适配器")
	}

	supportedTypes := factory.SupportedTypes()
	var found bool
	for _, ttype := range supportedTypes {
		if ttype == model.DatabaseKingBase {
			found = true
			break
		}
	}

	if !found {
		t.Error("SupportedTypes 中未包含 KingBase")
	}
}
