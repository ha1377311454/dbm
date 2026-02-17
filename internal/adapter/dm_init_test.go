package adapter

import (
	"database/sql"
	"fmt"
	"testing"

	"dbm/internal/model"

	_ "gitee.com/chunanyong/dm"
)

// getDMConnection 获取达梦数据库连接
func getDMConnection() (*sql.DB, error) {
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d", dmUser, dmPassword, dmHost, dmPort)
	db, err := sql.Open("dm", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// TestDMCreateDatabase 创建测试数据库（Schema）
func TestDMCreateDatabase(t *testing.T) {
	db, err := getDMConnection()
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer db.Close()

	// DM数据库中创建Schema
	_, err = db.Exec("CREATE SCHEMA TEST")
	if err != nil {
		t.Logf("创建Schema失败（可能已存在）: %v", err)
		// 尝试删除后重新创建
		db.Exec("DROP SCHEMA TEST CASCADE")
		_, err = db.Exec("CREATE SCHEMA TEST")
		if err != nil {
			t.Fatalf("重新创建Schema失败: %v", err)
		}
	}
	t.Log("Schema TEST 创建成功")
}

// TestDMCreateTable 创建测试表
func TestDMCreateTable(t *testing.T) {
	db, err := getDMConnection()
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer db.Close()

	// 设置当前Schema
	_, err = db.Exec("SET SCHEMA TEST")
	if err != nil {
		t.Fatalf("设置Schema失败: %v", err)
	}

	// 删除已存在的表
	db.Exec("DROP TABLE TEST.users")

	// 创建用户表
	createTableSQL := `
	CREATE TABLE TEST.users (
		id INT PRIMARY KEY,
		username VARCHAR(50) NOT NULL,
		email VARCHAR(100),
		age INT,
		score DECIMAL(10,2),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_active INT DEFAULT 1
	)
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}
	t.Log("表 TEST.users 创建成功")
}

// TestDMInsertData 插入100行测试数据
func TestDMInsertData(t *testing.T) {
	db, err := getDMConnection()
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer db.Close()

	// 准备插入语句
	insertSQL := `
	INSERT INTO TEST.users (id, username, email, age, score, is_active)
	VALUES (:1, :2, :3, :4, :5, :6)
	`

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		t.Fatalf("准备语句失败: %v", err)
	}
	defer stmt.Close()

	// 插入100行数据
	for i := 1; i <= 100; i++ {
		username := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user%d@test.com", i)
		age := 20 + (i % 40)
		score := float64(60 + (i % 40))
		isActive := 1
		if i%10 == 0 {
			isActive = 0
		}

		_, err := stmt.Exec(i, username, email, age, score, isActive)
		if err != nil {
			t.Errorf("插入第%d行失败: %v", i, err)
		}
	}

	t.Log("成功插入100行数据")
}

// TestDMQueryData 查询测试数据
func TestDMQueryData(t *testing.T) {
	db, err := getDMConnection()
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer db.Close()

	// 查询数据总数
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM TEST.users").Scan(&count)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}

	t.Logf("TEST.users 表中共有 %d 行数据", count)

	// 查询前5条数据
	rows, err := db.Query("SELECT id, username, email, age, score FROM TEST.users WHERE ROWNUM <= 5 ORDER BY id")
	if err != nil {
		t.Fatalf("查询数据失败: %v", err)
	}
	defer rows.Close()

	t.Log("前5条数据:")
	for rows.Next() {
		var id, age int
		var username, email string
		var score float64
		if err := rows.Scan(&id, &username, &email, &age, &score); err != nil {
			t.Errorf("扫描行失败: %v", err)
			continue
		}
		t.Logf("  ID: %d, 用户名: %s, 邮箱: %s, 年龄: %d, 分数: %.2f", id, username, email, age, score)
	}
}

// TestDMFullInit 完整初始化测试（一键执行所有操作）
func TestDMFullInit(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	adapter := NewDMAdapter()
	config := &model.ConnectionConfig{
		Host:     dmHost,
		Port:     dmPort,
		Username: dmUser,
		Password: dmPassword,
	}

	db, err := adapter.Connect(config)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer adapter.Close(db)

	t.Log("=== 开始初始化达梦测试数据 ===")

	// 1. 创建Schema
	t.Log("步骤1: 创建Schema TEST")
	_, err = db.Exec("CREATE SCHEMA TEST")
	if err != nil {
		t.Logf("Schema已存在，尝试重新创建: %v", err)
		db.Exec("DROP SCHEMA TEST CASCADE")
		_, err = db.Exec("CREATE SCHEMA TEST")
		if err != nil {
			t.Fatalf("创建Schema失败: %v", err)
		}
	}
	t.Log("✓ Schema TEST 创建成功")

	// 2. 创建表
	t.Log("步骤2: 创建表 users")
	db.Exec("DROP TABLE TEST.users")
	createTableSQL := `
	CREATE TABLE TEST.users (
		id INT PRIMARY KEY,
		username VARCHAR(50) NOT NULL,
		email VARCHAR(100),
		age INT,
		score DECIMAL(10,2),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_active INT DEFAULT 1
	)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		t.Fatalf("创建表失败: %v", err)
	}
	t.Log("✓ 表 TEST.users 创建成功")

	// 3. 插入100行数据
	t.Log("步骤3: 插入100行测试数据")
	insertSQL := `INSERT INTO TEST.users (id, username, email, age, score, is_active) VALUES (:1, :2, :3, :4, :5, :6)`
	stmt, _ := db.Prepare(insertSQL)
	defer stmt.Close()

	for i := 1; i <= 100; i++ {
		username := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user%d@test.com", i)
		age := 20 + (i % 40)
		score := float64(60 + (i % 40))
		isActive := 1
		if i%10 == 0 {
			isActive = 0
		}
		stmt.Exec(i, username, email, age, score, isActive)
	}
	t.Log("✓ 成功插入100行数据")

	// 4. 验证数据
	var count int
	db.QueryRow("SELECT COUNT(*) FROM TEST.users").Scan(&count)
	t.Logf("✓ 数据验证: 表中共有 %d 行数据", count)

	t.Log("=== 初始化完成 ===")
}
