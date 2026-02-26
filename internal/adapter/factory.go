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
		return NewOracleAdapter(), nil
	case model.DatabaseClickHouse:
		return NewClickHouseAdapter(), nil
	case model.DatabaseKingBase:
		return NewKingBaseAdapter(), nil
	case model.DatabaseDM:
		return NewDMAdapter(), nil
	case model.DatabaseMongoDB:
		return NewMongoDBAdapter(), nil
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
		model.DatabaseDM,
		model.DatabaseMongoDB,
		model.DatabaseOracle,
	}
}

// DefaultFactory 默认适配器工厂实例
var DefaultFactory = NewFactory()
