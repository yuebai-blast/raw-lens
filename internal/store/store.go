// Package store 用 SQLite 持久化抓到的请求。
// path 配 ":memory:" 时为内存库，进程重启即清空（兼容旧行为）。
package store

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "modernc.org/sqlite"
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

// Options 是打开 Store 的参数。
type Options struct {
	Path string // SQLite 文件路径；":memory:" 为内存库
	Max  int    // 最多保留多少条请求，<=0 兜底 500
}

// Store 是 SQLite 支撑的请求仓库，对外方法语义与原内存版一致。
type Store struct {
	db  *sql.DB
	max int
}

// selectCols 统一 SELECT 的列顺序，scanRow 依赖此顺序。
const selectCols = `id, captured_at, remote_addr, tls, request_line, method, target, proto, headers_json, body, raw`

const schema = `
CREATE TABLE IF NOT EXISTS captured_request (
  id           INTEGER PRIMARY KEY AUTOINCREMENT, -- 请求序号，自增主键
  captured_at  TEXT    NOT NULL,                  -- 抓到的时间，RFC3339Nano(UTC) 字符串
  remote_addr  TEXT    NOT NULL,                  -- 客户端来源地址
  tls          INTEGER NOT NULL,                  -- 是否经 TLS：0=明文 1=TLS
  request_line TEXT    NOT NULL,                  -- 原始请求行（保真）
  method       TEXT,                              -- 解析出的方法，可空
  target       TEXT,                              -- 解析出的请求目标，可空
  proto        TEXT,                              -- 协议版本，可空
  headers_json TEXT    NOT NULL,                  -- header 数组 JSON，保序保大小写 [["Name","Val"],...]
  body         BLOB,                              -- body 原始字节，可空
  raw          BLOB    NOT NULL                   -- 连接上读到的全量原始字节，逐字保真
);
CREATE INDEX IF NOT EXISTS idx_captured_request_id_desc ON captured_request(id DESC);
`

// New 打开（或创建）库并建表。
func New(opts Options) (*Store, error) {
	max := opts.Max
	if max <= 0 {
		max = 500
	}
	path := opts.Path
	if path == "" {
		path = "rawlens.db"
	}
	db, err := sql.Open("sqlite", dsnFor(path))
	if err != nil {
		return nil, err
	}
	// 单连接串行化所有访问：量小够用，且让 :memory: 共享同一库、写入不撞 SQLITE_BUSY。
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, max: max}, nil
}

// dsnFor 文件库附带 WAL 与 busy_timeout；:memory: 不需要。
func dsnFor(path string) string {
	if path == ":memory:" {
		return path
	}
	return path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// Add 写入一条请求，返回自增 id；出错记日志并返回 0。
func (s *Store) Add(cr *CapturedRequest) int64 {
	hj, err := json.Marshal(cr.Headers)
	if err != nil {
		log.Printf("store: 序列化 header 失败: %v", err)
		return 0
	}
	tlsInt := 0
	if cr.TLS {
		tlsInt = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO captured_request
		   (captured_at, remote_addr, tls, request_line, method, target, proto, headers_json, body, raw)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cr.Time.UTC().Format(time.RFC3339Nano), cr.RemoteAddr, tlsInt,
		cr.RequestLine, cr.Method, cr.Target, cr.Proto, string(hj), cr.Body, cr.Raw)
	if err != nil {
		log.Printf("store: 写入失败: %v", err)
		return 0
	}
	id, _ := res.LastInsertId()
	cr.ID = id
	s.prune()
	return id
}

// prune 按 max 条数保留最新记录，删掉更旧的。
func (s *Store) prune() {
	if s.max <= 0 {
		return
	}
	if _, err := s.db.Exec(
		`DELETE FROM captured_request
		 WHERE id <= (SELECT MAX(id) FROM captured_request) - ?`, s.max); err != nil {
		log.Printf("store: 保留策略清理失败: %v", err)
	}
}

// List 返回最近 max 条请求，按旧→新排序。
func (s *Store) List() []*CapturedRequest {
	rows, err := s.db.Query(
		`SELECT `+selectCols+` FROM captured_request ORDER BY id DESC LIMIT ?`, s.max)
	if err != nil {
		log.Printf("store: 查询失败: %v", err)
		return nil
	}
	defer rows.Close()
	var out []*CapturedRequest
	for rows.Next() {
		cr, err := scanRow(rows)
		if err != nil {
			log.Printf("store: 扫描失败: %v", err)
			continue
		}
		out = append(out, cr)
	}
	// 查询是新→旧，反转成旧→新，保持原内存版语义。
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Get 按 id 取一条，不存在返回 nil。
func (s *Store) Get(id int64) *CapturedRequest {
	row := s.db.QueryRow(`SELECT `+selectCols+` FROM captured_request WHERE id = ?`, id)
	cr, err := scanRow(row)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("store: 取 #%d 失败: %v", id, err)
		}
		return nil
	}
	return cr
}

// Clear 清空所有记录。
func (s *Store) Clear() {
	if _, err := s.db.Exec(`DELETE FROM captured_request`); err != nil {
		log.Printf("store: 清空失败: %v", err)
	}
}

// Close 关闭底层数据库。
func (s *Store) Close() error {
	return s.db.Close()
}

// scanner 同时被 *sql.Row 与 *sql.Rows 满足。
type scanner interface {
	Scan(dest ...any) error
}

// scanRow 把一行扫描成 CapturedRequest，列顺序须与 selectCols 一致。
func scanRow(sc scanner) (*CapturedRequest, error) {
	var (
		cr                    CapturedRequest
		capturedAt            string
		tlsInt                int
		headersJSON           string
		method, target, proto sql.NullString
	)
	if err := sc.Scan(&cr.ID, &capturedAt, &cr.RemoteAddr, &tlsInt, &cr.RequestLine,
		&method, &target, &proto, &headersJSON, &cr.Body, &cr.Raw); err != nil {
		return nil, err
	}
	cr.Method, cr.Target, cr.Proto = method.String, target.String, proto.String
	cr.TLS = tlsInt != 0
	if t, err := time.Parse(time.RFC3339Nano, capturedAt); err == nil {
		cr.Time = t
	}
	if err := json.Unmarshal([]byte(headersJSON), &cr.Headers); err != nil {
		return nil, err
	}
	return &cr, nil
}
