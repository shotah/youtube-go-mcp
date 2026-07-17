package ytmusic

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	// AuthPath is the headers.json path to auto-reload when the file's mtime changes.
	AuthPath string
	Now      func() time.Time
	// Sleep is used for throttle/backoff delays; defaults to time.Sleep.
	Sleep func(time.Duration)

	// MinRequestInterval spaces requests to reduce 429s. Default 250ms.
	MinRequestInterval time.Duration
	// MaxRetries is how many times to retry on 429/503 after the first attempt. Default 3.
	MaxRetries int

	limiter     *rateLimiter
	authMu      *sync.Mutex // pointer so Client value-copies (withLanguage/etc.) stay valid
	authModTime time.Time
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
		authMu:             &sync.Mutex{},
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
		_ = c.SetAuthPath(path)
	}
	return c
}

func (c *Client) ensureAuthMu() {
	if c.authMu == nil {
		c.authMu = &sync.Mutex{}
	}
}

// SetAuthPath configures the headers file used for browser auth and loads it immediately.
// Subsequent requests reload from disk when the file's modification time changes.
func (c *Client) SetAuthPath(path string) error {
	if c == nil {
		return ErrInvalidAuth
	}
	c.ensureAuthMu()
	c.authMu.Lock()
	defer c.authMu.Unlock()
	c.AuthPath = path
	return c.reloadAuthLocked(true)
}

// WithAuth returns a shallow copy using the given auth (shares the rate limiter).
// AuthPath is cleared so the copy does not keep reloading the previous file.
func (c *Client) WithAuth(auth *BrowserAuth) *Client {
	cp := *c
	cp.Auth = auth
	cp.AuthPath = ""
	cp.authModTime = time.Time{}
	cp.authMu = &sync.Mutex{}
	if cp.limiter == nil {
		cp.limiter = &rateLimiter{}
	}
	return &cp
}

// Authenticated reports whether browser credentials are configured.
func (c *Client) Authenticated() bool {
	if c == nil {
		return false
	}
	c.maybeReloadAuth()
	return c.Auth != nil && c.Auth.Cookie != "" && c.Auth.SAPISID != ""
}

// maybeReloadAuth reloads Auth from AuthPath when the file mtime advances.
func (c *Client) maybeReloadAuth() {
	if c == nil || c.AuthPath == "" {
		return
	}
	c.ensureAuthMu()
	c.authMu.Lock()
	defer c.authMu.Unlock()
	_ = c.reloadAuthLocked(false)
}

// reloadAuthIfChanged reloads when mtime changed. Returns true if Auth was replaced.
func (c *Client) reloadAuthIfChanged() bool {
	if c == nil || c.AuthPath == "" {
		return false
	}
	c.ensureAuthMu()
	c.authMu.Lock()
	defer c.authMu.Unlock()
	before := c.Auth
	beforeMod := c.authModTime
	_ = c.reloadAuthLocked(false)
	return c.Auth != before || !c.authModTime.Equal(beforeMod)
}

func (c *Client) reloadAuthLocked(force bool) error {
	path := c.AuthPath
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: stat %s: %w", ErrInvalidAuth, path, err)
	}
	mod := info.ModTime()
	if !force && c.Auth != nil && !c.authModTime.IsZero() && !mod.After(c.authModTime) {
		return nil
	}
	auth, err := LoadAuthFromFile(path)
	if err != nil {
		// Advance mtime so a broken rewrite does not retry on every request.
		c.authModTime = mod
		return err
	}
	c.Auth = auth
	c.authModTime = mod
	return nil
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
