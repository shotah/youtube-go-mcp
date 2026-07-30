package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

func TestSelfTestRequiresOAuth(t *testing.T) {
	c := ytmusic.NewClient()
	c.OAuth = nil
	if err := SelfTest(c); err == nil {
		t.Fatal("expected oauth required")
	}
}

func TestSelfTestOAuthDataAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":      map[string]string{"videoId": "searchvid01"},
					"snippet": map[string]string{"title": "Hit"},
				}},
			})
		case strings.Contains(r.URL.Path, "tokeninfo") || strings.Contains(r.URL.RawQuery, "access_token="):
			_ = json.NewEncoder(w).Encode(map[string]string{"scope": "https://www.googleapis.com/auth/youtube"})
		case strings.Contains(r.URL.Path, "/channels"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id": "UCme", "snippet": map[string]string{"title": "Me"},
					"contentDetails": map[string]any{
						"relatedPlaylists": map[string]string{"likes": "LL", "uploads": "UU"},
					},
				}},
			})
		case strings.Contains(r.URL.Path, "/videos"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id": "liked1", "snippet": map[string]string{"title": "Liked", "categoryId": "10"},
					"contentDetails": map[string]string{"duration": "PT1M"},
				}},
			})
		default:
			http.Error(w, "unexpected "+r.URL.String(), http.StatusBadRequest)
		}
	}))
	t.Cleanup(api.Close)

	client := ytmusic.NewClient()
	client.HTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: roundTripAllHosts{base: api.URL},
	}
	client.OAuth = &ytmusic.OAuthSession{
		Credentials: ytmusic.OAuthCredentials{ClientID: "c", ClientSecret: "s"},
		Token: &ytmusic.OAuthToken{
			AccessToken:  "tok",
			RefreshToken: "ref",
			TokenType:    "Bearer",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		},
		Now: time.Now,
	}

	if err := SelfTest(client); err != nil {
		t.Fatal(err)
	}
}

type roundTripAllHosts struct {
	base string
}

func (t roundTripAllHosts) RoundTrip(req *http.Request) (*http.Response, error) {
	base, err := url.Parse(strings.TrimRight(t.base, "/"))
	if err != nil {
		return nil, err
	}
	orig := *req.URL
	req = req.Clone(req.Context())
	req.URL.Scheme = base.Scheme
	req.URL.Host = base.Host
	req.URL.Path = orig.Path
	req.URL.RawQuery = orig.RawQuery
	req.Host = base.Host
	return http.DefaultTransport.RoundTrip(req)
}
