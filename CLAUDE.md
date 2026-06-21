# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

raw-lens 是一个**保真**的 HTTP 请求观察工具：直接监听裸 TCP/TLS，不经过 `net/http` 路由层，逐字节读 socket，因此能保留客户端真正发出的 header 顺序、大小写、重复项和原始 body。详细用法见 `README.md`。

## 常用命令

工具版本由 `mise.toml` 固定（Go 1.26.4、Node 20、pnpm 9），优先用 mise 任务：

```bash
mise run install       # 安装全部依赖（go mod download + pnpm install）
mise run api           # 本地启动后端（= go run ./cmd/rawlens，监听 :8080/:9090）
mise run webui         # 本地启动 Vite 开发服务器（HMR，/api 代理到 :9090）
mise run build         # 前端 pnpm build 后构建本机二进制到 bin/rawlens
mise run build-linux   # 交叉编译 linux/amd64 部署产物（CGO_ENABLED=0）
mise run test-web      # 前端质量检查（typecheck + lint + vitest）
mise run image         # 本地构建 Docker 镜像 rawlens:local（单架构，验证 Dockerfile）
go test ./...          # 运行全部 Go 测试
go test ./internal/capture -run TestXxx   # 跑单个 Go 测试
go vet ./...           # CI 会跑
gofmt -l .             # CI 用它判定格式，有输出即失败
```

本地开发双进程流程：先 `mise run api` 启动后端，再 `mise run webui` 启动 Vite（`http://localhost:5173`），Vite 会把 `/api/*` 代理到后端 `:9090`，前端改动 HMR 即时生效，无需重启后端。

提交前确保 `gofmt -l .` 无输出、`go vet ./...` 和 `go test ./...` 通过——CI（`.github/workflows/ci.yml`）会逐项检查，外加前端 typecheck/lint/test、`goreleaser check`，以及 `image-build` job（PR 阶段 `docker build` 只构建不推送，提前暴露 Dockerfile 损坏）。

## 架构要点

**双端口、单进程**（`cmd/rawlens/main.go`）：抓包端口（默认 `:8080`，对外暴露）和面板端口（默认 `:9090`，仅内网/本机）分开，避免打开面板页面本身的请求污染抓包列表。两个 server 共享同一个 `store.Store`。

依赖方向：`capture → store`、`dashboard → store + web`、`main → config + 三者`。

**保真的核心在 `internal/capture/capture.go`**——这是为什么不用 `net/http`：标准库会规范化 header（首字母大写、排序、去重）。这里自己实现：
- `readUntilHeaderEnd` 读到 `\r\n\r\n`（容忍裸 `\n\n`），1 MiB 上限；
- `bodyLength` 大小写不敏感地读 `Content-Length` / `Transfer-Encoding: chunked`；
- `readChunked` 连同分块框架字节一起原样保存；
- `parseCaptured` 解析出结构化字段时**保留原始顺序和大小写**，header 值只去掉冒号后惯例的一个前导空格。
- 改动这里时务必保持「字节保真」这个不变量，不要顺手做任何规范化。

`store.CapturedRequest` 同时保存 `Raw`（全量原始字节）和解析后的结构化字段（`Headers [][2]string` 保序）。`Store` 以 SQLite 持久化（驱动 `modernc.org/sqlite`，纯 Go、保 `CGO_ENABLED=0`），默认落盘到 `rawlens.db`，按条数保留最近 `max` 条（超出删最旧）；`store.path` 配 `":memory:"` 可恢复进程重启即清空的行为。

**TLS 抓包**（`internal/capture/tls.go`）：`tls.NewListener` 包一层后，握手由 raw-lens 终结，从 `tls.Conn` 读到的是**解密后的明文字节**，所以下游解析逻辑完全复用，无需区分 HTTP/HTTPS。`cert`/`key` 留空时生成内存自签名证书。

**连接模型**：每条连接处理一条请求，响应带 `Connection: close`，不做 keep-alive 复用。读超时 30s。

**前端架构**（`frontend/` → `web/dist/`）：前端是 Vue 3 + TypeScript + Vite + Pinia + Vue Router 项目，源码在 `frontend/src/`（App.vue、router、stores/captures.ts、components/、views/、utils/、types/、styles/global.css）。`pnpm build`（即 `mise run build` 的前半段）将产物输出到 `web/dist/`，再由 `web/embed.go` 的 `//go:embed all:dist` 编进 Go 二进制；`web/dist/.keep` 已提交，保证不构建前端时 `go build/test/vet ./...` 也能通过。`dashboard.go` 提供 JSON API（`GET /api/requests`、`GET /api/requests/{id}`、`POST /api/clear`）+ SPA fallback（非 API、非静态文件路径一律返回 index.html，使 `/r/:id` 刷新可用）；`Raw`/`Body` 经 base64 传给前端，前端提供 RAW/HEADERS/BODY/HEX 四视图。改前端后需重新 `mise run build` 才能更新内嵌产物；日常开发用双进程流程（见"常用命令"）。

**发布与镜像**：两条独立的 tag 触发流水线，均由 `v*.*.*` tag 触发、互不干扰。`release.yml`（goreleaser）出各平台二进制与 GitHub Release；`image.yml` 用 buildx 构建 `linux/amd64,arm64` 多架构镜像推到 `ghcr.io/yuebai-blast/raw-lens`（登录用内置 `GITHUB_TOKEN`，`latest` 仅 tag 触发且非预发布才打）。镜像由仓库根 `Dockerfile` 两阶段构建，**工具链版本只认根 `mise.toml`，不在 Dockerfile 里重复钉**（见全局规范 `monorepo-docker-build.md`）：构建阶段用干净的 `debian:12-slim` 底座，按官方写法 `curl https://mise.run | sh` 装进 mise，再 `mise install go node pnpm` 照 `mise.toml` 装工具链，依次跑 `pnpm build` 出前端、`CGO_ENABLED=0` 纯 Go 编译把 `web/dist` embed 进单二进制（embed 机制同上）→ 运行阶段 `distroless/static` 非 root（uid 65532）只带二进制，`WORKDIR /data`、`rawlens.db` 落该目录。**禁止**改回 `node:`/`golang:` 这类带运行时的基础镜像（会把版本钉死、绕过 `mise.toml` 单一来源）；Dockerfile 里唯一允许钉死的版本是 mise 自身的 `MISE_VERSION`（自举工具无法由 mise 管），升级 mise 时同步它。改 Dockerfile 后 `mise run image` 本地验证；PR 阶段 `ci.yml` 的 `image-build` job 会兜底。本项目**不做服务器自动部署**。

## 配置

所有运行时配置走 `config.yaml`（`internal/config/config.go`：文件不存在用内置默认值，存在则字段覆盖）。`-config /path/to.yaml` 是唯一的命令行 flag。部署只需二进制 + 一个 yaml。仓库里只提交模板 `config.example.yaml`，实际的 `config.yaml` 由各环境从模板复制而来、已在 `.gitignore` 中不入库（程序默认仍读 `config.yaml`，`config.go` 的 `DefaultPath` 不变）。
