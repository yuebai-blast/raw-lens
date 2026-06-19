package main

import (
	"crypto/tls"
	"flag"
	"log"

	"github.com/yuebai-blast/raw-lens/internal/capture"
	"github.com/yuebai-blast/raw-lens/internal/config"
	"github.com/yuebai-blast/raw-lens/internal/dashboard"
	"github.com/yuebai-blast/raw-lens/internal/store"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "YAML 配置文件路径")
	flag.Parse()

	cfg, loaded, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("配置: %v", err)
	}
	if loaded {
		log.Printf("已加载配置文件 %s", *configPath)
	} else {
		log.Printf("未找到 %s，使用内置默认值", *configPath)
	}

	st, err := store.New(store.Options{Path: cfg.Store.Path, Max: cfg.Store.Max})
	if err != nil {
		log.Fatalf("存储: %v", err)
	}
	defer st.Close()
	if cfg.Store.Path == ":memory:" {
		log.Printf("存储：内存模式（进程重启即清空）")
	} else {
		log.Printf("存储：SQLite 文件 %s（最多保留 %d 条）", cfg.Store.Path, cfg.Store.Max)
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

	log.Printf("raw-lens 启动：抓包 %s，面板 http://localhost%s", cfg.Capture.Addr, cfg.Dashboard.Addr)
	if err := dashboard.Serve(cfg.Dashboard.Addr, st); err != nil {
		log.Fatalf("dashboard server: %v", err)
	}
}
