# syntax=docker/dockerfile:1

# 阶段 1：构建前端 → /src/web/dist（vite outDir = ../web/dist）
FROM node:20.18.1-bookworm-slim AS web
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate
# 先装依赖，利用层缓存：lockfile 不变则不重装
COPY frontend/package.json frontend/pnpm-lock.yaml ./frontend/
RUN pnpm -C frontend install --frozen-lockfile
COPY frontend/ ./frontend/
RUN pnpm -C frontend build

# 阶段 2：编译内嵌前端产物的单二进制（CGO_ENABLED=0 纯 Go）
FROM golang:1.26.4-bookworm AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 用阶段 1 的真实前端产物覆盖占位 dist，保证 embed 拿到完整面板
COPY --from=web /src/web/dist ./web/dist
# 顺便建好运行时数据目录，供下一阶段带属主拷贝
RUN mkdir -p /data \
 && go build -trimpath -ldflags="-s -w" -o /out/rawlens ./cmd/rawlens

# 阶段 3：distroless 非 root 运行
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/rawlens /usr/local/bin/rawlens
# /data 归非 root 用户所有，rawlens.db 才可写
COPY --from=build --chown=65532:65532 /data /data
WORKDIR /data
USER 65532:65532
EXPOSE 8080 9090
# 默认无 config.yaml → 内置默认值；自定义配置挂到 /data/config.yaml 即被读取
ENTRYPOINT ["/usr/local/bin/rawlens"]
