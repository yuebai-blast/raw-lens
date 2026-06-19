# raw-lens

抓 HTTP 请求的**原始字节**——header 的顺序、大小写、重复项，以及 body，全部保真展示。

普通 HTTP 框架（Go `net/http`、Flask、Express）都会把 header 规范化：首字母大写、排序、合并重复项。
raw-lens 不走框架，直接监听裸 TCP，自己读 socket 字节、自己找 header 边界、按 `Content-Length` / `Transfer-Encoding: chunked` 读 body，所以你看到的就是客户端真正发出来的样子。

## 两个端口

| 端口 | 用途 |
|------|------|
| `:8080`（抓包） | **对外暴露这个**。客户端把请求发到这里，会被原样记录，并回一个最小的 `200`。 |
| `:9090`（面板） | 浏览器打开 `http://<host>:9090` 看抓到的请求。不要对公网暴露。 |

两个端口分开，是为了让你打开面板页面本身的请求不会污染抓包列表。

## 本地开发

```bash
mise run install      # 安装全部依赖（go mod download + pnpm install）
```

然后开**两个终端**：

```bash
# 终端 1：后端（监听 :8080 抓包、:9090 面板 API）
mise run api

# 终端 2：Vite 开发服务器（HMR，/api 代理到 :9090）
mise run webui
# 然后浏览器开 http://localhost:5173
```

向抓包端口发请求测试：

```bash
curl -X POST localhost:8080/hi -d 'hello'
```

## 配置

所有运行时配置走 `config.yaml`（启动时默认读当前目录的 `config.yaml`，或用 `-config /path/to.yaml` 指定）。文件不存在时使用内置默认值，只需写你想改的字段。

```yaml
capture:
  addr: ":8080"        # 裸 TCP 抓包监听地址，对外暴露这个端口
  tls:
    enabled: false     # 开启后抓包端口走 TLS，可抓 HTTPS 原始请求
    cert: ""           # 证书路径；留空则用内存自签名证书
    key: ""            # 私钥路径
dashboard:
  addr: ":9090"        # 前端面板监听地址，建议只在内网/本机访问
store:
  max: 500             # 最多保留多少条请求（超出删最旧）
  path: "rawlens.db"  # SQLite 文件路径，相对启动目录；":memory:" 则重启即清空
```

`-config` 是唯一的命令行 flag。请求默认落盘到 `rawlens.db` 持久化，进程重启后记录不丢失；如需重启即清空的旧行为，将 `store.path` 配为 `":memory:"`。

## 抓 HTTPS 原始请求

在 `config.yaml` 里把 `capture.tls.enabled` 设为 `true`。TLS 握手由 raw-lens 终结，握手后读到的是**解密后的明文字节**——和抓 HTTP 一样保真（顺序、大小写、重复 header 全保留）。

```yaml
capture:
  tls:
    enabled: true
    cert: ""   # 留空 = 内存自签名证书（测试用，客户端需 curl -k）
    # cert: /etc/letsencrypt/live/example.com/fullchain.pem   # 或用真证书
    # key:  /etc/letsencrypt/live/example.com/privkey.pem
```

```bash
mise run api                                   # 按上面的 config.yaml 启动后端
curl -k https://localhost:8080/secure -d hi    # -k 跳过自签名校验
```

`cert`/`key` 留空时自动生成一张内存自签名证书（含 `localhost` / `127.0.0.1`），仅供测试。客户端会报证书不可信，用 `curl -k` 或浏览器手动信任即可。

## 部署到服务器

先在本机构建，再上传：

```bash
mise run build              # 先构建前端（pnpm build），再编译 Go 二进制到 bin/rawlens
mise run build-linux        # 或：交叉编译 linux/amd64 产物 bin/rawlens-linux-amd64
scp bin/rawlens-linux-amd64 user@server:/usr/local/bin/rawlens
scp config.yaml user@server:/etc/rawlens/config.yaml
ssh user@server 'rawlens -config /etc/rawlens/config.yaml'   # 或配 systemd
```

部署只需二进制 + 一个 yaml（前端已通过 `go:embed` 编进二进制，无需额外文件）。运行时会在启动目录生成 `rawlens.db`（SQLite 持久化文件），可通过 `store.path` 指定路径或设为 `":memory:"` 改为内存模式。
防火墙放开 8080 给客户端；面板端口 9090 建议只在本机/内网访问（或用 SSH 隧道：`ssh -L 9090:localhost:9090 user@server`）。

## 目录结构

标准 Go 布局，前后端分离：

```
config.yaml                  运行时配置（端口 / TLS / 容量），外置可编辑
cmd/rawlens/main.go          入口：加载配置、起两个 server
internal/
  config/config.go           YAML 配置加载（默认值 + 文件覆盖）
  store/store.go             SQLite 持久化存储 + CapturedRequest 类型
  capture/capture.go         裸 TCP 抓包 + 原始解析（保真的核心）
  capture/tls.go             TLS 配置 / 自签名证书生成
  dashboard/dashboard.go     面板 JSON API + SPA fallback
frontend/                    前端源码（Vue 3 + TS + Vite + Pinia + Router）
  src/                       App.vue、router、stores、components、views、utils
web/
  embed.go                   //go:embed all:dist，把 dist/ 编进二进制
  dist/                      pnpm build 产物（gitignored，dist/.keep 已提交）
```

后端依赖方向：`capture → store`、`dashboard → store, web`、`main → config + 三者`。
配置外置（`config.yaml`），前端经 `go:embed` 内嵌——部署只需二进制 + 一个 yaml。

## 已知边界

- 每条连接处理一条请求（响应带 `Connection: close`），不做 keep-alive 复用。curl / 浏览器会自动开新连接，不影响使用。
- chunked body 按原始分块字节保存（含分块框架），保真优先。
- header 上限 1 MiB，超出报错但已收到的字节仍会保存。
