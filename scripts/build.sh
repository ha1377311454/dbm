#!/bin/bash

# DBM 构建脚本

set -e

VERSION=${VERSION:-dev}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  DBM Build Script${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# 检查依赖
echo -e "${YELLOW}检查依赖...${NC}"
command -v go >/dev/null 2>&1 || { echo -e "${RED}错误: Go 未安装${NC}" >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo -e "${RED}错误: Node.js/npm 未安装${NC}" >&2; exit 1; }
echo -e "${GREEN}✓ 依赖检查通过${NC}"
echo ""

# 构建前端
echo -e "${YELLOW}构建前端...${NC}"
cd web
npm install
npm run build
cd ..
echo -e "${GREEN}✓ 前端构建完成${NC}"
echo ""

# 构建后端
echo -e "${YELLOW}构建后端...${NC}"
mkdir -p dist

LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT}"

# 构建各平台版本
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"

    OUTPUT_NAME="dbm-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo -e "${YELLOW}构建 ${PLATFORM}...${NC}"

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "${LDFLAGS}" \
        -o "dist/${OUTPUT_NAME}" \
        ./cmd/dbm

    if [ $? -ne 0 ]; then
        echo -e "${RED}构建失败: ${PLATFORM}${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✓ 后端构建完成${NC}"
echo ""

# 计算校验和
echo -e "${YELLOW}计算校验和...${NC}"
cd dist
shasum -a 256 dbm-* > checksums.txt || true
cd ..
echo -e "${GREEN}✓ 校验和生成完成${NC}"
echo ""

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  构建完成！${NC}"
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}输出目录: ./dist/${NC}"
echo ""
