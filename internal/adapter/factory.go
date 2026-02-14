package adapter

import (
	"dbm/internal/model"
	"fmt"
)

// Factory 适配器工厂
type Factory struct{}

// NewFactory 创建适配器工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateAdapter 创建数据库适配器
func (f *Factory) CreateAdapter(dbType model.DatabaseType) (DatabaseAdapter, error) {
	switch dbType {
	case model.DatabaseMySQL:
		return NewMySQLAdapter(), nil
	case model.DatabasePostgreSQL:
		return NewPostgreSQLAdapter(), nil
	case model.DatabaseSQLite:
		return NewSQLiteAdapter(), nil
	case model.DatabaseMSSQL:
		return nil, fmt.Errorf("SQL Server 适配器尚未实现")
	case model.DatabaseOracle:
		// todo github.com/sijms/go-ora/v2 v2.8.19
		return nil, fmt.Errorf("Oracle 适配器尚未实现")
	case model.DatabaseClickHouse:
		return NewClickHouseAdapter(), nil
	case model.DatabaseKingBase:
		return NewKingBaseAdapter(), nil
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// SupportedTypes 返回支持的数据库类型
func (f *Factory) SupportedTypes() []model.DatabaseType {
	return []model.DatabaseType{
		model.DatabaseMySQL,
		model.DatabasePostgreSQL,
		model.DatabaseSQLite,
		model.DatabaseClickHouse,
		model.DatabaseKingBase,
	}
}

// DefaultFactory 默认适配器工厂实例
var DefaultFactory = NewFactory()
