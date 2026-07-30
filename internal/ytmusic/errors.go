package ytmusic

import "errors"

var (
	// ErrAuthRequired is returned when an authenticated endpoint is called without credentials.
	ErrAuthRequired = errors.New("ytmusic: authentication required (set YTMUSIC_OAUTH_PATH or YTMUSIC_HEADERS_PATH)")
	// ErrInvalidAuth is returned when credentials are present but unusable.
	ErrInvalidAuth = errors.New("ytmusic: invalid authentication credentials")
	// ErrSessionExpired is returned when InnerTube / OAuth rejects the session (401/403).
	ErrSessionExpired = errors.New("ytmusic: session expired or revoked — refresh OAuth (preferred) or re-export browser headers (see docs/auth.md)")
	// ErrRateLimited is returned after retries are exhausted on HTTP 429/503.
	ErrRateLimited = errors.New("ytmusic: rate limited by YouTube Music (HTTP 429/503) after retries")
	// ErrOAuthPending means the user has not finished the device consent screen yet.
	ErrOAuthPending = errors.New("ytmusic: oauth authorization_pending")
	// ErrOAuthSlowDown means the device-code poll interval should increase.
	ErrOAuthSlowDown = errors.New("ytmusic: oauth slow_down")
)

// AuthRefreshHint is short guidance for agents / operators when auth fails at runtime.
const AuthRefreshHint = "Prefer OAuth: ensure YTMUSIC_OAUTH_PATH + YTMUSIC_OAUTH_CLIENT_ID/SECRET are set (re-run: youtube-go-mcp auth oauth). Browser cookies: re-export headers.json and keep a dedicated minting profile — logging into YouTube in that profile often kills the jar."
