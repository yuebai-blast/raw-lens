# GitHub 发布流程

本文档描述 raw-lens 的标准 CI 与发布流程。仓库使用 GitHub Actions 自动运行检查，并在推送语义化版本 tag 时构建并推送 Docker 镜像。**本项目只发布 Docker 镜像这一种交付物**（不出独立二进制、无 GoReleaser）。

## 分支与版本约定

- 日常开发从 `main` 切 feature/fix 分支。
- Pull Request 合并前必须通过 CI。
- 发布版本使用语义化 tag：`vMAJOR.MINOR.PATCH`，例如 `v0.3.0`。
- 预发布版本使用带后缀的 tag：`vMAJOR.MINOR.PATCH-rc.N`，例如 `v0.3.0-rc.1`（带 `-` 即视为预发布，不会打 `latest`）。

## CI 流程

CI 由 `.github/workflows/ci.yml` 定义，在向 `main` 推送代码、创建或更新 Pull Request 时运行。拆成三个并行 job：

- **backend**：`gofmt` 检查、`go test -race ./...`（本项目并发处理连接、共享 store，开竞态检测）、`go vet ./...`、`go build ./...`。
- **frontend**：`pnpm typecheck`、`pnpm lint`、`pnpm test`、`pnpm build`。
- **image-build**：`docker build` 只构建不推送（单架构），合并前提前暴露 Dockerfile 损坏。

本地提交前建议运行：

```bash
gofmt -w .
go test -race ./...
go vet ./...
mise run test-web   # 前端 typecheck + lint + test
mise run build      # 前端 pnpm build + 编译二进制
mise run image      # 本地验证 Dockerfile（单架构）
```

## 发布流程

发布由 `.github/workflows/image.yml` 定义，在推送 `v*.*.*` tag 时运行，用 buildx 构建 `linux/amd64,linux/arm64` 多架构镜像并推到 GHCR：`ghcr.io/yuebai-blast/raw-lens`。

### 1. 准备

确认工作区干净，并从主分支拉取最新代码：

```bash
git checkout main
git pull --ff-only
git status --short
```

### 2. 本地验证

```bash
gofmt -w .
go test -race ./...
go vet ./...
mise run test-web
mise run image      # 本地构建镜像，验证 Dockerfile
```

如果本次发布包含前端面板改动，手动启动并验证面板（`mise run api` + `mise run webui`，面板地址 `http://localhost:5173` 开发态、生产态在 `:9090`）。

### 3. 创建并推送 tag

```bash
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

推送 tag 后，GitHub Actions 会自动：

1. 用 buildx 构建 `linux/amd64,linux/arm64` 多架构镜像。
2. 推送到 `ghcr.io/yuebai-blast/raw-lens`，打上 `{{version}}`、`{{major}}.{{minor}}` 等 tag。
3. 仅当 tag 非预发布（名字不含 `-`）时额外打 `latest`。

> 也可在 GitHub Actions 页面手动 `workflow_dispatch` 触发 `image.yml`（重推/回滚验证用）；日常 push `main` 不触发。

### 4. 发布后检查

- 在仓库 Packages 页面确认新镜像 tag 已出现。
- 拉取镜像并启动一次，确认可正常运行：
  ```bash
  docker run --rm -p 8080:8080 -p 9090:9090 ghcr.io/yuebai-blast/raw-lens:v0.3.0
  ```
- 首次发布后镜像在 GitHub Packages 默认 private，如需公开拉取，到仓库 Packages 设置里改为 public。

### 5. 回滚或撤回

如果 tag 已推送但镜像有问题，优先发布修复版本，例如 `v0.3.1`。

只有在错误 tag 尚未被用户使用时，才考虑删除远端 tag：

```bash
git tag -d v0.3.0
git push origin :refs/tags/v0.3.0
```

删除 tag 前应在团队内确认，避免破坏已经引用该版本的部署流程；GHCR 上已推送的镜像 tag 需另行在 Packages 页面删除。
