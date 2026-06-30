package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gosync/internal/config"
	"github.com/gosync/internal/server"
	"github.com/gosync/internal/watch"
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

	watcher := watch.New(hub, cfg.Watch)
	if err := watcher.Start(); err != nil {
		return err
	}

	var proxyTimeout time.Duration
	if cfg.ProxyTimeoutSecs != nil {
		proxyTimeout = time.Duration(*cfg.ProxyTimeoutSecs) * time.Second
	}

	srv := server.New(server.Config{
		Port:         cfg.Port,
		Dir:          cfg.Dir,
		Proxy:        cfg.Proxy,
		ProxyTimeout: proxyTimeout,
		TLSCert:      cfg.TLSCert,
		TLSKey:       cfg.TLSKey,
	}, hub)

	log.Printf("gosync starting (port=%s proxy=%q dir=%q watch=%v)", cfg.Port, cfg.Proxy, cfg.Dir, cfg.Watch)
	return srv.Start()
}

func main() {
	port := flag.String("port", "", "port to listen on (default 3001)")
	dir := flag.String("dir", "", "static files directory (default .)")
	proxyTarget := flag.String("proxy", "", "upstream proxy target (e.g. http://localhost:5173)")
	watchDirs := flag.String("watch", "", "comma-separated directories to watch (default .)")
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
	if *watchDirs != "" {
		parts := strings.Split(*watchDirs, ",")
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

func init() {
	// Silence usage of flag parse errors; main handles its own errors.
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: gosync [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
}
