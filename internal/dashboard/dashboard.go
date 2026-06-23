// Package dashboard 提供浏览抓到的请求的前端面板和 JSON API。
package dashboard

import (
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yuebai-blast/raw-lens/internal/config"
	"github.com/yuebai-blast/raw-lens/internal/store"
	"github.com/yuebai-blast/raw-lens/web"
)

type summaryDTO struct {
	ID          string `json:"id"`
	Time        string `json:"time"`
	RemoteAddr  string `json:"remoteAddr"`
	TLS         bool   `json:"tls"`
	Method      string `json:"method"`
	Target      string `json:"target"`
	Proto       string `json:"proto"`
	Name        string `json:"name"`
	HeaderCount int    `json:"headerCount"`
	BodySize    int    `json:"bodySize"`
	RawSize     int    `json:"rawSize"`
}

type headerDTO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type detailDTO struct {
	summaryDTO
	RequestLine string      `json:"requestLine"`
	Headers     []headerDTO `json:"headers"`
	RawBase64   string      `json:"rawBase64"`
	BodyBase64  string      `json:"bodyBase64"`
}

func toSummary(c *store.CapturedRequest) summaryDTO {
	return summaryDTO{
		ID:          c.ID,
		Time:        c.Time.Format(time.RFC3339Nano),
		RemoteAddr:  c.RemoteAddr,
		TLS:         c.TLS,
		Method:      c.Method,
		Target:      c.Target,
		Proto:       c.Proto,
		Name:        c.Name,
		HeaderCount: len(c.Headers),
		BodySize:    len(c.Body),
		RawSize:     len(c.Raw),
	}
}

// newHandler 组装路由：先挂鉴权端点，再挂数据 API 与静态层，最后用鉴权中间件包住。
func newHandler(st *store.Store, auth config.Auth) http.Handler {
	mux := http.NewServeMux()
	gate := newAuthGate(auth)

	mux.HandleFunc("POST /api/login", gate.handleLogin)
	mux.HandleFunc("POST /api/logout", gate.handleLogout)
	mux.HandleFunc("GET /api/session", gate.handleSession)

	mux.HandleFunc("GET /api/requests", func(w http.ResponseWriter, r *http.Request) {
		items := st.List()
		out := make([]summaryDTO, 0, len(items))
		for i := len(items) - 1; i >= 0; i-- { // 新的在前
			out = append(out, toSummary(items[i]))
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("GET /api/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		c := st.Get(r.PathValue("id"))
		if c == nil {
			http.NotFound(w, r)
			return
		}
		d := detailDTO{
			summaryDTO:  toSummary(c),
			RequestLine: c.RequestLine,
			RawBase64:   base64.StdEncoding.EncodeToString(c.Raw),
			BodyBase64:  base64.StdEncoding.EncodeToString(c.Body),
		}
		for _, h := range c.Headers {
			d.Headers = append(d.Headers, headerDTO{Name: h[0], Value: h[1]})
		}
		writeJSON(w, d)
	})

	mux.HandleFunc("PATCH /api/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		st.SetName(r.PathValue("id"), normalizeName(body.Name))
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("DELETE /api/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		st.Delete(r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/clear", func(w http.ResponseWriter, r *http.Request) {
		st.Clear()
		w.WriteHeader(http.StatusNoContent)
	})

	// 其余路径交给内嵌前端：命中 dist 中的文件就发文件，否则回退 index.html（支持 /r/:id 刷新）。
	mux.Handle("GET /", spaFileServer())
	return gate.middleware(mux)
}

// spaFileServer 提供内嵌前端的静态资源，未命中文件时回退 index.html（SPA history 模式）。
// 未构建前端（dist 仅占位、无 index.html）时返回可读的 503 提示而非 panic。
func spaFileServer() http.Handler {
	dist, err := web.DistFS()
	if err != nil {
		log.Printf("dashboard: 读取内嵌前端失败: %v", err)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "前端资源不可用", http.StatusServiceUnavailable)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if f, ferr := dist.Open(p); ferr == nil {
			_ = f.Close()
			http.FileServerFS(dist).ServeHTTP(w, r)
			return
		}
		// 未命中文件 → 回退 index.html。
		index, ierr := fs.ReadFile(dist, "index.html")
		if ierr != nil {
			http.Error(w, "前端未构建，请运行 `mise run build` 后重试。", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}

// Serve 在 addr 上提供前端 + API，按 auth 配置决定是否启用登录鉴权。
func Serve(addr string, st *store.Store, auth config.Auth) error {
	log.Printf("dashboard 监听 %s", addr)
	return http.ListenAndServe(addr, newHandler(st, auth))
}

// normalizeName 去掉名称首尾空白，并截断到 200 个字符（按 rune 计，避免截断多字节字符）。
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 200 {
		s = string(r[:200])
	}
	return s
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
