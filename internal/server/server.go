package server

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/gosync/internal/clientjs"
	"github.com/gosync/internal/inject"
	"github.com/gosync/internal/proxy"
	"github.com/gosync/internal/ws"
)

type Config struct {
	Port      string
	Dir       string
	Proxy     string
	TLSCert   string
	TLSKey    string
	ProxyOpts proxy.Options
}

type Server struct {
	config Config
	hub    *ws.Hub
}

func New(config Config, hub *ws.Hub) *Server {
	return &Server{config: config, hub: hub}
}

func (s *Server) Start() error {
	var handler http.Handler

	if s.config.Proxy != "" {
		proxyHandler, err := proxy.NewWithOptions(s.config.Proxy, s.config.ProxyOpts)
		if err != nil {
			return err
		}
		handler = proxyHandler
	} else {
		handler = http.FileServer(http.Dir(s.config.Dir))
	}

	mux := http.NewServeMux()

	mux.Handle("/__bs/ws", http.HandlerFunc(s.hub.HandleWS))
	mux.Handle("/__bs.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(clientjs.ClientJS))
	}))

	mux.Handle("/", inject.Middleware(handler))

	addr := ":" + s.config.Port

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if s.config.TLSCert != "" && s.config.TLSKey != "" {
		log.Printf("gosync listening on https://%s (TLS enabled)", addr)
		httpServer.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		}
		return httpServer.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
	}

	log.Printf("gosync listening on http://%s", addr)
	return httpServer.ListenAndServe()
}
