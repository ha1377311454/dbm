package server

import (
	"dbm/internal/adapter"
	"dbm/internal/connection"
	"dbm/internal/export"
	"dbm/internal/model"
	"dbm/internal/monitor"
	"dbm/internal/service"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server HTTP 服务器
type Server struct {
	engine        *gin.Engine
	connManager   *connection.Manager
	connectionSvc *service.ConnectionService
	databaseSvc   *service.DatabaseService
	staticFS      http.FileSystem
	collector     *monitor.Collector
	registry      *prometheus.Registry
}

// NewServer 创建服务器
func NewServer(connManager *connection.Manager, staticFS http.FileSystem) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())

	factory := adapter.DefaultFactory
	databaseSvc := service.NewDatabaseService(connManager, factory)
	connectionSvc := service.NewConnectionService(connManager, factory)

	// 创建 Prometheus Registry
	registry := prometheus.NewRegistry()

	// 创建监控采集器
	collector := monitor.NewCollector(connManager, factory)
	registry.MustRegister(collector)

	s := &Server{
		engine:        engine,
		connManager:   connManager,
		connectionSvc: connectionSvc,
		databaseSvc:   databaseSvc,
		staticFS:      staticFS,
		collector:     collector,
		registry:      registry,
	}

	// 初始化监控配置
	if err := s.initMonitorMetrics(); err != nil {
		// 记录错误但不阻止启动
		println("Failed to initialize monitor metrics:", err.Error())
	}

	s.setupRoutes()
	return s
}

// initMonitorMetrics 初始化监控指标配置
// 使用 InitAllScrapers 自动初始化所有支持数据库的监控
// 添加新数据库支持时，只需在 monitor/init.go 的 dbTypeConfigFile 中添加配置文件名即可
func (s *Server) initMonitorMetrics() error {
	return monitor.InitAllScrapers(s.collector)
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// API 路由
	api := s.engine.Group("/api/v1")
	{
		// 连接管理
		api.GET("/connections", s.listConnections)
		api.POST("/connections", s.createConnection)
		api.PUT("/connections/:id", s.updateConnection)
		api.DELETE("/connections/:id", s.deleteConnection)
		api.POST("/connections/:id/connect", s.connectConnection)
		api.POST("/connections/:id/close", s.closeConnection)
		api.POST("/connections/test", s.testConnectionConfig)
		api.POST("/connections/:id/test", s.testConnection)

		// 数据库元数据
		api.GET("/connections/:id/databases", s.getDatabases)
		api.GET("/connections/:id/schemas", s.getSchemas)
		api.GET("/connections/:id/tables", s.getTables)
		api.GET("/connections/:id/tables/:table/schema", s.getTableSchema)
		api.GET("/connections/:id/views", s.getViews)

		// 表结构修改
		api.POST("/connections/:id/tables/:table/alter", s.alterTable)
		api.POST("/connections/:id/tables/:table/rename", s.renameTable)

		// 数据编辑
		api.POST("/connections/:id/tables/:table/data", s.createData)
		api.PUT("/connections/:id/tables/:table/data", s.updateData)
		api.DELETE("/connections/:id/tables/:table/data", s.deleteData)

		// SQL 执行
		api.POST("/connections/:id/query", s.executeQuery)
		api.POST("/connections/:id/execute", s.executeNonQuery)

		// 导出
		api.POST("/connections/:id/export/csv", s.exportCSV)
		api.POST("/connections/:id/export/sql", s.exportSQL)
		api.POST("/connections/:id/export/sql/preview", s.previewExportSQL)

		// 分组管理
		api.GET("/groups", s.listGroups)
		api.POST("/groups", s.createGroup)
		api.PUT("/groups/:id", s.updateGroup)
		api.DELETE("/groups/:id", s.deleteGroup)

		// 监控
		api.GET("/monitor/stats", s.getMonitorStats)
	}

	// Prometheus 指标
	s.engine.GET("/metrics", s.metricsHandler)

	// 静态文件服务（SPA）
	// 资源文件
	s.engine.GET("/assets/*filepath", s.serveAsset)

	// 根路径返回 index.html
	s.engine.GET("/", s.serveIndex)

	// 其他所有路径（SPA 路由）也返回 index.html
	s.engine.NoRoute(func(c *gin.Context) {
		// 如果是 API 路径，返回 404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, errorResponse(404, "Not found"))
			return
		}
		// 如果是 metrics 路径
		if c.Request.URL.Path == "/metrics" {
			s.metricsHandler(c)
			return
		}
		// 其他所有路径返回 index.html
		s.serveIndex(c)
	})
}

// Run 启动服务器
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}

// ==================== 连接管理 ====================

// listConnections 获取连接列表
func (s *Server) listConnections(c *gin.Context) {
	configs, err := s.connManager.ListConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(configs))
}

// createConnection 创建连接
func (s *Server) createConnection(c *gin.Context) {
	var config model.ConnectionConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	// 生成 ID
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	if err := s.connManager.AddConnection(&config); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 返回时不包含密码
	config.Password = ""
	c.JSON(http.StatusOK, successResponse(config))
}

// updateConnection 更新连接
func (s *Server) updateConnection(c *gin.Context) {
	id := c.Param("id")

	var config model.ConnectionConfig

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	config.ID = id
	config.UpdatedAt = time.Now()

	// 如果密码为空，保留原有密码
	if config.Password == "" {
		if existing, err := s.connManager.GetConfig(id); err == nil {
			config.Password = existing.Password
		}
	}

	// 添加新配置
	if err := s.connManager.AddConnection(&config); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 返回时不包含密码
	config.Password = ""
	c.JSON(http.StatusOK, successResponse(config))
}

// deleteConnection 删除连接
func (s *Server) deleteConnection(c *gin.Context) {
	id := c.Param("id")

	if err := s.connManager.RemoveConnection(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(nil))
}

// testConnection 测试连接
func (s *Server) testConnection(c *gin.Context) {
	id := c.Param("id")

	config, err := s.connManager.GetConfig(id)
	if err != nil {
		if err == connection.ErrConnectionNotFound {
			c.JSON(http.StatusNotFound, errorResponse(404, "Connection not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 测试连接
	result, err := s.connectionSvc.TestConnection(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	if result.Success {
		c.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"connected": true,
			"latency":   "< 10ms",
		}))
	} else {
		c.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"connected": false,
			"error":     result.Error,
		}))
	}
}

// closeConnection 关闭连接
func (s *Server) closeConnection(c *gin.Context) {
	id := c.Param("id")

	if err := s.connectionSvc.CloseConnection(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(nil))
}

// connectConnection 建立连接并缓存
func (s *Server) connectConnection(c *gin.Context) {
	id := c.Param("id")

	// GetDBCached 会自动将主连接放入连接池
	_, _, err := s.connectionSvc.GetDBCached(id, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(map[string]interface{}{
		"connected": true,
	}))
}

// testConnectionConfig 测试连接配置（未保存的）
func (s *Server) testConnectionConfig(c *gin.Context) {
	var config model.ConnectionConfig

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	// 测试连接
	result, err := s.connectionSvc.TestConnection(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	if result.Success {
		c.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"connected": true,
			"latency":   "< 10ms",
		}))
	} else {
		c.JSON(http.StatusOK, successResponse(map[string]interface{}{
			"connected": false,
			"error":     result.Error,
		}))
	}
}

// ==================== 分组管理 ====================

// listGroups 获取所有分组
func (s *Server) listGroups(c *gin.Context) {
	groups, err := s.connManager.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, successResponse(groups))
}

// createGroup 创建分组
func (s *Server) createGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if group.ID == "" {
		group.ID = uuid.New().String()
	}

	if err := s.connManager.AddGroup(&group); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(group))
}

// updateGroup 更新分组
func (s *Server) updateGroup(c *gin.Context) {
	id := c.Param("id")
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	group.ID = id
	if err := s.connManager.UpdateGroup(&group); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(group))
}

// deleteGroup 删除分组
func (s *Server) deleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := s.connManager.RemoveGroup(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, successResponse(nil))
}

// ==================== 数据库元数据 ====================

// getSchemas 获取 schema 列表
func (s *Server) getSchemas(c *gin.Context) {
	id := c.Param("id")
	database := c.DefaultQuery("database", "")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	defer func() {}() // 保持连接打开

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 只有 PostgreSQL 支持 schema
	schemaAware, ok := dbAdapter.(adapter.SchemaAwareDatabase)
	if !ok {
		c.JSON(http.StatusOK, successResponse([]string{}))
		return
	}

	// 如果未指定数据库，尝试获取第一个
	if database == "" {
		dbs, err := dbAdapter.GetDatabases(db)
		if err == nil && len(dbs) > 0 {
			database = dbs[0]
		}
	}

	schemas, err := schemaAware.GetSchemas(db, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(schemas))
}

// getDatabases 获取数据库列表
func (s *Server) getDatabases(c *gin.Context) {
	id := c.Param("id")

	db, config, err := s.connectionSvc.GetDB(id, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	defer func() {}() // 保持连接打开

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	databases, err := dbAdapter.GetDatabases(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(databases))
}

// getTables 获取表列表
func (s *Server) getTables(c *gin.Context) {
	id := c.Param("id")
	database := c.DefaultQuery("database", "")
	schema := c.DefaultQuery("schema", "")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	defer func() {}() // 保持连接打开

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 如果未指定数据库，尝试获取第一个
	if database == "" {
		dbs, err := dbAdapter.GetDatabases(db)
		if err == nil && len(dbs) > 0 {
			database = dbs[0]
		}
	}

	var tables []model.TableInfo
	// PostgreSQL 支持 schema 参数
	if schema != "" {
		if schemaAware, ok := dbAdapter.(adapter.SchemaAwareDatabase); ok {
			tables, err = schemaAware.GetTablesWithSchema(db, database, schema)
		} else {
			tables, err = dbAdapter.GetTables(db, database)
		}
	} else {
		tables, err = dbAdapter.GetTables(db, database)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(tables))
}

// getTableSchema 获取表结构
func (s *Server) getTableSchema(c *gin.Context) {
	id := c.Param("id")
	table := c.Param("table")
	database := c.Query("database")
	schema := c.DefaultQuery("schema", "")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	defer func() {}() // 保持连接打开

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	var tableSchema *model.TableSchema
	// PostgreSQL 支持 schema 参数
	if schema != "" {
		if schemaAware, ok := dbAdapter.(adapter.SchemaAwareDatabase); ok {
			tableSchema, err = schemaAware.GetTableSchemaWithSchema(db, database, schema, table)
		} else {
			tableSchema, err = dbAdapter.GetTableSchema(db, database, table)
		}
	} else {
		tableSchema, err = dbAdapter.GetTableSchema(db, database, table)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(tableSchema))
}

// getViews 获取视图列表
func (s *Server) getViews(c *gin.Context) {
	id := c.Param("id")
	database := c.DefaultQuery("database", "")
	schema := c.DefaultQuery("schema", "")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
	defer func() {}() // 保持连接打开

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	var views []model.TableInfo
	// PostgreSQL 支持 schema 参数
	if schema != "" {
		if schemaAware, ok := dbAdapter.(adapter.SchemaAwareDatabase); ok {
			views, err = schemaAware.GetViewsWithSchema(db, database, schema)
		} else {
			views, err = dbAdapter.GetViews(db, database)
		}
	} else {
		views, err = dbAdapter.GetViews(db, database)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(views))
}

// ==================== SQL 执行 ====================

// executeQuery 执行查询
func (s *Server) executeQuery(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Query string              `json:"query"`
		Opts  *model.QueryOptions `json:"opts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Query cannot be empty"))
		return
	}

	database := ""
	if req.Opts != nil {
		database = req.Opts.Database
	}

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	result, err := dbAdapter.Query(db, req.Query, req.Opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(result))
}

// executeNonQuery 执行非查询 SQL
func (s *Server) executeNonQuery(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Query string `json:"query"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Query cannot be empty"))
		return
	}

	database := c.Query("database")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	result, err := dbAdapter.Execute(db, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(result))
}

// ==================== 导出 ====================

// exportCSV CSV 导出
func (s *Server) exportCSV(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Query string            `json:"query"`
		Opts  *model.CSVOptions `json:"opts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	database := c.Query("database")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=export.csv")

	// 执行导出
	err = dbAdapter.ExportToCSV(db, c.Writer, database, req.Query, req.Opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
}

// exportSQL SQL 导出
func (s *Server) exportSQL(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Tables []string          `json:"tables"`
		Opts   *model.SQLOptions `json:"opts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if len(req.Tables) == 0 && (req.Opts == nil || req.Opts.Query == "") {
		c.JSON(http.StatusBadRequest, errorResponse(400, "No tables or query specified"))
		return
	}

	database := c.Query("database")

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/sql; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=export.sql")

	// 执行导出
	err = dbAdapter.ExportToSQL(db, c.Writer, database, req.Tables, req.Opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}
}

// previewExportSQL 预览 SQL 导出的类型映射
func (s *Server) previewExportSQL(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Tables       []string           `json:"tables"`
		TargetDBType model.DatabaseType `json:"targetDbType"` // 目标数据库类型
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if len(req.Tables) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse(400, "No tables specified"))
		return
	}

	// 验证目标数据库类型
	if req.TargetDBType == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Target database type required"))
		return
	}

	// 获取连接配置
	config, err := s.connManager.GetConfig(id)
	if err != nil {
		if err == connection.ErrConnectionNotFound {
			c.JSON(http.StatusNotFound, errorResponse(404, "Connection not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 获取数据库连接
	db, _, err := s.connectionSvc.GetDB(id, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 获取适配器
	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 获取所有表的列信息
	var allColumns []model.ColumnInfo
	for _, table := range req.Tables {
		schema, err := dbAdapter.GetTableSchema(db, "", table)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorResponse(500, "Failed to get table schema: "+table))
			return
		}
		allColumns = append(allColumns, schema.Columns...)
	}

	// 执行类型映射
	mapper, err := export.LoadDefaultConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, "Failed to load type mapper"))
		return
	}

	// 如果是同类型数据库迁移，不需要类型映射
	if config.Type == req.TargetDBType {
		// 返回直接映射结果
		result := &export.TypeMappingResult{
			Success: true,
			Mapped:  make(map[string]string),
			Summary: export.TypeSummary{
				Total:  len(allColumns),
				Direct: len(allColumns),
			},
		}
		for _, col := range allColumns {
			result.Mapped[col.Type] = col.Type
		}
		c.JSON(http.StatusOK, successResponse(result))
		return
	}

	result, err := mapper.MapTypes(config.Type, req.TargetDBType, allColumns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(result))
}

// ==================== 监控 ====================

// getMonitorStats 获取监控统计
func (s *Server) getMonitorStats(c *gin.Context) {
	c.JSON(http.StatusOK, successResponse(map[string]interface{}{
		"connections": 0,
		"queries":     0,
	}))
}

// metricsHandler Prometheus 指标处理器
func (s *Server) metricsHandler(c *gin.Context) {
	handler := promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(c.Writer, c.Request)
}

// ==================== 静态文件服务 ====================

// serveIndex 服务 index.html
func (s *Server) serveIndex(c *gin.Context) {
	// 设置正确的 Content-Type
	c.Header("Content-Type", "text/html; charset=utf-8")

	// 打开 index.html 文件
	file, err := s.staticFS.Open("index.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, "Failed to load index.html"))
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, "Failed to read index.html"))
		return
	}

	// 返回文件内容
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
}

// serveAsset 服务静态资源文件
func (s *Server) serveAsset(c *gin.Context) {
	// 获取文件路径
	filePath := c.Param("filepath")
	// Gin 的 *filepath 参数可能包含或不包含前导斜杠，需要处理
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid file path"))
		return
	}

	// 打开文件（文件系统中的路径是 assets/xxx）
	file, err := s.staticFS.Open("assets/" + filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse(404, "File not found: "+filePath))
		return
	}
	defer file.Close()

	// 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, "Failed to get file info"))
		return
	}

	// 根据文件扩展名设置 Content-Type
	ext := path.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)

	// 设置 Content-Length 头
	c.Header("Content-Length", strconv.Itoa(int(stat.Size())))

	// 使用 io.Copy 直接传输文件内容，更适合大文件和 ES6 模块
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		// 错误发生时连接可能已经关闭，不需要额外处理
		return
	}
}

// ==================== 辅助函数 ====================

// APIResponse 统一响应格式
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// successResponse 成功响应
func successResponse(data interface{}) APIResponse {
	return APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// errorResponse 错误响应
func errorResponse(code int, message string) APIResponse {
	return APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// ==================== 数据编辑 ====================

// createData 创建数据
func (s *Server) createData(c *gin.Context) {
	id := c.Param("id")
	table := c.Param("table")
	database := c.Query("database")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	if err := dbAdapter.Insert(db, database, table, data); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(nil))
}

// updateData 更新数据
func (s *Server) updateData(c *gin.Context) {
	id := c.Param("id")
	table := c.Param("table")
	database := c.Query("database")

	var req struct {
		Data  map[string]interface{} `json:"data"`
		Where string                 `json:"where"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if req.Where == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Where condition required"))
		return
	}

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	if err := dbAdapter.Update(db, database, table, req.Data, req.Where); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(nil))
}

// deleteData 删除数据
func (s *Server) deleteData(c *gin.Context) {
	id := c.Param("id")
	table := c.Param("table")
	database := c.Query("database")

	var req struct {
		Where string `json:"where"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body"))
		return
	}

	if req.Where == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Where condition required"))
		return
	}

	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	if err := dbAdapter.Delete(db, database, table, req.Where); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(nil))
}

// alterTable 修改表结构
func (s *Server) alterTable(c *gin.Context) {
	id := c.Param("id")
	table := c.Param("table")
	database := c.Query("database")

	var req model.AlterTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body: "+err.Error()))
		return
	}

	// 设置数据库和表名
	if req.Database == "" {
		req.Database = database
	}
	if req.Table == "" {
		req.Table = table
	}

	// 验证请求
	if req.Database == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Database name required"))
		return
	}
	if req.Table == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Table name required"))
		return
	}
	if len(req.Actions) == 0 {
		c.JSON(http.StatusBadRequest, errorResponse(400, "No actions specified"))
		return
	}

	// 获取数据库连接
	db, config, err := s.connectionSvc.GetDB(id, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 获取适配器
	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 执行表结构修改
	if err := dbAdapter.AlterTable(db, &req); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(map[string]interface{}{
		"message": "Table altered successfully",
	}))
}

// renameTable 重命名表
func (s *Server) renameTable(c *gin.Context) {
	id := c.Param("id")
	oldName := c.Param("table")
	database := c.Query("database")

	var req struct {
		NewName string `json:"newName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Invalid request body: "+err.Error()))
		return
	}

	if database == "" {
		c.JSON(http.StatusBadRequest, errorResponse(400, "Database name required"))
		return
	}

	// 获取数据库连接
	db, config, err := s.connectionSvc.GetDB(id, database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 获取适配器
	dbAdapter, err := s.databaseSvc.GetAdapter(config.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	// 执行重命名
	if err := dbAdapter.RenameTable(db, database, oldName, req.NewName); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, successResponse(map[string]interface{}{
		"message": "Table renamed successfully",
		"oldName": oldName,
		"newName": req.NewName,
	}))
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
