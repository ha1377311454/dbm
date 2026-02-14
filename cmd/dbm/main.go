package main

import (
	"dbm/internal/assets"
	"dbm/internal/connection"
	"dbm/internal/server"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	// 命令行参数
	host := flag.String("host", "0.0.0.0", "监听地址")
	port := flag.Int("port", 2048, "监听端口")
	configPath := flag.String("config", "", "配置文件路径")
	dataPath := flag.String("data", "", "数据目录路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	showConfig := flag.Bool("config-path", false, "显示配置路径")

	flag.Parse()

	// 显示版本
	if *showVersion {
		log.Printf("DBM v%s (commit: %s)", version, commit)
		return
	}

	// 显示配置路径
	if *showConfig {
		log.Println(getConfigPath())
		return
	}

	// 初始化配置
	cfg, err := initConfig(*configPath, *dataPath)
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	// 创建数据目录
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	// 生成或加载加密密钥
	encryptionKey, err := getOrGenerateEncryptionKey(cfg)
	if err != nil {
		log.Fatalf("初始化加密密钥失败: %v", err)
	}

	// 创建连接管理器
	connManager, err := connection.NewManager(cfg.DataDir, encryptionKey)
	if err != nil {
		log.Fatalf("创建连接管理器失败: %v", err)
	}
	defer connManager.Close()

	// 获取前端文件系统
	staticFS := assets.FS()

	// 创建并启动服务器
	srv := server.NewServer(connManager, http.FS(staticFS))

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("DBM v%s 启动中...", version)
	log.Printf("监听地址: http://%s", addr)
	log.Printf("配置目录: %s", getConfigPath())
	log.Printf("数据目录: %s", cfg.DataDir)

	if err := srv.Run(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// Config 配置
type Config struct {
	DataDir string
}

// initConfig 初始化配置
func initConfig(configPath, dataPath string) (*Config, error) {
	cfg := &Config{}

	// 数据目录
	if dataPath != "" {
		cfg.DataDir = dataPath
	} else {
		homeDir, _ := os.UserHomeDir()
		cfg.DataDir = filepath.Join(homeDir, ".dbm")
	}

	return cfg, nil
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".dbm", "config.yaml")
}

// getOrGenerateEncryptionKey 获取或生成加密密钥
func getOrGenerateEncryptionKey(cfg *Config) (string, error) {
	keyFile := filepath.Join(cfg.DataDir, ".key")

	// 尝试读取现有密钥
	if data, err := os.ReadFile(keyFile); err == nil {
		return string(data), nil
	}

	// 生成新密钥
	key := uuid.New().String()

	// 保存密钥
	if err := os.WriteFile(keyFile, []byte(key), 0600); err != nil {
		return "", err
	}

	return key, nil
}
