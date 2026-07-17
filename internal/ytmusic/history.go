package ytmusic

import (
	"errors"
	"net/url"
)

// HistoryItem is a recently played track from the authenticated listening history.
type HistoryItem struct {
	VideoID    string      `json:"videoId"`
	PlaylistID string      `json:"playlistId,omitempty"`
	Title      string      `json:"title"`
	Artists    []Artist    `json:"artists"`
	Album      Album       `json:"album"`
	Duration   int         `json:"duration"`
	IsExplicit bool        `json:"isExplicit"`
	Thumbnails []Thumbnail `json:"thumbnails,omitempty"`
	// Played is the history shelf label (e.g. "Today", "Yesterday", "Friday").
	Played string `json:"played,omitempty"`
}

// GetHistory returns recently played tracks from YouTube Music history.
// Requires browser auth. limit <= 0 defaults to 50.
func GetHistory(limit int) ([]*HistoryItem, error) {
	syncDefaultGlobals()
	return Default.GetHistory(limit)
}

// GetHistory returns recently played tracks from YouTube Music history.
func (c *Client) GetHistory(limit int) ([]*HistoryItem, error) {
	if !c.Authenticated() {
		return nil, ErrAuthRequired
	}
	if limit <= 0 {
		limit = 50
	}

	page, err := c.makeRequest(
		"browse",
		map[string]any{"browseId": "FEmusic_history"},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	items := parseHistory(page)
	if items == nil {
		return nil, errors.New("ytmusic: couldn't parse listening history")
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func parseHistory(page any) []*HistoryItem {
	sections := historySections(page)
	if sections == nil {
		return nil
	}

	var out []*HistoryItem
	for _, section := range sections {
		if msg := getValue(section, path{"musicNotifierShelfRenderer"}); msg != nil {
			title := stringFromRuns(getValue(msg, path{"title", "runs"}))
			if title == "" {
				if t, ok := getValue(msg, path{"title", "simpleText"}).(string); ok {
					title = t
				}
			}
			if title != "" {
				// History disabled / empty notice — treat as empty result, not a hard failure.
				continue
			}
		}

		shelf := getValue(section, path{"musicShelfRenderer"})
		if shelf == nil {
			continue
		}
		played := stringFromRuns(getValue(shelf, path{"title", "runs"}))
		if played == "" {
			if t, ok := getValue(shelf, path{"title", "simpleText"}).(string); ok {
				played = t
			}
		}
		contents, _ := getValue(shelf, path{"contents"}).([]any)
		for _, item := range contents {
			track := parsePlaylistShelfTrack(item)
			if track == nil || track.VideoID == "" {
				continue
			}
			out = append(out, &HistoryItem{
				VideoID:    track.VideoID,
				PlaylistID: track.PlaylistID,
				Title:      track.Title,
				Artists:    track.Artists,
				Album:      track.Album,
				Duration:   track.Duration,
				IsExplicit: track.IsExplicit,
				Thumbnails: track.Thumbnails,
				Played:     played,
			})
		}
	}
	return out
}

func historySections(page any) []any {
	candidates := []path{
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 1, "tabRenderer", "content", "sectionListRenderer", "contents"},
		{"contents", "twoColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents"},
	}
	for _, p := range candidates {
		if v := getValue(page, p); v != nil {
			if sections, ok := v.([]any); ok {
				return sections
			}
		}
	}
	return nil
}
