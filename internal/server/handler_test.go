package server

import (
	"dbm/internal/export"
	"dbm/internal/model"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestTypeMappingPreviewHandler 测试类型映射预览 API
func TestTypeMappingPreviewHandler(t *testing.T) {
	// 创建测试用例
	testColumns := []model.ColumnInfo{
		{Name: "id", Type: "INT"},
		{Name: "name", Type: "VARCHAR(255)"},
		{Name: "status", Type: "ENUM"},
		{Name: "count", Type: "TINYINT"},
		{Name: "price", Type: "DECIMAL(10,2)"},
	}

	// 模拟请求
	req := export.SQLExportRequest{
		Tables:      []string{"test_table"},
		TargetDBType: model.DatabasePostgreSQL,
	}

	// 创建模拟上下文
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 设置目标数据库类型到查询参数
	c.Params = gin.Params{
		"id":         "test-id",
		"targetDbType": "postgresql",
	}

	// 创建模拟的连接管理器
	mockConfig := &model.ConnectionConfig{
		ID:     "test-id",
		Name:   "Test Connection",
		Type:   model.DatabaseMySQL,
	}

	// 创建模拟的数据库服务
	mockDB, _, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	defer mockDB.Close()

	mockAdapter := export.NewCSVExporter(&model.CSVOptions{})

	// 创建模拟的 Server
	s := &Server{}

	// 执行请求
	w.ServeHTTP(w, func(w http.ResponseWriter, r *http.Request) {
		c.Params = gin.Params{ // 从请求路径获取参数
			"id": "test-id",
		}

		s.previewExportSQL(c)

		// 检查响应状态
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试用例 2：无效的目标数据库类型
	t.Run("invalid target db type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Params = gin.Params{
			"id":         "test-id",
			"targetDbType": "invalid_db", // 无效类型
		}

		s.previewExportSQL(c)

		// 检查响应状态
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestTypeMappingPreviewHandler_Success 测试成功场景
func TestTypeMappingPreviewHandler_Success(t *testing.T) {
	testColumns := []model.ColumnInfo{
		{Name: "id", Type: "INT"},
		{Name: "name", Type: "VARCHAR(255)"},
	}

	req := export.SQLExportRequest{
		Tables:      []string{"test_table"},
		TargetDBType: model.DatabasePostgreSQL,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
			"id":         "test-id",
			"targetDbType": "postgresql",
		}

	s.previewExportSQL(c)

		assert.Equal(t, http.StatusOK, w.Code)
		result := parseTypeMappingResult(w.Body)

		assert.True(t, result.Success)
		assert.Equal(t, 4, result.Summary.Total)
		assert.Equal(t, 1, result.Summary.Direct)
		assert.Equal(t, 1, result.Summary.Fallback)
		assert.Equal(t, 1, result.Summary.LossyCount)
	})
}

// parseTypeMappingResult 辅助函数：解析响应
func parseTypeMappingResult(body []byte) export.TypeMappingResult {
	var result export.TypeMappingResult
	err := json.Unmarshal(body, &result)
	assert.NoError(t, err)
	return result
}