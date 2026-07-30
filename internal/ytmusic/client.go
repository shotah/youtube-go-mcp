package ytmusic

import (
	"net/http"
	"strings"
	"time"
)

// Client holds OAuth credentials for YouTube Data API access (via internal/youtube).
// InnerTube / browser-cookie auth has been removed.
type Client struct {
	HTTPClient *http.Client
	OAuth      *OAuthSession
	Now        func() time.Time
}

// NewClient returns a Client and loads OAuth from YOUTUBE_OAUTH_* (or legacy YTMUSIC_*) when set.
func NewClient() *Client {
	c := &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Now:        time.Now,
	}
	if path := oauthPathFromEnv(); path != "" {
		_ = c.SetOAuthPath(path, oauthClientIDFromEnv(), oauthClientSecretFromEnv())
	}
	return c
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// SetOAuthPath loads oauth.json and configures refresh with the given client credentials.
func (c *Client) SetOAuthPath(path, clientID, clientSecret string) error {
	if c == nil {
		return ErrInvalidAuth
	}
	creds := OAuthCredentials{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		HTTPClient:   c.HTTPClient,
	}
	if err := creds.validate(); err != nil {
		return err
	}
	tok, err := LoadOAuthTokenFromFile(path)
	if err != nil {
		return err
	}
	c.OAuth = &OAuthSession{
		Credentials: creds,
		Token:       tok,
		Path:        path,
		Now:         c.Now,
	}
	return nil
}

// WithOAuth returns a shallow copy using the given OAuth session.
func (c *Client) WithOAuth(session *OAuthSession) *Client {
	cp := *c
	cp.OAuth = session
	return &cp
}

// Authenticated reports whether a refreshable OAuth session is configured.
func (c *Client) Authenticated() bool {
	return c != nil && c.OAuth != nil && c.OAuth.Ready()
}
