package ytmusic

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetOAuthPathAndWithHelpers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")
	raw := `{
	  "access_token": "a",
	  "refresh_token": "r",
	  "expires_at": 9999999999,
	  "token_type": "Bearer"
	}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	c := NewClient()
	if err := c.SetOAuthPath(path, "cid", "secret"); err != nil {
		t.Fatal(err)
	}
	if c.OAuth == nil || !c.OAuth.Ready() {
		t.Fatal("oauth not ready")
	}
	if c.Auth != nil {
		t.Fatal("browser auth should be cleared")
	}

	authed := c.WithAuth(&BrowserAuth{Cookie: "c", SAPISID: "s", AuthUser: "0"})
	if authed.OAuth != nil || authed.Auth == nil || !authed.Authenticated() {
		t.Fatal("WithAuth")
	}
	oauthCopy := c.WithOAuth(c.OAuth)
	if oauthCopy.Auth != nil || oauthCopy.OAuth == nil {
		t.Fatal("WithOAuth")
	}
}

func TestPackageLibraryWrappersHTTP(t *testing.T) {
	c := withBrowserAuth(testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		switch {
		case strings.Contains(body, "FEmusic_liked_playlists"):
			writeJSON(w, fixtureLibraryPlaylists)
		case strings.Contains(body, "FEmusic_history"):
			writeJSON(w, fixtureHistoryBrowse)
		case strings.Contains(body, "VLPL") || strings.Contains(body, "VLLM"):
			writeJSON(w, fixturePlaylistBrowse)
		default:
			t.Fatalf("unexpected %s %s", r.URL.Path, body)
		}
	}))
	old, oldHTTP := Default, HTTPClient
	Default, HTTPClient = c, c.HTTPClient
	t.Cleanup(func() { Default, HTTPClient = old, oldHTTP })

	if _, err := GetLibraryPlaylists(5); err != nil {
		t.Fatal(err)
	}
	if _, err := GetHistory(5); err != nil {
		t.Fatal(err)
	}
	if _, err := GetPlaylist("PLTEST", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := GetLikedSongs(5); err != nil {
		t.Fatal(err)
	}
}

func TestFetchPlaylistContinuationsHTTP(t *testing.T) {
	first := `{
	  "contents": {
	    "twoColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "musicResponsiveHeaderRenderer": {
	                  "title": {"runs": [{"text": "Long Mix"}]},
	                  "secondSubtitle": {"runs": [{"text": "2 songs"}]}
	                }
	              }]
	            }
	          }
	        }
	      }],
	      "secondaryContents": {
	        "sectionListRenderer": {
	          "contents": [{
	            "musicPlaylistShelfRenderer": {
	              "contents": [
	                {
	                  "musicResponsiveListItemRenderer": {
	                    "playlistItemData": {"videoId": "vid1"},
	                    "flexColumns": [
	                      {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	                        {"text": "One", "navigationEndpoint": {"watchEndpoint": {"videoId": "vid1"}}}
	                      ]}}}
	                    ]
	                  }
	                },
	                {
	                  "continuationItemRenderer": {
	                    "continuationEndpoint": {
	                      "continuationCommand": {"token": "CONT1"}
	                    }
	                  }
	                }
	              ]
	            }
	          }]
	        }
	      }
	    }
	  }
	}`
	cont := `{
	  "onResponseReceivedActions": [{
	    "appendContinuationItemsAction": {
	      "continuationItems": [{
	        "musicResponsiveListItemRenderer": {
	          "playlistItemData": {"videoId": "vid2"},
	          "flexColumns": [
	            {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	              {"text": "Two", "navigationEndpoint": {"watchEndpoint": {"videoId": "vid2"}}}
	            ]}}}
	          ]
	        }
	      }]
	    }
	  }]
	}`
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		if strings.Contains(body, "CONT1") {
			writeJSON(w, cont)
			return
		}
		writeJSON(w, first)
	})
	detail, err := c.GetPlaylist("PLLONG", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tracks) != 2 {
		t.Fatalf("tracks=%d want 2", len(detail.Tracks))
	}
}

func TestSearchNextExistsAfterPage(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, fixtureSearchPage)
	})
	sc := c.TrackSearch("ncs")
	if !sc.NextExists() {
		t.Fatal("before")
	}
	if _, err := sc.Next(); err != nil {
		t.Fatal(err)
	}
	// No continuation in fixture → NextExists false after first page.
	if sc.NextExists() {
		t.Fatal("expected end after page without continuation")
	}
	if _, err := sc.Next(); err == nil {
		t.Fatal("expected end reached")
	}
}

func TestMapOAuthHTTPErrors(t *testing.T) {
	if !errors.Is(mapOAuthHTTPError(400, []byte(`{"error":"authorization_pending"}`)), ErrOAuthPending) {
		t.Fatal("pending")
	}
	if !errors.Is(mapOAuthHTTPError(400, []byte(`{"error":"slow_down"}`)), ErrOAuthSlowDown) {
		t.Fatal("slow_down")
	}
	if !errors.Is(mapOAuthHTTPError(400, []byte(`{"error":"access_denied"}`)), ErrSessionExpired) {
		t.Fatal("access_denied")
	}
	if !errors.Is(mapOAuthHTTPError(401, []byte(`{"error":"other"}`)), ErrSessionExpired) {
		t.Fatal("401")
	}
	if !errors.Is(mapOAuthHTTPError(400, []byte(`{"error":"invalid_client"}`)), ErrInvalidAuth) {
		t.Fatal("invalid_client")
	}
}
