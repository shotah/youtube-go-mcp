package ytmusic

import (
	"net/url"
)

func (c *Client) getWatchPlaylist(videoID string) ([]*TrackItem, error) {
	page, err := c.makeRequest(
		"next",
		map[string]any{
			"videoId":                       videoID,
			"playlistId":                    "RDAMVM" + videoID,
			"enablePersistentPlaylistPanel": true,
			"isAudioOnly":                   true,
		},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}
	return parseWatchPlaylist(page), nil
}
