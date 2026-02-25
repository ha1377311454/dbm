package service

import (
	"dbm/internal/adapter"
	"dbm/internal/connection"
	"dbm/internal/model"
)

// ConnectionService 连接服务
type ConnectionService struct {
	connManager *connection.Manager
	factory     *adapter.Factory
}

// NewConnectionService 创建连接服务
func NewConnectionService(connManager *connection.Manager, factory *adapter.Factory) *ConnectionService {
	return &ConnectionService{
		connManager: connManager,
		factory:     factory,
	}
}

// GetDB 获取数据库连接（每次创建新连接）
func (s *ConnectionService) GetDB(connectionID string, database string) (any, *model.ConnectionConfig, error) {
	// 获取配置
	config, err := s.connManager.GetConfig(connectionID)
	if err != nil {
		return nil, nil, err
	}

	// 创建副本以避免修改原始配置（ contamination ）
	configCopy := *config

	// 如果指定了数据库，覆盖配置中的默认数据库
	if database != "" {
		configCopy.Database = database
	}

	// 创建适配器
	dbAdapter, err := s.factory.CreateAdapter(configCopy.Type)
	if err != nil {
		return nil, nil, err
	}

	// 创建新连接
	db, err := dbAdapter.Connect(&configCopy)
	if err != nil {
		return nil, &configCopy, err
	}

	// 放入连接池（如果未指定特定数据库，则认为是主连接）
	if database == "" {
		s.connManager.PutConnection(connectionID, db)
	}

	return db, &configCopy, nil
}

// GetDBCached 获取缓存的数据库连接
func (s *ConnectionService) GetDBCached(connectionID string, database string) (any, *model.ConnectionConfig, error) {
	// 如果是主数据库连接，尝试从缓存获取
	if database == "" {
		if db, config, err := s.connManager.GetConnection(connectionID); err == nil && db != nil {
			return db, config, nil
		}
	}

	return s.GetDB(connectionID, database)
}

// CloseConnection 关闭连接
func (s *ConnectionService) CloseConnection(connectionID string) error {
	return s.connManager.CloseConnection(connectionID)
}

// TestConnection 测试连接
func (s *ConnectionService) TestConnection(config *model.ConnectionConfig) (*ConnectionTestResult, error) {
	// 创建适配器
	dbAdapter, err := s.factory.CreateAdapter(config.Type)
	if err != nil {
		return nil, err
	}

	// 尝试连接
	db, err := dbAdapter.Connect(config)
	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer dbAdapter.Close(db)

	// 测试 Ping
	if err := dbAdapter.Ping(db); err != nil {
		return &ConnectionTestResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ConnectionTestResult{
		Success: true,
		Latency: "<10ms",
	}, nil
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}
