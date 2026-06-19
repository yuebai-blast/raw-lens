// Package web 把前端构建产物（Vite 输出到 dist/）编进二进制，部署时无需额外文件。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedFS embed.FS

// DistFS 返回 dist 子目录的文件系统。未构建前端时 dist 仅含占位文件，
// 此时打开 index.html 会失败，由 dashboard 层优雅提示而非 panic。
func DistFS() (fs.FS, error) {
	return fs.Sub(embedFS, "dist")
}
