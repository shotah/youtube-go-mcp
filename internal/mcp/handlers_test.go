package mcp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

func TestHelpers(t *testing.T) {
	if got := clampLimit(0, 10, 50); got != 10 {
		t.Fatalf("default: %d", got)
	}
	if got := clampLimit(100, 10, 50); got != 50 {
		t.Fatalf("max: %d", got)
	}
	if got := clampLimit(25, 10, 50); got != 25 {
		t.Fatalf("ok: %d", got)
	}

	out := trackToOut(&ytmusic.TrackItem{
		VideoID:  "abc123DEFG1",
		Title:    "Song",
		Artists:  []ytmusic.Artist{{Name: "A"}, {Name: ""}},
		Duration: 120,
	})
	if out.VideoID != "abc123DEFG1" || out.VideoIDSnake != out.VideoID {
		t.Fatalf("%+v", out)
	}
	if out.URL == "" || out.MusicURL == "" || len(out.Artists) != 1 {
		t.Fatalf("%+v", out)
	}

	detail := trackDetailToOut(&ytmusic.TrackDetail{
		VideoID:   "abc123DEFG1",
		Title:     "Song",
		Artists:   []ytmusic.Artist{{Name: "A"}},
		Album:     ytmusic.Album{Name: "Alb"},
		Lyrics:    "la",
		HasLyrics: true,
	})
	if detail.Album != "Alb" || !detail.HasLyrics || detail.Lyrics != "la" {
		t.Fatalf("%+v", detail)
	}

	pl := playlistToOut(&ytmusic.PlaylistDetail{
		ID:    "PL1",
		Title: "T",
		Tracks: []*ytmusic.TrackItem{
			{VideoID: "v1", Title: "one"},
			nil,
			{VideoID: "", Title: "skip"},
		},
	})
	if len(pl.Tracks) != 1 || pl.Tracks[0].VideoID != "v1" {
		t.Fatalf("%+v", pl)
	}

	errRes := toolError("boom")
	if errRes == nil || !errRes.IsError {
		t.Fatal("expected tool error")
	}
	if toolErrFrom(ytmusic.ErrAuthRequired) == nil {
		t.Fatal("expected auth tool error")
	}
	if toolErrFrom(ytmusic.ErrRateLimited) == nil {
		t.Fatal("expected rate tool error")
	}
	if toolErrFrom(errors.Join(ytmusic.ErrSessionExpired, errors.New("gone"))) == nil {
		t.Fatal("expected session tool error")
	}

	s := New(nil)
	if s.Client == nil || s.Log == nil {
		t.Fatal("New should default client/logger")
	}
}

func TestFormatCastTarget(t *testing.T) {
	s := New(ytmusic.NewClient())
	ctx := context.Background()
	res, _, err := s.formatCastTarget(ctx, nil, castTargetInput{})
	if err != nil || res == nil || !res.IsError {
		t.Fatalf("empty videoId: res=%v err=%v", res, err)
	}
	res, out, err := s.formatCastTarget(ctx, nil, castTargetInput{VideoID: "abcdefghijk"})
	if err != nil || res != nil {
		t.Fatalf("ok: res=%v err=%v", res, err)
	}
	if out.VideoID != "abcdefghijk" || out.VideoIDSnake != out.VideoID || out.CastHint == "" {
		t.Fatalf("%+v", out)
	}
}

func TestHandlersValidation(t *testing.T) {
	s := New(ytmusic.NewClient())
	ctx := context.Background()

	if res, _, _ := s.searchTracks(ctx, nil, searchTracksInput{}); res == nil || !res.IsError {
		t.Fatal("search requires query")
	}
	if res, _, _ := s.getPlaylist(ctx, nil, getPlaylistInput{}); res == nil || !res.IsError {
		t.Fatal("playlist requires id")
	}
	if res, _, _ := s.getWatchPlaylist(ctx, nil, watchPlaylistInput{}); res == nil || !res.IsError {
		t.Fatal("watch requires videoId")
	}
	if res, _, _ := s.getTrack(ctx, nil, getTrackInput{}); res == nil || !res.IsError {
		t.Fatal("track requires videoId")
	}
	if res, _, _ := s.getLyrics(ctx, nil, getLyricsInput{}); res == nil || !res.IsError {
		t.Fatal("lyrics requires videoId")
	}
	if res, _, _ := s.getLibraryPlaylists(ctx, nil, libraryPlaylistsInput{}); res == nil || !res.IsError {
		t.Fatal("library requires auth")
	}
	if res, _, _ := s.getLikedSongs(ctx, nil, likedSongsInput{}); res == nil || !res.IsError {
		t.Fatal("liked requires auth")
	}
	if res, _, _ := s.getHistory(ctx, nil, historyInput{}); res == nil || !res.IsError {
		t.Fatal("history requires auth")
	}
}

func TestHandlersWithHTTPFixtures(t *testing.T) {
	searchBody := `{
	  "contents": {
	    "tabbedSearchResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "itemSectionRenderer": {
	                  "contents": [{
	                    "musicResponsiveListItemRenderer": {
	                      "playlistItemData": {"videoId": "track1"},
	                      "flexColumns": [
	                        {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	                          {"text": "Song One", "navigationEndpoint": {"watchEndpoint": {"videoId": "track1"}}}
	                        ]}}},
	                        {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	                          {"text": "Artist", "navigationEndpoint": {"browseEndpoint": {
	                            "browseId": "UCa",
	                            "browseEndpointContextSupportedConfigs": {
	                              "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
	                            }
	                          }}}
	                        ]}}}
	                      ],
	                      "overlay": {
	                        "musicItemThumbnailOverlayRenderer": {
	                          "content": {
	                            "musicPlayButtonRenderer": {
	                              "playNavigationEndpoint": {
	                                "watchEndpoint": {
	                                  "watchEndpointMusicSupportedConfigs": {
	                                    "watchEndpointMusicConfig": {"musicVideoType": "MUSIC_VIDEO_TYPE_ATV"}
	                                  }
	                                }
	                              }
	                            }
	                          }
	                        }
	                      }
	                    }
	                  }]
	                }
	              }]
	            }
	          }
	        }
	      }]
	    }
	  }
	}`
	playlistBody := `{
	  "contents": {
	    "twoColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "musicResponsiveHeaderRenderer": {
	                  "title": {"runs": [{"text": "Chill Mix"}]},
	                  "secondSubtitle": {"runs": [{"text": "1 song"}]}
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
	              "contents": [{
	                "musicResponsiveListItemRenderer": {
	                  "playlistItemData": {"videoId": "abc123"},
	                  "flexColumns": [
	                    {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [{
	                      "text": "Night Drive",
	                      "navigationEndpoint": {"watchEndpoint": {"videoId": "abc123"}}
	                    }]}}}
	                  ]
	                }
	              }]
	            }
	          }]
	        }
	      }
	    }
	  }
	}`
	libraryBody := `{
	  "contents": {
	    "singleColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "gridRenderer": {
	                  "items": [{
	                    "musicTwoRowItemRenderer": {
	                      "title": {"runs": [{"text": "Road Trip"}]},
	                      "navigationEndpoint": {"browseEndpoint": {"browseId": "VLPLABCDEF"}}
	                    }
	                  }]
	                }
	              }]
	            }
	          }
	        }
	      }]
	    }
	  }
	}`
	historyBody := `{
	  "contents": {
	    "singleColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "musicShelfRenderer": {
	                  "title": {"runs": [{"text": "Today"}]},
	                  "contents": [{
	                    "musicResponsiveListItemRenderer": {
	                      "playlistItemData": {"videoId": "hist1"},
	                      "flexColumns": [
	                        {"musicResponsiveListItemFlexColumnRenderer": {
	                          "text": {"runs": [{"text": "Song A", "navigationEndpoint": {"watchEndpoint": {"videoId": "hist1"}}}]}
	                        }}
	                      ]
	                    }
	                  }]
	                }
	              }]
	            }
	          }
	        }
	      }]
	    }
	  }
	}`
	nextBody := `{
	  "contents": {
	    "singleColumnMusicWatchNextResultsRenderer": {
	      "tabbedRenderer": {
	        "watchNextTabbedResultsRenderer": {
	          "tabs": [{
	            "tabRenderer": {
	              "content": {
	                "musicQueueRenderer": {
	                  "content": {
	                    "playlistPanelRenderer": {
	                      "contents": [{
	                        "playlistPanelVideoRenderer": {
	                          "title": {"runs": [{"text": "Test Song"}]},
	                          "navigationEndpoint": {
	                            "watchEndpoint": {"videoId": "abc123", "playlistId": "RDAMVMabc123"}
	                          },
	                          "longBylineText": {"runs": [{"text": "Artist Z"}]},
	                          "lengthText": {"runs": [{"text": "2:05"}]}
	                        }
	                      }]
	                    }
	                  }
	                }
	              }
	            }
	          }]
	        }
	      }
	    }
	  }
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		switch {
		case strings.Contains(r.URL.Path, "/search"):
			_, _ = w.Write([]byte(searchBody))
		case strings.Contains(r.URL.Path, "/next"):
			_, _ = w.Write([]byte(nextBody))
		case strings.Contains(body, "FEmusic_liked_playlists"):
			_, _ = w.Write([]byte(libraryBody))
		case strings.Contains(body, "FEmusic_history"):
			_, _ = w.Write([]byte(historyBody))
		case strings.Contains(body, "VLPL") || strings.Contains(body, "VLLM"):
			_, _ = w.Write([]byte(playlistBody))
		default:
			http.Error(w, "unexpected "+r.URL.Path+" "+body, http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	client := ytmusic.NewClient()
	client.HTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: rewriteMusicHost{base: srv.URL},
	}
	client.Sleep = func(time.Duration) {}
	client.MinRequestInterval = 0
	client.MaxRetries = 0
	client.Auth = &ytmusic.BrowserAuth{
		Cookie:   "VISITOR=1; __Secure-3PAPISID=x",
		SAPISID:  "x",
		AuthUser: "0",
	}

	s := New(client)
	ctx := context.Background()

	res, searchOut, err := s.searchTracks(ctx, nil, searchTracksInput{Query: "ncs", Limit: 5})
	if err != nil || res != nil || len(searchOut.Tracks) != 1 {
		t.Fatalf("search: res=%v err=%v out=%+v", res, err, searchOut)
	}
	if searchOut.Tracks[0].VideoIDSnake == "" {
		t.Fatal("missing video_id mirror")
	}

	res, libOut, err := s.getLibraryPlaylists(ctx, nil, libraryPlaylistsInput{Limit: 5})
	if err != nil || res != nil || len(libOut.Playlists) != 1 {
		t.Fatalf("library: res=%v err=%v out=%+v", res, err, libOut)
	}

	res, plOut, err := s.getPlaylist(ctx, nil, getPlaylistInput{PlaylistID: "PLTEST", Limit: 10})
	if err != nil || res != nil || len(plOut.Tracks) != 1 {
		t.Fatalf("playlist: res=%v err=%v out=%+v", res, err, plOut)
	}

	res, likedOut, err := s.getLikedSongs(ctx, nil, likedSongsInput{Limit: 10})
	if err != nil || res != nil || len(likedOut.Tracks) != 1 {
		t.Fatalf("liked: res=%v err=%v out=%+v", res, err, likedOut)
	}

	res, histOut, err := s.getHistory(ctx, nil, historyInput{Limit: 10})
	if err != nil || res != nil || len(histOut.Tracks) != 1 {
		t.Fatalf("history: res=%v err=%v out=%+v", res, err, histOut)
	}

	res, watchOut, err := s.getWatchPlaylist(ctx, nil, watchPlaylistInput{VideoID: "abc123", Limit: 5})
	if err != nil || res != nil || len(watchOut.Tracks) == 0 {
		t.Fatalf("watch: res=%v err=%v out=%+v", res, err, watchOut)
	}

	res, trackOut, err := s.getTrack(ctx, nil, getTrackInput{VideoID: "abc123"})
	if err != nil || res != nil || trackOut.Title == "" {
		t.Fatalf("track: res=%v err=%v out=%+v", res, err, trackOut)
	}

	res, lyricsOut, err := s.getLyrics(ctx, nil, getLyricsInput{VideoID: "abc123"})
	if err != nil || res != nil {
		t.Fatalf("lyrics: res=%v err=%v", res, err)
	}
	if lyricsOut.Available {
		t.Fatalf("expected no lyrics: %+v", lyricsOut)
	}
}

type rewriteMusicHost struct {
	base string
}

func (t rewriteMusicHost) RoundTrip(req *http.Request) (*http.Response, error) {
	base, err := url.Parse(strings.TrimRight(t.base, "/"))
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = base.Scheme
	req.URL.Host = base.Host
	req.Host = base.Host
	return http.DefaultTransport.RoundTrip(req)
}
