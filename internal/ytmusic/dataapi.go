package ytmusic

import (
	"fmt"

	"github.com/shotah/youtube-go-mcp/internal/youtube"
)

// DataAPIProbe summarizes YouTube Data API v3 identity + liked-videos smoke.
type DataAPIProbe struct {
	ChannelID      string `json:"channelId,omitempty"`
	ChannelTitle   string `json:"channelTitle,omitempty"`
	LikedVideos    int    `json:"likedVideos"`
	LikedSample    string `json:"likedSampleTitle,omitempty"`
	Scope          string `json:"scope,omitempty"`
	MusicCategoryN int    `json:"likedMusicCategoryCount,omitempty"`
	Hint           string `json:"hint,omitempty"`
}

// ProbeDataAPI refreshes OAuth if needed, then calls Data API v3
// (channels.list?mine=true + videos.list?myRating=like) via internal/youtube.
func (c *Client) ProbeDataAPI() (*DataAPIProbe, error) {
	if c == nil || c.OAuth == nil || !c.OAuth.Ready() {
		return nil, fmt.Errorf("%w: ProbeDataAPI requires OAuth", ErrAuthRequired)
	}
	yc := youtube.New(c.OAuth)
	if c.HTTPClient != nil {
		yc.HTTPClient = c.HTTPClient
	}

	out := &DataAPIProbe{}
	if info, err := c.WhoAmI(); err == nil && info != nil {
		out.ChannelID = info.ChannelID
		out.ChannelTitle = info.ChannelTitle
		out.Scope = info.Scope
	} else {
		ch, chErr := yc.ChannelMine()
		if chErr != nil {
			return out, fmt.Errorf("channels.list: %w", chErr)
		}
		out.ChannelID = ch.ID
		out.ChannelTitle = ch.Title
	}

	liked, err := yc.ListLikedVideos(youtube.ListOptions{MaxResults: 25})
	if err != nil {
		return out, fmt.Errorf("videos.list myRating=like: %w", err)
	}
	out.LikedVideos = len(liked)
	if len(liked) > 0 {
		out.LikedSample = liked[0].Title
	}
	for i := range liked {
		if liked[i].CategoryID == youtube.CategoryMusic {
			out.MusicCategoryN++
		}
	}

	switch {
	case out.ChannelID == "":
		out.Hint = "OAuth works but channels.list returned no channel — check the Google account that approved the device code."
	case out.LikedVideos == 0:
		out.Hint = "No liked videos via Data API (YouTube thumbs-up / LL). This is not YouTube Music Liked Songs. Like videos on youtube.com, or use music-leaning search."
	case out.MusicCategoryN == 0 && out.LikedVideos > 0:
		out.Hint = "Liked videos present but none with categoryId=10 (Music) in this page — taste may still be useful; filter heuristics can help."
	default:
		out.Hint = "Data API identity + liked videos OK. Music-leaning count is categoryId=10 on this page only."
	}
	return out, nil
}
