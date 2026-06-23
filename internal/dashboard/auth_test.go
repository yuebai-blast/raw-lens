package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yuebai-blast/raw-lens/internal/config"
)

func TestSessionCreateAndValid(t *testing.T) {
	s := newSessionStore(time.Hour)
	tok, err := s.create()
	if err != nil {
		t.Fatalf("create 失败: %v", err)
	}
	if tok == "" {
		t.Fatal("token 不应为空")
	}
	if !s.valid(tok) {
		t.Fatal("刚创建的 token 应有效")
	}
}

func TestSessionInvalidForUnknownAndEmpty(t *testing.T) {
	s := newSessionStore(time.Hour)
	if s.valid("") {
		t.Fatal("空 token 应无效")
	}
	if s.valid("nope") {
		t.Fatal("未知 token 应无效")
	}
}

func TestSessionExpires(t *testing.T) {
	s := newSessionStore(-time.Second) // 立即过期
	tok, _ := s.create()
	if s.valid(tok) {
		t.Fatal("过期 token 应无效")
	}
}

func TestSessionDestroy(t *testing.T) {
	s := newSessionStore(time.Hour)
	tok, _ := s.create()
	s.destroy(tok)
	if s.valid(tok) {
		t.Fatal("destroy 后 token 应无效")
	}
}

func TestSessionTokensUnique(t *testing.T) {
	s := newSessionStore(time.Hour)
	a, _ := s.create()
	b, _ := s.create()
	if a == b {
		t.Fatal("两次 create 的 token 应不同")
	}
}

// enabledHandler 造一个开启鉴权、账号 admin/secret 的 handler。
func enabledHandler(t *testing.T) http.Handler {
	t.Helper()
	return newHandler(newTestStore(t), config.Auth{
		Enabled: true, Username: "admin", Password: "secret", SessionTTLHours: 168,
	})
}

func TestLoginSuccessSetsCookie(t *testing.T) {
	h := enabledHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("正确账号密码应 200，得到 %d", rec.Code)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("登录成功应种 cookie")
	}
}

func TestLoginWrongPassword401(t *testing.T) {
	h := enabledHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"username":"admin","password":"wrong"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码应 401，得到 %d", rec.Code)
	}
}

func TestGatedAPIBlockedWithoutSession(t *testing.T) {
	h := enabledHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("无会话访问数据 API 应 401，得到 %d", rec.Code)
	}
}

func TestGatedAPIAllowedAfterLogin(t *testing.T) {
	h := enabledHandler(t)
	// 先登录拿 cookie
	loginReq := httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"username":"admin","password":"secret"}`))
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, loginReq)
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("登录应返回 cookie")
	}
	// 带 cookie 访问数据 API
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("带有效会话访问数据 API 应 200，得到 %d", rec.Code)
	}
}

func TestUnauthedAllowsShellAndAuthEndpoints(t *testing.T) {
	h := enabledHandler(t)
	// 静态外壳 / 必须放行（未构建前端时为 200 或 503，但绝不应是 401）
	for _, p := range []string{"/", "/r/1"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusUnauthorized {
			t.Fatalf("%s 不应被鉴权拦截", p)
		}
	}
	// /api/session 放行
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/session 应 200，得到 %d", rec.Code)
	}
}

func TestSessionEndpointReports(t *testing.T) {
	h := enabledHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `"enabled":true`) || !strings.Contains(body, `"authenticated":false`) {
		t.Fatalf("未登录时 session 应 enabled=true authenticated=false，得到 %s", body)
	}
}

func TestDisabledAuthAllowsEverything(t *testing.T) {
	h := newHandler(newTestStore(t), config.Auth{Enabled: false})
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("关闭鉴权时数据 API 应直接 200，得到 %d", rec.Code)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	h := enabledHandler(t)
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"username":"admin","password":"secret"}`)))
	cookie := loginRec.Result().Cookies()[0]
	// 登出
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	logoutReq.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), logoutReq)
	// 再用旧 cookie 访问数据 API 应被拦
	req := httptest.NewRequest(http.MethodGet, "/api/requests", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("登出后旧会话应失效，得到 %d", rec.Code)
	}
}

// login 函数：向给定 handler 发一次正确的登录请求，返回收到的 cookie。
func login(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/login",
		strings.NewReader(`{"username":"admin","password":"secret"}`)))
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("登录应返回 cookie")
	}
	return cookies[0]
}

func TestLoginCookieSecureDefaultOff(t *testing.T) {
	// 默认（CookieSecure 未配=false）：cookie 不带 Secure，便于内网 HTTP / 本地开发访问。
	h := enabledHandler(t)
	if c := login(t, h); c.Secure {
		t.Fatalf("默认不应设 Secure")
	}
}

func TestLoginCookieSecureOn(t *testing.T) {
	// CookieSecure=true（公网 HTTPS 反代场景）：cookie 必须带 Secure。
	h := newHandler(newTestStore(t), config.Auth{
		Enabled: true, Username: "admin", Password: "secret", SessionTTLHours: 168, CookieSecure: true,
	})
	if c := login(t, h); !c.Secure {
		t.Fatalf("CookieSecure=true 时 cookie 应带 Secure")
	}
}
