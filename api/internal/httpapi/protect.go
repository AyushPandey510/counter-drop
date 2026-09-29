package httpapi

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"counter-drop/api/internal/ratelimit"
)

// --- security headers -------------------------------------------------------------------

// contentSecurityPolicy allows only this site's own scripts, styles, fonts and connections, plus the
// file-storage origin that phones upload to directly. No inline scripts, no framing, no plugins.
func (s *Server) contentSecurityPolicy() string {
	connect := []string{"'self'"}
	if o := originOf(s.Cfg.PublicAPIURL); o != "" {
		connect = append(connect, o)
	}
	if o := s.wsOrigin(); o != "" {
		connect = append(connect, o)
	}
	if st := s.Cfg.Storage; st.S3Enabled() {
		if o := originOf(st.Endpoint); o != "" {
			connect = append(connect, o)
		} else {
			// Native AWS S3: presigned URLs use the bucket's regional host.
			if region := st.Region; region != "" && region != "auto" {
				connect = append(connect, "https://"+st.Bucket+".s3."+region+".amazonaws.com", "https://s3."+region+".amazonaws.com")
			} else {
				connect = append(connect, "https://*.amazonaws.com")
			}
		}
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'", // React style attributes (progress bars)
		"img-src 'self' data: blob:",
		"font-src 'self' data:", // small font files are embedded in the CSS by the build
		"connect-src " + strings.Join(connect, " "),
		"worker-src 'self' blob:",
		"media-src 'self' blob:",
		"manifest-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}, "; ")
}

func originOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	csp := s.contentSecurityPolicy()
	prod := s.Cfg.Env == "prod"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		// Customer files (PDFs, photos) open in the browser's own viewer, which a CSP would break.
		// They can only be allowed file types, and nosniff stops a browser treating them as anything else.
		if !strings.HasPrefix(r.URL.Path, "/api/v1/cd/files/") {
			h.Set("Content-Security-Policy", csp)
		}
		if prod {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// --- rate limits ------------------------------------------------------------------------

type limitRule struct {
	name      string
	perMinute int
	burst     int
	match     func(r *http.Request) bool
}

const apiPrefix = "/api/v1/cd"

func pathIs(method, path string) func(*http.Request) bool {
	return func(r *http.Request) bool { return r.Method == method && r.URL.Path == apiPrefix+path }
}

// Per client IP. The first rule that matches applies; static files (the app itself) are never limited.
var limitRules = []limitRule{
	{"login", 10, 10, pathIs("POST", "/shop/login")},
	{"setup", 20, 10, func(r *http.Request) bool {
		return r.Method == "POST" && strings.HasPrefix(r.URL.Path, apiPrefix+"/shop/setup")
	}},
	{"staff-names", 30, 15, pathIs("GET", "/shop/staff-names")},
	{"create-job", 10, 6, func(r *http.Request) bool {
		return r.Method == "POST" && strings.HasPrefix(r.URL.Path, apiPrefix+"/shops/") && strings.HasSuffix(r.URL.Path, "/jobs")
	}},
	{"upload", 60, 30, func(r *http.Request) bool {
		return r.Method == "PUT" && strings.HasPrefix(r.URL.Path, apiPrefix+"/files/")
	}},
	{"events", 30, 10, func(r *http.Request) bool {
		return r.Method == "GET" && (strings.HasSuffix(r.URL.Path, "/events") || r.URL.Path == apiPrefix+"/ws")
	}},
	{"guest-write", 120, 60, func(r *http.Request) bool {
		return r.Method != "GET" && strings.HasPrefix(r.URL.Path, apiPrefix+"/jobs/")
	}},
	{"api", 600, 200, func(r *http.Request) bool { return strings.HasPrefix(r.URL.Path, "/api/") }},
}

// clientIP is the connecting address, or — behind a trusted proxy — the address that proxy saw
// (the last entry of X-Forwarded-For; earlier entries can be forged by the client).
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	if !s.Cfg.RateLimit {
		return next
	}
	if s.limiter == nil {
		s.limiter = ratelimit.New()
		go s.limiter.Run(time.Minute, nil)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		for _, rule := range limitRules {
			if !rule.match(r) {
				continue
			}
			ip := clientIP(r, s.Cfg.TrustProxy)
			if ok, wait := s.limiter.Allow(rule.name+"|"+ip, rule.perMinute, rule.burst); !ok {
				secs := int(math.Ceil(wait.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				s.Logger.Warn("rate limited", "rule", rule.name, "ip", ip, "path", r.URL.Path)
				failCode(w, http.StatusTooManyRequests, "rate_limited", fmt.Sprintf("Too many requests. Please wait %d seconds and try again.", secs))
				return
			}
			break
		}
		next.ServeHTTP(w, r)
	})
}
