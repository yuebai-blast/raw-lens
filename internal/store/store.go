// Package store 在内存里保存抓到的请求（环形缓冲，进程重启即清空）。
package store

import (
	"sync"
	"sync/atomic"
	"time"
)

// CapturedRequest 保存一条抓到的请求。
// Raw 是连接上读到的原始字节（请求行 + header 块 + body），完全保真。
// Headers 是按收到的顺序、原始大小写解析出来的，方便前端结构化展示。
type CapturedRequest struct {
	ID          int64
	Time        time.Time
	RemoteAddr  string
	TLS         bool
	Raw         []byte
	RequestLine string
	Method      string
	Target      string
	Proto       string
	Headers     [][2]string
	Body        []byte
}

type Store struct {
	mu     sync.RWMutex
	max    int
	items  []*CapturedRequest
	nextID int64
}

func New(max int) *Store {
	if max <= 0 {
		max = 500
	}
	return &Store{max: max}
}

func (s *Store) Add(cr *CapturedRequest) int64 {
	id := atomic.AddInt64(&s.nextID, 1)
	cr.ID = id
	s.mu.Lock()
	s.items = append(s.items, cr)
	if len(s.items) > s.max {
		s.items = s.items[len(s.items)-s.max:]
	}
	s.mu.Unlock()
	return id
}

// List 返回当前所有请求（旧到新），调用方自行决定展示顺序。
func (s *Store) List() []*CapturedRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*CapturedRequest, len(s.items))
	copy(out, s.items)
	return out
}

func (s *Store) Get(id int64) *CapturedRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.items {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (s *Store) Clear() {
	s.mu.Lock()
	s.items = nil
	s.mu.Unlock()
}
