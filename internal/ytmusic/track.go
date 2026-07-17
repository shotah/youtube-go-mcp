package ytmusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrNoLyrics means the track has no lyrics tab / synced lyrics on YouTube Music.
var ErrNoLyrics = errors.New("ytmusic: lyrics not available for this track")

// TrackDetail is metadata for a single videoId, optionally with lyrics.
type TrackDetail struct {
	VideoID    string      `json:"videoId"`
	PlaylistID string      `json:"playlistId,omitempty"`
	Title      string      `json:"title"`
	Artists    []Artist    `json:"artists"`
	Album      Album       `json:"album"`
	Duration   int         `json:"duration"`
	IsExplicit bool        `json:"isExplicit"`
	Thumbnails []Thumbnail `json:"thumbnails,omitempty"`
	Lyrics     string      `json:"lyrics,omitempty"`
	HasLyrics  bool        `json:"hasLyrics"`
}

// GetTrack returns metadata for a videoId. When includeLyrics is true, lyrics are
// fetched when YouTube Music exposes them (empty string + HasLyrics=false otherwise).
func GetTrack(videoID string, includeLyrics bool) (*TrackDetail, error) {
	syncDefaultGlobals()
	return Default.GetTrack(videoID, includeLyrics)
}

// GetTrack returns metadata for a videoId.
func (c *Client) GetTrack(videoID string, includeLyrics bool) (*TrackDetail, error) {
	if videoID == "" {
		return nil, errors.New("ytmusic: video id is required")
	}

	page, err := c.makeRequest(
		"next",
		map[string]any{
			"videoId":                       videoID,
			"enablePersistentPlaylistPanel": true,
			"isAudioOnly":                   true,
		},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	detail := parseTrackDetail(page, videoID)
	if detail == nil {
		return nil, fmt.Errorf("ytmusic: couldn't parse track %s", videoID)
	}

	if includeLyrics {
		lyrics, err := c.lyricsFromNextPage(page)
		switch {
		case errors.Is(err, ErrNoLyrics):
			detail.HasLyrics = false
		case err != nil:
			return nil, err
		default:
			detail.Lyrics = lyrics
			detail.HasLyrics = lyrics != ""
		}
	}
	return detail, nil
}

func (c *Client) getLyrics(videoID string) (string, error) {
	if videoID == "" {
		return "", errors.New("ytmusic: video id is required")
	}
	page, err := c.makeRequest(
		"next",
		map[string]any{
			"videoId":                       videoID,
			"enablePersistentPlaylistPanel": true,
			"isAudioOnly":                   true,
		},
		url.Values{},
	)
	if err != nil {
		return "", err
	}
	return c.lyricsFromNextPage(page)
}

func (c *Client) lyricsFromNextPage(page any) (string, error) {
	browseID := lyricsBrowseID(page)
	if browseID == "" {
		return "", ErrNoLyrics
	}

	lyricsPage, err := c.makeRequest(
		"browse",
		map[string]any{"browseId": browseID},
		url.Values{},
	)
	if err != nil {
		return "", err
	}

	lyrics := extractLyricsText(lyricsPage)
	if lyrics == "" {
		return "", ErrNoLyrics
	}
	return lyrics, nil
}

func lyricsBrowseID(page any) string {
	candidates := []path{
		{"contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs", 1, "tabRenderer", "endpoint", "browseEndpoint", "browseId"},
		{"contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs", 2, "tabRenderer", "endpoint", "browseEndpoint", "browseId"},
	}
	for _, p := range candidates {
		if id, ok := getValue(page, p).(string); ok && id != "" {
			return id
		}
	}

	tabs, _ := getValue(page, path{"contents", "singleColumnMusicWatchNextResultsRenderer", "tabbedRenderer", "watchNextTabbedResultsRenderer", "tabs"}).([]any)
	for _, tab := range tabs {
		id, _ := getValue(tab, path{"tabRenderer", "endpoint", "browseEndpoint", "browseId"}).(string)
		if id == "" {
			continue
		}
		title := stringFromRuns(getValue(tab, path{"tabRenderer", "title", "runs"}))
		if title == "" {
			if t, ok := getValue(tab, path{"tabRenderer", "title"}).(string); ok {
				title = t
			}
		}
		if strings.HasPrefix(id, "MPLY") || strings.Contains(strings.ToLower(title), "lyric") {
			return id
		}
	}
	return ""
}

func extractLyricsText(page any) string {
	paths := []path{
		{"contents", "sectionListRenderer", "contents", 0, "musicDescriptionShelfRenderer", "description", "runs", 0, "text"},
		{"contents", "sectionListRenderer", "contents", 0, "musicDescriptionShelfRenderer", "description", "simpleText"},
		{"contents", "messageRenderer", "text", "runs", 0, "text"},
	}
	for _, p := range paths {
		if s, ok := getValue(page, p).(string); ok && s != "" {
			if strings.Contains(strings.ToLower(s), "not available") {
				return ""
			}
			return s
		}
	}
	return ""
}

func parseTrackDetail(page any, fallbackID string) *TrackDetail {
	tracks := parseWatchPlaylist(page)
	var track *TrackItem
	for _, t := range tracks {
		if t == nil {
			continue
		}
		if t.VideoID == fallbackID {
			track = t
			break
		}
		if track == nil {
			track = t
		}
	}
	if track == nil {
		return &TrackDetail{VideoID: fallbackID}
	}
	return &TrackDetail{
		VideoID:    firstNonEmpty(track.VideoID, fallbackID),
		PlaylistID: track.PlaylistID,
		Title:      track.Title,
		Artists:    track.Artists,
		Album:      track.Album,
		Duration:   track.Duration,
		IsExplicit: track.IsExplicit,
		Thumbnails: track.Thumbnails,
	}
}
