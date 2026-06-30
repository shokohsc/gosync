package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type HubOptions struct {
	RateLimitConns       *int `yaml:"rate_limit_conns" json:"RateLimitConns"`
	MaxMsgSizeBytes      *int `yaml:"max_msg_size_bytes" json:"MaxMsgSizeBytes"`
	PingPongIntervalSecs *int `yaml:"ping_pong_interval_seconds" json:"PingPongIntervalSeconds"`
	PongWaitSecs         *int `yaml:"pong_wait_seconds" json:"PongWaitSeconds"`
	WriteWaitSecs        *int `yaml:"write_wait_seconds" json:"WriteWaitSeconds"`
}

type Config struct {
	Port               string     `yaml:"port" json:"port"`
	Dir                string     `yaml:"dir" json:"dir"`
	Proxy              string     `yaml:"proxy" json:"proxy"`
	Watch              []string   `yaml:"watch" json:"watch"`
	TLSCert            string     `yaml:"tls_cert" json:"tls_cert"`
	TLSKey             string     `yaml:"tls_key" json:"tls_key"`
	ProxyTimeoutSecs   *int       `yaml:"proxy_timeout_seconds" json:"proxy_timeout_seconds"`
	HubOpts            HubOptions `yaml:"hub_options" json:"hub_options"`
}

func DefaultConfigPath() string {
	if p := os.Getenv("GOSYNC_CONFIG"); p != "" {
		return p
	}
	return ".gosync.yaml"
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func (cfg *Config) ApplyEnvVars() {
	if v, ok := lookupEnv("GOSYNC_PORT"); ok {
		cfg.Port = v
	}
	if v, ok := lookupEnv("GOSYNC_DIR"); ok {
		cfg.Dir = v
	}
	if v, ok := lookupEnv("GOSYNC_PROXY"); ok {
		cfg.Proxy = v
	}
	if v, ok := lookupEnv("GOSYNC_WATCH"); ok {
		parts := strings.Split(v, ",")
		var dirs []string
		seen := make(map[string]bool)
		for _, d := range parts {
			d = strings.TrimSpace(d)
			if d != "" && !seen[d] {
				dirs = append(dirs, d)
				seen[d] = true
			}
		}
		cfg.Watch = dirs
	}
	if v, ok := lookupEnv("GOSYNC_TLS_CERT"); ok {
		cfg.TLSCert = v
	}
	if v, ok := lookupEnv("GOSYNC_TLS_KEY"); ok {
		cfg.TLSKey = v
	}
	if v, ok := lookupEnv("GOSYNC_PROXY_TIMEOUT_SECONDS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ProxyTimeoutSecs = &n
		}
	}

	cfg.applyHubOptsEnv()
}

func (cfg *Config) applyHubOptsEnv() {
	if v, ok := lookupEnv("GOSYNC_HUB_OPTIONS"); ok {
		var envOpts HubOptions
		if err := json.Unmarshal([]byte(v), &envOpts); err == nil {
			cfg.HubOpts = mergePtrOpts(cfg.HubOpts, envOpts)
		}
	}

	if v, ok := lookupEnv("GOSYNC_RATE_LIMIT_CONNS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HubOpts.RateLimitConns = &n
		}
	}
	if v, ok := lookupEnv("GOSYNC_MAX_MSG_SIZE_BYTES"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HubOpts.MaxMsgSizeBytes = &n
		}
	}
	if v, ok := lookupEnv("GOSYNC_PING_PONG_INTERVAL_SECONDS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HubOpts.PingPongIntervalSecs = &n
		}
	}
	if v, ok := lookupEnv("GOSYNC_PONG_WAIT_SECONDS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HubOpts.PongWaitSecs = &n
		}
	}
	if v, ok := lookupEnv("GOSYNC_WRITE_WAIT_SECONDS"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.HubOpts.WriteWaitSecs = &n
		}
	}
}

func (cfg *Config) ApplyDefaults() {
	if cfg.Port == "" {
		cfg.Port = "3001"
	}
	if cfg.Dir == "" {
		cfg.Dir = "."
	}
	if len(cfg.Watch) == 0 {
		cfg.Watch = []string{"."}
	}
}

func lookupEnv(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func mergePtrOpts(base, override HubOptions) HubOptions {
	if override.RateLimitConns != nil {
		base.RateLimitConns = override.RateLimitConns
	}
	if override.MaxMsgSizeBytes != nil {
		base.MaxMsgSizeBytes = override.MaxMsgSizeBytes
	}
	if override.PingPongIntervalSecs != nil {
		base.PingPongIntervalSecs = override.PingPongIntervalSecs
	}
	if override.PongWaitSecs != nil {
		base.PongWaitSecs = override.PongWaitSecs
	}
	if override.WriteWaitSecs != nil {
		base.WriteWaitSecs = override.WriteWaitSecs
	}
	return base
}
