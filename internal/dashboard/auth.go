package dashboard

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
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
