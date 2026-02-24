package connection

import (
	"database/sql"
	"dbm/internal/model"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Manager 连接管理器
type Manager struct {
	mu                 sync.RWMutex
	connections        map[string]*sql.DB // key: connectionID
	configs            map[string]*model.ConnectionConfig
	decryptedPasswords map[string]string       // 缓存解密后的密码，避免频繁调用昂贵的 Argon2
	groups             map[string]*model.Group // key: groupID
	crypto             *Encryptor
	dataPath           string
}

// NewManager 创建连接管理器
func NewManager(dataPath, encryptionKey string) (*Manager, error) {
	crypto, err := NewEncryptor(encryptionKey)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		connections:        make(map[string]*sql.DB),
		configs:            make(map[string]*model.ConnectionConfig),
		decryptedPasswords: make(map[string]string),
		groups:             make(map[string]*model.Group),
		crypto:             crypto,
		dataPath:           dataPath,
	}

	// 加载配置
	if err := m.loadConfigs(); err != nil {
		return nil, err
	}

	return m, nil
}

// AddConnection 添加连接
func (m *Manager) AddConnection(config *model.ConnectionConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建副本以避免外部修改（如 masking）影响内部存储
	configCopy := *config

	// 加密密码
	// If password looks exactly like base64, assume it might be already encrypted? No, avoid double encryption.
	// Better: Only encrypt if it's not empty. (But empty password is valid... handled by Connect)
	// Actually, simple way: always encrypt what's given. The persistence layer saves encrypted.
	// Wait, updateConnection sends encrypted password from existing? NO.
	// updateConnection retrieves existing (decrypted), so it's plaintext.
	// So config.Password here is Plaintext. We must encrypt it.
	encryptedPassword, err := m.crypto.Encrypt(configCopy.Password)
	if err != nil {
		return err
	}
	configCopy.Password = encryptedPassword

	m.configs[configCopy.ID] = &configCopy
	m.decryptedPasswords[configCopy.ID] = config.Password // 缓存原始明文密码

	// Save to file
	return m.saveConfigs()
}

// RemoveConnection 移除连接
func (m *Manager) RemoveConnection(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭数据库连接
	if db, exists := m.connections[id]; exists {
		_ = db.Close()
		delete(m.connections, id)
	}

	delete(m.configs, id)
	delete(m.decryptedPasswords, id)

	// Save to file
	return m.saveConfigs()
}

// GetConnection 获取已缓存的数据库连接
func (m *Manager) GetConnection(id string) (*sql.DB, *model.ConnectionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[id]
	if !exists {
		return nil, nil, ErrConnectionNotFound
	}

	// 如果连接池中已有连接，直接返回
	if db, exists := m.connections[id]; exists {
		return db, config, nil
	}

	return nil, config, nil
}

// PutConnection 将连接放入管理器
func (m *Manager) PutConnection(id string, db *sql.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[id] = db
}

// CloseConnection 关闭特定连接
func (m *Manager) CloseConnection(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if db, exists := m.connections[id]; exists {
		err := db.Close()
		delete(m.connections, id)
		return err
	}
	return nil
}

// IsConnectionActive 检查连接是否活跃
func (m *Manager) IsConnectionActive(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[id]
	return exists
}

// GetConfig 获取连接配置（解密密码）
func (m *Manager) GetConfig(id string) (*model.ConnectionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[id]
	if !exists {
		return nil, ErrConnectionNotFound
	}
	// 先检查缓存
	if decrypted, exists := m.decryptedPasswords[id]; exists {
		configCopy := *config
		configCopy.Password = decrypted
		return &configCopy, nil
	}

	// 解密密码
	decryptedPassword, err := m.crypto.Decrypt(config.Password)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	m.decryptedPasswords[id] = decryptedPassword

	// 返回配置的副本
	configCopy := *config
	configCopy.Password = decryptedPassword

	return &configCopy, nil
}

// ListConfigs 获取所有连接配置列表
func (m *Manager) ListConfigs() ([]*model.ConnectionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]*model.ConnectionConfig, 0, len(m.configs))
	for id, config := range m.configs {
		// 返回时不包含密码
		configCopy := *config
		configCopy.Password = ""
		// 设置连接状态
		_, configCopy.Connected = m.connections[id]
		configs = append(configs, &configCopy)
	}

	return configs, nil
}

// Close 关闭所有连接
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, db := range m.connections {
		_ = db.Close()
		delete(m.connections, id)
	}

	return nil
}

// ==================== 分组管理 ====================

// AddGroup 添加分组
func (m *Manager) AddGroup(group *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.groups[group.ID] = group
	return m.saveConfigs()
}

// UpdateGroup 更新分组
func (m *Manager) UpdateGroup(group *model.Group) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.groups[group.ID] = group
	return m.saveConfigs()
}

// RemoveGroup 移除分组
func (m *Manager) RemoveGroup(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.groups, id)
	return m.saveConfigs()
}

// ListGroups 获取所有分组
func (m *Manager) ListGroups() ([]*model.Group, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make([]*model.Group, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}

	return groups, nil
}

// saveConfigs 保存配置到文件
func (m *Manager) saveConfigs() error {
	if m.dataPath == "" {
		return nil
	}

	configFile := filepath.Join(m.dataPath, "connections.json")
	groupFile := filepath.Join(m.dataPath, "groups.json")

	// 保存连接配置
	configs := make([]*model.ConnectionConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	configData, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		return err
	}

	// 保存分组配置
	groups := make([]*model.Group, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	groupData, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(groupFile, groupData, 0600)
}

// loadConfigs 从文件加载配置
func (m *Manager) loadConfigs() error {
	if m.dataPath == "" {
		return nil
	}

	configFile := filepath.Join(m.dataPath, "connections.json")
	groupFile := filepath.Join(m.dataPath, "groups.json")

	// 加载连接配置
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err == nil {
			var configs []*model.ConnectionConfig
			if err := json.Unmarshal(data, &configs); err == nil {
				for _, config := range configs {
					m.configs[config.ID] = config
				}
			}
		}
	}

	// 加载分组配置
	if _, err := os.Stat(groupFile); err == nil {
		data, err := os.ReadFile(groupFile)
		if err == nil {
			var groups []*model.Group
			if err := json.Unmarshal(data, &groups); err == nil {
				for _, group := range groups {
					m.groups[group.ID] = group
				}
			}
		}
	}

	return nil
}
