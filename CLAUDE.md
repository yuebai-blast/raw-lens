# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

raw-lens 是一个**保真**的 HTTP 请求观察工具：直接监听裸 TCP/TLS，不经过 `net/http` 路由层，逐字节读 socket，因此能保留客户端真正发出的 header 顺序、大小写、重复项和原始 body。详细用法见 `README.md`，贡献规范见 `AGENTS.md`。

## 常用命令

工具版本由 `mise.toml` 固定（Go 1.26.4），优先用 mise 任务：

```bash
mise run run           # 本地运行（= go run ./cmd/rawlens，读 ./config.yaml）
mise run build         # 构建本机二进制到 bin/rawlens
mise run build-linux   # 交叉编译 linux/amd64 部署产物（CGO_ENABLED=0）
go test ./...          # 运行全部测试
go test ./internal/capture -run TestXxx   # 跑单个测试
go vet ./...           # CI 会跑
gofmt -l .             # CI 用它判定格式，有输出即失败
```

提交前确保 `gofmt -l .` 无输出、`go vet ./...` 和 `go test ./...` 通过——CI（`.github/workflows/ci.yml`）会逐项检查，外加 `goreleaser check`。

## 架构要点

**双端口、单进程**（`cmd/rawlens/main.go`）：抓包端口（默认 `:8080`，对外暴露）和面板端口（默认 `:9090`，仅内网/本机）分开，避免打开面板页面本身的请求污染抓包列表。两个 server 共享同一个 `store.Store`。

依赖方向：`capture → store`、`dashboard → store + web`、`main → config + 三者`。

**保真的核心在 `internal/capture/capture.go`**——这是为什么不用 `net/http`：标准库会规范化 header（首字母大写、排序、去重）。这里自己实现：
- `readUntilHeaderEnd` 读到 `\r\n\r\n`（容忍裸 `\n\n`），1 MiB 上限；
- `bodyLength` 大小写不敏感地读 `Content-Length` / `Transfer-Encoding: chunked`；
- `readChunked` 连同分块框架字节一起原样保存；
- `parseCaptured` 解析出结构化字段时**保留原始顺序和大小写**，header 值只去掉冒号后惯例的一个前导空格。
- 改动这里时务必保持「字节保真」这个不变量，不要顺手做任何规范化。

`store.CapturedRequest` 同时保存 `Raw`（全量原始字节）和解析后的结构化字段（`Headers [][2]string` 保序）。`Store` 是带锁的环形缓冲（`store.Max` 条上限，溢出丢最旧），数据仅在内存，进程重启即清空。

**TLS 抓包**（`internal/capture/tls.go`）：`tls.NewListener` 包一层后，握手由 raw-lens 终结，从 `tls.Conn` 读到的是**解密后的明文字节**，所以下游解析逻辑完全复用，无需区分 HTTP/HTTPS。`cert`/`key` 留空时生成内存自签名证书。

**连接模型**：每条连接处理一条请求，响应带 `Connection: close`，不做 keep-alive 复用。读超时 30s。

**前端零构建**（`web/`）：原生 HTML/CSS/JS，通过 `web/embed.go` 的 `go:embed` 编进二进制，部署无需额外文件。`dashboard.go` 提供 JSON API（`GET /api/requests`、`GET /api/requests/{id}`、`POST /api/clear`）并托管静态资源；`Raw`/`Body` 经 base64 传给前端，前端提供 RAW/HEADERS/BODY/HEX 四视图。改完前端直接重新编译即可。

## 配置

所有运行时配置走 `config.yaml`（`internal/config/config.go`：文件不存在用内置默认值，存在则字段覆盖）。`-config /path/to.yaml` 是唯一的命令行 flag。部署只需二进制 + 一个 yaml。
