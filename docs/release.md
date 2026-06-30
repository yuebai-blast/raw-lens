# GitHub 发布流程

本文档描述 raw-lens 的标准 CI 与发布流程。仓库使用 GitHub Actions 自动运行检查，并在推送语义化版本 tag 时构建并推送 Docker 镜像、发布一个 GitHub Release。**交付物是 Docker 镜像（在 GHCR）+ 随每次发版的 GitHub Release（纯发布说明、无附件）**（不出独立二进制、无 GoReleaser）。

## 分支与版本约定

- 日常开发从 `main` 切 feature/fix 分支。
- Pull Request 合并前必须通过 CI。
- 发布版本使用语义化 tag：`vMAJOR.MINOR.PATCH`，例如 `v0.3.0`。
- 预发布版本使用带后缀的 tag：`vMAJOR.MINOR.PATCH-rc.N`，例如 `v0.3.0-rc.1`（带 `-` 即视为预发布，不会打 `latest`）。

## CI 流程

CI 由 `.github/workflows/ci.yml` 定义，在向 `main` 推送代码、创建或更新 Pull Request 时运行。两个并行 job（所有检查命令统一走 `mise run`，与本地、Dockerfile 单一来源）：

- **quality**：复用 `quality.yml`（与 `release.yml` 发版前门禁共用同一份定义），内含两个并行子 job：
  - **backend**：`mise run fmt-backend`（gofmt 校验）、`mise run test-backend`（`go test -race ./...`，本项目并发处理连接、共享 store，开竞态检测）、`mise run vet-backend`、`mise run build-backend`。
  - **frontend**：`mise run install-frontend --frozen`、`mise run test-frontend`（typecheck + lint + vitest）、`mise run build-frontend`。
- **image-build**：`docker build` 只构建不推送（单架构），合并前提前暴露 Dockerfile 损坏。

本地提交前建议运行：

```bash
gofmt -w .            # 先把格式问题就地修掉（fmt-backend 只校验不修改）
mise run fmt-backend  # gofmt 校验
mise run vet-backend  # go vet
mise run test-backend # go test -race ./...
mise run test-frontend # 前端 typecheck + lint + test
mise run build        # 前端构建 + 编译二进制
mise run image        # 本地验证 Dockerfile（单架构）
```

## 发布流程

发布由 `.github/workflows/release.yml`（workflow `name: Release`）定义，在推送 `v*.*.*` tag 时运行，三个 job 串联：`verify`（复用 `quality.yml` 做发版前质量门禁，不过不构建）→ `image`（用 buildx 构建**单架构** `linux/amd64` 镜像并推到 GHCR：`ghcr.io/yuebai-blast/raw-lens`）→ `release`（发 GitHub Release：annotated tag 注释置顶 + 按提交类型过滤的变更日志）。

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
mise run fmt-backend
mise run vet-backend
mise run test-backend
mise run test-frontend
mise run image      # 本地构建镜像，验证 Dockerfile
```

如果本次发布包含前端面板改动，手动启动并验证面板（`mise run api` + `mise run webui`，面板地址 `http://localhost:5173` 开发态、生产态在 `:9101`）。

### 3. 发版（打 tag 并推送）

发版动作收口在 `mise run release`，它会做发版前校验（在 main、工作区干净、tag 不重复、main 不落后 origin），再打 tag + 先推 main 后推 tag：

```bash
mise run release                     # 版本号省略 → 取最新稳定 tag 的 patch+1，无说明（轻量 tag）
mise run release "更新说明"           # patch+1 且带说明（→ annotated tag，注释会被置顶到 Release 正文）
mise run release v0.4.0              # 显式指定版本号（升 minor/major 或发预发布时用）
mise run release v0.4.0 "更新说明"    # 指定版本号 + 说明
```

推送 tag 后，GitHub Actions 会自动：

1. `verify` 跑后端 + 前端质量门禁，不过则整个发版中止。
2. `image` 用 buildx 构建**单架构** `linux/amd64` 镜像，推送到 `ghcr.io/yuebai-blast/raw-lens`，标签由 `docker/metadata-action` 从 `vX.Y.Z` 剥出 `X.Y.Z`。
3. 仅当 tag 非预发布（名字不含 `-`）时由 `flavor: latest=auto` 额外打 `latest`。
4. `release` 发 GitHub Release：annotated tag 注释置顶 + 按提交类型过滤（只收 `feat`/`fix`/`perf`/`refactor`/`revert`）的变更日志。

> 也可在 GitHub Actions 页面手动 `workflow_dispatch` 触发 `release.yml`（重推/回滚验证用，无 tag 故只构建推镜像、不发 Release）；日常 push `main` 不触发。

### 4. 发布后检查

- 在仓库 Packages 页面确认新镜像 tag 已出现。
- 拉取镜像并启动一次，确认可正常运行：
  ```bash
  docker run --rm -p 9100:9100 -p 9101:9101 ghcr.io/yuebai-blast/raw-lens:v0.3.0
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
