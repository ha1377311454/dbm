package adapter

import (
	"context"
	"dbm/internal/model"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDBAdapter MongoDB 适配器
type MongoDBAdapter struct {
	BaseAdapter
}

// NewMongoDBAdapter 创建 MongoDB 适配器
func NewMongoDBAdapter() *MongoDBAdapter {
	return &MongoDBAdapter{}
}

// Connect 连接数据库
func (a *MongoDBAdapter) Connect(config *model.ConnectionConfig) (any, error) {
	var uri string
	if config.Host != "" {
		if config.Username != "" && config.Password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", config.Username, config.Password, config.Host, config.Port)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%d", config.Host, config.Port)
		}
	} else if config.Params["uri"] != "" {
		uri = config.Params["uri"]
	} else {
		return nil, fmt.Errorf("host or uri is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 验证连接
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

// Close 关闭连接
func (a *MongoDBAdapter) Close(db any) error {
	client := db.(*mongo.Client)
	return client.Disconnect(context.Background())
}

// Ping 检查连接
func (a *MongoDBAdapter) Ping(db any) error {
	client := db.(*mongo.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.Ping(ctx, nil)
}

// GetDatabases 获取数据库列表
func (a *MongoDBAdapter) GetDatabases(db any) ([]string, error) {
	client := db.(*mongo.Client)
	return client.ListDatabaseNames(context.Background(), bson.M{})
}

// GetTables 获取表（集合）列表
func (a *MongoDBAdapter) GetTables(db any, database string) ([]model.TableInfo, error) {
	client := db.(*mongo.Client)
	collections, err := client.Database(database).ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}

	tableInfos := make([]model.TableInfo, 0, len(collections))
	for _, name := range collections {
		tableInfos = append(tableInfos, model.TableInfo{
			Name:      name,
			Database:  database,
			TableType: "COLLECTION",
		})
	}
	return tableInfos, nil
}

// GetTableSchema 获取集合结构（采样推断）
func (a *MongoDBAdapter) GetTableSchema(db any, database, table string) (*model.TableSchema, error) {
	client := db.(*mongo.Client)
	collection := client.Database(database).Collection(table)

	// 采样第一条文档来推断结构
	var doc bson.M
	err := collection.FindOne(context.Background(), bson.M{}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &model.TableSchema{
				Database: database,
				Table:    table,
				Columns:  []model.ColumnInfo{},
			}, nil
		}
		return nil, err
	}

	columns := make([]model.ColumnInfo, 0, len(doc))
	for k, v := range doc {
		colType := fmt.Sprintf("%T", v)
		columns = append(columns, model.ColumnInfo{
			Name: k,
			Type: colType,
		})
	}

	return &model.TableSchema{
		Database: database,
		Table:    table,
		Columns:  columns,
	}, nil
}

// GetViews 获取视图列表
func (a *MongoDBAdapter) GetViews(db any, database string) ([]model.TableInfo, error) {
	return []model.TableInfo{}, nil
}

// GetIndexes 获取索引列表
func (a *MongoDBAdapter) GetIndexes(db any, database, table string) ([]model.IndexInfo, error) {
	client := db.(*mongo.Client)
	cursor, err := client.Database(database).Collection(table).Indexes().List(context.Background())
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		return nil, err
	}

	indexInfos := make([]model.IndexInfo, 0, len(results))
	for _, res := range results {
		name, _ := res["name"].(string)
		key, _ := res["key"].(bson.M)
		columns := make([]string, 0, len(key))
		for k := range key {
			columns = append(columns, k)
		}

		unique, _ := res["unique"].(bool)
		indexInfos = append(indexInfos, model.IndexInfo{
			Name:    name,
			Columns: columns,
			Unique:  unique,
		})
	}

	return indexInfos, nil
}

// Execute 执行命令
func (a *MongoDBAdapter) Execute(db any, query string, args ...interface{}) (*model.ExecuteResult, error) {
	// 简单实现：将 query 解析为 BSON 并作为 RunCommand 执行
	client := db.(*mongo.Client)
	var command bson.M
	if err := bson.UnmarshalExtJSON([]byte(query), false, &command); err != nil {
		return nil, fmt.Errorf("invalid MongoDB command JSON: %w", err)
	}

	start := time.Now()
	// 默认在 'admin' 数据库执行，或者从参数中解析？
	dbName := "admin"
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			dbName = s
		}
	}

	var result bson.M
	err := client.Database(dbName).RunCommand(context.Background(), command).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &model.ExecuteResult{
		TimeCost: time.Since(start),
		Message:  "Command executed successfully",
	}, nil
}

// Query 执行查询
func (a *MongoDBAdapter) Query(db any, query string, opts *model.QueryOptions) (*model.QueryResult, error) {
	client := db.(*mongo.Client)
	var command bson.D
	if err := bson.UnmarshalExtJSON([]byte(query), false, &command); err != nil {
		// 如果不是有效的 JSON，尝试将其视为集合名称并构建默认 find 命令
		command = bson.D{
			{Key: "find", Value: query},
		}
	}

	// 合并 QueryOptions
	if opts != nil {
		// 检查 command 中是否已有 find 键
		isFind := false
		for _, e := range command {
			if e.Key == "find" {
				isFind = true
				break
			}
		}

		if isFind {
			if opts.PageSize > 0 {
				command = append(command, bson.E{Key: "batchSize", Value: opts.PageSize})
				command = append(command, bson.E{Key: "limit", Value: opts.PageSize})
			}
			if opts.Page > 1 && opts.PageSize > 0 {
				command = append(command, bson.E{Key: "skip", Value: (opts.Page - 1) * opts.PageSize})
			}
		}
	}

	start := time.Now()
	var result bson.M
	err := client.Database(opts.Database).RunCommand(context.Background(), command).Decode(&result)
	if err != nil {
		return nil, err
	}

	// 获取 cursor 和 firstBatch
	var firstBatch bson.A
	if cursorVal, ok := result["cursor"]; ok {
		var cursorMap map[string]any
		switch v := cursorVal.(type) {
		case map[string]any:
			cursorMap = v
		case bson.D:
			cursorMap = make(map[string]any)
			for _, e := range v {
				cursorMap[e.Key] = e.Value
			}
		}

		if cursorMap != nil {
			if batch, ok := cursorMap["firstBatch"].(bson.A); ok {
				firstBatch = batch
			} else if batch, ok := cursorMap["nextBatch"].(bson.A); ok {
				firstBatch = batch
			}
		}
	}

	if firstBatch == nil {
		// 如果没有 cursor，返回原始结果作为单行
		return &model.QueryResult{
			Columns:  []string{"result"},
			Rows:     []map[string]any{result},
			Total:    1,
			TimeCost: time.Since(start),
		}, nil
	}

	rows := make([]map[string]any, 0, len(firstBatch))
	allKeysMap := make(map[string]bool)
	var columns []string

	for i, item := range firstBatch {
		var row map[string]any
		switch v := item.(type) {
		case map[string]any:
			row = v
		case bson.D:
			row = make(map[string]any)
			for _, e := range v {
				row[e.Key] = e.Value
			}
		}

		if row != nil {
			rows = append(rows, row)
			// 如果是第一行，尝试按顺序获取列名
			if i == 0 {
				switch v := item.(type) {
				case bson.D:
					for _, e := range v {
						columns = append(columns, e.Key)
						allKeysMap[e.Key] = true
					}
				default:
					// bson.M 无序，只能收集
					for k := range row {
						if !allKeysMap[k] {
							columns = append(columns, k)
							allKeysMap[k] = true
						}
					}
				}
			} else {
				// 后续行补充新发现的列
				for k := range row {
					if !allKeysMap[k] {
						columns = append(columns, k)
						allKeysMap[k] = true
					}
				}
			}
		}
	}

	// 优化列排序：确保 _id 在第一位，其他列按字母排序(除了已发现的顺序)
	if len(columns) > 0 {
		finalCols := make([]string, 0, len(columns))
		hasId := false
		for _, col := range columns {
			if col == "_id" {
				hasId = true
				break
			}
		}
		if hasId {
			finalCols = append(finalCols, "_id")
		}
		for _, col := range columns {
			if col != "_id" {
				finalCols = append(finalCols, col)
			}
		}
		columns = finalCols
	}

	return &model.QueryResult{
		Columns:  columns,
		Rows:     rows,
		Total:    int64(len(rows)),
		TimeCost: time.Since(start),
	}, nil
}

// Insert 插入数据
func (a *MongoDBAdapter) Insert(db any, database, table string, data map[string]interface{}) error {
	client := db.(*mongo.Client)
	_, err := client.Database(database).Collection(table).InsertOne(context.Background(), data)
	return err
}

// Update 更新数据
func (a *MongoDBAdapter) Update(db any, database, table string, data map[string]interface{}, where string) error {
	client := db.(*mongo.Client)
	var filter bson.M
	if err := bson.UnmarshalExtJSON([]byte(where), true, &filter); err != nil {
		return fmt.Errorf("invalid MongoDB filter JSON: %w", err)
	}

	_, err := client.Database(database).Collection(table).UpdateMany(context.Background(), filter, bson.M{"$set": data})
	return err
}

// Delete 删除数据
func (a *MongoDBAdapter) Delete(db any, database, table, where string) error {
	client := db.(*mongo.Client)
	var filter bson.M
	if err := bson.UnmarshalExtJSON([]byte(where), true, &filter); err != nil {
		return fmt.Errorf("invalid MongoDB filter JSON: %w", err)
	}

	_, err := client.Database(database).Collection(table).DeleteMany(context.Background(), filter)
	return err
}

// ExportToCSV 导出 CSV
func (a *MongoDBAdapter) ExportToCSV(db any, writer io.Writer, database, query string, opts *model.CSVOptions) error {
	qOpts := &model.QueryOptions{
		Database: database,
	}
	result, err := a.Query(db, query, qOpts)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	if opts != nil && opts.Separator != "" {
		csvWriter.Comma = rune(opts.Separator[0])
	}

	// 写入表头
	if err := csvWriter.Write(result.Columns); err != nil {
		return err
	}

	// 写入数据
	for _, row := range result.Rows {
		record := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			val := row[col]
			if val == nil {
				record[i] = ""
				continue
			}
			record[i] = fmt.Sprintf("%v", val)
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

// ExportToSQL 导出 SQL
func (a *MongoDBAdapter) ExportToSQL(db any, writer io.Writer, database string, tables []string, opts *model.SQLOptions) error {
	return fmt.Errorf("not applicable for MongoDB")
}

// GetCreateTableSQL 获取集合创建语句 (JSON options)
func (a *MongoDBAdapter) GetCreateTableSQL(db any, database, table string) (string, error) {
	return "{ \"create\": \"" + table + "\" }", nil
}

// AlterTable 修改表结构 (NOT SUPPORTED)
func (a *MongoDBAdapter) AlterTable(db any, request *model.AlterTableRequest) error {
	return fmt.Errorf("AlterTable is not supported for MongoDB")
}

// RenameTable 重命名集合
func (a *MongoDBAdapter) RenameTable(db any, database, oldName, newName string) error {
	client := db.(*mongo.Client)
	// MongoDB renameCollection is a command
	command := bson.D{
		{Key: "renameCollection", Value: database + "." + oldName},
		{Key: "to", Value: database + "." + newName},
	}
	return client.Database("admin").RunCommand(context.Background(), command).Err()
}
