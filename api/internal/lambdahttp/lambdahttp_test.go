package lambdahttp

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func event(method, path, query, body string, b64 bool, headers map[string]string) events.APIGatewayV2HTTPRequest {
	e := events.APIGatewayV2HTTPRequest{RawPath: path, RawQueryString: query, Body: body, IsBase64Encoded: b64, Headers: headers}
	e.RequestContext.HTTP.Method = method
	e.RequestContext.HTTP.SourceIP = "198.51.100.7"
	e.RequestContext.RequestID = "req-123"
	return e
}

func TestRequestAndResponse(t *testing.T) {
	var seen *http.Request
	var seenBody string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r
		b, _ := io.ReadAll(r.Body)
		seenBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Add("Set-Cookie", "a=1")
		w.Header().Add("Set-Cookie", "b=2")
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Accept")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{}}`))
	})
	res, err := Handler(h)(context.Background(), event("POST", "/api/v1/cd/jobs/j1/submit", "a=1&b=two", base64.StdEncoding.EncodeToString([]byte(`{"x":1}`)), true,
		map[string]string{"host": "counterdrop.cloudsuggest.in", "x-ticket-secret": "s3cret", "content-type": "application/json"}))
	if err != nil {
		t.Fatal(err)
	}
	if seen.Method != "POST" || seen.URL.Path != "/api/v1/cd/jobs/j1/submit" || seen.URL.Query().Get("b") != "two" {
		t.Fatalf("request line: %s %s", seen.Method, seen.URL)
	}
	if seenBody != `{"x":1}` || seen.Header.Get("X-Ticket-Secret") != "s3cret" || seen.Host != "counterdrop.cloudsuggest.in" {
		t.Fatalf("body/headers: %q %v %q", seenBody, seen.Header, seen.Host)
	}
	if seen.RemoteAddr != "198.51.100.7:0" || seen.Header.Get("X-Request-ID") != "req-123" {
		t.Fatalf("remote %q id %q", seen.RemoteAddr, seen.Header.Get("X-Request-ID"))
	}
	if res.StatusCode != 201 || res.Body != `{"data":{}}` || res.IsBase64Encoded {
		t.Fatalf("response: %+v", res)
	}
	if len(res.Cookies) != 2 || res.Headers["Vary"] != "Origin, Accept" {
		t.Fatalf("headers: %v cookies %v", res.Headers, res.Cookies)
	}
}

func TestBinaryAndDefaults(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte{0x25, 0x50, 0x44, 0x46, 0x00, 0xff})
	})
	res, _ := Handler(h)(context.Background(), event("GET", "/x", "", "", false, nil))
	if res.StatusCode != 200 || !res.IsBase64Encoded {
		t.Fatalf("binary: %+v", res)
	}
	if b, _ := base64.StdEncoding.DecodeString(res.Body); len(b) != 6 || b[5] != 0xff {
		t.Fatalf("binary body %v", b)
	}
	empty, _ := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))(context.Background(), event("DELETE", "/x", "", "", false, nil))
	if empty.StatusCode != 204 || empty.Body != "" {
		t.Fatalf("empty: %+v", empty)
	}
	bad, _ := Handler(h)(context.Background(), event("POST", "/x", "", "!!not base64", true, nil))
	if bad.StatusCode != 400 {
		t.Fatalf("bad base64: %d", bad.StatusCode)
	}
}
