package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoadConfigFileNotFound(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/gosync.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	f, err := os.CreateTemp("", "gosync-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("invalid: yaml: [unbalanced")
	f.Close()

	_, err = LoadConfig(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadConfigValidYAML(t *testing.T) {
	f, err := os.CreateTemp("", "gosync-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(`
port: "8080"
dir: /var/www
proxy: http://localhost:3000
watch:
  - src
  - assets
tls_cert: /certs/cert.pem
tls_key: /certs/key.pem
proxy_timeout_seconds: 30
hub_options:
  rate_limit_conns: 200
  max_msg_size_bytes: 8192
  pong_wait_seconds: 120
  write_wait_seconds: 5
`)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Port)
	}
	if cfg.Dir != "/var/www" {
		t.Errorf("expected dir /var/www, got %s", cfg.Dir)
	}
	if cfg.Proxy != "http://localhost:3000" {
		t.Errorf("expected proxy http://localhost:3000, got %s", cfg.Proxy)
	}
	if len(cfg.Watch) != 2 || cfg.Watch[0] != "src" || cfg.Watch[1] != "assets" {
		t.Errorf("expected watch [src assets], got %v", cfg.Watch)
	}
	if cfg.TLSCert != "/certs/cert.pem" {
		t.Errorf("expected tls_cert /certs/cert.pem, got %s", cfg.TLSCert)
	}
	if cfg.TLSKey != "/certs/key.pem" {
		t.Errorf("expected tls_key /certs/key.pem, got %s", cfg.TLSKey)
	}
	if cfg.ProxyTimeoutSecs == nil || *cfg.ProxyTimeoutSecs != 30 {
		t.Errorf("expected proxy_timeout_seconds 30, got %v", cfg.ProxyTimeoutSecs)
	}
	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 200 {
		t.Errorf("expected rate_limit_conns 200")
	}
	if cfg.HubOpts.MaxMsgSizeBytes == nil || *cfg.HubOpts.MaxMsgSizeBytes != 8192 {
		t.Errorf("expected max_msg_size_bytes 8192")
	}
	if cfg.HubOpts.PongWaitSecs == nil || *cfg.HubOpts.PongWaitSecs != 120 {
		t.Errorf("expected pong_wait_seconds 120")
	}
	if cfg.HubOpts.WriteWaitSecs == nil || *cfg.HubOpts.WriteWaitSecs != 5 {
		t.Errorf("expected write_wait_seconds 5")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	f, err := os.CreateTemp("", "gosync-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString("")
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "" {
		t.Errorf("expected empty port, got %s", cfg.Port)
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()
	if cfg.Port != "3001" {
		t.Errorf("expected port 3001, got %s", cfg.Port)
	}
	if cfg.Dir != "." {
		t.Errorf("expected dir '.', got %s", cfg.Dir)
	}
	if len(cfg.Watch) != 1 || cfg.Watch[0] != "." {
		t.Errorf("expected watch [.], got %v", cfg.Watch)
	}
}

func TestApplyDefaultsPartial(t *testing.T) {
	cfg := &Config{Port: "9090", Dir: "/app"}
	cfg.ApplyDefaults()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.Dir != "/app" {
		t.Errorf("expected dir /app, got %s", cfg.Dir)
	}
	if len(cfg.Watch) != 1 || cfg.Watch[0] != "." {
		t.Errorf("expected watch [.], got %v", cfg.Watch)
	}
}

func TestApplyDefaultsExistingWatch(t *testing.T) {
	cfg := &Config{Watch: []string{"src", "lib"}}
	cfg.ApplyDefaults()
	if len(cfg.Watch) != 2 || cfg.Watch[0] != "src" || cfg.Watch[1] != "lib" {
		t.Errorf("expected watch [src lib], got %v", cfg.Watch)
	}
}

func TestApplyEnvVarsPort(t *testing.T) {
	os.Setenv("GOSYNC_PORT", "5555")
	defer os.Unsetenv("GOSYNC_PORT")

	cfg := &Config{Port: "3001"}
	cfg.ApplyEnvVars()
	if cfg.Port != "5555" {
		t.Errorf("expected port 5555, got %s", cfg.Port)
	}
}

func TestApplyEnvVarsDir(t *testing.T) {
	os.Setenv("GOSYNC_DIR", "/app/www")
	defer os.Unsetenv("GOSYNC_DIR")

	cfg := &Config{Dir: "."}
	cfg.ApplyEnvVars()
	if cfg.Dir != "/app/www" {
		t.Errorf("expected dir /app/www, got %s", cfg.Dir)
	}
}

func TestApplyEnvVarsProxy(t *testing.T) {
	os.Setenv("GOSYNC_PROXY", "http://backend:8080")
	defer os.Unsetenv("GOSYNC_PROXY")

	cfg := &Config{Proxy: "http://old:3000"}
	cfg.ApplyEnvVars()
	if cfg.Proxy != "http://backend:8080" {
		t.Errorf("expected proxy http://backend:8080, got %s", cfg.Proxy)
	}
}

func TestApplyEnvVarsWatch(t *testing.T) {
	os.Setenv("GOSYNC_WATCH", "src, assets, src")
	defer os.Unsetenv("GOSYNC_WATCH")

	cfg := &Config{Watch: []string{"."}}
	cfg.ApplyEnvVars()
	if len(cfg.Watch) != 2 || cfg.Watch[0] != "src" || cfg.Watch[1] != "assets" {
		t.Errorf("expected watch [src assets] (deduped), got %v", cfg.Watch)
	}
}

func TestApplyEnvVarsTLSCert(t *testing.T) {
	os.Setenv("GOSYNC_TLS_CERT", "/etc/certs/cert.pem")
	defer os.Unsetenv("GOSYNC_TLS_CERT")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.TLSCert != "/etc/certs/cert.pem" {
		t.Errorf("expected TLS cert /etc/certs/cert.pem, got %s", cfg.TLSCert)
	}
}

func TestApplyEnvVarsTLSKey(t *testing.T) {
	os.Setenv("GOSYNC_TLS_KEY", "/etc/certs/key.pem")
	defer os.Unsetenv("GOSYNC_TLS_KEY")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.TLSKey != "/etc/certs/key.pem" {
		t.Errorf("expected TLS key /etc/certs/key.pem, got %s", cfg.TLSKey)
	}
}

func TestApplyEnvVarsProxyTimeout(t *testing.T) {
	os.Setenv("GOSYNC_PROXY_TIMEOUT_SECONDS", "45")
	defer os.Unsetenv("GOSYNC_PROXY_TIMEOUT_SECONDS")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.ProxyTimeoutSecs == nil || *cfg.ProxyTimeoutSecs != 45 {
		t.Errorf("expected proxy_timeout 45, got %v", cfg.ProxyTimeoutSecs)
	}
}

func TestApplyEnvVarsProxyTimeoutInvalid(t *testing.T) {
	os.Setenv("GOSYNC_PROXY_TIMEOUT_SECONDS", "not-a-number")
	defer os.Unsetenv("GOSYNC_PROXY_TIMEOUT_SECONDS")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.ProxyTimeoutSecs != nil {
		t.Errorf("expected nil proxy_timeout on invalid value, got %v", *cfg.ProxyTimeoutSecs)
	}
}

func TestApplyEnvVarsEmptyIgnored(t *testing.T) {
	os.Setenv("GOSYNC_PORT", "")
	defer os.Unsetenv("GOSYNC_PORT")

	cfg := &Config{Port: "3001"}
	cfg.ApplyEnvVars()
	if cfg.Port != "3001" {
		t.Errorf("expected port to remain 3001, got %s", cfg.Port)
	}
}

func TestApplyEnvVarsHubOptionsJSON(t *testing.T) {
	opts := HubOptions{RateLimitConns: intPtr(300), MaxMsgSizeBytes: intPtr(16384)}
	data, _ := json.Marshal(opts)
	os.Setenv("GOSYNC_HUB_OPTIONS", string(data))
	defer os.Unsetenv("GOSYNC_HUB_OPTIONS")

	cfg := &Config{HubOpts: HubOptions{
		RateLimitConns:  intPtr(100),
		PongWaitSecs:    intPtr(60),
	}}
	cfg.ApplyEnvVars()

	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 300 {
		t.Errorf("expected RateLimitConns 300, got %v", cfg.HubOpts.RateLimitConns)
	}
	if cfg.HubOpts.MaxMsgSizeBytes == nil || *cfg.HubOpts.MaxMsgSizeBytes != 16384 {
		t.Errorf("expected MaxMsgSizeBytes 16384, got %v", cfg.HubOpts.MaxMsgSizeBytes)
	}
	if cfg.HubOpts.PongWaitSecs == nil || *cfg.HubOpts.PongWaitSecs != 60 {
		t.Errorf("expected PongWaitSecs to remain 60, got %v", cfg.HubOpts.PongWaitSecs)
	}
}

func TestApplyEnvVarsHubOptionsInvalidJSON(t *testing.T) {
	os.Setenv("GOSYNC_HUB_OPTIONS", "not-json")
	defer os.Unsetenv("GOSYNC_HUB_OPTIONS")

	cfg := &Config{HubOpts: HubOptions{
		RateLimitConns: intPtr(100),
	}}
	cfg.ApplyEnvVars()

	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 100 {
		t.Errorf("expected RateLimitConns to remain 100 on invalid JSON, got %v", cfg.HubOpts.RateLimitConns)
	}
}

func TestApplyEnvVarsHubIndividualOverride(t *testing.T) {
	os.Setenv("GOSYNC_RATE_LIMIT_CONNS", "500")
	os.Setenv("GOSYNC_MAX_MSG_SIZE_BYTES", "8192")
	defer os.Unsetenv("GOSYNC_RATE_LIMIT_CONNS")
	defer os.Unsetenv("GOSYNC_MAX_MSG_SIZE_BYTES")

	cfg := &Config{HubOpts: HubOptions{
		RateLimitConns:  intPtr(100),
		MaxMsgSizeBytes: intPtr(4096),
	}}
	cfg.ApplyEnvVars()

	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 500 {
		t.Errorf("expected RateLimitConns 500, got %v", cfg.HubOpts.RateLimitConns)
	}
	if cfg.HubOpts.MaxMsgSizeBytes == nil || *cfg.HubOpts.MaxMsgSizeBytes != 8192 {
		t.Errorf("expected MaxMsgSizeBytes 8192, got %v", cfg.HubOpts.MaxMsgSizeBytes)
	}
}

func TestApplyEnvVarsHubIndividualOverridesJSON(t *testing.T) {
	opts := HubOptions{RateLimitConns: intPtr(300), MaxMsgSizeBytes: intPtr(16384)}
	data, _ := json.Marshal(opts)
	os.Setenv("GOSYNC_HUB_OPTIONS", string(data))
	os.Setenv("GOSYNC_RATE_LIMIT_CONNS", "999")
	defer os.Unsetenv("GOSYNC_HUB_OPTIONS")
	defer os.Unsetenv("GOSYNC_RATE_LIMIT_CONNS")

	cfg := &Config{HubOpts: HubOptions{
		PongWaitSecs: intPtr(60),
	}}
	cfg.ApplyEnvVars()

	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 999 {
		t.Errorf("expected RateLimitConns 999 (individual trumps JSON), got %v", cfg.HubOpts.RateLimitConns)
	}
	if cfg.HubOpts.MaxMsgSizeBytes == nil || *cfg.HubOpts.MaxMsgSizeBytes != 16384 {
		t.Errorf("expected MaxMsgSizeBytes 16384 from JSON, got %v", cfg.HubOpts.MaxMsgSizeBytes)
	}
	if cfg.HubOpts.PongWaitSecs == nil || *cfg.HubOpts.PongWaitSecs != 60 {
		t.Errorf("expected PongWaitSecs 60 from config, got %v", cfg.HubOpts.PongWaitSecs)
	}
}

func TestDefaultConfigPathEnvVar(t *testing.T) {
	os.Setenv("GOSYNC_CONFIG", "/custom/path/config.yaml")
	defer os.Unsetenv("GOSYNC_CONFIG")

	path := DefaultConfigPath()
	if path != "/custom/path/config.yaml" {
		t.Errorf("expected /custom/path/config.yaml, got %s", path)
	}
}

func TestDefaultConfigPathDefault(t *testing.T) {
	path := DefaultConfigPath()
	if path != ".gosync.yaml" {
		t.Errorf("expected .gosync.yaml, got %s", path)
	}
}

func TestApplyEnvVarsPingPongInterval(t *testing.T) {
	os.Setenv("GOSYNC_PING_PONG_INTERVAL_SECONDS", "30")
	defer os.Unsetenv("GOSYNC_PING_PONG_INTERVAL_SECONDS")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.HubOpts.PingPongIntervalSecs == nil || *cfg.HubOpts.PingPongIntervalSecs != 30 {
		t.Errorf("expected PingPongIntervalSecs 30, got %v", cfg.HubOpts.PingPongIntervalSecs)
	}
}

func TestApplyEnvVarsPongWait(t *testing.T) {
	os.Setenv("GOSYNC_PONG_WAIT_SECONDS", "90")
	defer os.Unsetenv("GOSYNC_PONG_WAIT_SECONDS")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.HubOpts.PongWaitSecs == nil || *cfg.HubOpts.PongWaitSecs != 90 {
		t.Errorf("expected PongWaitSecs 90, got %v", cfg.HubOpts.PongWaitSecs)
	}
}

func TestApplyEnvVarsWriteWait(t *testing.T) {
	os.Setenv("GOSYNC_WRITE_WAIT_SECONDS", "15")
	defer os.Unsetenv("GOSYNC_WRITE_WAIT_SECONDS")

	cfg := &Config{}
	cfg.ApplyEnvVars()
	if cfg.HubOpts.WriteWaitSecs == nil || *cfg.HubOpts.WriteWaitSecs != 15 {
		t.Errorf("expected WriteWaitSecs 15, got %v", cfg.HubOpts.WriteWaitSecs)
	}
}

func TestFullPipelineFileOverEnv(t *testing.T) {
	f, err := os.CreateTemp("", "gosync-*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(`
port: "8080"
dir: /from/file
proxy: http://file:3000
watch:
  - src
hub_options:
  rate_limit_conns: 200
`)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	os.Setenv("GOSYNC_PORT", "9090")
	os.Setenv("GOSYNC_DIR", "/from/env")
	os.Setenv("GOSYNC_RATE_LIMIT_CONNS", "999")
	defer os.Unsetenv("GOSYNC_PORT")
	defer os.Unsetenv("GOSYNC_DIR")
	defer os.Unsetenv("GOSYNC_RATE_LIMIT_CONNS")

	cfg.ApplyEnvVars()
	cfg.ApplyDefaults()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090 (env override), got %s", cfg.Port)
	}
	if cfg.Dir != "/from/env" {
		t.Errorf("expected dir /from/env (env override), got %s", cfg.Dir)
	}
	if cfg.Proxy != "http://file:3000" {
		t.Errorf("expected proxy http://file:3000 (file-only, no env), got %s", cfg.Proxy)
	}
	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 999 {
		t.Errorf("expected RateLimitConns 999 (env override), got %v", cfg.HubOpts.RateLimitConns)
	}
	if len(cfg.Watch) != 1 || cfg.Watch[0] != "src" {
		t.Errorf("expected watch [src] (file-only), got %v", cfg.Watch)
	}
}

func TestApplyEnvVarsIndividualBadValue(t *testing.T) {
	os.Setenv("GOSYNC_RATE_LIMIT_CONNS", "abc")
	defer os.Unsetenv("GOSYNC_RATE_LIMIT_CONNS")

	cfg := &Config{HubOpts: HubOptions{
		RateLimitConns: intPtr(100),
	}}
	cfg.ApplyEnvVars()
	if cfg.HubOpts.RateLimitConns == nil || *cfg.HubOpts.RateLimitConns != 100 {
		t.Errorf("expected RateLimitConns to remain 100 on bad value, got %v", cfg.HubOpts.RateLimitConns)
	}
}

func intPtr(n int) *int { return &n }
