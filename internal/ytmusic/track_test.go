package ytmusic

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseTrackDetailFixture(t *testing.T) {
	raw := `{
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
	                "endpoint": {
	                  "browseEndpoint": {"browseId": "MPLYt_test"}
	                }
	              }
	            }
	          ]
	        }
	      }
	    }
	  }
	}`
	var page any
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}
	detail := parseTrackDetail(page, "abc123")
	if detail.Title != "Test Song" || detail.VideoID != "abc123" {
		t.Fatalf("%+v", detail)
	}
	if len(detail.Artists) == 0 || detail.Artists[0].Name != "Artist Z" {
		t.Fatalf("artists=%+v", detail.Artists)
	}
	if detail.Duration != 125 {
		t.Fatalf("duration=%d", detail.Duration)
	}
	if id := lyricsBrowseID(page); id != "MPLYt_test" {
		t.Fatalf("browseId=%q", id)
	}
}

func TestExtractLyricsText(t *testing.T) {
	raw := `{
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
	var page any
	_ = json.Unmarshal([]byte(raw), &page)
	if got := extractLyricsText(page); got != "line one\nline two" {
		t.Fatalf("got %q", got)
	}
}

func TestGetTrackRequiresVideoID(t *testing.T) {
	_, err := NewClient().GetTrack("", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetLyricsNoLyricsIsSentinel(t *testing.T) {
	c := NewClient()
	_, err := c.lyricsFromNextPage(map[string]any{})
	if !errors.Is(err, ErrNoLyrics) {
		t.Fatalf("got %v", err)
	}
}
