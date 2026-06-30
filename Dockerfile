# syntax=docker/dockerfile:1

# ---- 构建阶段：干净 slim 底座 + curl 装 mise，工具链版本全由根 mise.toml 决定 ----
# 不用 node:/golang: 这类带运行时的基础镜像（会把版本钉死在 Dockerfile，绕过 mise.toml 单一来源）
FROM debian:13-slim AS build

# mise 自身版本在此钉死——自举工具无法由 mise.toml 管，这是 Dockerfile 里唯一允许的版本钉死
ENV MISE_VERSION=v2026.6.0
ENV MISE_DATA_DIR=/mise MISE_CONFIG_DIR=/mise MISE_CACHE_DIR=/mise/cache \
    MISE_INSTALL_PATH=/usr/local/bin/mise PATH=/mise/shims:$PATH
# 关掉「mise run 跑 task 前自动装全部 [tools]」：本镜像只显式装所需工具（见下方 mise install），
# 让下面的 mise run 直接用已装工具，不再额外触发安装
ENV MISE_TASK_RUN_AUTO_INSTALL=false
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl git ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && curl https://mise.run | sh

WORKDIR /src
# 工具链版本唯一来源：根 mise.toml；只装本镜像需要的 go/node/pnpm，不 `mise install` 全装
COPY mise.toml ./
RUN mise trust && mise install go node pnpm

# 层缓存：依赖清单先于源码 COPY，清单不变则不重装（见 monorepo-docker-build.md §6）。
# 安装/构建命令一律走 mise run task，与本地、CI 单一来源（不在 Dockerfile 里重抄 pnpm/go 命令）。
# 1) 前端依赖（pnpm-lock.yaml 不变则命中缓存）
COPY frontend/package.json frontend/pnpm-lock.yaml ./frontend/
RUN mise run install-frontend --frozen
# 2) Go 依赖（go.mod/go.sum 不变则命中缓存）
COPY go.mod go.sum ./
RUN mise run install-backend

# 3) 拷源码后一条 mise task 收口构建：build 依赖 build-frontend 先出 web/dist，
#    再瘦身编译（CGO_ENABLED=0 纯 Go + -trimpath -ldflags=-s -w）把 web/dist embed 进单二进制到 bin/rawlens。
#    命令与本地、CI 单一来源，不在 Dockerfile 里重抄 go 命令。
COPY . .
RUN mise run build

# ---- 运行阶段：普通 debian-slim 底座（默认 root），只带二进制 ----
# 用普通镜像而非 distroless 非 root：以 root 跑，直接可写 bind 挂载进来的目录，
# 部署时无需 chown 宿主目录、也无需在 compose 里指定 user。构建期的 mise/工具链不进这一层。
FROM debian:13-slim AS runtime
# 装 wget 供 docker-compose 的 healthcheck 探活面板端口（slim 底座默认不带 HTTP 客户端）
RUN apt-get update \
 && apt-get install -y --no-install-recommends wget \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/bin/rawlens /usr/local/bin/rawlens
# WORKDIR=/app：默认读 /app/config.yaml，db 落 /app/data/db、日志落 /app/data/logs
# （部署时把宿主 ./config.yaml、./data 分别挂到 /app/config.yaml、/app/data）。
WORKDIR /app
# EXPOSE 仅为元数据/文档，不发布也不绑定端口；这里标的是内置默认端口
# （capture :9100 / dashboard :9101）。实际监听端口由 config.yaml 决定，
# 改了配置请以配置为准，并自行用 docker run -p 映射对应端口。
EXPOSE 9100 9101
ENTRYPOINT ["/usr/local/bin/rawlens"]
