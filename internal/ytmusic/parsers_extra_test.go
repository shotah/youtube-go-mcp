package ytmusic

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestParseArtistAndAlbumItems(t *testing.T) {
	artistRaw := `{
	  "musicResponsiveListItemRenderer": {
	    "navigationEndpoint": {"browseEndpoint": {"browseId": "UCartist"}},
	    "flexColumns": [{
	      "musicResponsiveListItemFlexColumnRenderer": {
	        "text": {"runs": [{"text": "Neon"}]}
	      }
	    }],
	    "menu": {
	      "menuRenderer": {
	        "items": [{
	          "menuNavigationItemRenderer": {
	            "navigationEndpoint": {
	              "watchPlaylistEndpoint": {"playlistId": "RDshuffle"}
	            }
	          }
	        }]
	      }
	    },
	    "thumbnail": {
	      "musicThumbnailRenderer": {
	        "thumbnail": {"thumbnails": [{"url": "https://example.com/a.jpg", "width": 60}]}
	      }
	    }
	  }
	}`
	var artistItem any
	if err := json.Unmarshal([]byte(artistRaw), &artistItem); err != nil {
		t.Fatal(err)
	}
	artist := parseArtistItem(artistItem)
	if artist.BrowseID != "UCartist" || artist.Artist != "Neon" || artist.ShuffleID != "RDshuffle" {
		t.Fatalf("%+v", artist)
	}

	albumRaw := `{
	  "musicResponsiveListItemRenderer": {
	    "navigationEndpoint": {"browseEndpoint": {"browseId": "MPalbum"}},
	    "flexColumns": [
	      {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [{"text": "Night LP"}]}}},
	      {"musicResponsiveListItemFlexColumnRenderer": {"text": {"runs": [
	        {"text": "Album"},
	        {"text": "Neon", "navigationEndpoint": {"browseEndpoint": {
	          "browseId": "UCartist",
	          "browseEndpointContextSupportedConfigs": {
	            "browseEndpointContextMusicConfig": {"pageType": "MUSIC_PAGE_TYPE_ARTIST"}
	          }
	        }}},
	        {"text": "2020"}
	      ]}}}
	    ],
	    "badges": [{
	      "musicInlineBadgeRenderer": {"icon": {"iconType": "MUSIC_EXPLICIT_BADGE"}}
	    }]
	  }
	}`
	var albumItem any
	if err := json.Unmarshal([]byte(albumRaw), &albumItem); err != nil {
		t.Fatal(err)
	}
	album := parseAlbumItem(albumItem)
	if album.BrowseID != "MPalbum" || album.Title != "Night LP" || !album.IsExplicit {
		t.Fatalf("%+v", album)
	}
	if len(album.Artists) == 0 || album.Artists[0].Name != "Neon" {
		t.Fatalf("artists=%+v", album.Artists)
	}
}

func TestParseSearchSuggestions(t *testing.T) {
	raw := `{
	  "contents": [{
	    "searchSuggestionsSectionRenderer": {
	      "contents": [
	        {"searchSuggestionRenderer": {
	          "navigationEndpoint": {"searchEndpoint": {"query": "night drive"}}
	        }},
	        {"searchSuggestionRenderer": {
	          "navigationEndpoint": {"searchEndpoint": {"query": "neon"}}
	        }}
	      ]
	    }
	  }]
	}`
	var page any
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}
	got := parseSearchSuggestions(page)
	if len(got) != 2 || got[0] != "night drive" || got[1] != "neon" {
		t.Fatalf("%v", got)
	}
}

func TestGetSearchSuggestionsHTTP(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "get_search_suggestions") {
			t.Fatalf("path=%s", r.URL.Path)
		}
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
	})
	got, err := c.GetSearchSuggestions("nc")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "ncs" {
		t.Fatalf("%v", got)
	}
}
