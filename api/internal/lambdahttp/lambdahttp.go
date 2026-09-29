// Package lambdahttp runs an ordinary http.Handler behind API Gateway HTTP APIs (payload format 2.0),
// so the Lambda serves exactly the same routes, middleware and tests as the long-running server.
package lambdahttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"mime"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Handler adapts h to the Lambda handler signature for an HTTP API.
func Handler(h http.Handler) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		r, err := Request(ctx, req)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusBadRequest, Body: `{"error":{"code":"bad_request","message":"Malformed request."}}`,
				Headers: map[string]string{"Content-Type": "application/json"}}, nil
		}
		w := newRecorder()
		h.ServeHTTP(w, r)
		return w.response(), nil
	}
}

// Request builds the *http.Request for an API Gateway event.
func Request(ctx context.Context, req events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	body := []byte(req.Body)
	if req.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, err
		}
		body = b
	}
	path := req.RawPath
	if path == "" {
		path = req.RequestContext.HTTP.Path
	}
	url := path
	if req.RawQueryString != "" {
		url += "?" + req.RawQueryString
	}
	method := req.RequestContext.HTTP.Method
	r, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		// API Gateway joins repeated headers with commas, which is also valid HTTP.
		r.Header.Set(k, v)
	}
	if len(req.Cookies) > 0 {
		r.Header.Set("Cookie", strings.Join(req.Cookies, "; "))
	}
	r.Host = req.Headers["host"]
	r.RequestURI = url
	r.ContentLength = int64(len(body))
	// The address API Gateway saw. Behind CloudFront that is the edge location; the viewer's address
	// comes from CloudFront-Viewer-Address (CD_CLIENT_IP_HEADER).
	r.RemoteAddr = req.RequestContext.HTTP.SourceIP + ":0"
	if id := req.RequestContext.RequestID; id != "" && r.Header.Get("X-Request-ID") == "" {
		r.Header.Set("X-Request-ID", id)
	}
	return r, nil
}

type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newRecorder() *recorder { return &recorder{header: http.Header{}} }

func (w *recorder) Header() http.Header { return w.header }

func (w *recorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(b)
}

func (w *recorder) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *recorder) response() events.APIGatewayV2HTTPResponse {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	out := events.APIGatewayV2HTTPResponse{StatusCode: w.status, Headers: map[string]string{}}
	for k, vs := range w.header {
		if http.CanonicalHeaderKey(k) == "Set-Cookie" {
			out.Cookies = append(out.Cookies, vs...)
			continue
		}
		out.Headers[k] = strings.Join(vs, ", ")
	}
	if isText(w.header.Get("Content-Type")) || w.body.Len() == 0 {
		out.Body = w.body.String()
	} else {
		out.Body = base64.StdEncoding.EncodeToString(w.body.Bytes())
		out.IsBase64Encoded = true
	}
	return out
}

func isText(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mt, "text/") || mt == "application/json" || mt == "application/javascript" ||
		strings.HasSuffix(mt, "+json") || strings.HasSuffix(mt, "+xml") || mt == "application/xml"
}
