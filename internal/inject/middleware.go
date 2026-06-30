package inject

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

var bsScript = `<script src="/__bs.js"></script>`

type responseWriter struct {
	w      http.ResponseWriter
	buf    bytes.Buffer
	header http.Header
	code   int
}

func (rw *responseWriter) Header() http.Header {
	return rw.header
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.buf.Write(b)
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.code = code
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip injection for our own endpoints
		if r.URL.Path == "/__bs" || r.URL.Path == "/__bs.js" || strings.HasPrefix(r.URL.Path, "/__bs/") || r.URL.Path == "/__browser_sync__" {
			next.ServeHTTP(w, r)
			return
		}

		ct := r.Header.Get("Accept")
		if ct != "" && !strings.Contains(ct, "text/html") {
			acceptsHTML := false
			for _, accept := range strings.Split(ct, ",") {
				if strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*") {
					acceptsHTML = true
					break
				}
			}
			if !acceptsHTML {
				next.ServeHTTP(w, r)
				return
			}
		}

		rw := &responseWriter{
			w:      w,
			header: make(http.Header),
		}

		next.ServeHTTP(rw, r)

		respCT := rw.header.Get("Content-Type")

		// Normalize status code: if handler never called WriteHeader, default to 200
		code := rw.code
		if code == 0 {
			code = http.StatusOK
		}

		if strings.Contains(respCT, "text/html") && code == http.StatusOK {
			body := rw.buf.String()
			if idx := strings.LastIndex(body, "</body>"); idx != -1 {
				body = body[:idx] + bsScript + body[idx:]
			}

			for k, v := range rw.header {
				if k == "Content-Length" {
					continue
				}
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(code)
			io.WriteString(w, body)
		} else {
			for k, v := range rw.header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}
			w.WriteHeader(code)
			w.Write(rw.buf.Bytes())
		}
	})
}
