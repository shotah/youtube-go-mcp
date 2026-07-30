package ytmusic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOAuthDeviceFlowAndRefresh(t *testing.T) {
	var tokenHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/device/code"):
			if !strings.Contains(string(body), "client_id=cid") {
				t.Errorf("missing client_id in code request: %s", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":      "dev-code",
				"user_code":        "ABC-DEF",
				"expires_in":       1800,
				"interval":         1,
				"verification_url": "https://www.google.com/device",
			})
		case strings.Contains(r.URL.Path, "/token"):
			tokenHits.Add(1)
			form := string(body)
			if strings.Contains(form, "grant_type=refresh_token") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "access-2",
					"expires_in":   3600,
					"scope":        oauthScope,
					"token_type":   "Bearer",
				})
				return
			}
			if tokenHits.Load() == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "access-1",
				"refresh_token": "refresh-1",
				"expires_in":    3600,
				"scope":         oauthScope,
				"token_type":    "Bearer",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Point package URLs at the test server by temporarily swapping via local helpers.
	origCode, origToken := oauthCodeURL, oauthTokenURL
	t.Cleanup(func() {
		// constants are not vars — use credential HTTP against rewritten transport instead.
		_ = origCode
		_ = origToken
	})

	creds := OAuthCredentials{
		ClientID:     "cid",
		ClientSecret: "sec",
		HTTPClient:   &http.Client{Transport: rewriteOAuthHost{base: srv.URL, next: http.DefaultTransport}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tok, err := RunDeviceAuthFlow(ctx, creds, nil)
	if err != nil {
		t.Fatalf("RunDeviceAuthFlow: %v", err)
	}
	if tok.AccessToken != "access-1" || tok.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected token: %+v", tok)
	}

	session := &OAuthSession{
		Credentials: creds,
		Token: &OAuthToken{
			AccessToken:  "stale",
			RefreshToken: "refresh-1",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(-time.Minute).Unix(),
		},
		Path: filepath.Join(t.TempDir(), "oauth.json"),
		Now:  time.Now,
	}
	if err := session.EnsureAccessToken(false); err != nil {
		t.Fatalf("EnsureAccessToken: %v", err)
	}
	if session.Token.AccessToken != "access-2" {
		t.Fatalf("access=%q", session.Token.AccessToken)
	}
	if _, err := os.Stat(session.Path); err != nil {
		t.Fatalf("expected persisted oauth file: %v", err)
	}
}

func TestOAuthApplySetsBearer(t *testing.T) {
	session := &OAuthSession{
		Credentials: OAuthCredentials{ClientID: "c", ClientSecret: "s"},
		Token: &OAuthToken{
			AccessToken:  "tok",
			RefreshToken: "ref",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		},
		Now: time.Now,
	}
	req, _ := http.NewRequest(http.MethodPost, "https://music.youtube.com/youtubei/v1/browse", http.NoBody)
	if err := session.Apply(req); err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer tok" {
		t.Fatalf("Authorization=%q", got)
	}
	if req.Header.Get("X-Goog-Request-Time") == "" {
		t.Fatal("missing X-Goog-Request-Time")
	}
}

func TestLoadOAuthTokenFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")
	raw := `{
	  "scope": "https://www.googleapis.com/auth/youtube",
	  "token_type": "Bearer",
	  "access_token": "a",
	  "refresh_token": "r",
	  "expires_at": 1,
	  "expires_in": 3600
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	tok, err := LoadOAuthTokenFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken != "r" {
		t.Fatalf("%+v", tok)
	}
}

func TestMapOAuthPending(t *testing.T) {
	err := mapOAuthHTTPError(400, []byte(`{"error":"authorization_pending"}`))
	if !errors.Is(err, ErrOAuthPending) {
		t.Fatalf("got %v", err)
	}
}

func TestOAuthDeviceCodeHelpers(t *testing.T) {
	code := &DeviceCode{
		VerificationURL: "https://www.google.com/device",
		UserCode:        "ABC",
	}
	if got := code.VerificationLink(); !strings.Contains(got, "google.com/device") {
		t.Fatalf("link=%q", got)
	}
	session := &OAuthSession{
		Credentials: OAuthCredentials{ClientID: "c", ClientSecret: "s"},
		Token:       &OAuthToken{RefreshToken: "r"},
	}
	if !session.Ready() {
		t.Fatal("expected Ready with refresh token + credentials")
	}
	if (&OAuthSession{Token: &OAuthToken{RefreshToken: "r"}}).Ready() {
		t.Fatal("Ready requires client credentials")
	}
	if intFromAny(float64(3)) != 3 || intFromAny(int(4)) != 4 || intFromAny(int64(5)) != 5 {
		t.Fatal("intFromAny")
	}
	if intFromAny(true) != 0 {
		t.Fatal("intFromAny bad type")
	}
}

type rewriteOAuthHost struct {
	base string
	next http.RoundTripper
}

func (t rewriteOAuthHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u := *req.URL
	base := strings.TrimRight(t.base, "/")
	req = req.Clone(req.Context())
	switch {
	case strings.Contains(u.Path, "/device/code") || strings.Contains(u.String(), "/device/code"):
		req.URL, _ = req.URL.Parse(base + "/o/oauth2/device/code")
	default:
		req.URL, _ = req.URL.Parse(base + "/token")
	}
	req.URL.RawQuery = u.RawQuery
	req.Host = req.URL.Host
	return t.next.RoundTrip(req)
}
