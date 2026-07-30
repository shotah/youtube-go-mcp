package ytmusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	oauthScope   = "https://www.googleapis.com/auth/youtube"
	oauthCodeURL = "https://www.youtube.com/o/oauth2/device/code"
	//nolint:gosec // G101: OAuth token endpoint URL, not a credential
	oauthTokenURL  = "https://oauth2.googleapis.com/token"
	oauthUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:88.0) Gecko/20100101 Firefox/88.0 Cobalt/Version"

	envOAuthPath     = "YTMUSIC_OAUTH_PATH"
	envOAuthClientID = "YTMUSIC_OAUTH_CLIENT_ID"
	//nolint:gosec // G101: env var name, not a secret value
	envOAuthClientSecret = "YTMUSIC_OAUTH_CLIENT_SECRET"

	oauthRefreshSkew   = 60 * time.Second
	oauthErrorSnippetN = 200
)

// OAuthCredentials is a Google OAuth client registered as
// "TVs and Limited Input devices" with the YouTube Data API enabled.
type OAuthCredentials struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

// OAuthToken is the ytmusicapi-compatible oauth.json shape.
type OAuthToken struct {
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
}

// DeviceCode is the first step of the TV/device OAuth flow.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	VerificationURL string `json:"verification_url"`
}

// VerificationLink returns the URL the user should open to approve access.
func (d DeviceCode) VerificationLink() string {
	base := strings.TrimSpace(d.VerificationURL)
	if base == "" {
		base = "https://www.google.com/device"
	}
	if d.UserCode == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "user_code=" + url.QueryEscape(d.UserCode)
}

// OAuthSession holds a refreshable Bearer token and optional on-disk cache.
type OAuthSession struct {
	Credentials OAuthCredentials
	Token       *OAuthToken
	Path        string
	Now         func() time.Time

	mu sync.Mutex
}

func (c *OAuthCredentials) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *OAuthCredentials) validate() error {
	if c == nil || strings.TrimSpace(c.ClientID) == "" || strings.TrimSpace(c.ClientSecret) == "" {
		return fmt.Errorf("%w: OAuth requires client_id and client_secret (Google Cloud → OAuth client type \"TVs and Limited Input devices\")", ErrInvalidAuth)
	}
	return nil
}

// GetDeviceCode starts the TV/device OAuth flow.
func (c *OAuthCredentials) GetDeviceCode() (*DeviceCode, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	form := url.Values{
		"client_id": {c.ClientID},
		"scope":     {oauthScope},
	}
	resp, err := c.postForm(oauthCodeURL, form)
	if err != nil {
		return nil, err
	}
	var code DeviceCode
	if err := json.Unmarshal(resp, &code); err != nil {
		return nil, fmt.Errorf("%w: decode device code: %w", ErrInvalidAuth, err)
	}
	if code.DeviceCode == "" || code.UserCode == "" {
		return nil, fmt.Errorf("%w: device code response missing device_code/user_code: %s", ErrInvalidAuth, truncate(string(resp)))
	}
	return &code, nil
}

// TokenFromDeviceCode exchanges an approved device_code for tokens.
func (c *OAuthCredentials) TokenFromDeviceCode(deviceCode string) (*OAuthToken, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	form := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"http://oauth.net/grant_type/device/1.0"},
		"code":          {deviceCode},
	}
	resp, err := c.postForm(oauthTokenURL, form)
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(resp, true)
}

// RefreshAccessToken requests a new access_token for the given refresh_token.
func (c *OAuthCredentials) RefreshAccessToken(refreshToken string) (*OAuthToken, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	form := url.Values{
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	resp, err := c.postForm(oauthTokenURL, form)
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(resp, false)
}

func (c *OAuthCredentials) postForm(endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", oauthUserAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapOAuthHTTPError(resp.StatusCode, body)
	}
	return body, nil
}

func mapOAuthHTTPError(status int, body []byte) error {
	var parsed struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &parsed)
	snippet := truncate(string(body))
	switch parsed.Error {
	case "invalid_client":
		return fmt.Errorf("%w: invalid OAuth client (check client_id/secret and that YouTube Data API is enabled): %s", ErrInvalidAuth, snippet)
	case "unauthorized_client":
		return fmt.Errorf("%w: unauthorized OAuth client (token was minted with a different client_id): %s", ErrInvalidAuth, snippet)
	case "authorization_pending":
		return fmt.Errorf("%w: authorization_pending", ErrOAuthPending)
	case "slow_down":
		return fmt.Errorf("%w: slow_down", ErrOAuthSlowDown)
	case "expired_token", "access_denied":
		return fmt.Errorf("%w: device authorization %s — %s", ErrSessionExpired, parsed.Error, snippet)
	default:
		if status == 401 || status == 403 {
			return fmt.Errorf("%w: oauth HTTP %d: %s", ErrSessionExpired, status, snippet)
		}
		return fmt.Errorf("%w: oauth HTTP %d: %s", ErrInvalidAuth, status, snippet)
	}
}

func parseTokenResponse(body []byte, requireRefresh bool) (*OAuthToken, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: decode token: %w", ErrInvalidAuth, err)
	}
	tok := &OAuthToken{
		Scope:        stringFromAny(raw["scope"]),
		TokenType:    firstNonEmpty(stringFromAny(raw["token_type"]), "Bearer"),
		AccessToken:  stringFromAny(raw["access_token"]),
		RefreshToken: stringFromAny(raw["refresh_token"]),
		ExpiresIn:    intFromAny(raw["expires_in"]),
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("%w: token response missing access_token: %s", ErrInvalidAuth, truncate(string(body)))
	}
	if requireRefresh && tok.RefreshToken == "" {
		return nil, fmt.Errorf("%w: token response missing refresh_token: %s", ErrInvalidAuth, truncate(string(body)))
	}
	if tok.Scope == "" {
		tok.Scope = oauthScope
	}
	tok.ExpiresAt = time.Now().Unix() + int64(tok.ExpiresIn)
	return tok, nil
}

// LoadOAuthTokenFromFile loads a ytmusicapi-compatible oauth.json.
func LoadOAuthTokenFromFile(path string) (*OAuthToken, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G703: path comes from operator-configured env/CLI
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrInvalidAuth, path, err)
	}
	var tok OAuthToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("%w: parse oauth json: %w", ErrInvalidAuth, err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		return nil, fmt.Errorf("%w: oauth.json must include access_token and refresh_token", ErrInvalidAuth)
	}
	if tok.TokenType == "" {
		tok.TokenType = "Bearer"
	}
	if tok.Scope == "" {
		tok.Scope = oauthScope
	}
	return &tok, nil
}

// Save writes the token to path (0600), ytmusicapi-compatible.
func (t *OAuthToken) Save(path string) error {
	if t == nil {
		return fmt.Errorf("%w: nil oauth token", ErrInvalidAuth)
	}
	//nolint:gosec // G117: oauth.json intentionally persists access/refresh tokens for the operator
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (s *OAuthSession) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *OAuthSession) isExpiring() bool {
	if s == nil || s.Token == nil || s.Token.AccessToken == "" {
		return true
	}
	if s.Token.ExpiresAt <= 0 {
		return false
	}
	return s.now().Unix() >= s.Token.ExpiresAt-int64(oauthRefreshSkew/time.Second)
}

// EnsureAccessToken refreshes the access token when it is missing or near expiry.
func (s *OAuthSession) EnsureAccessToken(force bool) error {
	if s == nil {
		return ErrAuthRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Token == nil || s.Token.RefreshToken == "" {
		return fmt.Errorf("%w: oauth session has no refresh_token", ErrInvalidAuth)
	}
	if !force && !s.isExpiring() {
		return nil
	}
	fresh, err := s.Credentials.RefreshAccessToken(s.Token.RefreshToken)
	if err != nil {
		return err
	}
	s.Token.AccessToken = fresh.AccessToken
	s.Token.ExpiresIn = fresh.ExpiresIn
	s.Token.ExpiresAt = fresh.ExpiresAt
	if fresh.TokenType != "" {
		s.Token.TokenType = fresh.TokenType
	}
	if fresh.Scope != "" {
		s.Token.Scope = fresh.Scope
	}
	// Google omits refresh_token on refresh responses; keep the existing one.
	if s.Path != "" {
		if err := s.Token.Save(s.Path); err != nil {
			return fmt.Errorf("persist refreshed oauth token: %w", err)
		}
	}
	return nil
}

// Apply sets Authorization: Bearer <access_token>, refreshing if needed.
func (s *OAuthSession) Apply(req *http.Request) error {
	if s == nil || req == nil {
		return ErrAuthRequired
	}
	if err := s.EnsureAccessToken(false); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	typ := firstNonEmpty(s.Token.TokenType, "Bearer")
	req.Header.Set("Authorization", typ+" "+s.Token.AccessToken)
	req.Header.Set("X-Origin", origin)
	req.Header.Set("Origin", origin)
	return nil
}

// Ready reports whether the session can produce a Bearer token.
func (s *OAuthSession) Ready() bool {
	return s != nil && s.Token != nil && s.Token.RefreshToken != "" &&
		strings.TrimSpace(s.Credentials.ClientID) != "" &&
		strings.TrimSpace(s.Credentials.ClientSecret) != ""
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= oauthErrorSnippetN {
		return s
	}
	return s[:oauthErrorSnippetN] + "…"
}
