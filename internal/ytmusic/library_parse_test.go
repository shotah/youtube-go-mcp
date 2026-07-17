package ytmusic

import (
	"encoding/json"
	"testing"
)

func TestParseLibraryPlaylistsFixture(t *testing.T) {
	raw := `{
	  "contents": {
	    "singleColumnBrowseResultsRenderer": {
	      "tabs": [{
	        "tabRenderer": {
	          "content": {
	            "sectionListRenderer": {
	              "contents": [{
	                "gridRenderer": {
	                  "items": [
	                    {
	                      "musicTwoRowItemRenderer": {
	                        "title": {"runs": [{"text": "New playlist"}]}
	                      }
	                    },
	                    {
	                      "musicTwoRowItemRenderer": {
	                        "title": {"runs": [{"text": "Road Trip"}]},
	                        "subtitle": {"runs": [{"text": "12 tracks"}]},
	                        "navigationEndpoint": {
	                          "browseEndpoint": {"browseId": "VLPLABCDEF"}
	                        },
	                        "thumbnailRenderer": {
	                          "musicThumbnailRenderer": {
	                            "thumbnail": {
	                              "thumbnails": [{"url": "https://example.com/a.jpg", "width": 60, "height": 60}]
	                            }
	                          }
	                        }
	                      }
	                    }
	                  ]
	                }
	              }]
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
	playlists := parseLibraryPlaylists(page)
	if len(playlists) != 1 {
		t.Fatalf("len=%d want 1", len(playlists))
	}
	if playlists[0].PlaylistID != "PLABCDEF" {
		t.Fatalf("playlistId=%q", playlists[0].PlaylistID)
	}
	if playlists[0].Title != "Road Trip" {
		t.Fatalf("title=%q", playlists[0].Title)
	}
}
