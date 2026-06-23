package store

import (
	"bytes"
	"database/sql"
	"sync"
	"testing"
	"time"
)

func sampleReq() *CapturedRequest {
	return &CapturedRequest{
		Time:        time.Date(2026, 6, 19, 10, 0, 0, 123456789, time.UTC),
		RemoteAddr:  "127.0.0.1:5555",
		TLS:         true,
		Raw:         []byte("POST /x HTTP/1.1\r\nHost: a\r\nX-Dup: 1\r\nx-dup: 2\r\n\r\nbody-bytes"),
		RequestLine: "POST /x HTTP/1.1",
		Method:      "POST",
		Target:      "/x",
		Proto:       "HTTP/1.1",
		Headers:     [][2]string{{"Host", "a"}, {"X-Dup", "1"}, {"x-dup", "2"}},
		Body:        []byte("body-bytes"),
	}
}

func newMemStore(t *testing.T, max int) *Store {
	t.Helper()
	s, err := New(Options{Path: ":memory:", Max: max})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAddGetRoundTripFidelity(t *testing.T) {
	s := newMemStore(t, 500)
	want := sampleReq()
	id := s.Add(want)
	if id <= 0 {
		t.Fatalf("Add 返回非法 id: %d", id)
	}
	got := s.Get(id)
	if got == nil {
		t.Fatal("Get 返回 nil")
	}
	if !bytes.Equal(got.Raw, want.Raw) {
		t.Errorf("Raw 不一致:\n got=%q\nwant=%q", got.Raw, want.Raw)
	}
	if !bytes.Equal(got.Body, want.Body) {
		t.Errorf("Body 不一致: got=%q want=%q", got.Body, want.Body)
	}
	if len(got.Headers) != len(want.Headers) {
		t.Fatalf("Headers 数量不一致: got=%d want=%d", len(got.Headers), len(want.Headers))
	}
	for i := range want.Headers {
		if got.Headers[i] != want.Headers[i] {
			t.Errorf("Headers[%d] 不一致(顺序/大小写): got=%v want=%v", i, got.Headers[i], want.Headers[i])
		}
	}
	if got.Method != "POST" || got.Target != "/x" || got.Proto != "HTTP/1.1" || !got.TLS {
		t.Errorf("结构化字段不一致: %+v", got)
	}
	if got.Time.UTC().Format(time.RFC3339Nano) != want.Time.Format(time.RFC3339Nano) {
		t.Errorf("Time 不一致: got=%v want=%v", got.Time, want.Time)
	}
}

func TestListOrderAndLimit(t *testing.T) {
	s := newMemStore(t, 3)
	var ids []int64
	for i := 0; i < 5; i++ {
		ids = append(ids, s.Add(sampleReq()))
	}
	list := s.List()
	if len(list) != 3 {
		t.Fatalf("List 应返回最近 3 条，得到 %d", len(list))
	}
	// 旧→新：应为最后插入的三个 id
	wantIDs := ids[2:]
	for i, cr := range list {
		if cr.ID != wantIDs[i] {
			t.Errorf("List[%d].ID=%d，期望 %d（应旧→新）", i, cr.ID, wantIDs[i])
		}
	}
	// 被裁掉的最旧记录取不到
	if s.Get(ids[0]) != nil {
		t.Errorf("最旧记录 id=%d 应已被保留策略删除", ids[0])
	}
}

func TestClear(t *testing.T) {
	s := newMemStore(t, 500)
	id := s.Add(sampleReq())
	s.Clear()
	if len(s.List()) != 0 {
		t.Error("Clear 后 List 应为空")
	}
	if s.Get(id) != nil {
		t.Error("Clear 后 Get 应返回 nil")
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	path := t.TempDir() + "/p.db"
	s1, err := New(Options{Path: path, Max: 500})
	if err != nil {
		t.Fatal(err)
	}
	s1.Add(sampleReq())
	s1.Add(sampleReq())
	s1.Close()

	s2, err := New(Options{Path: path, Max: 500})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if len(s2.List()) != 2 {
		t.Fatalf("重开后应有 2 条历史，得到 %d", len(s2.List()))
	}
	if id := s2.Add(sampleReq()); id != 3 {
		t.Errorf("自增 id 应续接为 3，得到 %d", id)
	}
}

func TestSetName(t *testing.T) {
	s := newMemStore(t, 500)
	id := s.Add(sampleReq())
	// 新抓到的请求默认无名称
	if got := s.Get(id); got == nil || got.Name != "" {
		t.Fatalf("新记录 Name 应为空串，得到 %q", got.Name)
	}
	s.SetName(id, "登录接口")
	got := s.Get(id)
	if got == nil || got.Name != "登录接口" {
		t.Errorf("SetName 后 Name 应为 \"登录接口\"，得到 %q", got.Name)
	}
	// List 也应带回名称
	list := s.List()
	if len(list) != 1 || list[0].Name != "登录接口" {
		t.Errorf("List 应带回名称，得到 %+v", list)
	}
}

func TestDelete(t *testing.T) {
	s := newMemStore(t, 500)
	id1 := s.Add(sampleReq())
	id2 := s.Add(sampleReq())
	s.Delete(id1)
	if s.Get(id1) != nil {
		t.Errorf("Delete 后 id=%d 应取不到", id1)
	}
	if s.Get(id2) == nil {
		t.Errorf("未删除的 id=%d 应仍在", id2)
	}
	if n := len(s.List()); n != 1 {
		t.Errorf("Delete 后应剩 1 条，得到 %d", n)
	}
	// 删不存在的 id 不应 panic / 影响其它记录（幂等）
	s.Delete(99999)
	if n := len(s.List()); n != 1 {
		t.Errorf("删不存在 id 后应仍剩 1 条，得到 %d", n)
	}
}

func TestMigrateAddsNameColumnToOldDB(t *testing.T) {
	path := t.TempDir() + "/old.db"
	// 模拟老库：建一张不含 name 列的旧表，塞一条历史。
	old, err := sql.Open("sqlite", dsnFor(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`CREATE TABLE captured_request (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  captured_at TEXT NOT NULL, remote_addr TEXT NOT NULL, tls INTEGER NOT NULL,
	  request_line TEXT NOT NULL, method TEXT, target TEXT, proto TEXT,
	  headers_json TEXT NOT NULL, body BLOB, raw BLOB NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`INSERT INTO captured_request
	  (captured_at, remote_addr, tls, request_line, headers_json, raw)
	  VALUES ('2026-06-19T10:00:00Z','127.0.0.1:1',0,'GET / HTTP/1.1','[]','raw')`); err != nil {
		t.Fatal(err)
	}
	old.Close()

	// New 应平滑补上 name 列，老数据可读、可设名。
	s, err := New(Options{Path: path, Max: 500})
	if err != nil {
		t.Fatalf("New 应能升级老库: %v", err)
	}
	defer s.Close()
	list := s.List()
	if len(list) != 1 || list[0].Name != "" {
		t.Fatalf("升级后老记录应可读且 Name 为空，得到 %+v", list)
	}
	s.SetName(list[0].ID, "历史请求")
	if got := s.Get(list[0].ID); got == nil || got.Name != "历史请求" {
		t.Errorf("升级后应能设名，得到 %+v", got)
	}
}

func TestConcurrentAdd(t *testing.T) {
	s := newMemStore(t, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Add(sampleReq())
		}()
	}
	wg.Wait()
	if n := len(s.List()); n != 50 {
		t.Errorf("并发写后应有 50 条，得到 %d", n)
	}
}
