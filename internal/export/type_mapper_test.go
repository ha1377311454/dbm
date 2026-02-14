package export

import (
	"testing"

	"dbm/internal/model"
)

func TestTypeMapper_MapTypes_DirectMapping(t *testing.T) {
	mapper := &TypeMapper{
		mappings: map[string]map[string]*TypeRule{
			"mysql_to_postgresql": {
				"VARCHAR": &TypeRule{
					TargetType:    "VARCHAR",
					PrecisionLoss: false,
				},
			},
		},
	}

	columns := []model.ColumnInfo{
		{Name: "name", Type: "VARCHAR"},
		{Name: "age", Type: "INT"},
	}

	result, err := mapper.MapTypes(model.DatabaseMySQL, model.DatabasePostgreSQL, columns)
	if err != nil {
		t.Fatalf("MapTypes failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected success")
	}

	// VARCHAR 有规则（1 Direct），INT 无规则（1 Direct）
	if result.Summary.Direct != 2 {
		t.Errorf("Expected 2 direct mapping, got %d", result.Summary.Direct)
	}
}

func TestTypeMapper_MapTypes_FallbackWithPrecisionLoss(t *testing.T) {
	mapper := &TypeMapper{
		mappings: map[string]map[string]*TypeRule{
			"mysql_to_postgresql": {
				"TINYINT": &TypeRule{
					TargetType:    "SMALLINT",
					SafeFallback: "INTEGER",
					PrecisionLoss: true,
				},
			},
		},
	}

	columns := []model.ColumnInfo{
		{Name: "count", Type: "TINYINT"},
	}

	result, err := mapper.MapTypes(model.DatabaseMySQL, model.DatabasePostgreSQL, columns)
	if err != nil {
		t.Fatalf("MapTypes failed: %v", err)
	}

	// 根据需求："选择最安全的类型"，应该返回 INTEGER（SafeFallback）
	if result.Mapped["TINYINT"] != "INTEGER" {
		t.Errorf("Expected INTEGER (safe fallback), got %s", result.Mapped["TINYINT"])
	}

	if result.Summary.Fallback != 1 {
		t.Errorf("Expected 1 fallback, got %d", result.Summary.Fallback)
	}

	if result.Summary.LossyCount != 1 {
		t.Errorf("Expected 1 lossy type, got %d", result.Summary.LossyCount)
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestTypeMapper_MapTypes_RequiresUserChoice(t *testing.T) {
	mapper := &TypeMapper{
		mappings: map[string]map[string]*TypeRule{
			"mysql_to_postgresql": {
				"ENUM": &TypeRule{
					TargetType:   "TEXT",
					RequiresUser: true,
					UserOptions: []TypeOption{
						{Label: "转换为 TEXT", Value: "TEXT"},
						{Label: "转换为 VARCHAR(255)", Value: "VARCHAR(255)"},
					},
				},
			},
		},
	}

	columns := []model.ColumnInfo{
		{Name: "status", Type: "ENUM"},
	}

	result, err := mapper.MapTypes(model.DatabaseMySQL, model.DatabasePostgreSQL, columns)
	if err != nil {
		t.Fatalf("MapTypes failed: %v", err)
	}

	if result.Summary.UserChoice != 1 {
		t.Errorf("Expected 1 user choice required, got %d", result.Summary.UserChoice)
	}

	if len(result.RequiresUser) != 1 {
		t.Errorf("Expected 1 type requiring user, got %d", len(result.RequiresUser))
	}

	// 应该使用默认的 TEXT
	if result.Mapped["ENUM"] != "TEXT" {
		t.Errorf("Expected default TEXT, got %s", result.Mapped["ENUM"])
	}
}

func TestTypeMapper_ApplyUserChoices(t *testing.T) {
	mapper := &TypeMapper{
		mappings: map[string]map[string]*TypeRule{
			"mysql_to_postgresql": {
				"ENUM": &TypeRule{
					TargetType:   "TEXT",
					RequiresUser: true,
					UserOptions: []TypeOption{
						{Label: "转换为 TEXT", Value: "TEXT"},
						{Label: "转换为 VARCHAR(255)", Value: "VARCHAR(255)"},
					},
				},
			},
		},
	}

	columns := []model.ColumnInfo{
		{Name: "status", Type: "ENUM"},
	}

	result, _ := mapper.MapTypes(model.DatabaseMySQL, model.DatabasePostgreSQL, columns)

	// 用户选择了 VARCHAR(255)
	choices := map[string]string{
		"ENUM": "VARCHAR(255)",
	}

	err := mapper.ApplyUserChoices(result, choices)
	if err != nil {
		t.Fatalf("ApplyUserChoices failed: %v", err)
	}

	if result.Mapped["ENUM"] != "VARCHAR(255)" {
		t.Errorf("Expected VARCHAR(255), got %s", result.Mapped["ENUM"])
	}

	if result.Summary.UserChoice != 0 {
		t.Errorf("Expected 0 user choice remaining, got %d", result.Summary.UserChoice)
	}
}

func TestTypeMapper_ApplyUserChoices_InvalidChoice(t *testing.T) {
	mapper := &TypeMapper{
		mappings: map[string]map[string]*TypeRule{
			"mysql_to_postgresql": {
				"ENUM": &TypeRule{
					TargetType:   "TEXT",
					RequiresUser: true,
					UserOptions: []TypeOption{
						{Label: "转换为 TEXT", Value: "TEXT"},
					},
				},
			},
		},
	}

	columns := []model.ColumnInfo{
		{Name: "status", Type: "ENUM"},
	}

	result, _ := mapper.MapTypes(model.DatabaseMySQL, model.DatabasePostgreSQL, columns)

	// 用户选择了一个无效的选项
	choices := map[string]string{
		"ENUM": "INVALID_TYPE",
	}

	err := mapper.ApplyUserChoices(result, choices)
	if err == nil {
		t.Error("Expected error for invalid choice")
	}
}