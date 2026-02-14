.PHONY: build dev clean test lint install-web-deps run

# 版本信息
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

# 目录
WEB_DIR := web
DIST_DIR := dist
CMD_DIR := cmd/dbm

# 默认目标
.DEFAULT_GOAL := build

## dev: 启动开发模式（后端热重载）
dev:
	@echo "启动开发服务器..."
	@go run $(CMD_DIR)/main.go --host 0.0.0.0 --port 8080

## dev-web: 启动前端开发服务器
dev-web:
	@echo "启动前端开发服务器..."
	@cd $(WEB_DIR) && npm run dev

## install-web-deps: 安装前端依赖
install-web-deps:
	@echo "安装前端依赖..."
	@cd $(WEB_DIR) && npm install

## build-web: 构建前端
build-web: install-web-deps
	@echo "构建前端..."
	@cd $(WEB_DIR) && npm run build
	@echo "复制前端资源..."
	@mkdir -p internal/assets/web/dist
	@cp -r $(WEB_DIR)/dist/* internal/assets/web/dist/

## build: 构建所有平台版本
build: clean build-web
	@echo "构建后端..."
	@mkdir -p $(DIST_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm $(CMD_DIR)/main.go
	@echo "构建完成: $(DIST_DIR)/dbm"

## build-linux: 构建 Linux 版本
build-linux: build-web
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm-linux-amd64 $(CMD_DIR)/main.go
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm-linux-arm64 $(CMD_DIR)/main.go

## build-darwin: 构建 macOS 版本
build-darwin: build-web
	@mkdir -p $(DIST_DIR)
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm-darwin-amd64 $(CMD_DIR)/main.go
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm-darwin-arm64 $(CMD_DIR)/main.go

## build-windows: 构建 Windows 版本
build-windows: build-web
	@mkdir -p $(DIST_DIR)
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/dbm-windows-amd64.exe $(CMD_DIR)/main.go

## build-all: 构建所有平台
build-all: build-linux build-darwin build-windows
	@echo "生成校验和..."
	@cd $(DIST_DIR) && shasum -a 256 dbm-* > checksums.txt 2>/dev/null || true

## test: 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

## lint: 代码检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run ./... || true

## clean: 清理构建产物
clean:
	@echo "清理..."
	@rm -rf $(DIST_DIR)
	@rm -rf $(WEB_DIR)/dist
	@rm -rf $(WEB_DIR)/node_modules
	@rm -rf internal/assets/web/dist/*
	@mkdir -p internal/assets/web/dist

## run: 运行构建后的程序
run: build
	@echo "运行 DBM..."
	@$(DIST_DIR)/dbm

## help: 显示帮助信息
help:
	@echo "可用命令:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
