package ytmusic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWhoAmI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "tokeninfo") || strings.Contains(r.URL.RawQuery, "access_token="):
			_ = json.NewEncoder(w).Encode(map[string]string{
				"email": "me@example.com",
				"sub":   "123",
				"scope": oauthScope,
				"aud":   "cid",
			})
		case strings.Contains(r.URL.Path, "channels"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":      "UCme",
					"snippet": map[string]string{"title": "My Channel"},
				}},
			})
		default:
			http.Error(w, "unexpected "+r.URL.String(), http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		orig := *req.URL
		req = req.Clone(req.Context())
		base, _ := url.Parse(strings.TrimRight(srv.URL, "/"))
		req.URL.Scheme = base.Scheme
		req.URL.Host = base.Host
		req.URL.Path = orig.Path
		req.URL.RawQuery = orig.RawQuery
		req.Host = base.Host
		return http.DefaultTransport.RoundTrip(req)
	})

	c := &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second, Transport: transport},
		OAuth: &OAuthSession{
			Credentials: OAuthCredentials{ClientID: "c", ClientSecret: "s"},
			Token: &OAuthToken{
				AccessToken:  "tok",
				RefreshToken: "ref",
				TokenType:    "Bearer",
				ExpiresAt:    time.Now().Add(time.Hour).Unix(),
			},
			Now: time.Now,
		},
	}
	info, err := c.WhoAmI()
	if err != nil {
		t.Fatal(err)
	}
	if info.Email != "me@example.com" || info.ChannelTitle != "My Channel" || info.ChannelID != "UCme" {
		t.Fatalf("%+v", info)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
