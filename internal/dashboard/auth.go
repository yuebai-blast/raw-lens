package dashboard

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yuebai-blast/raw-lens/internal/config"
)

// sessionCookieName 是面板登录会话 cookie 的名字。
const sessionCookieName = "rawlens_session"

// sessionStore 是内存会话表：token → 过期时刻。并发安全。
// 进程重启即清空——单实例内网工具可接受，重启后需重新登录。
type sessionStore struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]time.Time
}

func newSessionStore(ttl time.Duration) *sessionStore {
	return &sessionStore{ttl: ttl, m: make(map[string]time.Time)}
}

// create 生成一个随机 token 并记录过期时刻。
func (s *sessionStore) create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.m[tok] = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return tok, nil
}

// valid 判断 token 是否存在且未过期；顺手惰性清理已过期项。
func (s *sessionStore) valid(tok string) bool {
	if tok == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.m[tok]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.m, tok)
		return false
	}
	return true
}

// destroy 删除一个会话（登出用）。
func (s *sessionStore) destroy(tok string) {
	s.mu.Lock()
	delete(s.m, tok)
	s.mu.Unlock()
}

// authGate 持有鉴权配置与会话表，提供登录/登出/会话端点和拦截中间件。
type authGate struct {
	cfg      config.Auth
	sessions *sessionStore
}

func newAuthGate(cfg config.Auth) *authGate {
	return &authGate{
		cfg:      cfg,
		sessions: newSessionStore(time.Duration(cfg.SessionTTLHours) * time.Hour),
	}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// writeJSONStatus 先写状态码再写 JSON 体（避免在 WriteHeader 后再设 Content-Type 无效）。
func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// authed 判断本次请求是否已登录。关闭鉴权时恒为 true。
func (a *authGate) authed(r *http.Request) bool {
	if !a.cfg.Enabled {
		return true
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return a.sessions.valid(c.Value)
}

func (a *authGate) handleLogin(w http.ResponseWriter, r *http.Request) {
	// 限制请求体大小，防止超大 body 占用内存（登录体仅需几十字节，1 MiB 已极为宽裕）。
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]any{"authenticated": false})
		return
	}
	// 用户名、密码都用常量时间比较，避免短路与计时旁路；不区分错在哪以防账号枚举。
	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(a.cfg.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.cfg.Password)) == 1
	if !(userOK && passOK) {
		writeJSONStatus(w, http.StatusUnauthorized, map[string]any{"authenticated": false})
		return
	}
	tok, err := a.sessions.create()
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   a.cfg.SessionTTLHours * 3600,
	})
	writeJSONStatus(w, http.StatusOK, map[string]any{"authenticated": true})
}

func (a *authGate) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		a.sessions.destroy(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/",
		HttpOnly: true, Secure: a.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *authGate) handleSession(w http.ResponseWriter, r *http.Request) {
	writeJSONStatus(w, http.StatusOK, map[string]bool{
		"enabled":       a.cfg.Enabled,
		"authenticated": a.authed(r),
	})
}

// isGated 标记必须有有效会话才能访问的数据 API（白名单式拦截，避免误拦静态资源与认证端点）。
func isGated(path string) bool {
	return path == "/api/requests" ||
		strings.HasPrefix(path, "/api/requests/") ||
		path == "/api/clear"
}

// middleware 包住面板 handler：仅当开启鉴权、命中数据 API 且未登录时拦截，其余放行。
func (a *authGate) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.Enabled && isGated(r.URL.Path) && !a.authed(r) {
			writeJSONStatus(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
