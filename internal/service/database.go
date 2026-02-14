package service

import (
	"dbm/internal/adapter"
	"dbm/internal/connection"
	"dbm/internal/model"
)

// DatabaseService 数据库服务
type DatabaseService struct {
	connManager *connection.Manager
	factory     *adapter.Factory
}

// NewDatabaseService 创建数据库服务
func NewDatabaseService(connManager *connection.Manager, factory *adapter.Factory) *DatabaseService {
	return &DatabaseService{
		connManager: connManager,
		factory:     factory,
	}
}

// GetConnectionService 获取连接服务
func (s *DatabaseService) GetConnectionService() *ConnectionService {
	return NewConnectionService(s.connManager, s.factory)
}

// GetAdapter 获取数据库适配器
func (s *DatabaseService) GetAdapter(dbType model.DatabaseType) (adapter.DatabaseAdapter, error) {
	return s.factory.CreateAdapter(dbType)
}
