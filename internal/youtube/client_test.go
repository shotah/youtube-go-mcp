package youtube

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChannelMineSearchLikedPlaylists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			http.Error(w, "bad auth "+got, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/channels"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":      "UCme",
					"snippet": map[string]string{"title": "My Channel"},
					"contentDetails": map[string]any{
						"relatedPlaylists": map[string]string{
							"likes":   "LLme",
							"uploads": "UUme",
						},
					},
				}},
			})
		case strings.HasPrefix(r.URL.Path, "/search"):
			if r.URL.Query().Get("videoCategoryId") != CategoryMusic {
				http.Error(w, "expected music category", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id": map[string]string{"videoId": "abc12345678"},
					"snippet": map[string]string{
						"title": "Cool Song", "channelTitle": "Artist - Topic",
					},
				}},
			})
		case strings.HasPrefix(r.URL.Path, "/videos"):
			if r.URL.Query().Get("myRating") == "like" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{{
						"id": "likedvid123",
						"snippet": map[string]string{
							"title": "Liked Track", "categoryId": "10", "channelTitle": "X",
						},
						"contentDetails": map[string]string{"duration": "PT3M1S"},
					}},
				})
				return
			}
			if r.URL.Query().Get("id") == "vid11111111" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"items": []map[string]any{{
						"id": "vid11111111",
						"snippet": map[string]string{
							"title": "Essay", "categoryId": "22", "channelTitle": "Creator",
						},
						"contentDetails": map[string]string{"duration": "PT10M"},
					}},
				})
				return
			}
			http.Error(w, "unexpected videos "+r.URL.RawQuery, http.StatusBadRequest)
		case strings.HasPrefix(r.URL.Path, "/playlists"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":             "PLtest",
					"snippet":        map[string]string{"title": "Mix"},
					"contentDetails": map[string]any{"itemCount": 3},
				}},
			})
		case strings.HasPrefix(r.URL.Path, "/playlistItems"):
			if r.URL.Query().Get("playlistId") != "PLtest" {
				http.Error(w, "bad playlist", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"snippet": map[string]any{
						"title":      "Item One",
						"resourceId": map[string]string{"videoId": "itemvid0001"},
					},
					"contentDetails": map[string]string{"videoId": "itemvid0001"},
				}},
			})
		default:
			http.Error(w, "unexpected "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(StaticToken("test-token"))
	c.BaseURL = srv.URL
	c.HTTPClient = srv.Client()

	ch, err := c.ChannelMine()
	if err != nil {
		t.Fatal(err)
	}
	if ch.ID != "UCme" || ch.LikesPlaylist != "LLme" {
		t.Fatalf("%+v", ch)
	}

	hits, err := c.SearchVideos(SearchOptions{Query: "cool", MaxResults: 5, MusicOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].VideoID != "abc12345678" || hits[0].MusicURL == "" {
		t.Fatalf("search: %+v", hits)
	}

	liked, err := c.ListLikedVideos(ListOptions{MaxResults: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(liked) != 1 || liked[0].DurationSec != 181 || liked[0].MusicURL == "" {
		t.Fatalf("liked: %+v", liked)
	}

	v, err := c.GetVideo("vid11111111")
	if err != nil {
		t.Fatal(err)
	}
	if v.Title != "Essay" || v.MusicURL != "" || v.URL == "" {
		t.Fatalf("get: %+v", v)
	}

	pls, err := c.ListMyPlaylists(ListOptions{MaxResults: 10})
	if err != nil || len(pls) != 1 || pls[0].ID != "PLtest" {
		t.Fatalf("playlists: %v %+v", err, pls)
	}

	items, err := c.ListPlaylistItems("VLPLtest", ListOptions{MaxResults: 10})
	if err != nil || len(items) != 1 || items[0].VideoID != "itemvid0001" {
		t.Fatalf("items: %v %+v", err, items)
	}
}

func TestAuthRequired(t *testing.T) {
	c := New(nil)
	if _, err := c.ChannelMine(); err == nil {
		t.Fatal("expected auth error")
	}
	c = New(StaticToken(""))
	if _, err := c.ChannelMine(); err == nil {
		t.Fatal("expected empty token error")
	}
}
