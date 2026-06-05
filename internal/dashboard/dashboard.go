// Package dashboard 提供浏览抓到的请求的前端面板和 JSON API。
package dashboard

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"rawlens/internal/store"
	"rawlens/web"
)

type summaryDTO struct {
	ID          int64  `json:"id"`
	Time        string `json:"time"`
	RemoteAddr  string `json:"remoteAddr"`
	TLS         bool   `json:"tls"`
	Method      string `json:"method"`
	Target      string `json:"target"`
	Proto       string `json:"proto"`
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
		HeaderCount: len(c.Headers),
		BodySize:    len(c.Body),
		RawSize:     len(c.Raw),
	}
}

// Serve 在 addr 上提供前端 + API。
func Serve(addr string, st *store.Store) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/requests", func(w http.ResponseWriter, r *http.Request) {
		items := st.List()
		out := make([]summaryDTO, 0, len(items))
		for i := len(items) - 1; i >= 0; i-- { // 新的在前
			out = append(out, toSummary(items[i]))
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("GET /api/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		c := st.Get(id)
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

	mux.HandleFunc("POST /api/clear", func(w http.ResponseWriter, r *http.Request) {
		st.Clear()
		w.WriteHeader(http.StatusNoContent)
	})

	// 其余路径交给内嵌的前端静态资源（/ 返回 index.html）。
	files := http.FileServerFS(web.FS)
	mux.Handle("GET /", files)

	log.Printf("dashboard 监听 %s", addr)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
