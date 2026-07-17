package ytmusic

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMinRequestInterval = 250 * time.Millisecond
	defaultMaxRetries         = 3
	defaultBackoffBase        = 1 * time.Second
	defaultBackoffMax         = 16 * time.Second
)

// rateLimiter spaces outbound InnerTube calls and supports retry backoff.
type rateLimiter struct {
	mu      sync.Mutex
	lastReq time.Time
}

func (c *Client) throttle() {
	interval := c.MinRequestInterval
	if interval <= 0 {
		interval = defaultMinRequestInterval
	}
	sleep := c.sleepFunc()
	if c.limiter == nil {
		c.limiter = &rateLimiter{}
	}

	c.limiter.mu.Lock()
	var wait time.Duration
	if !c.limiter.lastReq.IsZero() {
		wait = interval - c.nowFunc().Sub(c.limiter.lastReq)
	}
	c.limiter.mu.Unlock()

	if wait > 0 {
		sleep(wait)
	}

	c.limiter.mu.Lock()
	c.limiter.lastReq = c.nowFunc()
	c.limiter.mu.Unlock()
}

func (c *Client) nowFunc() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) sleepFunc() func(time.Duration) {
	if c != nil && c.Sleep != nil {
		return c.Sleep
	}
	return time.Sleep
}

func (c *Client) maxRetries() int {
	if c != nil && c.MaxRetries > 0 {
		return c.MaxRetries
	}
	return defaultMaxRetries
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable
}

func isAuthFailureStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

// retryDelay picks how long to wait before the next attempt.
// Prefer Retry-After when present; otherwise exponential backoff from attempt index (0-based).
func retryDelay(resp *http.Response, attempt int, now time.Time) time.Duration {
	if resp != nil {
		if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), now); ok {
			return d
		}
	}
	mult := math.Pow(2, float64(attempt))
	d := time.Duration(float64(defaultBackoffBase) * mult)
	return min(d, defaultBackoffMax)
}

func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0, false
		}
		d := time.Duration(secs) * time.Second
		return min(d, defaultBackoffMax), true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := t.Sub(now)
		if d < 0 {
			return 0, true
		}
		return min(d, defaultBackoffMax), true
	}
	return 0, false
}
