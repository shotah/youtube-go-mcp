package ytmusic

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultClientName    = "WEB_REMIX"
	defaultClientVersion = "1.20250326.01.00"
	defaultLanguage      = "en"
	defaultRegion        = "US"
	envHeadersPath       = "YTMUSIC_HEADERS_PATH"
	envClientVersion     = "YTMUSIC_CLIENT_VERSION"
	envMinIntervalMS     = "YTMUSIC_MIN_REQUEST_INTERVAL_MS"
	envMaxRetries        = "YTMUSIC_MAX_RETRIES"
)

// Client talks to YouTube Music InnerTube endpoints.
type Client struct {
	HTTPClient    *http.Client
	Language      string
	Region        string
	ClientName    string
	ClientVersion string
	Auth          *BrowserAuth
	Now           func() time.Time
	// Sleep is used for throttle/backoff delays; defaults to time.Sleep.
	Sleep func(time.Duration)

	// MinRequestInterval spaces requests to reduce 429s. Default 250ms.
	MinRequestInterval time.Duration
	// MaxRetries is how many times to retry on 429/503 after the first attempt. Default 3.
	MaxRetries int

	limiter *rateLimiter
}

// NewClient returns a Client with sensible defaults.
// If YTMUSIC_HEADERS_PATH is set, browser auth is loaded automatically.
func NewClient() *Client {
	c := &Client{
		HTTPClient:         &http.Client{Timeout: 30 * time.Second},
		Language:           defaultLanguage,
		Region:             defaultRegion,
		ClientName:         defaultClientName,
		ClientVersion:      defaultClientVersion,
		Now:                time.Now,
		Sleep:              time.Sleep,
		MinRequestInterval: defaultMinRequestInterval,
		MaxRetries:         defaultMaxRetries,
		limiter:            &rateLimiter{},
	}
	if v := os.Getenv(envClientVersion); v != "" {
		c.ClientVersion = v
	}
	if v := os.Getenv(envMinIntervalMS); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms >= 0 {
			c.MinRequestInterval = time.Duration(ms) * time.Millisecond
		}
	}
	if v := os.Getenv(envMaxRetries); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			c.MaxRetries = n
		}
	}
	if path := os.Getenv(envHeadersPath); path != "" {
		if auth, err := LoadAuthFromFile(path); err == nil {
			c.Auth = auth
		}
	}
	return c
}

// WithAuth returns a shallow copy using the given auth (shares the rate limiter).
func (c *Client) WithAuth(auth *BrowserAuth) *Client {
	cp := *c
	cp.Auth = auth
	if cp.limiter == nil {
		cp.limiter = &rateLimiter{}
	}
	return &cp
}

// Authenticated reports whether browser credentials are configured.
func (c *Client) Authenticated() bool {
	return c != nil && c.Auth != nil && c.Auth.Cookie != "" && c.Auth.SAPISID != ""
}

// Default is the package-level client used by convenience helpers.
var Default = NewClient()

// Package-level knobs kept for seed-client compatibility.
var (
	Language   = defaultLanguage
	Region     = defaultRegion
	HTTPClient = Default.HTTPClient
)

func syncDefaultGlobals() {
	Default.Language = Language
	Default.Region = Region
	Default.HTTPClient = HTTPClient
}
