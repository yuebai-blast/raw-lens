package config

import (
	"os"
	"testing"
)

func TestDefaultHasSQLitePath(t *testing.T) {
	cfg := Default()
	if cfg.Store.Path != "rawlens.db" {
		t.Fatalf("默认 path 应为 rawlens.db，得到 %q", cfg.Store.Path)
	}
	if cfg.Store.Max != 500 {
		t.Fatalf("默认 max 应为 500，得到 %d", cfg.Store.Max)
	}
}

func TestLoadFillsEmptyPath(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	if err := os.WriteFile(f, []byte("store:\n  max: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, loaded, err := Load(f)
	if err != nil || !loaded {
		t.Fatalf("Load 失败: loaded=%v err=%v", loaded, err)
	}
	if cfg.Store.Path != "rawlens.db" {
		t.Fatalf("空 path 应兜底为 rawlens.db，得到 %q", cfg.Store.Path)
	}
	if cfg.Store.Max != 10 {
		t.Fatalf("max 应被覆盖为 10，得到 %d", cfg.Store.Max)
	}
}
