package export

import (
	"fmt"
	"os"
	"path/filepath"

	"dbm/internal/model"

	"gopkg.in/yaml.v3"
)

// TypeMapperConfig 配置文件结构
type TypeMapperConfig struct {
	TypeMappings map[string]map[string]TypeRule
}

// TypeRule 类型转换规则
type TypeRule struct {
	TargetType   string
	SafeFallback string
	PrecisionLoss bool
	RequiresUser  bool
	UserOptions  []TypeOption
	Note         string
}

// TypeOption 用户可选择的类型选项
type TypeOption struct {
	Label string
	Value string
}

// TypeMappingResult 类型映射结果
type TypeMappingResult struct {
	Success      bool                `json:"success"`
	Mapped       map[string]string   `json:"mapped"`         // 源类型 → 目标类型
	Warnings     []string            `json:"warnings"`
	RequiresUser map[string]*TypeRule `json:"requiresUser"` // 需要用户选择的类型
	Summary      TypeSummary        `json:"summary"`
}

// TypeSummary 映射摘要
type TypeSummary struct {
	Total       int `json:"total"`
	Direct      int `json:"direct"`       // 直接映射
	Fallback    int `json:"fallback"`     // 安全降级
	UserChoice  int `json:"userChoice"`   // 需要用户选择
	LossyCount  int `json:"lossyCount"`  // 有精度损失
}

// TypeMapper 类型映射器
type TypeMapper struct {
	configPath string
	mappings   map[string]map[string]*TypeRule // key: "mysql_to_postgresql"
}

// NewTypeMapper 创建类型映射器
func NewTypeMapper(configPath string) (*TypeMapper, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read type mapping config: %w", err)
	}

	var config TypeMapperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse type mapping config: %w", err)
	}

	// 转换为指针类型，方便后续使用
	mappings := make(map[string]map[string]*TypeRule)
	for key, mapping := range config.TypeMappings {
		ptrMapping := make(map[string]*TypeRule)
		for typeKey, rule := range mapping {
			ruleCopy := rule
			ptrMapping[typeKey] = &ruleCopy
		}
		mappings[key] = ptrMapping
	}

	return &TypeMapper{
		configPath: configPath,
		mappings:   mappings,
	}, nil
}

// MapTypes 映射类型
// sourceDB: 源数据库类型
// targetDB: 目标数据库类型
// columns: 源表列信息
func (m *TypeMapper) MapTypes(sourceDB, targetDB model.DatabaseType, columns []model.ColumnInfo) (*TypeMappingResult, error) {
	result := &TypeMappingResult{
		Success:      true,
		Mapped:       make(map[string]string),
		RequiresUser: make(map[string]*TypeRule),
		Warnings:     []string{},
		Summary: TypeSummary{
			Total: len(columns),
		},
	}

	mappingKey := getMappingKey(sourceDB, targetDB)
	mapping, exists := m.mappings[mappingKey]

	if !exists {
		// 无映射配置，返回原始类型
		for _, col := range columns {
			result.Mapped[col.Type] = col.Type
		}
		result.Summary.Direct = len(columns)
		return result, nil
	}

	for _, col := range columns {
		rule, hasRule := mapping[col.Type]

		if !hasRule {
			// 无规则，保持原类型
			result.Mapped[col.Type] = col.Type
			result.Summary.Direct++
			continue
		}

		if rule.RequiresUser {
			// 需要用户选择
			result.RequiresUser[col.Type] = rule
			result.Summary.UserChoice++
			// 暂时使用默认值
			result.Mapped[col.Type] = rule.TargetType
		} else if rule.PrecisionLoss {
			// 有精度损失，使用安全降级
			result.Mapped[col.Type] = rule.SafeFallback
			result.Summary.Fallback++
			result.Summary.LossyCount++
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s → %s (有精度损失)", col.Type, rule.SafeFallback))
		} else {
			// 直接映射
			result.Mapped[col.Type] = rule.TargetType
			result.Summary.Direct++
		}
	}

	return result, nil
}

// ApplyUserChoices 应用用户选择的类型映射
func (m *TypeMapper) ApplyUserChoices(result *TypeMappingResult, choices map[string]string) error {
	for sourceType, targetType := range choices {
		if rule, exists := result.RequiresUser[sourceType]; exists {
			// 验证选择是否有效
			valid := false
			for _, opt := range rule.UserOptions {
				if opt.Value == targetType {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid choice for type %s: %s", sourceType, targetType)
			}

			result.Mapped[sourceType] = targetType
			result.Summary.UserChoice--
		}
	}
	return nil
}

// getMappingKey 获取映射键
func getMappingKey(sourceDB, targetDB model.DatabaseType) string {
	return fmt.Sprintf("%s_to_%s", sourceDB, targetDB)
}

// LoadDefaultConfig 加载默认配置
func LoadDefaultConfig() (*TypeMapper, error) {
	// 尝试从用户目录加载
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".dbm", "type_mapping.yaml")

	if _, err := os.Stat(configPath); err == nil {
		return NewTypeMapper(configPath)
	}

	// 使用内置配置
	configPath = filepath.Join("configs", "type_mapping.yaml")
	return NewTypeMapper(configPath)
}