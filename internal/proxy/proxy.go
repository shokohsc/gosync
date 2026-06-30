package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	Timeout            time.Duration
	ChangeOrigin       bool
	AutoRewrite        bool
	StripCookiesDomain bool
	RewriteLinks       bool
	RequestHeaders     map[string]string
	InsecureSkipVerify bool
}

type Proxy struct {
	target  *url.URL
	handler http.Handler
	opts    Options
}

func New(target string) (http.Handler, error) {
	return NewWithOptions(target, Options{})
}

func NewWithOptions(target string, opts Options) (http.Handler, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy target %q: %w", target, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("proxy target scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("proxy target must have a valid host")
	}

	p := &Proxy{
		target: u,
		opts:   opts,
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	if opts.Timeout > 0 || opts.InsecureSkipVerify {
		transport := &http.Transport{
			ResponseHeaderTimeout: opts.Timeout,
		}
		if opts.InsecureSkipVerify {
			if transport.TLSClientConfig == nil {
				transport.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
			} else {
				transport.TLSClientConfig.InsecureSkipVerify = true
			}
		}
		rp.Transport = transport
	}

	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)

		originalHost := req.Host

		if opts.ChangeOrigin {
			req.Host = u.Host
		}

		if clientIP := req.Header.Get("X-Forwarded-For"); clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP+", "+req.RemoteAddr)
		} else {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}

		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}

		if originalHost != "" {
			req.Header.Set("X-Forwarded-Host", originalHost)
		}

		for k, v := range opts.RequestHeaders {
			req.Header.Set(k, v)
		}
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	rp.ModifyResponse = func(resp *http.Response) error {
		if opts.AutoRewrite && isRedirect(resp.StatusCode) {
			loc := resp.Header.Get("Location")
			if loc != "" {
				locURL, err := url.Parse(loc)
				if err == nil && locURL.Host == u.Host {
					locURL.Host = resp.Request.Host
					if resp.Request.TLS != nil {
						locURL.Scheme = "https"
					} else {
						locURL.Scheme = "http"
					}
					resp.Header.Set("Location", locURL.String())
				}
			}
		}

		if opts.StripCookiesDomain {
			cookies := resp.Header.Values("Set-Cookie")
			if len(cookies) > 0 {
				resp.Header.Del("Set-Cookie")
				for _, c := range cookies {
					resp.Header.Add("Set-Cookie", stripCookieDomain(c))
				}
			}
		}

		if opts.RewriteLinks && isHTML(resp) && !isCompressed(resp) {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				return err
			}
			resp.Body.Close()

			body = p.rewriteLinks(body, resp.Request)

			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
			resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		}

		return nil
	}

	p.handler = rp

	return p, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.handler.ServeHTTP(w, r)
}

func isRedirect(code int) bool {
	return code == 301 || code == 302 || code == 307 || code == 308
}

func isHTML(resp *http.Response) bool {
	ct := resp.Header.Get("Content-Type")
	return strings.Contains(ct, "text/html")
}

func isCompressed(resp *http.Response) bool {
	ce := resp.Header.Get("Content-Encoding")
	return ce == "gzip" || ce == "deflate" || ce == "br"
}

func stripCookieDomain(c string) string {
	parts := strings.Split(c, ";")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "domain=") {
			continue
		}
		result = append(result, part)
	}
	return strings.Join(result, "; ")
}

func (p *Proxy) rewriteLinks(body []byte, req *http.Request) []byte {
	targetHost := p.target.Host
	reqHost := req.Host

	if targetHost == "" || reqHost == "" || targetHost == reqHost {
		return body
	}

	body = bytes.ReplaceAll(body, []byte("http://"+targetHost), []byte("http://"+reqHost))
	body = bytes.ReplaceAll(body, []byte("https://"+targetHost), []byte("https://"+reqHost))
	body = bytes.ReplaceAll(body, []byte("//"+targetHost), []byte("//"+reqHost))

	return body
}
