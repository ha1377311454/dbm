package model

import (
	"encoding/json"
	"time"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	DatabaseMySQL      DatabaseType = "mysql"
	DatabasePostgreSQL DatabaseType = "postgresql"
	DatabaseSQLite     DatabaseType = "sqlite"
	DatabaseMSSQL      DatabaseType = "mssql"
	DatabaseOracle     DatabaseType = "oracle"
	DatabaseClickHouse DatabaseType = "clickhouse"
	DatabaseKingBase   DatabaseType = "kingbase"
	DatabaseDM         DatabaseType = "dm"      // 达梦数据库
	DatabaseMongoDB    DatabaseType = "mongodb" // MongoDB
)

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`     // 连接名称
	Type              DatabaseType      `json:"type"`     // 数据库类型
	Host              string            `json:"host"`     // 主机地址
	Port              int               `json:"port"`     // 端口
	Username          string            `json:"username"` // 用户名
	Password          string            `json:"password"` // 数据库密码（加密存储，仅内部使用）
	Database          string            `json:"database"` // 数据库名
	Params            map[string]string `json:"params"`   // 额外参数
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
	GroupID           string            `json:"groupId"`           // 所属分组 ID
	Connected         bool              `json:"connected"`         // 运行时状态：是否已连接
	MonitoringEnabled bool              `json:"monitoringEnabled"` // 是否启用监控
}

// ConnectionParams 序列化参数
func (c *ConnectionConfig) ConnectionParams() string {
	if c.Params == nil {
		return "{}"
	}
	data, _ := json.Marshal(c.Params)
	return string(data)
}

// ParseConnectionParams 解析参数
func (c *ConnectionConfig) ParseConnectionParams(params string) error {
	if params == "" || params == "{}" {
		c.Params = make(map[string]string)
		return nil
	}
	return json.Unmarshal([]byte(params), &c.Params)
}
