package config

import (
	"os"
	"testing"
)

func TestDefaultUsesSQLiteMode(t *testing.T) {
	cfg := Default()
	if cfg.Store.Mode != SQLITE {
		t.Fatalf("默认 mode 应为 SQLITE，得到 %q", cfg.Store.Mode)
	}
	if cfg.Store.Max != 500 {
		t.Fatalf("默认 max 应为 500，得到 %d", cfg.Store.Max)
	}
}

func TestLoadFillsEmptyMode(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	if err := os.WriteFile(f, []byte("store:\n  max: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, loaded, err := Load(f)
	if err != nil || !loaded {
		t.Fatalf("Load 失败: loaded=%v err=%v", loaded, err)
	}
	if cfg.Store.Mode != SQLITE {
		t.Fatalf("空 mode 应兜底为 SQLITE，得到 %q", cfg.Store.Mode)
	}
	if cfg.Store.Max != 10 {
		t.Fatalf("max 应被覆盖为 10，得到 %d", cfg.Store.Max)
	}
}

func TestLoadAcceptsMemoryMode(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	if err := os.WriteFile(f, []byte("store:\n  mode: MEMORY\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(f)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Store.Mode != MEMORY {
		t.Fatalf("mode 应为 MEMORY，得到 %q", cfg.Store.Mode)
	}
}

func TestDefaultLogToFile(t *testing.T) {
	cfg := Default()
	if cfg.Log.File != "data/logs/rawlens.log" {
		t.Fatalf("默认日志文件应为 data/logs/rawlens.log，得到 %q", cfg.Log.File)
	}
	if cfg.Log.MaxAgeDays != 14 {
		t.Fatalf("默认保留天数应为 14，得到 %d", cfg.Log.MaxAgeDays)
	}
}

func TestLoadFillsLogRotationDefaults(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	// 只给文件路径，滚动参数留空 → 应兜底。
	if err := os.WriteFile(f, []byte("log:\n  file: \"x/y.log\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(f)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Log.MaxSizeMB != 10 || cfg.Log.MaxBackups != 5 || cfg.Log.MaxAgeDays != 14 {
		t.Fatalf("滚动参数兜底不对：%+v", cfg.Log)
	}
}

func TestLoadRejectsInvalidMode(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	if err := os.WriteFile(f, []byte("store:\n  mode: postgres\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(f); err == nil {
		t.Fatal("非法 mode 应返回错误，却成功了")
	}
}
