package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"os"

	"github.com/yuebai-blast/raw-lens/internal/capture"
	"github.com/yuebai-blast/raw-lens/internal/config"
	"github.com/yuebai-blast/raw-lens/internal/dashboard"
	"github.com/yuebai-blast/raw-lens/internal/store"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "YAML 配置文件路径")
	flag.Parse()

	cfg, loaded, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("配置: %v", err)
	}
	// 日志：始终打 stdout；配了 log.file 就再 tee 一份到文件（lumberjack 按大小滚动）。
	if cfg.Log.File != "" {
		lj := &lumberjack.Logger{
			Filename:   cfg.Log.File,
			MaxSize:    cfg.Log.MaxSizeMB,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAgeDays,
			Compress:   true,
		}
		defer lj.Close()
		log.SetOutput(io.MultiWriter(os.Stderr, lj))
	}
	if loaded {
		log.Printf("已加载配置文件 %s", *configPath)
	} else {
		log.Printf("未找到 %s，使用内置默认值", *configPath)
	}
	if cfg.Log.File != "" {
		log.Printf("日志：同时输出到 stdout 与 %s（按大小滚动，最多 %d 份 / %d 天）",
			cfg.Log.File, cfg.Log.MaxBackups, cfg.Log.MaxAgeDays)
	}

	// 存储模式映射到底层 store 的路径：MEMORY 用 ":memory:"，SQLITE 走默认文件 data/db/rawlens.db。
	storePath := config.DefaultDBPath
	if cfg.Store.Mode == config.MEMORY {
		storePath = ":memory:"
	}
	st, err := store.New(store.Options{Path: storePath, Max: cfg.Store.Max})
	if err != nil {
		log.Fatalf("存储: %v", err)
	}
	defer st.Close()
	if cfg.Store.Mode == config.MEMORY {
		log.Printf("存储：内存模式（进程重启即清空）")
	} else {
		log.Printf("存储：SQLite 文件 %s（最多保留 %d 条）", storePath, cfg.Store.Max)
	}

	var tlsConf *tls.Config
	if cfg.Capture.TLS.Enabled {
		tlsConf, err = capture.BuildTLSConfig(cfg.Capture.TLS.Cert, cfg.Capture.TLS.Key)
		if err != nil {
			log.Fatalf("TLS 配置: %v", err)
		}
		if cfg.Capture.TLS.Cert == "" {
			log.Printf("抓包端口使用内存自签名证书，客户端请用 curl -k / 浏览器跳过校验")
		}
	}

	go func() {
		if err := capture.Serve(cfg.Capture.Addr, st, tlsConf); err != nil {
			log.Fatalf("capture server: %v", err)
		}
	}()

	log.Printf("raw-lens 启动：抓包 %s（展示地址 %s），面板 http://localhost%s",
		cfg.Capture.Addr, cfg.CaptureURL(), cfg.Dashboard.Addr)
	if err := dashboard.Serve(cfg.Dashboard.Addr, st, cfg.Auth, cfg.CaptureURL()); err != nil {
		log.Fatalf("dashboard server: %v", err)
	}
}
