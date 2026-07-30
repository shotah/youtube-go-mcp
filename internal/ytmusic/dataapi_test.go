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

func TestProbeDataAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "tokeninfo") || strings.Contains(r.URL.RawQuery, "access_token="):
			_ = json.NewEncoder(w).Encode(map[string]string{
				"scope": oauthScope,
				"aud":   "cid",
			})
		case strings.Contains(r.URL.Path, "/channels"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":      "UCme",
					"snippet": map[string]string{"title": "My Channel"},
					"contentDetails": map[string]any{
						"relatedPlaylists": map[string]string{"likes": "LLme", "uploads": "UUme"},
					},
				}},
			})
		case strings.Contains(r.URL.Path, "/videos"):
			if r.URL.Query().Get("myRating") != "like" {
				http.Error(w, "expected myRating=like", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "a", "snippet": map[string]string{"title": "Song A", "categoryId": "10"}, "contentDetails": map[string]string{"duration": "PT1M"}},
					{"id": "b", "snippet": map[string]string{"title": "Vlog", "categoryId": "22"}, "contentDetails": map[string]string{"duration": "PT2M"}},
				},
			})
		default:
			http.Error(w, "unexpected "+r.URL.String(), http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	c := oauthTestClient(t, srv.URL)
	probe, err := c.ProbeDataAPI()
	if err != nil {
		t.Fatal(err)
	}
	if probe.ChannelID != "UCme" || probe.ChannelTitle != "My Channel" {
		t.Fatalf("channel: %+v", probe)
	}
	if probe.LikedVideos != 2 || probe.MusicCategoryN != 1 || probe.LikedSample != "Song A" {
		t.Fatalf("liked: %+v", probe)
	}
}

func TestEnvFirstPrefersYouTubeNames(t *testing.T) {
	t.Setenv(EnvOAuthPath, "/new/oauth.json")
	t.Setenv(envOAuthPathLegacy, "/legacy/oauth.json")
	if got := oauthPathFromEnv(); got != "/new/oauth.json" {
		t.Fatalf("got %q", got)
	}
	t.Setenv(EnvOAuthPath, "")
	if got := oauthPathFromEnv(); got != "/legacy/oauth.json" {
		t.Fatalf("legacy got %q", got)
	}
}

func oauthTestClient(t *testing.T, apiBase string) *Client {
	t.Helper()
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		orig := *req.URL
		req = req.Clone(req.Context())
		base, _ := url.Parse(strings.TrimRight(apiBase, "/"))
		req.URL.Scheme = base.Scheme
		req.URL.Host = base.Host
		req.URL.Path = orig.Path
		req.URL.RawQuery = orig.RawQuery
		req.Host = base.Host
		return http.DefaultTransport.RoundTrip(req)
	})
	return &Client{
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
}
