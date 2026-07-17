package ytmusic

import (
	"encoding/json"
	"testing"
)

func TestParseSearchPageItemSections(t *testing.T) {
	raw := `{
	  "contents": {
	    "tabbedSearchResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [
	                {
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
	                },
	                {
	                  "itemSectionRenderer": {
	                    "contents": [{
	                      "musicResponsiveListItemRenderer": {
	                        "playlistItemData": {"videoId": "vid1"},
	                        "flexColumns": [
	                          {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	                            {"text": "Video One", "navigationEndpoint": {"watchEndpoint": {"videoId": "vid1"}}}
	                          ]}}}
	                        ],
	                        "overlay": {
	                          "musicItemThumbnailOverlayRenderer": {
	                            "content": {
	                              "musicPlayButtonRenderer": {
	                                "playNavigationEndpoint": {
	                                  "watchEndpoint": {
	                                    "watchEndpointMusicSupportedConfigs": {
	                                      "watchEndpointMusicConfig": {"musicVideoType": "MUSIC_VIDEO_TYPE_OMV"}
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
	                },
	                {
	                  "itemSectionRenderer": {
	                    "contents": [{
	                      "musicResponsiveListItemRenderer": {
	                        "navigationEndpoint": {
	                          "browseEndpoint": {
	                            "browseId": "PLABC",
	                            "browseEndpointContextSupportedConfigs": {
	                              "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_PLAYLIST"}
	                            }
	                          }
	                        },
	                        "flexColumns": [
	                          {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [{"text": "My Playlist"}]}}}
	                        ]
	                      }
	                    }]
	                  }
	                }
	              ]
	            }
	          }
	        }
	      }]
	    }
	  }
	}`
	var page any
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}
	result, _ := parseSearchPage(page)
	if result == nil {
		t.Fatal("nil result")
	}
	if len(result.Tracks) != 1 || result.Tracks[0].VideoID != "track1" {
		t.Fatalf("tracks=%+v", result.Tracks)
	}
	if len(result.Videos) != 1 || result.Videos[0].VideoID != "vid1" {
		t.Fatalf("videos=%+v", result.Videos)
	}
	if len(result.Playlists) != 1 || result.Playlists[0].BrowseID != "PLABC" {
		t.Fatalf("playlists=%+v", result.Playlists)
	}
}

func TestParseTrackItemVideoIDFromPlaylistItemData(t *testing.T) {
	raw := `{
	  "musicResponsiveListItemRenderer": {
	    "playlistItemData": {"videoId": "onlyInPlaylistData"},
	    "flexColumns": [
	      {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	        {"text": "No Watch Endpoint"}
	      ]}}},
	      {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	        {"text": "Artist", "navigationEndpoint": {"browseEndpoint": {
	          "browseId": "UCa",
	          "browseEndpointContextSupportedConfigs": {
	            "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
	          }
	        }}}
	      ]}}}
	    ]
	  }
	}`
	var item any
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		t.Fatal(err)
	}
	track := parseTrackItem(item)
	if track == nil || track.VideoID != "onlyInPlaylistData" {
		t.Fatalf("got %+v", track)
	}
}
