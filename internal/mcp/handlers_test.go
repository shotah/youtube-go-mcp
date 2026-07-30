package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shotah/youtube-go-mcp/internal/youtube"
)

func TestFormatCastTarget(t *testing.T) {
	s := New(nil)
	ctx := context.Background()
	res, _, err := s.formatCastTarget(ctx, nil, castTargetInput{})
	if err != nil || res == nil || !res.IsError {
		t.Fatal("expected error for empty videoId")
	}
	res, out, err := s.formatCastTarget(ctx, nil, castTargetInput{VideoID: "abcdefghijk"})
	if err != nil || res != nil {
		t.Fatalf("res=%v err=%v", res, err)
	}
	if out.VideoID != "abcdefghijk" || out.VideoIDSnake != out.VideoID || out.CastHint == "" || out.URL == "" {
		t.Fatalf("%+v", out)
	}
}

func TestVideoOutCastFields(t *testing.T) {
	music := videoToOut(youtube.Video{
		VideoID: "musicvid001", Title: "Song", CategoryID: "10",
		URL: youtube.WatchURL("musicvid001"), MusicURL: youtube.MusicWatchURL("musicvid001"),
	})
	if music.VideoIDSnake != music.VideoID || music.URL == "" || music.MusicURL == "" || !music.MusicLikely {
		t.Fatalf("music row: %+v", music)
	}
	plain := videoToOut(youtube.Video{
		VideoID: "plainvid001", Title: "Talk", CategoryID: "22",
		URL: youtube.WatchURL("plainvid001"),
	})
	if plain.VideoIDSnake != plain.VideoID || plain.URL == "" || plain.MusicURL != "" || plain.MusicLikely {
		t.Fatalf("plain row should omit music flags: %+v", plain)
	}
}

func TestHandlersRequireAuth(t *testing.T) {
	s := New(nil)
	ctx := context.Background()
	if res, _, _ := s.searchVideos(ctx, nil, searchVideosInput{Query: "x"}); res == nil || !res.IsError {
		t.Fatal("search requires auth")
	}
	if res, _, _ := s.getVideo(ctx, nil, getVideoInput{VideoID: "x"}); res == nil || !res.IsError {
		t.Fatal("get requires auth")
	}
	if res, _, _ := s.listLikedVideos(ctx, nil, likedVideosInput{}); res == nil || !res.IsError {
		t.Fatal("liked requires auth")
	}
}

func TestHandlersDataAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":      map[string]string{"videoId": "searchvid01"},
					"snippet": map[string]string{"title": "Hit", "channelTitle": "Ch"},
				}},
			})
		case strings.Contains(r.URL.Path, "/videos") && r.URL.Query().Get("myRating") == "like":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":             "likedvid001",
					"snippet":        map[string]string{"title": "Liked", "categoryId": "10"},
					"contentDetails": map[string]string{"duration": "PT2M"},
				}},
			})
		case strings.Contains(r.URL.Path, "/videos"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":             "getvid00001",
					"snippet":        map[string]string{"title": "Meta", "categoryId": "22"},
					"contentDetails": map[string]string{"duration": "PT5M"},
				}},
			})
		case strings.Contains(r.URL.Path, "/playlists"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"id":             "PLabc",
					"snippet":        map[string]string{"title": "My PL"},
					"contentDetails": map[string]any{"itemCount": 2},
				}},
			})
		case strings.Contains(r.URL.Path, "/playlistItems"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"snippet":        map[string]any{"title": "Row", "resourceId": map[string]string{"videoId": "plitem00001"}},
					"contentDetails": map[string]string{"videoId": "plitem00001"},
				}},
			})
		default:
			http.Error(w, "unexpected "+r.URL.String(), http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	yt := youtube.New(youtube.StaticToken("tok"))
	yt.BaseURL = srv.URL
	yt.HTTPClient = srv.Client()
	s := New(yt)
	ctx := context.Background()

	res, searchOut, err := s.searchVideos(ctx, nil, searchVideosInput{Query: "q", Limit: 5})
	if err != nil || res != nil || len(searchOut.Videos) != 1 {
		t.Fatalf("search: res=%v err=%v out=%+v", res, err, searchOut)
	}

	res, getOut, err := s.getVideo(ctx, nil, getVideoInput{VideoID: "getvid00001"})
	if err != nil || res != nil || getOut.Video.Title != "Meta" {
		t.Fatalf("get: res=%v err=%v out=%+v", res, err, getOut)
	}

	res, plOut, err := s.getPlaylist(ctx, nil, getPlaylistInput{PlaylistID: "PLabc", Limit: 10})
	if err != nil || res != nil || len(plOut.Videos) != 1 {
		t.Fatalf("playlist: res=%v err=%v out=%+v", res, err, plOut)
	}

	res, libOut, err := s.listPlaylists(ctx, nil, libraryPlaylistsInput{Limit: 10})
	if err != nil || res != nil || len(libOut.Playlists) != 1 {
		t.Fatalf("library: res=%v err=%v out=%+v", res, err, libOut)
	}

	res, likedOut, err := s.listLikedVideos(ctx, nil, likedVideosInput{Limit: 10, MusicOnly: true})
	if err != nil || res != nil || len(likedOut.Videos) != 1 || !likedOut.Videos[0].MusicLikely {
		t.Fatalf("liked: res=%v err=%v out=%+v", res, err, likedOut)
	}
}
