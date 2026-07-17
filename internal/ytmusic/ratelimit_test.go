package ytmusic

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseRetryAfterSeconds(t *testing.T) {
	d, ok := parseRetryAfter("2", time.Unix(0, 0))
	if !ok || d != 2*time.Second {
		t.Fatalf("got %v ok=%v", d, ok)
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	future := now.Add(3 * time.Second).Format(http.TimeFormat)
	d, ok := parseRetryAfter(future, now)
	if !ok || d != 3*time.Second {
		t.Fatalf("got %v ok=%v", d, ok)
	}
}

func TestRetryDelayFallsBackToBackoff(t *testing.T) {
	d0 := retryDelay(nil, 0, time.Now())
	d1 := retryDelay(nil, 1, time.Now())
	if d0 != defaultBackoffBase {
		t.Fatalf("attempt0=%v", d0)
	}
	if d1 != 2*defaultBackoffBase {
		t.Fatalf("attempt1=%v", d1)
	}
}

func TestThrottleAndRetryOn429(t *testing.T) {
	var calls atomic.Int32
	var slept atomic.Int64
	client := NewClient()
	client.MinRequestInterval = 10 * time.Millisecond
	client.MaxRetries = 2
	client.Sleep = func(d time.Duration) {
		slept.Add(int64(d))
	}
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if n < 3 {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{"Retry-After": []string{"1"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"rate"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			}, nil
		}),
	}

	result, err := client.makeRequest("search", map[string]any{"query": "x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
	if slept.Load() < int64(time.Second) {
		t.Fatalf("expected Retry-After sleep, slept=%d", slept.Load())
	}
	m, ok := result.(map[string]any)
	if !ok || m["ok"] != true {
		t.Fatalf("result=%v", result)
	}
}

func TestRateLimitedAfterExhaustedRetries(t *testing.T) {
	client := NewClient()
	client.MinRequestInterval = 0
	client.MaxRetries = 1
	client.Sleep = func(time.Duration) {}
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`nope`)),
			}, nil
		}),
	}

	_, err := client.makeRequest("search", map[string]any{"query": "x"}, nil)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("got %v", err)
	}
}

func TestSessionExpiredOn401(t *testing.T) {
	client := NewClient()
	client.MinRequestInterval = 0
	client.MaxRetries = 0
	client.Auth = &BrowserAuth{
		Cookie:   "x=__Secure-3PAPISID=sap; other=1",
		AuthUser: "0",
		SAPISID:  "sap",
	}
	client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`auth`)),
			}, nil
		}),
	}

	_, err := client.makeRequest("browse", map[string]any{"browseId": "FEmusic_liked_playlists"}, nil)
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "docs/auth.md") && !strings.Contains(err.Error(), AuthRefreshHint[:20]) {
		t.Fatalf("missing refresh hint: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
