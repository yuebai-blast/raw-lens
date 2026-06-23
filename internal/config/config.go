// Package config 从 YAML 文件加载 raw-lens 的运行时配置。
//
// 加载策略：先填内置默认值，再用配置文件里出现的字段覆盖。
// 文件不存在时直接用默认值（不报错），方便零配置启动。
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultPath 是不指定 -config 时查找的默认路径（相对当前工作目录）。
const DefaultPath = "config.yaml"

// DefaultDBPath 是 SQLITE 模式下的默认库文件路径（相对当前工作目录，落在专属 data 子目录）。
const DefaultDBPath = "data/db/rawlens.db"

type TLS struct {
	Enabled bool   `yaml:"enabled"` // 开启后抓包端口走 TLS（抓 HTTPS 原始请求）
	Cert    string `yaml:"cert"`    // 证书路径，留空则用内存自签名证书
	Key     string `yaml:"key"`     // 私钥路径
}

type Capture struct {
	Addr string `yaml:"addr"` // 裸 TCP 抓包监听地址，对外暴露这个端口
	TLS  TLS    `yaml:"tls"`
}

type Dashboard struct {
	Addr string `yaml:"addr"` // 前端面板监听地址
}

// Mode 是存储模式枚举：成员名与字符串取值均为 SCREAMING_SNAKE_CASE 且逐字一致。
type Mode string

const (
	SQLITE Mode = "SQLITE" // 落盘到默认 SQLite 文件 rawlens.db
	MEMORY Mode = "MEMORY" // 内存库，进程重启即清空
)

type Store struct {
	Max  int  `yaml:"max"`  // 最多保留多少条请求（超出删最旧）
	Mode Mode `yaml:"mode"` // 存储模式：SQLITE=落盘默认文件 data/db/rawlens.db；MEMORY=内存库（重启即清空）
}

// Log 控制日志文件输出。日志始终打到 stdout；File 非空时再额外写一份到该文件，
// 用 lumberjack 按大小滚动（生成 rawlens.log + rawlens-时间戳.log 等备份）。
type Log struct {
	File       string `yaml:"file"`         // 日志文件路径；留空则只输出到 stdout，不落文件
	MaxSizeMB  int    `yaml:"max_size_mb"`  // 单个文件多大（MB）就滚动
	MaxBackups int    `yaml:"max_backups"`  // 最多保留多少个滚动备份
	MaxAgeDays int    `yaml:"max_age_days"` // 备份最多保留多少天
}

// Auth 控制面板登录鉴权。Enabled=false（默认）时面板免登录，行为同历史版本。
// Enabled=true 时访问面板数据 API 需先用 Username/Password 登录。密码为明文。
type Auth struct {
	Enabled         bool   `yaml:"enabled"`           // true 才开启面板登录鉴权
	Username        string `yaml:"username"`          // 登录用户名
	Password        string `yaml:"password"`          // 登录密码（明文）
	SessionTTLHours int    `yaml:"session_ttl_hours"` // 会话有效期（小时），重启进程后会话清空需重登
	CookieSecure    bool   `yaml:"cookie_secure"`     // true 时会话 cookie 加 Secure（仅 HTTPS 下发送）；公网经 HTTPS 反代访问应设 true
}

type Config struct {
	Capture   Capture   `yaml:"capture"`
	Dashboard Dashboard `yaml:"dashboard"`
	Store     Store     `yaml:"store"`
	Log       Log       `yaml:"log"`
	Auth      Auth      `yaml:"auth"`
}

// Default 返回内置默认配置。
func Default() *Config {
	return &Config{
		Capture:   Capture{Addr: ":9100"},
		Dashboard: Dashboard{Addr: ":9101"},
		Store:     Store{Max: 500, Mode: SQLITE},
		Log:       Log{File: "data/logs/rawlens.log", MaxSizeMB: 10, MaxBackups: 5, MaxAgeDays: 14},
		Auth:      Auth{Enabled: false, SessionTTLHours: 168},
	}
}

// Load 从 path 读取配置并覆盖到默认值上。
// 返回的 bool 表示是否真的读到了文件（false = 文件不存在，用的默认值）。
func Load(path string) (*Config, bool, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, false, nil
		}
		return nil, false, fmt.Errorf("读取配置文件 %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, false, fmt.Errorf("解析配置文件 %s: %w", path, err)
	}
	if cfg.Store.Max <= 0 {
		cfg.Store.Max = 500
	}
	switch cfg.Store.Mode {
	case "":
		cfg.Store.Mode = SQLITE
	case SQLITE, MEMORY:
		// 合法取值
	default:
		return nil, false, fmt.Errorf("store.mode 取值非法 %q（仅支持 SQLITE / MEMORY）", cfg.Store.Mode)
	}
	// 日志落文件时（File 非空）滚动参数兜底，避免 0 值导致异常滚动行为。
	if cfg.Log.File != "" {
		if cfg.Log.MaxSizeMB <= 0 {
			cfg.Log.MaxSizeMB = 10
		}
		if cfg.Log.MaxBackups <= 0 {
			cfg.Log.MaxBackups = 5
		}
		if cfg.Log.MaxAgeDays <= 0 {
			cfg.Log.MaxAgeDays = 14
		}
	}
	// 开启鉴权时账号密码必填，否则既进不去也拦不住，属配置错误。
	if cfg.Auth.Enabled {
		if cfg.Auth.Username == "" || cfg.Auth.Password == "" {
			return nil, false, fmt.Errorf("auth.enabled 为 true 时必须配置 auth.username 与 auth.password")
		}
		if cfg.Auth.SessionTTLHours <= 0 {
			cfg.Auth.SessionTTLHours = 168
		}
	}
	return cfg, true, nil
}
