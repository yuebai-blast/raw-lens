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

type Store struct {
	Max int `yaml:"max"` // 内存里最多保留多少条请求
}

type Config struct {
	Capture   Capture   `yaml:"capture"`
	Dashboard Dashboard `yaml:"dashboard"`
	Store     Store     `yaml:"store"`
}

// Default 返回内置默认配置。
func Default() *Config {
	return &Config{
		Capture:   Capture{Addr: ":8080"},
		Dashboard: Dashboard{Addr: ":9090"},
		Store:     Store{Max: 500},
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
	return cfg, true, nil
}
