package ytmusic

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestParseHistoryFixture(t *testing.T) {
	raw := `{
	  "contents": {
	    "singleColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [
	                {
	                  "musicShelfRenderer": {
	                    "title": {"runs": [{"text": "Today"}]},
	                    "contents": [{
	                      "musicResponsiveListItemRenderer": {
	                        "playlistItemData": {"videoId": "hist1"},
	                        "flexColumns": [
	                          {
	                            "musicResponsiveListItemFlexColumnRenderer": {
	                              "text": {"runs": [{"text": "Song A", "navigationEndpoint": {"watchEndpoint": {"videoId": "hist1"}}}]}
	                            }
	                          },
	                          {
	                            "musicResponsiveListItemFlexColumnRenderer": {
	                              "text": {"runs": [
	                                {"text": "Artist A", "navigationEndpoint": {"browseEndpoint": {"browseId": "UC1", "browseEndpointContextSupportedConfigs": {"browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}}}}},
	                                {"text": " • "},
	                                {"text": "3:00"}
	                              ]}
	                            }
	                          }
	                        ],
	                        "fixedColumns": [{
	                          "musicResponsiveListItemFixedColumnRenderer": {
	                            "text": {"simpleText": "3:00"}
	                          }
	                        }]
	                      }
	                    }]
	                  }
	                },
	                {
	                  "musicShelfRenderer": {
	                    "title": {"runs": [{"text": "Yesterday"}]},
	                    "contents": [{
	                      "musicResponsiveListItemRenderer": {
	                        "playlistItemData": {"videoId": "hist2"},
	                        "flexColumns": [{
	                          "musicResponsiveListItemFlexColumnRenderer": {
	                            "text": {"runs": [{"text": "Song B"}]}
	                          }
	                        }]
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
	items := parseHistory(page)
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].VideoID != "hist1" || items[0].Played != "Today" || items[0].Title != "Song A" {
		t.Fatalf("item0=%+v", items[0])
	}
	if items[0].Duration != 180 {
		t.Fatalf("duration=%d", items[0].Duration)
	}
	if items[1].VideoID != "hist2" || items[1].Played != "Yesterday" {
		t.Fatalf("item1=%+v", items[1])
	}
}

func TestGetHistoryRequiresAuth(t *testing.T) {
	c := NewClient()
	c.Auth = nil
	_, err := c.GetHistory(10)
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("got %v", err)
	}
}
