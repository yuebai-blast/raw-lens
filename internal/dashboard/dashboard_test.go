package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yuebai-blast/raw-lens/internal/config"
	"github.com/yuebai-blast/raw-lens/internal/store"
)

func seedReq(st *store.Store) string {
	return st.Add(&store.CapturedRequest{
		Time:        time.Now(),
		RemoteAddr:  "127.0.0.1:1",
		RequestLine: "GET / HTTP/1.1",
		Method:      "GET",
		Target:      "/",
		Proto:       "HTTP/1.1",
		Raw:         []byte("GET / HTTP/1.1\r\n\r\n"),
	})
}

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
	h := newHandler(newTestStore(t), config.Auth{}, "")
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
	h := newHandler(newTestStore(t), config.Auth{}, "")
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
	h := newHandler(newTestStore(t), config.Auth{}, "")
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

// PATCH /api/requests/{id} 设置名称：204，且后续 GET 详情/列表能读回。
func TestPatchSetsName(t *testing.T) {
	st := newTestStore(t)
	h := newHandler(st, config.Auth{}, "")
	id := seedReq(st)

	req := httptest.NewRequest(http.MethodPatch, "/api/requests/"+id, strings.NewReader(`{"name":"  登录接口  "}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH 期望 204，得到 %d", rec.Code)
	}

	// 详情应带回去掉首尾空格的名称
	detail := getJSON(t, h, "/api/requests/"+id)
	if detail["name"] != "登录接口" {
		t.Errorf("详情 name 期望 \"登录接口\"，得到 %v", detail["name"])
	}
	// 列表也应带回名称
	list := getJSONArray(t, h, "/api/requests")
	if len(list) != 1 || list[0]["name"] != "登录接口" {
		t.Errorf("列表应带回名称，得到 %v", list)
	}
}

// PATCH 未知 id：id 现在是随机串，任意非空形状都合法，查不到即无操作返回 204。
func TestPatchUnknownID(t *testing.T) {
	h := newHandler(newTestStore(t), config.Auth{}, "")
	req := httptest.NewRequest(http.MethodPatch, "/api/requests/deadbeef0000", strings.NewReader(`{"name":"x"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH 未知 id 期望 204，得到 %d", rec.Code)
	}
}

// GET 未知 id 仍返回 404。
func TestGetUnknownID(t *testing.T) {
	h := newHandler(newTestStore(t), config.Auth{}, "")
	req := httptest.NewRequest(http.MethodGet, "/api/requests/deadbeef0000", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET 未知 id 期望 404，得到 %d", rec.Code)
	}
}

// DELETE /api/requests/{id} 删除单条：204，且后续取不到、列表少一条。
func TestDeleteRemovesOne(t *testing.T) {
	st := newTestStore(t)
	h := newHandler(st, config.Auth{}, "")
	id1 := seedReq(st)
	seedReq(st)

	req := httptest.NewRequest(http.MethodDelete, "/api/requests/"+id1, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE 期望 204，得到 %d", rec.Code)
	}
	if list := getJSONArray(t, h, "/api/requests"); len(list) != 1 {
		t.Errorf("删除后列表应剩 1 条，得到 %d", len(list))
	}
	// 删不存在的也应 204（幂等）
	req = httptest.NewRequest(http.MethodDelete, "/api/requests/ffffffffffff", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("删不存在期望 204，得到 %d", rec.Code)
	}
}

func getJSON(t *testing.T, h http.Handler, path string) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("解析 %s 响应失败: %v (body=%s)", path, err, rec.Body.String())
	}
	return m
}

func getJSONArray(t *testing.T, h http.Handler, path string) []map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var a []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil {
		t.Fatalf("解析 %s 响应失败: %v (body=%s)", path, err, rec.Body.String())
	}
	return a
}

// GET /api/meta 返回注入的抓包展示地址，且鉴权开启时也放行（不在 isGated 列表内）。
func TestMetaReturnsCaptureURL(t *testing.T) {
	h := newHandler(newTestStore(t), config.Auth{
		Enabled: true, Username: "admin", Password: "secret", SessionTTLHours: 168,
	}, "https://xxx.xx.com:9100")
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/meta 期望 200（应放行），得到 %d", rec.Code)
	}
	m := getJSON(t, h, "/api/meta")
	if m["captureUrl"] != "https://xxx.xx.com:9100" {
		t.Fatalf("captureUrl 期望 https://xxx.xx.com:9100，得到 %v", m["captureUrl"])
	}
}
