package ytmusic

import "errors"

var (
	// ErrAuthRequired is returned when an authenticated endpoint is called without credentials.
	ErrAuthRequired = errors.New("ytmusic: authentication required (set YOUTUBE_OAUTH_PATH + client id/secret)")
	// ErrInvalidAuth is returned when credentials are present but unusable.
	ErrInvalidAuth = errors.New("ytmusic: invalid authentication credentials")
	// ErrSessionExpired is returned when OAuth is rejected (401/403).
	ErrSessionExpired = errors.New("ytmusic: session expired or revoked — refresh OAuth (see docs/auth.md)")
	// ErrOAuthPending means the user has not finished the device consent screen yet.
	ErrOAuthPending = errors.New("ytmusic: oauth authorization_pending")
	// ErrOAuthSlowDown means the device-code poll interval should increase.
	ErrOAuthSlowDown = errors.New("ytmusic: oauth slow_down")
)

// AuthRefreshHint is short guidance for agents / operators when auth fails at runtime.
const AuthRefreshHint = "Prefer OAuth: set YOUTUBE_OAUTH_PATH + YOUTUBE_OAUTH_CLIENT_ID/SECRET (legacy YTMUSIC_OAUTH_* still works). Re-run: youtube-go-mcp auth oauth. Access tokens expire ~1h; refresh_token is the long-lived credential."
