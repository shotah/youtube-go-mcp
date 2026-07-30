package youtube

import "time"

// CategoryMusic is YouTube's video category id for Music.
const CategoryMusic = "10"

// Channel is the authenticated user's channel (channels.list?mine=true).
type Channel struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	LikesPlaylist string `json:"likesPlaylistId,omitempty"`
	Uploads       string `json:"uploadsPlaylistId,omitempty"`
}

// Video is an agent-friendly Data API video row.
type Video struct {
	VideoID      string        `json:"videoId"`
	Title        string        `json:"title"`
	ChannelID    string        `json:"channelId,omitempty"`
	ChannelTitle string        `json:"channelTitle,omitempty"`
	CategoryID   string        `json:"categoryId,omitempty"`
	Duration     time.Duration `json:"-"`
	DurationSec  int           `json:"durationSec,omitempty"`
	PublishedAt  string        `json:"publishedAt,omitempty"`
	URL          string        `json:"url"`
	MusicURL     string        `json:"musicUrl,omitempty"`
}

// Playlist is a playlists.list row.
type Playlist struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ItemCount   int64  `json:"itemCount,omitempty"`
	URL         string `json:"url,omitempty"`
}

// SearchOptions controls search.list.
type SearchOptions struct {
	Query      string
	MaxResults int  // default 25, max 50 per page; client pages until MaxResults total when > 50
	MusicOnly  bool // sets videoCategoryId=10
	PageToken  string
}

// ListOptions is shared pagination for list endpoints.
type ListOptions struct {
	MaxResults int
	PageToken  string
	// MusicOnly keeps only music-leaning rows (category / Topic / title heuristics).
	MusicOnly bool
}

// Page is a single API page.
type Page[T any] struct {
	Items         []T
	NextPageToken string
}
