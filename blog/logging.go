package blog

import (
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
)

// LogReqInfo describes the HTTP request.
type HTTPReqInfo struct {
	Method    string
	Uri       string
	Referer   string
	Ip        string
	Code      int
	Size      int64
	Duration  time.Duration
	UserAgent string
}

// ipAddrFromRemoteAddr removes the port from the address.
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// requestGetRemoteAddress returns the IP Address of the client making the
// request, taking into account HTTP proxies.
func requestGetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIP == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}

func (env *Env) logRequestHandler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ri := &HTTPReqInfo{
			Method:    r.Method,
			Uri:       r.URL.String(),
			Referer:   r.Header.Get("Referer"),
			UserAgent: r.Header.Get("User-Agent"),
		}

		ri.Ip = requestGetRemoteAddress(r)

		// this runs handler h and captures information about
		// HTTP request
		m := httpsnoop.CaptureMetrics(h, w, r)

		ri.Code = m.Code
		ri.Size = m.Written
		ri.Duration = m.Duration
		env.logHTTPReq(ri)
	}

	// http.HandlerFunc wraps a function so that it
	// implements http.Handler interface
	return http.HandlerFunc(fn)
}

func (env *Env) logHTTPReq(ri *HTTPReqInfo) {
	sugar := env.sugar
	if ri.Referer != "" {
		sugar = sugar.With("referer", ri.Referer)
	}
	sugar.Infow("http logging",
		"method", ri.Method,
		"uri", ri.Uri,
		"ip", ri.Ip,
		"code", ri.Code,
		"size", ri.Size,
		"duration", ri.Duration.Milliseconds(),
		"ua", ri.UserAgent,
	)
}
