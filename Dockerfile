# syntax=docker/dockerfile:1

# ---- 构建阶段：干净 slim 底座 + curl 装 mise，工具链版本全由根 mise.toml 决定 ----
# 不用 node:/golang: 这类带运行时的基础镜像（会把版本钉死在 Dockerfile，绕过 mise.toml 单一来源）
FROM debian:12-slim AS build

# mise 自身版本在此钉死——自举工具无法由 mise.toml 管，这是 Dockerfile 里唯一允许的版本钉死
ENV MISE_VERSION=v2026.6.0
ENV MISE_DATA_DIR=/mise MISE_CONFIG_DIR=/mise MISE_CACHE_DIR=/mise/cache \
    MISE_INSTALL_PATH=/usr/local/bin/mise PATH=/mise/shims:$PATH
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl git ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && curl https://mise.run | sh

WORKDIR /src
# 工具链版本唯一来源：根 mise.toml；只装本镜像需要的 go/node/pnpm，不 `mise install` 全装
COPY mise.toml ./
RUN mise trust && mise install go node pnpm

# 层缓存：依赖清单先于源码 COPY，清单不变则不重装（见 monorepo-docker-build.md §6）
# 1) 前端依赖（pnpm-lock.yaml 不变则命中缓存）
COPY frontend/package.json frontend/pnpm-lock.yaml ./frontend/
RUN pnpm -C frontend install --frozen-lockfile
# 2) Go 依赖（go.mod/go.sum 不变则命中缓存）
COPY go.mod go.sum ./
RUN go mod download

# 3) 拷源码：先前端 build 出 web/dist，再 CGO_ENABLED=0 纯 Go 编译把它 embed 进单二进制
COPY . .
RUN pnpm -C frontend build \
 && mkdir -p /data \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rawlens ./cmd/rawlens

# ---- 运行阶段：distroless 非 root，只带二进制，构建期的 mise/工具链都不进最终镜像 ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rawlens /usr/local/bin/rawlens
# /data 归非 root 用户所有，rawlens.db 才可写
COPY --from=build --chown=65532:65532 /data /data
WORKDIR /data
USER 65532:65532
# EXPOSE 仅为元数据/文档，不发布也不绑定端口；这里标的是内置默认端口
# （capture :8080 / dashboard :9090）。实际监听端口由 config.yaml 决定，
# 改了配置请以配置为准，并自行用 docker run -p 映射对应端口。
EXPOSE 8080 9090
# 默认无 config.yaml → 内置默认值；自定义配置挂到 /data/config.yaml 即被读取
ENTRYPOINT ["/usr/local/bin/rawlens"]
