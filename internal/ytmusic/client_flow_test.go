package ytmusic

import (
	"net/http"
	"strings"
	"testing"
)

const fixtureSearchPage = `{
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

const fixtureLibraryPlaylists = `{
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
                      "subtitle": {"runs": [{"text": "12 tracks"}]},
                      "navigationEndpoint": {
                        "browseEndpoint": {"browseId": "VLPLABCDEF"}
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

const fixturePlaylistBrowse = `{
  "contents": {
    "twoColumnBrowseResultsRenderer": {
      "tabs": [{
        "tabRenderer": {
          "content": {
            "sectionListRenderer": {
              "contents": [{
                "musicResponsiveHeaderRenderer": {
                  "title": {"runs": [{"text": "Chill Mix"}]},
                  "facepile": {"avatarStackViewModel": {"text": {"content": "YouTube Music"}}},
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
                      "navigationEndpoint": {"watchEndpoint": {"videoId": "abc123", "playlistId": "PLTEST"}}
                    }]}}},
                    {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
                      {"text": "Neon", "navigationEndpoint": {"browseEndpoint": {
                        "browseId": "UCartist",
                        "browseEndpointContextSupportedConfigs": {
                          "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
                        }
                      }}},
                      {"text": " • "}, {"text": "3:21"}
                    ]}}}
                  ],
                  "fixedColumns": [{
                    "musicResponsiveListItemFixedColumnRenderer": {
                      "text": {"simpleText": "3:21"}
                    }
                  }]
                }
              }]
            }
          }]
        }
      }
    }
  }
}`

const fixtureHistoryBrowse = `{
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
                        }},
                        {"musicResponsiveListItemFlexColumnRenderer": {
                          "text": {"runs": [
                            {"text": "Artist A", "navigationEndpoint": {"browseEndpoint": {
                              "browseId": "UC1",
                              "browseEndpointContextSupportedConfigs": {
                                "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
                              }
                            }}},
                            {"text": " • "}, {"text": "3:00"}
                          ]}
                        }}
                      ],
                      "fixedColumns": [{
                        "musicResponsiveListItemFixedColumnRenderer": {
                          "text": {"simpleText": "3:00"}
                        }
                      }]
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

const fixtureNextTrack = `{
  "contents": {
    "singleColumnMusicWatchNextResultsRenderer": {
      "tabbedRenderer": {
        "watchNextTabbedResultsRenderer": {
          "tabs": [
            {
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
                            "longBylineText": {"runs": [
                              {"text": "Artist Z", "navigationEndpoint": {"browseEndpoint": {
                                "browseId": "UCz",
                                "browseEndpointContextSupportedConfigs": {
                                  "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
                                }
                              }}}
                            ]},
                            "lengthText": {"runs": [{"text": "2:05"}]}
                          }
                        }]
                      }
                    }
                  }
                }
              }
            },
            {
              "tabRenderer": {
                "title": "Lyrics",
                "endpoint": {"browseEndpoint": {"browseId": "MPLYt_test"}}
              }
            }
          ]
        }
      }
    }
  }
}`

const fixtureLyricsBrowse = `{
  "contents": {
    "sectionListRenderer": {
      "contents": [{
        "musicDescriptionShelfRenderer": {
          "description": {"runs": [{"text": "line one\nline two"}]}
        }
      }]
    }
  }
}`

func TestClientTrackSearchHTTP(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/search") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if !strings.Contains(readBody(r), `"query":"ncs"`) {
			t.Fatalf("missing query in body")
		}
		writeJSON(w, fixtureSearchPage)
	})
	result, err := c.TrackSearch("ncs").Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tracks) != 1 || result.Tracks[0].VideoID != "track1" {
		t.Fatalf("tracks=%+v", result.Tracks)
	}
	sc := c.TrackSearch("x")
	if !sc.NextExists() {
		t.Fatal("expected NextExists before first page")
	}
	lang := c.withLanguage("de")
	if lang.Language != "de" {
		t.Fatalf("language=%q", lang.Language)
	}
	reg := c.withRegion("GB")
	if reg.Region != "GB" {
		t.Fatalf("region=%q", reg.Region)
	}
}

func TestClientLibraryPlaylistHistoryHTTP(t *testing.T) {
	c := withBrowserAuth(testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		switch {
		case strings.Contains(body, "FEmusic_liked_playlists"):
			writeJSON(w, fixtureLibraryPlaylists)
		case strings.Contains(body, `"browseId":"VLPLTEST"`) || strings.Contains(body, `"browseId":"VLPLABCDEF"`):
			writeJSON(w, fixturePlaylistBrowse)
		case strings.Contains(body, "FEmusic_history"):
			writeJSON(w, fixtureHistoryBrowse)
		case strings.Contains(body, `"browseId":"VLLM"`):
			writeJSON(w, fixturePlaylistBrowse)
		default:
			t.Fatalf("unexpected browse body: %s", body)
		}
	}))

	playlists, err := c.GetLibraryPlaylists(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 1 || playlists[0].PlaylistID != "PLABCDEF" {
		t.Fatalf("%+v", playlists)
	}

	detail, err := c.GetPlaylist("PLTEST", 10)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Title != "Chill Mix" || len(detail.Tracks) != 1 {
		t.Fatalf("%+v", detail)
	}

	liked, err := c.GetLikedSongs(10)
	if err != nil {
		t.Fatal(err)
	}
	if liked == nil || len(liked.Tracks) == 0 {
		t.Fatalf("%+v", liked)
	}

	hist, err := c.GetHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) == 0 || hist[0].VideoID != "hist1" {
		t.Fatalf("%+v", hist)
	}
}

func TestClientGetTrackAndLyricsHTTP(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		switch {
		case strings.Contains(r.URL.Path, "/next"):
			writeJSON(w, fixtureNextTrack)
		case strings.Contains(body, "MPLYt_test"):
			writeJSON(w, fixtureLyricsBrowse)
		default:
			t.Fatalf("unexpected: path=%s body=%s", r.URL.Path, body)
		}
	})

	detail, err := c.GetTrack("abc123", true)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Title != "Test Song" || !detail.HasLyrics || detail.Lyrics == "" {
		t.Fatalf("%+v", detail)
	}

	lyrics, err := c.GetLyrics("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if lyrics == "" {
		t.Fatal("empty lyrics")
	}

	tracks, err := c.GetWatchPlaylist("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) == 0 || tracks[0].VideoID != "abc123" {
		t.Fatalf("%+v", tracks)
	}
}

func TestClientAuthRequiredPaths(t *testing.T) {
	c := testClient(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("should not call network without auth")
	})
	if _, err := c.GetLibraryPlaylists(1); err == nil {
		t.Fatal("expected auth error")
	}
	if _, err := c.GetHistory(1); err == nil {
		t.Fatal("expected auth error")
	}
	if _, err := c.GetLikedSongs(1); err == nil {
		t.Fatal("expected auth error")
	}
}

func TestPackageLevelWrappersHTTP(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		switch {
		case strings.Contains(r.URL.Path, "get_search_suggestions"):
			writeJSON(w, `{
			  "contents": [{
			    "searchSuggestionsSectionRenderer": {
			      "contents": [
			        {"searchSuggestionRenderer": {
			          "navigationEndpoint": {"searchEndpoint": {"query": "ncs"}}
			        }}
			      ]
			    }
			  }]
			}`)
		case strings.Contains(r.URL.Path, "/youtubei/v1/search"):
			writeJSON(w, fixtureSearchPage)
		case strings.Contains(r.URL.Path, "/youtubei/v1/next"):
			writeJSON(w, fixtureNextTrack)
		case strings.Contains(r.URL.Path, "/youtubei/v1/browse") && strings.Contains(body, "MPLYt_test"):
			writeJSON(w, fixtureLyricsBrowse)
		default:
			t.Fatalf("unexpected path=%s body=%s", r.URL.Path, body)
		}
	})
	old := Default
	oldHTTP := HTTPClient
	Default = c
	HTTPClient = c.HTTPClient // syncDefaultGlobals() copies this onto Default
	t.Cleanup(func() {
		Default = old
		HTTPClient = oldHTTP
	})

	for _, sc := range []*SearchClient{
		Search("ncs"),
		TrackSearch("ncs"),
		AlbumSearch("ncs"),
		ArtistSearch("ncs"),
		PlaylistSearch("ncs"),
		VideoSearch("ncs"),
	} {
		if _, err := sc.Next(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GetWatchPlaylist("abc123"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetSearchSuggestions("nc"); err != nil {
		t.Fatal(err)
	}
	lyrics, err := GetLyrics("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if lyrics == "" {
		t.Fatal("empty lyrics from package wrapper")
	}
	if _, err := GetTrack("abc123", false); err != nil {
		t.Fatal(err)
	}
}
