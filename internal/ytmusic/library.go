package ytmusic

import (
	"net/url"
	"strings"
)

// LibraryPlaylist is a playlist from the authenticated user's library.
type LibraryPlaylist struct {
	PlaylistID string      `json:"playlistId"`
	Title      string      `json:"title"`
	Count      string      `json:"count,omitempty"`
	Thumbnails []Thumbnail `json:"thumbnails,omitempty"`
}

// GetLibraryPlaylists returns playlists from the signed-in YouTube Music library.
func GetLibraryPlaylists(limit int) ([]*LibraryPlaylist, error) {
	syncDefaultGlobals()
	return Default.GetLibraryPlaylists(limit)
}

// GetLibraryPlaylists returns playlists from the signed-in YouTube Music library.
// limit <= 0 means return all playlists found on the first page (and continuations not yet implemented beyond first page batch).
func (c *Client) GetLibraryPlaylists(limit int) ([]*LibraryPlaylist, error) {
	if !c.Authenticated() {
		return nil, ErrAuthRequired
	}

	page, err := c.makeRequest(
		"browse",
		map[string]any{
			"browseId": "FEmusic_liked_playlists",
		},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	playlists := parseLibraryPlaylists(page)
	if limit > 0 && len(playlists) > limit {
		playlists = playlists[:limit]
	}
	return playlists, nil
}

func parseLibraryPlaylists(page any) []*LibraryPlaylist {
	items := libraryGridItems(page)
	if items == nil {
		return nil
	}

	var out []*LibraryPlaylist
	for i, item := range items {
		// First grid item is usually "New playlist" / create shortcut — skip empty IDs.
		pl := parseLibraryPlaylistItem(item)
		if pl == nil || pl.PlaylistID == "" {
			if i == 0 {
				continue
			}
			continue
		}
		out = append(out, pl)
	}
	return out
}

func libraryGridItems(page any) []any {
	candidates := []path{
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "gridRenderer", "items"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 1, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "gridRenderer", "items"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 2, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "gridRenderer", "items"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "itemSectionRenderer", "contents", 0, "gridRenderer", "items"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 1, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "itemSectionRenderer", "contents", 0, "gridRenderer", "items"},
	}
	for _, p := range candidates {
		if v := getValue(page, p); v != nil {
			if items, ok := v.([]any); ok {
				return items
			}
		}
	}

	// Fallback: walk section list for any gridRenderer items.
	sectionLists := []path{
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 1, "tabRenderer", "content", "sectionListRenderer", "contents"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 2, "tabRenderer", "content", "sectionListRenderer", "contents"},
	}
	for _, p := range sectionLists {
		sections := getValue(page, p)
		list, ok := sections.([]any)
		if !ok {
			continue
		}
		for _, section := range list {
			if items := getValue(section, path{"gridRenderer", "items"}); items != nil {
				if arr, ok := items.([]any); ok {
					return arr
				}
			}
			if items := getValue(section, path{"itemSectionRenderer", "contents", 0, "gridRenderer", "items"}); items != nil {
				if arr, ok := items.([]any); ok {
					return arr
				}
			}
		}
	}
	return nil
}

func parseLibraryPlaylistItem(item any) *LibraryPlaylist {
	data := getValue(item, path{"musicTwoRowItemRenderer"})
	if data == nil {
		data = item
	}

	title := stringFromRuns(getValue(data, path{"title", "runs"}))
	if title == "" {
		if t, ok := getValue(data, path{"title", "simpleText"}).(string); ok {
			title = t
		}
	}

	playlistID := ""
	if id, ok := getValue(data, path{"navigationEndpoint", "watchPlaylistEndpoint", "playlistId"}).(string); ok {
		playlistID = id
	}
	if playlistID == "" {
		if id, ok := getValue(data, path{"navigationEndpoint", "browseEndpoint", "browseId"}).(string); ok {
			playlistID = strings.TrimPrefix(id, "VL")
		}
	}
	if playlistID == "" {
		if id, ok := getValue(data, path{"title", "runs", 0, "navigationEndpoint", "browseEndpoint", "browseId"}).(string); ok {
			playlistID = strings.TrimPrefix(id, "VL")
		}
	}
	// menu play action
	if playlistID == "" {
		if id, ok := getValue(data, path{"menu", "menuRenderer", "items", 0, "menuNavigationItemRenderer", "navigationEndpoint", "watchPlaylistEndpoint", "playlistId"}).(string); ok {
			playlistID = id
		}
	}

	count := ""
	if subtitle := getValue(data, path{"subtitle", "runs"}); subtitle != nil {
		count = stringFromRuns(subtitle)
	}

	var thumbs []Thumbnail
	if thumbnails := getValue(data, path{"thumbnailRenderer", "musicThumbnailRenderer", "thumbnail", "thumbnails"}); thumbnails != nil {
		thumbs = parseThumbnails(thumbnails)
	}

	if playlistID == "" && title == "" {
		return nil
	}
	return &LibraryPlaylist{
		PlaylistID: playlistID,
		Title:      title,
		Count:      count,
		Thumbnails: thumbs,
	}
}

func stringFromRuns(runsIface any) string {
	runs, ok := runsIface.([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, run := range runs {
		if text, ok := getValue(run, path{"text"}).(string); ok {
			b.WriteString(text)
		}
	}
	return b.String()
}
