package ytmusic

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWhoAmI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.RawQuery, "access_token=") || strings.Contains(r.URL.Path, "tokeninfo"):
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
		authMu:     &sync.Mutex{},
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

func TestProbeLibrary(t *testing.T) {
	c := withBrowserAuth(testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		b := string(body)
		switch {
		case strings.Contains(b, "FEmusic_liked_playlists"):
			writeJSON(w, fixtureLibraryPlaylists)
		case strings.Contains(b, "FEmusic_history"):
			writeJSON(w, fixtureHistoryBrowse)
		case strings.Contains(b, "VLLM"):
			writeJSON(w, fixturePlaylistBrowse)
		default:
			t.Fatalf("unexpected %s %s", r.URL.Path, b)
		}
	}))
	probe, err := c.ProbeLibrary()
	if err != nil {
		t.Fatal(err)
	}
	if probe.AuthMode != "browser" || !probe.VisitorIDPresent {
		t.Fatalf("%+v", probe)
	}
	if probe.LibraryPlaylists < 1 || !probe.LikedShelfFound || probe.LikedTracksParsed < 1 {
		t.Fatalf("%+v", probe)
	}
	if probe.HistoryItems < 1 {
		t.Fatalf("history=%+v", probe)
	}
}
