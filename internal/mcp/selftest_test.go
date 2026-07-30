package mcp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shotah/youtube-go-mcp/internal/ytmusic"
)

func TestSelfTestUnauthedSearch(t *testing.T) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/search") {
			http.Error(w, "unexpected "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(searchBody))
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
	client.Auth = nil
	client.OAuth = nil

	if err := SelfTest(client); err != nil {
		t.Fatal(err)
	}
}

func TestSelfTestAuthed(t *testing.T) {
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
	playlistBody := `{
	  "contents": {
	    "twoColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "musicResponsiveHeaderRenderer": {
	                  "title": {"runs": [{"text": "Liked"}]},
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
	                  "playlistItemData": {"videoId": "liked1"},
	                  "flexColumns": [
	                    {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [{
	                      "text": "Fav",
	                      "navigationEndpoint": {"watchEndpoint": {"videoId": "liked1"}}
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/search"):
			_, _ = w.Write([]byte(searchBody))
		case strings.Contains(body, "FEmusic_liked_playlists"):
			_, _ = w.Write([]byte(libraryBody))
		case strings.Contains(body, "FEmusic_history"):
			_, _ = w.Write([]byte(historyBody))
		case strings.Contains(body, "VLLM") || strings.Contains(body, "VLPL"):
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

	if err := SelfTest(client); err != nil {
		t.Fatal(err)
	}
}
