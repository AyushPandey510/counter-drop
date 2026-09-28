package httpapi_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	e := setup(t)
	res, err := http.Get(e.url + "/api/v1/cd/shops/demo-print")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	h := res.Header
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	csp := h.Get("Content-Security-Policy")
	for _, part := range []string{"default-src 'self'", "script-src 'self'", "frame-ancestors 'none'", "object-src 'none'"} {
		if !strings.Contains(csp, part) {
			t.Errorf("CSP missing %q: %s", part, csp)
		}
	}
	if !strings.Contains(h.Get("Permissions-Policy"), "camera=(self)") {
		t.Errorf("camera must stay allowed for the in-app scanner: %q", h.Get("Permissions-Policy"))
	}
}

func TestRateLimits(t *testing.T) {
	rateLimitInTests = true
	defer func() { rateLimitInTests = false }()
	e := setup(t)

	login := func(ip string) int {
		req, _ := http.NewRequest("POST", e.url+"/api/v1/cd/shop/login", strings.NewReader(`{"shop":"demo-print","name":"Nobody","pin":"0000"}`))
		req.Header.Set("Content-Type", "application/json")
		if ip != "" {
			req.Header.Set("X-Forwarded-For", ip)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode == 429 && res.Header.Get("Retry-After") == "" {
			t.Fatal("429 without Retry-After")
		}
		return res.StatusCode
	}
	// 10 tries allowed in a burst, the 11th is refused.
	for i := 0; i < 10; i++ {
		if s := login(""); s != 401 {
			t.Fatalf("attempt %d: %d", i+1, s)
		}
	}
	if s := login(""); s != 429 {
		t.Fatalf("11th attempt: %d, want 429", s)
	}
	// Other routes have their own budgets: reading a shop still works.
	res, _ := http.Get(e.url + "/api/v1/cd/shops/demo-print")
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("shop page limited too: %d", res.StatusCode)
	}
	// Without CD_TRUST_PROXY a forged X-Forwarded-For doesn't reset the budget.
	if s := login("198.51.100.7"); s != 429 {
		t.Fatalf("forged X-Forwarded-For bypassed the limit: %d", s)
	}
}
