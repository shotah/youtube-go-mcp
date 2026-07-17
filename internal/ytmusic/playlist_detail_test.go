package ytmusic

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParsePlaylistDetailFixture(t *testing.T) {
	raw := `{
	  "contents": {
	    "twoColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "musicResponsiveHeaderRenderer": {
	                  "title": {"runs": [{"text": "Chill Mix"}]},
	                  "facepile": {
	                    "avatarStackViewModel": {
	                      "text": {"content": "YouTube Music"}
	                    }
	                  },
	                  "secondSubtitle": {"runs": [{"text": "42 songs"}]}
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
	                    "playlistItemData": {"videoId": "abc123"},
	                    "flexColumns": [
	                      {
	                        "musicResponsiveListItemFlexColumnRenderer": {
	                          "text": {
	                            "runs": [{
	                              "text": "Night Drive",
	                              "navigationEndpoint": {
	                                "watchEndpoint": {"videoId": "abc123", "playlistId": "PLTEST"}
	                              }
	                            }]
	                          }
	                        }
	                      },
	                      {
	                        "musicResponsiveListItemFlexColumnRenderer": {
	                          "text": {
	                            "runs": [{
	                              "text": "Neon",
	                              "navigationEndpoint": {
	                                "browseEndpoint": {
	                                  "browseId": "UCartist",
	                                  "browseEndpointContextSupportedConfigs": {
	                                    "browseEndpointContextMusicConfig": {
	                                      "pageType": "MUSIC_PAGE_TYPE_ARTIST"
	                                    }
	                                  }
	                                }
	                              }
	                            }, {"text": " • "}, {"text": "3:21"}]
	                          }
	                        }
	                      }
	                    ],
	                    "fixedColumns": [{
	                      "musicResponsiveListItemFixedColumnRenderer": {
	                        "text": {"simpleText": "3:21"}
	                      }
	                    }]
	                  }
	                },
	                {
	                  "continuationItemRenderer": {
	                    "continuationEndpoint": {
	                      "continuationCommand": {"token": "CONT_TOKEN"}
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
	var page any
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}

	detail := parsePlaylistDetail(page, "PLTEST")
	if detail.Title != "Chill Mix" {
		t.Fatalf("title=%q", detail.Title)
	}
	if detail.TrackCount != 42 {
		t.Fatalf("trackCount=%d", detail.TrackCount)
	}
	if detail.Author != "YouTube Music" {
		t.Fatalf("author=%q", detail.Author)
	}

	shelf := playlistShelf(page)
	contents, _ := getValue(shelf, path{"contents"}).([]any)
	tracks := parsePlaylistTracks(contents)
	if len(tracks) != 1 {
		t.Fatalf("tracks=%d", len(tracks))
	}
	if tracks[0].VideoID != "abc123" || tracks[0].Title != "Night Drive" {
		t.Fatalf("track=%+v", tracks[0])
	}
	if tracks[0].Duration != 201 {
		t.Fatalf("duration=%d", tracks[0].Duration)
	}
	if token := continuationToken(contents); token != "CONT_TOKEN" {
		t.Fatalf("token=%q", token)
	}
}

func TestNormalizePlaylistID(t *testing.T) {
	if got := normalizePlaylistID("VLPLABC"); got != "PLABC" {
		t.Fatalf("got %q", got)
	}
	if got := normalizePlaylistID("LM"); got != "LM" {
		t.Fatalf("got %q", got)
	}
}

func TestGetLikedSongsRequiresAuth(t *testing.T) {
	c := NewClient()
	c.Auth = nil
	_, err := c.GetLikedSongs(10)
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestDurationToIntSafe(t *testing.T) {
	if durationToInt("3:21") != 201 {
		t.Fatal("3:21")
	}
	if durationToInt("not-a-duration") != 0 {
		t.Fatal("non-duration")
	}
	if durationToInt(nil) != 0 {
		t.Fatal("nil")
	}
}

func TestGetPlaylistLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live InnerTube")
	}
	search := PlaylistSearch("lofi")
	result, err := search.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Playlists) == 0 {
		t.Fatal("no playlists from search")
	}
	id := result.Playlists[0].BrowseID
	detail, err := GetPlaylist(id, 5)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Title == "" {
		t.Fatal("missing playlist title")
	}
	if len(detail.Tracks) == 0 {
		t.Fatalf("no tracks for playlist %s (%s)", id, detail.Title)
	}
	if detail.Tracks[0].VideoID == "" || detail.Tracks[0].Title == "" {
		t.Fatalf("incomplete track: %+v", detail.Tracks[0])
	}
	t.Logf("playlist=%s title=%q tracks=%d first=%s", detail.ID, detail.Title, len(detail.Tracks), detail.Tracks[0].Title)
}
