# Repository Guidelines

## 项目结构与模块组织

本仓库是一个 Go 实现的原始 HTTP 请求观察工具，前端为零构建的原生 HTML/CSS/JS，并通过 `go:embed` 编进二进制。

- `cmd/rawlens/main.go`：程序入口，加载配置并启动抓包端口与面板端口。
- `internal/config/`：`config.yaml` 的默认值、读取与覆盖逻辑。
- `internal/capture/`：裸 TCP/TLS 请求读取与原始字节解析，是保真抓包核心。
- `internal/store/`：内存环形存储与 `CapturedRequest` 数据结构。
- `internal/dashboard/`：面板 API 与静态资源服务。
- `web/`：嵌入式前端资源，包含 `index.html`、`styles.css`、`app.js`、`embed.go`。
- `bin/`：本地构建产物目录，不应作为源码修改入口。

## 构建、测试与本地开发命令

优先使用 `mise` 任务，工具版本由 `mise.toml` 固定。

```bash
mise run run           # 本地运行，等价于 go run ./cmd/rawlens
mise run build         # 构建本机二进制到 bin/rawlens
mise run build-linux   # 构建 linux/amd64 部署产物
go test ./...          # 运行全部 Go 测试
```

本地启动后，抓包端口默认是 `:8080`，面板默认是 `http://localhost:9090`。配置改动优先放在 `config.yaml`，或通过 `-config /path/to.yaml` 指定文件。

## 代码风格与命名约定

Go 代码必须通过 `gofmt` 格式化，包名使用简短小写名，文件名使用小写单词或下划线。导出类型、函数和字段使用 `PascalCase`，未导出标识符使用 `camelCase`。前端保持原生 HTML/CSS/JS，不引入构建链，除非有明确需求。

## 测试指南

当前仓库没有专门测试文件；新增后端逻辑时应添加 Go 单元测试，文件命名为 `*_test.go`，测试函数命名为 `TestXxx`。对 `capture`、`config`、`store` 的改动应覆盖边界输入，例如重复 header、chunked body、配置缺省值和环形缓冲溢出。

## 提交与 Pull Request 规范

提交历史使用类似 Conventional Commits 的格式，例如 `chore(gitignore): 添加 .idea/ 目录到 .gitignore`。建议继续使用 `type(scope): 简短说明`，常见类型包括 `feat`、`fix`、`test`、`docs`、`chore`。

PR 应说明变更目的、主要实现、验证命令；涉及前端面板时附截图或录屏；涉及协议解析、TLS、配置兼容性时列出手动验证样例。

## 安全与配置提示

不要把真实私钥、证书或生产配置提交到仓库。`dashboard.addr` 不建议直接暴露到公网；部署时只开放抓包端口，并通过内网或 SSH 隧道访问面板。
