// Package web 把前端静态资源编进二进制，部署时无需额外文件。
package web

import "embed"

//go:embed index.html styles.css app.js
var FS embed.FS
