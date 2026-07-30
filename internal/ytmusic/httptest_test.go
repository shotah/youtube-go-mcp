package ytmusic

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// rewriteMusicHost redirects music.youtube.com InnerTube calls to an httptest base.
type rewriteMusicHost struct {
	base string
	next http.RoundTripper
}

func (t rewriteMusicHost) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	base, err := url.Parse(strings.TrimRight(t.base, "/"))
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = base.Scheme
	req.URL.Host = base.Host
	req.Host = base.Host
	return next.RoundTrip(req)
}

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{
		HTTPClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: rewriteMusicHost{base: srv.URL},
		},
		Language:           defaultLanguage,
		Region:             defaultRegion,
		ClientName:         defaultClientName,
		ClientVersion:      defaultClientVersion,
		Now:                time.Now,
		Sleep:              func(time.Duration) {},
		MinRequestInterval: 0,
		MaxRetries:         0,
		limiter:            &rateLimiter{},
		authMu:             &sync.Mutex{},
		visitorID:          "test-visitor", // skip live music.youtube.com HTML fetch
	}
}

func withBrowserAuth(c *Client) *Client {
	c.Auth = &BrowserAuth{
		Cookie:   "VISITOR=1; __Secure-3PAPISID=sapisid-test",
		SAPISID:  "sapisid-test",
		AuthUser: "0",
	}
	return c
}

func readBody(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}
