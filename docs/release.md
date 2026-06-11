# GitHub 发布流程

本文档描述 raw-lens 的标准 CI 与发布流程。仓库使用 GitHub Actions 自动运行检查，并通过 GoReleaser 在推送语义化版本 tag 时生成 GitHub Release。

## 分支与版本约定

- 日常开发从 `main` 切 feature/fix 分支。
- Pull Request 合并前必须通过 CI。
- 发布版本使用语义化 tag：`vMAJOR.MINOR.PATCH`，例如 `v0.3.0`。
- 预发布版本使用带后缀的 tag：`vMAJOR.MINOR.PATCH-rc.N`，例如 `v0.3.0-rc.1`。

## CI 流程

CI 由 `.github/workflows/ci.yml` 定义，在以下场景运行：

- 向 `main` 推送代码。
- 创建或更新 Pull Request。

CI 执行内容：

1. Checkout 代码。
2. 安装 Go `1.26.4`。
3. 检查 `gofmt`。
4. 下载 Go modules。
5. 运行 `go test ./...`。
6. 运行 `go vet ./...`。
7. 构建本机二进制：`go build -o bin/rawlens ./cmd/rawlens`。
8. 校验 GoReleaser 配置：`goreleaser check`。

本地提交前建议运行：

```bash
gofmt -w .
go test ./...
go vet ./...
mise run build
```

## 发布流程

发布由 `.github/workflows/release.yml` 和 `.goreleaser.yaml` 定义，在推送 `v*.*.*` tag 时运行。

### 1. 准备发布分支

确认工作区干净，并从主分支拉取最新代码：

```bash
git checkout main
git pull --ff-only
git status --short
```

### 2. 本地验证

```bash
gofmt -w .
go test ./...
go vet ./...
mise run build
```

如果本次发布包含前端面板改动，手动启动并验证面板：

```bash
mise run run
```

默认面板地址是 `http://localhost:9090`，抓包端口默认是 `:8080`。

### 3. 创建并推送 tag

```bash
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

推送 tag 后，GitHub Actions 会自动：

1. 使用 GoReleaser 为 Linux、macOS、Windows 构建二进制。
2. 打包 `.tar.gz` 或 `.zip`。
3. 生成 `checksums.txt`。
4. 根据提交历史生成 Release notes。
5. 创建 GitHub Release 并上传产物。

### 4. 发布后检查

在 GitHub Release 页面确认：

- Release notes 已生成。
- 所有平台产物存在。
- `checksums.txt` 存在。
- 下载目标平台产物后可以正常启动。

### 5. 回滚或撤回

如果 tag 已推送但发布有问题，优先发布修复版本，例如 `v0.3.1`。

只有在错误 tag 尚未被用户使用时，才考虑删除远端 tag：

```bash
git tag -d v0.3.0
git push origin :refs/tags/v0.3.0
```

删除 tag 前应在团队内确认，避免破坏已经引用该版本的部署流程。
