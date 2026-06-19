package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yuebai-blast/raw-lens/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.New(store.Options{Path: ":memory:", Max: 100})
	if err != nil {
		t.Fatalf("建测试 store 失败: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// /api/* 不被静态层拦截：未知 id 应由 API handler 返回 404，而非回退 index.html。
func TestAPIRouteNotSwallowedBySPAFallback(t *testing.T) {
	h := newHandler(newTestStore(t))
	req := httptest.NewRequest(http.MethodGet, "/api/requests/999999", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知 id 应 404，得到 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct == "text/html; charset=utf-8" {
		t.Fatalf("/api 路径不应回退 HTML，Content-Type=%q", ct)
	}
}

// 未构建前端（dist 仅占位）时，非 API 路径应给出可读提示而非 panic / 500 空响应。
func TestSPAFallbackWhenFrontendNotBuilt(t *testing.T) {
	h := newHandler(newTestStore(t))
	for _, path := range []string{"/", "/r/123"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK && rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s 期望 200 或 503，得到 %d", path, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("%s 响应体不应为空", path)
		}
	}
}

// /api/requests 正常返回 JSON 数组。
func TestAPIRequestsStillJSON(t *testing.T) {
	h := newHandler(newTestStore(t))
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("期望 200，得到 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type 期望 JSON，得到 %q", ct)
	}
}
