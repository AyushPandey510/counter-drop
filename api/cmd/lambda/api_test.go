package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/lambdahttp"

	"github.com/aws/aws-lambda-go/events"
)

const originSecret = "cloudfront-origin-secret-value"

type invoker func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	endpoint := os.Getenv("CD_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("CD_TEST_DYNAMODB_ENDPOINT not set")
	}
	cfg := config.Load()
	cfg.Env = "prod"
	cfg.Dynamo.Endpoint = endpoint
	cfg.Dynamo.Table = fmt.Sprintf("cd-lambda-%d", time.Now().UnixNano())
	cfg.Dynamo.Region = "ap-south-1"
	cfg.DemoSeed = true
	cfg.RateLimit = true
	cfg.ClientIPHeader = "CloudFront-Viewer-Address"
	cfg.OriginSecret = originSecret
	cfg.Realtime.WSURL = "wss://ws.counterdrop.cloudsuggest.in"
	cfg.Realtime.Key = strings.Repeat("k", 32)
	cfg.PublicAPIURL = "https://counterdrop.cloudsuggest.in"
	cfg.Storage.Dir = t.TempDir()
	cfg.Storage.SigningKey = "test-signing-key-1234567890"
	return cfg
}

// newInstance builds the API exactly as a Lambda cold start does (minus the AWS-only checks).
func newInstance(t *testing.T, cfg config.Config) invoker {
	t.Helper()
	srv, err := buildAPI(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if srv.SharedLimits == nil || !srv.NoSSE {
		t.Fatal("Lambda API must use shared limits and no SSE")
	}
	return lambdahttp.Handler(srv.Handler())
}

type call struct {
	method, path string
	body         []byte
	headers      map[string]string
	viewer       string // CloudFront-Viewer-Address
	noOrigin     bool
}

func invoke(t *testing.T, h invoker, c call) (int, map[string]json.RawMessage) {
	t.Helper()
	u, _ := url.Parse(c.path)
	ev := events.APIGatewayV2HTTPRequest{RawPath: u.Path, RawQueryString: u.RawQuery, Headers: map[string]string{"host": "counterdrop.cloudsuggest.in"}}
	ev.RequestContext.HTTP.Method = c.method
	ev.RequestContext.HTTP.SourceIP = "130.176.0.1" // a CloudFront edge, not the viewer
	if c.body != nil {
		ev.Body, ev.IsBase64Encoded = base64.StdEncoding.EncodeToString(c.body), true
		ev.Headers["content-type"] = "application/json"
	}
	if !c.noOrigin {
		ev.Headers["x-origin-verify"] = originSecret
	}
	viewer := c.viewer
	if viewer == "" {
		viewer = "203.0.113.10:51234"
	}
	ev.Headers["cloudfront-viewer-address"] = viewer
	for k, v := range c.headers {
		ev.Headers[strings.ToLower(k)] = v
	}
	res, err := h(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	body := res.Body
	if res.IsBase64Encoded {
		b, _ := base64.StdEncoding.DecodeString(body)
		body = string(b)
	}
	var out map[string]json.RawMessage
	_ = json.Unmarshal([]byte(body), &out)
	return res.StatusCode, out
}

func jsonBody(v any) []byte { b, _ := json.Marshal(v); return b }

func TestLambdaAPIFlow(t *testing.T) {
	cfg := testConfig(t)
	h := newInstance(t, cfg)

	if code, _ := invoke(t, h, call{method: "GET", path: "/health", noOrigin: true}); code != 403 {
		t.Fatalf("request that bypassed CloudFront: %d, want 403", code)
	}
	if code, _ := invoke(t, h, call{method: "GET", path: "/ready"}); code != 200 {
		t.Fatalf("/ready = %d", code)
	}

	// Customer flow through the adapter: create → upload (binary body) → complete → submit.
	code, out := invoke(t, h, call{method: "POST", path: "/api/v1/cd/shops/demo-print/jobs", body: jsonBody(map[string]any{
		"customerName": "Asha", "files": []map[string]any{{"clientId": "0", "filename": "a.pdf", "size": 120, "mime": "application/pdf"}}})})
	if code != 201 {
		t.Fatalf("create job: %d %s", code, out["error"])
	}
	var created struct {
		Ticket  struct{ Job struct{ ID string } }
		Secret  string
		Uploads []struct {
			FileID  string
			URL     string
			Headers map[string]string
		}
	}
	_ = json.Unmarshal(out["data"], &created)
	guest := map[string]string{"X-Ticket-Secret": created.Secret}
	up := created.Uploads[0]
	upURL, _ := url.Parse(up.URL)
	upHeaders := map[string]string{}
	for k, v := range up.Headers {
		upHeaders[k] = v
	}
	if code, out := invoke(t, h, call{method: "PUT", path: upURL.RequestURI(), body: []byte(strings.Repeat("x", 120)), headers: upHeaders}); code != 200 {
		t.Fatalf("upload: %d %s", code, out["error"])
	}
	jobPath := "/api/v1/cd/jobs/" + created.Ticket.Job.ID
	if code, out := invoke(t, h, call{method: "POST", path: jobPath + "/files/" + up.FileID + "/complete", body: jsonBody(map[string]int{"pages": 2}), headers: guest}); code != 200 {
		t.Fatalf("complete: %d %s", code, out["error"])
	}
	_, out = invoke(t, h, call{method: "GET", path: jobPath, headers: guest})
	var tk struct{ Quote struct{ PriceVersion string } }
	_ = json.Unmarshal(out["data"], &tk)
	if code, out := invoke(t, h, call{method: "POST", path: jobPath + "/submit", body: jsonBody(map[string]string{"priceVersion": tk.Quote.PriceVersion}), headers: guest}); code != 200 {
		t.Fatalf("submit: %d %s", code, out["error"])
	}

	// Live updates: WebSocket ticket, and the SSE endpoint is off.
	code, out = invoke(t, h, call{method: "POST", path: jobPath + "/live", headers: guest})
	var live struct{ Mode, URL, Ticket string }
	_ = json.Unmarshal(out["data"], &live)
	if code != 200 || live.Mode != "ws" || live.URL != cfg.Realtime.WSURL || live.Ticket == "" {
		t.Fatalf("live ticket: %d %+v", code, live)
	}
	if code, _ := invoke(t, h, call{method: "GET", path: jobPath + "/events?secret=" + created.Secret}); code != 404 {
		t.Fatalf("SSE on Lambda: %d, want 404", code)
	}
}

// Two Lambda instances must share the sign-in limit (DynamoDB), keyed by the viewer's address from
// CloudFront — not by the edge address API Gateway sees.
func TestLambdaSharedLoginLimit(t *testing.T) {
	cfg := testConfig(t)
	a, b := newInstance(t, cfg), newInstance(t, cfg)
	// A name that doesn't exist: every try fails without triggering the per-account PIN lockout.
	login := jsonBody(map[string]string{"shop": "demo-print", "name": "Nobody", "pin": "9999"})
	for i := 0; i < 10; i++ {
		h := a
		if i%2 == 1 {
			h = b
		}
		if code, _ := invoke(t, h, call{method: "POST", path: "/api/v1/cd/shop/login", body: login, viewer: "203.0.113.50:1111"}); code == 429 {
			t.Fatalf("limited too early at attempt %d", i+1)
		}
	}
	if code, _ := invoke(t, b, call{method: "POST", path: "/api/v1/cd/shop/login", body: login, viewer: "203.0.113.50:2222"}); code != 429 {
		t.Fatalf("11th sign-in across two instances: %d, want 429", code)
	}
	// Another viewer behind the same CloudFront edge is unaffected.
	if code, _ := invoke(t, a, call{method: "POST", path: "/api/v1/cd/shop/login", body: login, viewer: "[2001:db8::7]:3333"}); code == 429 {
		t.Fatal("a different viewer was limited")
	}
}

func TestLambdaConfigChecks(t *testing.T) {
	ok := config.Config{}
	ok.Realtime.WSURL, ok.Realtime.Key, ok.Storage.Bucket = "wss://ws.counterdrop.cloudsuggest.in", strings.Repeat("k", 32), "counter-drop-prod-files"
	if err := checkAPIConfig(ok); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}
	for name, mut := range map[string]func(*config.Config){
		"sse":       func(c *config.Config) { c.Realtime.WSURL = "" },
		"local ws":  func(c *config.Config) { c.Realtime.WSURL = "local" },
		"short key": func(c *config.Config) { c.Realtime.Key = "short" },
		"no bucket": func(c *config.Config) { c.Storage.Bucket = "" },
		"endpoint":  func(c *config.Config) { c.Dynamo.Endpoint = "http://localhost:8000" },
	} {
		c := ok
		mut(&c)
		if err := checkAPIConfig(c); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

// The scheduled sweeper deletes a removed file's bytes (and verifies they're gone).
func TestLambdaSweeperDeletesDueFiles(t *testing.T) {
	cfg := testConfig(t)
	h := newInstance(t, cfg)
	code, out := invoke(t, h, call{method: "POST", path: "/api/v1/cd/shops/demo-print/jobs", body: jsonBody(map[string]any{
		"customerName": "Ravi", "files": []map[string]any{{"clientId": "0", "filename": "a.pdf", "size": 64, "mime": "application/pdf"}}})})
	if code != 201 {
		t.Fatalf("create job: %d", code)
	}
	var created struct {
		Ticket  struct{ Job struct{ ID string } }
		Secret  string
		Uploads []struct {
			FileID  string
			URL     string
			Headers map[string]string
		}
	}
	_ = json.Unmarshal(out["data"], &created)
	up := created.Uploads[0]
	upURL, _ := url.Parse(up.URL)
	if code, _ := invoke(t, h, call{method: "PUT", path: upURL.RequestURI(), body: []byte(strings.Repeat("y", 64)), headers: up.Headers}); code != 200 {
		t.Fatalf("upload: %d", code)
	}
	countFiles := func() int {
		n := 0
		_ = filepathWalk(cfg.Storage.Dir, func() { n++ })
		return n
	}
	if countFiles() == 0 {
		t.Fatal("upload not stored")
	}
	guest := map[string]string{"X-Ticket-Secret": created.Secret}
	if code, _ := invoke(t, h, call{method: "DELETE", path: "/api/v1/cd/jobs/" + created.Ticket.Job.ID + "/files/" + up.FileID, headers: guest}); code != 200 {
		t.Fatalf("remove file: %d", code)
	}
	d, err := buildSweeper(context.Background(), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("sweeper run: %v", err)
	}
	if n := countFiles(); n != 0 {
		t.Fatalf("%d stored file(s) left after the sweeper ran", n)
	}
}

func filepathWalk(dir string, onFile func()) error {
	return filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && !strings.HasSuffix(d.Name(), ".type") {
			onFile()
		}
		return err
	})
}
