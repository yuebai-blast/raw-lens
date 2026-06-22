package dashboard

import (
	"testing"
	"time"
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
