package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gosync/internal/config"
	"github.com/gosync/internal/proxy"
	"github.com/gosync/internal/server"
	"github.com/gosync/internal/ws"
)

func run(cfg *config.Config) error {
	hubOpts := ws.HubOptions{
		MaxConnections:      intVal(cfg.HubOpts.RateLimitConns),
		MaxMessageSizeBytes: intVal(cfg.HubOpts.MaxMsgSizeBytes),
	}
	if cfg.HubOpts.PongWaitSecs != nil {
		hubOpts.PongWait = time.Duration(*cfg.HubOpts.PongWaitSecs) * time.Second
	}
	if cfg.HubOpts.WriteWaitSecs != nil {
		hubOpts.WriteWait = time.Duration(*cfg.HubOpts.WriteWaitSecs) * time.Second
	}
	if cfg.HubOpts.PingPongIntervalSecs != nil {
		hubOpts.PingInterval = time.Duration(*cfg.HubOpts.PingPongIntervalSecs) * time.Second
	}

	hub := ws.NewHubWithOptions(hubOpts)

	var proxyTimeout time.Duration
	if cfg.ProxyTimeoutSecs != nil {
		proxyTimeout = time.Duration(*cfg.ProxyTimeoutSecs) * time.Second
	}

	srv := server.New(server.Config{
		Port:    cfg.Port,
		Dir:     cfg.Dir,
		Proxy:   cfg.Proxy,
		TLSCert: cfg.TLSCert,
		TLSKey:  cfg.TLSKey,
		ProxyOpts: proxy.Options{
			Timeout:            proxyTimeout,
			ChangeOrigin:       boolVal(cfg.ProxyChangeOrigin),
			AutoRewrite:        boolVal(cfg.ProxyAutoRewrite),
			StripCookiesDomain: boolVal(cfg.ProxyStripCookies),
			RewriteLinks:       boolVal(cfg.ProxyRewriteLinks),
			InsecureSkipVerify: boolVal(cfg.ProxyInsecure),
		},
	}, hub)

	log.Printf("gosync starting (port=%s proxy=%q dir=%q)", cfg.Port, cfg.Proxy, cfg.Dir)
	return srv.Start()
}

func main() {
	port := flag.String("port", "", "port to listen on (default 3001)")
	dir := flag.String("dir", "", "static files directory (default .)")
	proxyTarget := flag.String("proxy", "", "upstream proxy target (e.g. http://localhost:5173)")
	tlsCert := flag.String("tls-cert", "", "TLS certificate file path (enables HTTPS)")
	tlsKey := flag.String("tls-key", "", "TLS private key file path (enables HTTPS)")
	flag.Parse()

	cfgPath := config.DefaultConfigPath()
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cfg.ApplyEnvVars()

	if *port != "" {
		cfg.Port = *port
	}
	if *dir != "" {
		cfg.Dir = *dir
	}
	if *proxyTarget != "" {
		cfg.Proxy = *proxyTarget
	}
	if *tlsCert != "" {
		cfg.TLSCert = *tlsCert
	}
	if *tlsKey != "" {
		cfg.TLSKey = *tlsKey
	}

	cfg.ApplyDefaults()

	if err := run(cfg); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func intVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func boolVal(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gosync [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
}
