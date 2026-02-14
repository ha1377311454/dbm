package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var webFS embed.FS

// FS 返回嵌入的前端文件系统
func FS() fs.FS {
	sub, _ := fs.Sub(webFS, "web/dist")
	return sub
}