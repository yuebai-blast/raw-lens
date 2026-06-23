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

func TestDefaultAuthDisabled(t *testing.T) {
	cfg := Default()
	if cfg.Auth.Enabled {
		t.Fatalf("默认 auth 应关闭")
	}
	if cfg.Auth.SessionTTLHours != 168 {
		t.Fatalf("默认会话有效期应为 168 小时，得到 %d", cfg.Auth.SessionTTLHours)
	}
}

func TestLoadAuthEnabledRequiresCredentials(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	if err := os.WriteFile(f, []byte("auth:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(f); err == nil {
		t.Fatalf("auth.enabled=true 但缺账号密码时应报错")
	}
}

func TestLoadAuthEnabledOK(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	body := "auth:\n  enabled: true\n  username: admin\n  password: secret\n"
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(f)
	if err != nil {
		t.Fatalf("合法 auth 配置不应报错: %v", err)
	}
	if !cfg.Auth.Enabled || cfg.Auth.Username != "admin" || cfg.Auth.Password != "secret" {
		t.Fatalf("auth 字段未正确加载: %+v", cfg.Auth)
	}
	if cfg.Auth.SessionTTLHours != 168 {
		t.Fatalf("未配 ttl 应保留默认 168，得到 %d", cfg.Auth.SessionTTLHours)
	}
}

func TestLoadAuthTTLFallback(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/config.yaml"
	body := "auth:\n  enabled: true\n  username: a\n  password: b\n  session_ttl_hours: 0\n"
	if err := os.WriteFile(f, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load(f)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if cfg.Auth.SessionTTLHours != 168 {
		t.Fatalf("ttl<=0 应回落 168，得到 %d", cfg.Auth.SessionTTLHours)
	}
}

func TestCaptureURLWithDomain(t *testing.T) {
	cfg := Default()
	cfg.Capture.Addr = ":9100"
	cfg.Capture.Domain = "https://xxx.xx.com"
	if got := cfg.CaptureURL(); got != "https://xxx.xx.com:9100" {
		t.Fatalf("配域名应拼成 https://xxx.xx.com:9100，得到 %q", got)
	}
}

func TestCaptureURLFallbackHTTP(t *testing.T) {
	cfg := Default()
	cfg.Capture.Addr = ":9100"
	cfg.Capture.Domain = ""
	if got := cfg.CaptureURL(); got != "http://localhost:9100" {
		t.Fatalf("未配域名应回退 http://localhost:9100，得到 %q", got)
	}
}

func TestCaptureURLFallbackHTTPSWhenTLS(t *testing.T) {
	cfg := Default()
	cfg.Capture.Addr = "0.0.0.0:9443"
	cfg.Capture.Domain = ""
	cfg.Capture.TLS.Enabled = true
	if got := cfg.CaptureURL(); got != "https://localhost:9443" {
		t.Fatalf("开 TLS 未配域名应回退 https://localhost:9443，得到 %q", got)
	}
}
