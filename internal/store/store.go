// Package store 用 SQLite 持久化抓到的请求。
// path 配 ":memory:" 时为内存库，进程重启即清空（兼容旧行为）。
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// CapturedRequest 保存一条抓到的请求。
// Raw 是连接上读到的原始字节（请求行 + header 块 + body），完全保真。
// Headers 是按收到的顺序、原始大小写解析出来的，方便前端结构化展示。
type CapturedRequest struct {
	ID          string // 对外公开标识，12 位随机小写 hex
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
	Name        string // 用户给该请求起的备注名，可空
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
const selectCols = `id, captured_at, remote_addr, tls, request_line, method, target, proto, headers_json, body, raw, name`

// schema 是当前结构：seq 内部自增序号（仅排序/保留策略用），id 对外随机标识。
const schema = `
CREATE TABLE IF NOT EXISTS captured_request (
  seq          INTEGER PRIMARY KEY AUTOINCREMENT, -- 内部序号，仅用于排序与保留策略，不对外
  id           TEXT    NOT NULL UNIQUE,           -- 对外公开标识，12 位随机小写 hex
  captured_at  TEXT    NOT NULL,                  -- 抓到的时间，RFC3339Nano(UTC) 字符串
  remote_addr  TEXT    NOT NULL,                  -- 客户端来源地址
  tls          INTEGER NOT NULL,                  -- 是否经 TLS：0=明文 1=TLS
  request_line TEXT    NOT NULL,                  -- 原始请求行（保真）
  method       TEXT,                              -- 解析出的方法，可空
  target       TEXT,                              -- 解析出的请求目标，可空
  proto        TEXT,                              -- 协议版本，可空
  headers_json TEXT    NOT NULL,                  -- header 数组 JSON，保序保大小写 [["Name","Val"],...]
  body         BLOB,                              -- body 原始字节，可空
  raw          BLOB    NOT NULL,                  -- 连接上读到的全量原始字节，逐字保真
  name         TEXT    NOT NULL DEFAULT ''        -- 用户给该请求起的备注名，默认空串
);
`

// New 打开（或创建）库并建表。
func New(opts Options) (*Store, error) {
	max := opts.Max
	if max <= 0 {
		max = 500
	}
	path := opts.Path
	if path == "" {
		path = "data/db/rawlens.db"
	}
	// 文件库需先确保父目录存在，否则 SQLite 打不开（:memory: 不涉及目录）。
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
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
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, max: max}, nil
}

// migrate 对老库做平滑升级，按代际顺序执行：
//  1. 缺 name 列则补上（CREATE TABLE IF NOT EXISTS 不会给已存在的表加列）；
//  2. 缺 seq 列（旧的自增整数 id 结构）则重建为随机 id 结构。
//
// 先做 1 再做 2，保证重建时 SELECT 能安全引用 name 列。
func migrate(db *sql.DB) error {
	if !hasColumn(db, "captured_request", "name") {
		if _, err := db.Exec(`ALTER TABLE captured_request ADD COLUMN name TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !hasColumn(db, "captured_request", "seq") {
		return rebuildWithRandomID(db)
	}
	return nil
}

// rebuildWithRandomID 把旧的「id 自增整数主键」表重建为「seq 自增主键 + id 随机串」结构。
// 在事务内：建新表 → 按旧 id 顺序灌入（seq 自然续上、时序不变，id 用 SQLite randomblob 现场生成
// 12 位小写 hex）→ 删旧表 → 改名。旧抓包数据全部保留。
func rebuildWithRandomID(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		`CREATE TABLE captured_request_new (
		   seq INTEGER PRIMARY KEY AUTOINCREMENT,
		   id TEXT NOT NULL UNIQUE,
		   captured_at TEXT NOT NULL, remote_addr TEXT NOT NULL, tls INTEGER NOT NULL,
		   request_line TEXT NOT NULL, method TEXT, target TEXT, proto TEXT,
		   headers_json TEXT NOT NULL, body BLOB, raw BLOB NOT NULL,
		   name TEXT NOT NULL DEFAULT '')`,
		`INSERT INTO captured_request_new
		   (id, captured_at, remote_addr, tls, request_line, method, target, proto, headers_json, body, raw, name)
		 SELECT lower(hex(randomblob(6))), captured_at, remote_addr, tls, request_line, method, target, proto,
		        headers_json, body, raw, name
		 FROM captured_request ORDER BY id`,
		`DROP TABLE captured_request`,
		`ALTER TABLE captured_request_new RENAME TO captured_request`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// newID 生成 12 位随机小写 hex（crypto/rand 6 字节）。
func newID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hasColumn 查表是否已有某列。
func hasColumn(db *sql.DB, table, col string) bool {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil && name == col {
			return true
		}
	}
	return false
}

// dsnFor 文件库附带 WAL 与 busy_timeout；:memory: 不需要。
func dsnFor(path string) string {
	if path == ":memory:" {
		return path
	}
	return path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// Add 写入一条请求，返回随机生成的对外 id；出错记日志并返回空串。
// id 命中 UNIQUE 冲突时重新生成重试（500 条量级几乎不会发生，做几次兜底）。
func (s *Store) Add(cr *CapturedRequest) string {
	hj, err := json.Marshal(cr.Headers)
	if err != nil {
		log.Printf("store: 序列化 header 失败: %v", err)
		return ""
	}
	tlsInt := 0
	if cr.TLS {
		tlsInt = 1
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		id, err := newID()
		if err != nil {
			log.Printf("store: 生成 id 失败: %v", err)
			return ""
		}
		_, err = s.db.Exec(
			`INSERT INTO captured_request
			   (id, captured_at, remote_addr, tls, request_line, method, target, proto, headers_json, body, raw, name)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, cr.Time.UTC().Format(time.RFC3339Nano), cr.RemoteAddr, tlsInt,
			cr.RequestLine, cr.Method, cr.Target, cr.Proto, string(hj), cr.Body, cr.Raw, cr.Name)
		if err == nil {
			cr.ID = id
			s.prune()
			return id
		}
		lastErr = err // 多半是 id 撞 UNIQUE，换一个再试
	}
	log.Printf("store: 写入失败: %v", lastErr)
	return ""
}

// prune 按 max 条数保留最新记录，删掉更旧的。
func (s *Store) prune() {
	if s.max <= 0 {
		return
	}
	if _, err := s.db.Exec(
		`DELETE FROM captured_request
		 WHERE seq <= (SELECT MAX(seq) FROM captured_request) - ?`, s.max); err != nil {
		log.Printf("store: 保留策略清理失败: %v", err)
	}
}

// List 返回最近 max 条请求，按旧→新排序。
func (s *Store) List() []*CapturedRequest {
	rows, err := s.db.Query(
		`SELECT `+selectCols+` FROM captured_request ORDER BY seq DESC LIMIT ?`, s.max)
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
	if err := rows.Err(); err != nil {
		log.Printf("store: 查询迭代失败: %v", err)
	}
	// 查询是新→旧，反转成旧→新，保持原内存版语义。
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Get 按 id 取一条，不存在返回 nil。
func (s *Store) Get(id string) *CapturedRequest {
	row := s.db.QueryRow(`SELECT `+selectCols+` FROM captured_request WHERE id = ?`, id)
	cr, err := scanRow(row)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("store: 取 #%s 失败: %v", id, err)
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

// SetName 给指定记录设置备注名；出错记日志。
func (s *Store) SetName(id string, name string) {
	if _, err := s.db.Exec(`UPDATE captured_request SET name = ? WHERE id = ?`, name, id); err != nil {
		log.Printf("store: 设置 #%s 名称失败: %v", id, err)
	}
}

// Delete 删除指定记录；删不存在的 id 不视为错误（幂等）。出错记日志。
func (s *Store) Delete(id string) {
	if _, err := s.db.Exec(`DELETE FROM captured_request WHERE id = ?`, id); err != nil {
		log.Printf("store: 删除 #%s 失败: %v", id, err)
	}
}

// Close 关闭底层数据库。
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping 校验底层数据库连接是否可用，供健康检查端点判断后端是否真正就绪。
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
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
		&method, &target, &proto, &headersJSON, &cr.Body, &cr.Raw, &cr.Name); err != nil {
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
