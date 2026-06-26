package store

import (
	"bytes"
	"database/sql"
	"regexp"
	"sync"
	"testing"
	"time"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

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
	if !idPattern.MatchString(id) {
		t.Fatalf("Add 返回的 id 应为 12 位 hex 串，得到 %q", id)
	}
	got := s.Get(id)
	if got == nil {
		t.Fatal("Get 返回 nil")
	}
	if got.ID != id {
		t.Errorf("Get 回来的 ID 不一致: got=%q want=%q", got.ID, id)
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

func TestAddGeneratesUniqueRandomIDs(t *testing.T) {
	s := newMemStore(t, 500)
	seen := make(map[string]bool)
	for i := 0; i < 200; i++ {
		id := s.Add(sampleReq())
		if !idPattern.MatchString(id) {
			t.Fatalf("id 格式非法: %q", id)
		}
		if seen[id] {
			t.Fatalf("id 重复: %q", id)
		}
		seen[id] = true
	}
}

func TestListOrderAndLimit(t *testing.T) {
	s := newMemStore(t, 3)
	var ids []string
	for i := 0; i < 5; i++ {
		ids = append(ids, s.Add(sampleReq()))
	}
	list := s.List()
	if len(list) != 3 {
		t.Fatalf("List 应返回最近 3 条，得到 %d", len(list))
	}
	// 旧→新：应为最后插入的三个 id（时序由内部 seq 维持，与随机 id 无关）
	wantIDs := ids[2:]
	for i, cr := range list {
		if cr.ID != wantIDs[i] {
			t.Errorf("List[%d].ID=%q，期望 %q（应旧→新）", i, cr.ID, wantIDs[i])
		}
	}
	// 被裁掉的最旧记录取不到
	if s.Get(ids[0]) != nil {
		t.Errorf("最旧记录 id=%q 应已被保留策略删除", ids[0])
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

func TestSetName(t *testing.T) {
	s := newMemStore(t, 500)
	id := s.Add(sampleReq())
	if got := s.Get(id); got == nil || got.Name != "" {
		t.Fatalf("新记录 Name 应为空串，得到 %q", got.Name)
	}
	s.SetName(id, "登录接口")
	got := s.Get(id)
	if got == nil || got.Name != "登录接口" {
		t.Errorf("SetName 后 Name 应为 \"登录接口\"，得到 %q", got.Name)
	}
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
		t.Errorf("Delete 后 id=%q 应取不到", id1)
	}
	if s.Get(id2) == nil {
		t.Errorf("未删除的 id=%q 应仍在", id2)
	}
	if n := len(s.List()); n != 1 {
		t.Errorf("Delete 后应剩 1 条，得到 %d", n)
	}
	// 删不存在的 id 不应 panic / 影响其它记录（幂等）
	s.Delete("ffffffffffff")
	if n := len(s.List()); n != 1 {
		t.Errorf("删不存在 id 后应仍剩 1 条，得到 %d", n)
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	path := t.TempDir() + "/p.db"
	s1, err := New(Options{Path: path, Max: 500})
	if err != nil {
		t.Fatal(err)
	}
	id1 := s1.Add(sampleReq())
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
	// 重开后历史 id 仍可读
	if s2.Get(id1) == nil {
		t.Errorf("重开后应能取到历史 id=%q", id1)
	}
	// 新增仍得到合法随机 id
	if id := s2.Add(sampleReq()); !idPattern.MatchString(id) {
		t.Errorf("重开后 Add 应返回合法随机 id，得到 %q", id)
	}
}

func TestMigrateOldAutoIncrementDB(t *testing.T) {
	path := t.TempDir() + "/old.db"
	// 模拟旧库：id 为自增整数主键、无 seq、无 name，塞两条历史。
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
	for i, addr := range []string{"10.0.0.1:1", "10.0.0.2:2"} {
		if _, err := old.Exec(`INSERT INTO captured_request
		  (captured_at, remote_addr, tls, request_line, headers_json, raw)
		  VALUES (?, ?, 0, 'GET / HTTP/1.1', '[]', ?)`,
			time.Date(2026, 6, 19, 10, i, 0, 0, time.UTC).Format(time.RFC3339Nano), addr, "raw"+addr); err != nil {
			t.Fatal(err)
		}
	}
	old.Close()

	// New 应平滑重建为随机 id 结构：数据条数与时序保留，id 变为 12 位 hex。
	s, err := New(Options{Path: path, Max: 500})
	if err != nil {
		t.Fatalf("New 应能升级旧库: %v", err)
	}
	defer s.Close()
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("升级后应保留 2 条历史，得到 %d", len(list))
	}
	// 时序保留：旧→新应为 10.0.0.1 在前
	if list[0].RemoteAddr != "10.0.0.1:1" || list[1].RemoteAddr != "10.0.0.2:2" {
		t.Errorf("升级后时序应保留，得到 %q,%q", list[0].RemoteAddr, list[1].RemoteAddr)
	}
	for _, cr := range list {
		if !idPattern.MatchString(cr.ID) {
			t.Errorf("升级后 id 应为 12 位 hex，得到 %q", cr.ID)
		}
		if cr.Name != "" {
			t.Errorf("升级后 name 应为空串，得到 %q", cr.Name)
		}
	}
	// 升级后仍可正常增删改
	s.SetName(list[0].ID, "历史请求")
	if got := s.Get(list[0].ID); got == nil || got.Name != "历史请求" {
		t.Errorf("升级后应能设名，得到 %+v", got)
	}
	newID := s.Add(sampleReq())
	if !idPattern.MatchString(newID) {
		t.Errorf("升级后 Add 应返回合法随机 id，得到 %q", newID)
	}
}

func TestSetLocked(t *testing.T) {
	s := newMemStore(t, 500)
	id := s.Add(sampleReq())
	if got := s.Get(id); got == nil || got.Locked {
		t.Fatalf("新记录 Locked 应为 false，得到 %+v", got)
	}
	s.SetLocked(id, true)
	got := s.Get(id)
	if got == nil || !got.Locked {
		t.Errorf("SetLocked(true) 后 Locked 应为 true，得到 %+v", got)
	}
	if list := s.List(); len(list) != 1 || !list[0].Locked {
		t.Errorf("List 应带回 Locked=true，得到 %+v", list)
	}
	s.SetLocked(id, false)
	if got := s.Get(id); got == nil || got.Locked {
		t.Errorf("SetLocked(false) 后 Locked 应为 false，得到 %+v", got)
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

func TestClearKeepsLocked(t *testing.T) {
	s := newMemStore(t, 500)
	keep := s.Add(sampleReq())
	drop := s.Add(sampleReq())
	s.SetLocked(keep, true)
	s.Clear()
	if s.Get(keep) == nil {
		t.Errorf("Clear 后锁定记录 id=%q 应保留", keep)
	}
	if s.Get(drop) != nil {
		t.Errorf("Clear 后未锁定记录 id=%q 应被清空", drop)
	}
}

func TestDeleteSkipsLocked(t *testing.T) {
	s := newMemStore(t, 500)
	id := s.Add(sampleReq())
	s.SetLocked(id, true)
	s.Delete(id)
	if s.Get(id) == nil {
		t.Errorf("锁定记录不应被 Delete 删除")
	}
	s.SetLocked(id, false)
	s.Delete(id)
	if s.Get(id) != nil {
		t.Errorf("解锁后应可正常删除")
	}
}

func TestPruneExemptsLocked(t *testing.T) {
	s := newMemStore(t, 3)
	locked := s.Add(sampleReq()) // 最旧且锁定
	s.SetLocked(locked, true)
	var unlocked []string
	for i := 0; i < 5; i++ {
		unlocked = append(unlocked, s.Add(sampleReq()))
	}
	if s.Get(locked) == nil {
		t.Errorf("锁定记录应豁免自动淘汰（即使最旧）")
	}
	kept := 0
	for _, id := range unlocked {
		if s.Get(id) != nil {
			kept++
		}
	}
	if kept != 3 {
		t.Errorf("未锁定记录应只保留最近 3 条，实际保留 %d", kept)
	}
	if n := len(s.List()); n != 4 {
		t.Errorf("List 应返回 4 条（1 锁定 + 3 未锁定），得到 %d", n)
	}
}
