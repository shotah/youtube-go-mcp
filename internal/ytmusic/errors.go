package ytmusic

import "errors"

var (
	// ErrAuthRequired is returned when an authenticated endpoint is called without credentials.
	ErrAuthRequired = errors.New("ytmusic: authentication required (set YTMUSIC_HEADERS_PATH or pass BrowserAuth)")
	// ErrInvalidAuth is returned when headers/cookies are present but unusable.
	ErrInvalidAuth = errors.New("ytmusic: invalid authentication headers (missing cookie or SAPISID)")
	// ErrSessionExpired is returned when InnerTube rejects the browser session (401/403).
	ErrSessionExpired = errors.New("ytmusic: browser session expired or revoked — re-export headers from music.youtube.com (see docs/auth.md)")
	// ErrRateLimited is returned after retries are exhausted on HTTP 429/503.
	ErrRateLimited = errors.New("ytmusic: rate limited by YouTube Music (HTTP 429/503) after retries")
)

// AuthRefreshHint is short guidance for agents / operators when auth fails at runtime.
const AuthRefreshHint = "Re-run: youtube-go-mcp auth --out headers.json (or replace the mounted headers file). The MCP reloads that file when its mtime changes; restart only if the process still has no AuthPath. Do not log out of music.youtube.com in the browser that minted the session unless you intend to invalidate it."
