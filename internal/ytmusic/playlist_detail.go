package ytmusic

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// LikedSongsPlaylistID is the special playlist id for the user's Liked Songs.
const LikedSongsPlaylistID = "LM"

// PlaylistDetail is a playlist with its tracks.
type PlaylistDetail struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Author     string       `json:"author,omitempty"`
	TrackCount int          `json:"trackCount,omitempty"`
	Tracks     []*TrackItem `json:"tracks"`
}

// GetPlaylist returns playlist metadata and tracks for the given playlist id.
// Accepts bare ids (PL…, LM, RD…) or VL-prefixed browse ids.
// limit <= 0 defaults to 100. Continuations are followed until limit is reached.
func GetPlaylist(playlistID string, limit int) (*PlaylistDetail, error) {
	syncDefaultGlobals()
	return Default.GetPlaylist(playlistID, limit)
}

// GetLikedSongs returns tracks from the authenticated user's Liked Songs playlist.
func GetLikedSongs(limit int) (*PlaylistDetail, error) {
	syncDefaultGlobals()
	return Default.GetLikedSongs(limit)
}

// GetPlaylist returns playlist metadata and tracks for the given playlist id.
func (c *Client) GetPlaylist(playlistID string, limit int) (*PlaylistDetail, error) {
	playlistID = normalizePlaylistID(playlistID)
	if playlistID == "" {
		return nil, errors.New("ytmusic: playlist id is required")
	}
	if limit <= 0 {
		limit = 100
	}

	browseID := playlistID
	if !strings.HasPrefix(browseID, "VL") {
		browseID = "VL" + browseID
	}

	page, err := c.makeRequest(
		"browse",
		map[string]any{"browseId": browseID},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	detail := parsePlaylistDetail(page, playlistID)
	if detail == nil {
		return nil, fmt.Errorf("ytmusic: couldn't parse playlist %s", playlistID)
	}

	shelf := playlistShelf(page)
	if shelf != nil {
		contents, _ := getValue(shelf, path{"contents"}).([]any)
		detail.Tracks = append(detail.Tracks, parsePlaylistTracks(contents)...)
		if token := continuationToken(contents); token != "" && len(detail.Tracks) < limit {
			more, err := c.fetchPlaylistContinuations(token, limit-len(detail.Tracks))
			if err != nil {
				return nil, err
			}
			detail.Tracks = append(detail.Tracks, more...)
		}
	}

	if len(detail.Tracks) > limit {
		detail.Tracks = detail.Tracks[:limit]
	}
	if detail.TrackCount == 0 {
		detail.TrackCount = len(detail.Tracks)
	}
	return detail, nil
}

// GetLikedSongs returns tracks from the authenticated user's Liked Songs playlist.
func (c *Client) GetLikedSongs(limit int) (*PlaylistDetail, error) {
	if !c.Authenticated() {
		return nil, ErrAuthRequired
	}
	return c.GetPlaylist(LikedSongsPlaylistID, limit)
}

func (c *Client) fetchPlaylistContinuations(token string, remaining int) ([]*TrackItem, error) {
	var out []*TrackItem
	for token != "" && remaining > 0 {
		page, err := c.makeRequest(
			"browse",
			map[string]any{"continuation": token},
			url.Values{},
		)
		if err != nil {
			return out, err
		}
		items, _ := getValue(page, path{"onResponseReceivedActions", 0, "appendContinuationItemsAction", "continuationItems"}).([]any)
		if len(items) == 0 {
			break
		}
		tracks := parsePlaylistTracks(items)
		if len(tracks) == 0 {
			break
		}
		if len(tracks) > remaining {
			tracks = tracks[:remaining]
		}
		out = append(out, tracks...)
		remaining -= len(tracks)
		token = continuationToken(items)
	}
	return out, nil
}

func normalizePlaylistID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "VL")
	return id
}

func playlistShelf(page any) any {
	candidates := []path{
		{"contents", "twoColumnBrowseResultsRenderer", "secondaryContents", "sectionListRenderer", "contents", 0, "musicPlaylistShelfRenderer"},
		{"contents", "twoColumnBrowseResultsRenderer", "secondaryContents", "sectionListRenderer", "contents", 0, "itemSectionRenderer", "contents", 0, "musicPlaylistShelfRenderer"},
		{"contents", "singleColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicPlaylistShelfRenderer"},
	}
	for _, p := range candidates {
		if v := getValue(page, p); v != nil {
			return v
		}
	}
	// Walk section list for a playlist shelf.
	sections := getValue(page, path{"contents", "twoColumnBrowseResultsRenderer", "secondaryContents", "sectionListRenderer", "contents"})
	list, ok := sections.([]any)
	if !ok {
		return nil
	}
	for _, section := range list {
		if v := getValue(section, path{"musicPlaylistShelfRenderer"}); v != nil {
			return v
		}
		if v := getValue(section, path{"itemSectionRenderer", "contents", 0, "musicPlaylistShelfRenderer"}); v != nil {
			return v
		}
	}
	return nil
}

func parsePlaylistDetail(page any, fallbackID string) *PlaylistDetail {
	detail := &PlaylistDetail{ID: fallbackID, Tracks: []*TrackItem{}}

	headerPaths := []path{
		{"contents", "twoColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicResponsiveHeaderRenderer"},
		{"contents", "twoColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicEditablePlaylistDetailHeaderRenderer", "header", "musicResponsiveHeaderRenderer"},
		{"header", "musicDetailHeaderRenderer"},
	}
	var header any
	for _, p := range headerPaths {
		if v := getValue(page, p); v != nil {
			header = v
			break
		}
	}
	editable := getValue(page, path{"contents", "twoColumnBrowseResultsRenderer", "tabs", 0, "tabRenderer", "content", "sectionListRenderer", "contents", 0, "musicEditablePlaylistDetailHeaderRenderer"})
	if editable != nil {
		if id, ok := getValue(editable, path{"playlistId"}).(string); ok && id != "" {
			detail.ID = normalizePlaylistID(id)
		}
		if header == nil {
			header = getValue(editable, path{"header", "musicResponsiveHeaderRenderer"})
		}
	}

	applyPlaylistHeader(detail, header)

	if detail.Title == "" && fallbackID == LikedSongsPlaylistID {
		detail.Title = "Liked Songs"
	}
	if detail.ID == "" {
		detail.ID = fallbackID
	}
	return detail
}

func applyPlaylistHeader(detail *PlaylistDetail, header any) {
	if detail == nil || header == nil {
		return
	}
	if title := stringFromRuns(getValue(header, path{"title", "runs"})); title != "" {
		detail.Title = title
	}
	if author, ok := getValue(header, path{"facepile", "avatarStackViewModel", "text", "content"}).(string); ok {
		detail.Author = author
	}
	if detail.Author == "" {
		if runs := getValue(header, path{"subtitle", "runs"}); runs != nil {
			detail.Author = stringFromRuns(runs)
		}
	}
	if runs, ok := getValue(header, path{"secondSubtitle", "runs"}).([]any); ok && len(runs) > 0 {
		if text, ok := getValue(runs[0], path{"text"}).(string); ok {
			detail.TrackCount = extractLeadingInt(text)
		}
	}
}

func parsePlaylistTracks(contents []any) []*TrackItem {
	var out []*TrackItem
	for _, item := range contents {
		if getValue(item, path{"continuationItemRenderer"}) != nil {
			continue
		}
		track := parsePlaylistShelfTrack(item)
		if track == nil || track.VideoID == "" {
			continue
		}
		out = append(out, track)
	}
	return out
}

func parsePlaylistShelfTrack(item any) *TrackItem {
	data := getValue(item, path{"musicResponsiveListItemRenderer"})
	if data == nil {
		return nil
	}

	track := parseTrackItem(item)
	if track == nil {
		track = &TrackItem{}
	}

	if track.VideoID == "" {
		if v, ok := getValue(data, path{"playlistItemData", "videoId"}).(string); ok {
			track.VideoID = v
		}
	}
	if track.VideoID == "" {
		if v, ok := getValue(data, path{"overlay", "musicItemThumbnailOverlayRenderer", "content", "musicPlayButtonRenderer", "playNavigationEndpoint", "watchEndpoint", "videoId"}).(string); ok {
			track.VideoID = v
		}
	}
	if track.Title == "" {
		track.Title = stringFromRuns(getValue(data, path{"flexColumns", 0, "musicResponsiveListItemFlexColumnRenderer", "text", "runs"}))
	}
	if track.Title == "Song deleted" {
		return nil
	}

	if track.Duration == 0 {
		if d, ok := getValue(data, path{"fixedColumns", 0, "musicResponsiveListItemFixedColumnRenderer", "text", "simpleText"}).(string); ok {
			track.Duration = durationToInt(d)
		} else if d, ok := getValue(data, path{"fixedColumns", 0, "musicResponsiveListItemFixedColumnRenderer", "text", "runs", 0, "text"}).(string); ok {
			track.Duration = durationToInt(d)
		}
	}

	if len(track.Artists) == 0 {
		track.Artists = artistsFromFlexRuns(getValue(data, path{"flexColumns", 1, "musicResponsiveListItemFlexColumnRenderer", "text", "runs"}))
	}

	if track.VideoID == "" {
		return nil
	}
	return track
}

func artistsFromFlexRuns(runsIface any) []Artist {
	runs, ok := runsIface.([]any)
	if !ok {
		return nil
	}
	var artists []Artist
	for _, run := range runs {
		text, _ := getValue(run, path{"text"}).(string)
		if text == "" || text == " • " || text == "•" || strings.Contains(text, ":") {
			continue
		}
		if getValue(run, path{"navigationEndpoint", "browseEndpoint", "browseEndpointContextSupportedConfigs", "browseEndpointContextMusicConfig", "pageType"}) != nil {
			pageType, _ := getValue(run, path{"navigationEndpoint", "browseEndpoint", "browseEndpointContextSupportedConfigs", "browseEndpointContextMusicConfig", "pageType"}).(string)
			if pageType == "MUSIC_PAGE_TYPE_ALBUM" || pageType == "MUSIC_PAGE_TYPE_AUDIOBOOK" {
				continue
			}
		}
		artist := Artist{Name: text}
		if id, ok := getValue(run, path{"navigationEndpoint", "browseEndpoint", "browseId"}).(string); ok {
			artist.ID = id
		}
		artists = append(artists, artist)
	}
	return artists
}

func continuationToken(items []any) string {
	if len(items) == 0 {
		return ""
	}
	last := items[len(items)-1]
	if token, ok := getValue(last, path{"continuationItemRenderer", "continuationEndpoint", "continuationCommand", "token"}).(string); ok && token != "" {
		return token
	}
	commands, _ := getValue(last, path{"continuationItemRenderer", "continuationEndpoint", "commandExecutorCommand", "commands"}).([]any)
	for _, cmd := range commands {
		if req, ok := getValue(cmd, path{"continuationCommand", "request"}).(string); ok && req == "CONTINUATION_REQUEST_TYPE_BROWSE" {
			if token, ok := getValue(cmd, path{"continuationCommand", "token"}).(string); ok {
				return token
			}
		}
		if token, ok := getValue(cmd, path{"continuationCommand", "token"}).(string); ok && token != "" {
			return token
		}
	}
	return ""
}

func extractLeadingInt(s string) int {
	n := 0
	found := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			found = true
			n = n*10 + int(r-'0')
			continue
		}
		if found {
			break
		}
	}
	return n
}
