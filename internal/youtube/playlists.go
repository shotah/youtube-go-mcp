package youtube

import (
	"net/url"
	"strconv"
	"strings"
)

// ListMyPlaylists returns playlists.list?mine=true.
func (c *Client) ListMyPlaylists(opts ListOptions) ([]Playlist, error) {
	want := opts.MaxResults
	if want <= 0 {
		want = 25
	}
	var out []Playlist
	pageToken := opts.PageToken
	for len(out) < want {
		pageSize := clampPageSize(want - len(out))
		q := url.Values{}
		q.Set("part", "snippet,contentDetails")
		q.Set("mine", "true")
		q.Set("maxResults", strconv.Itoa(pageSize))
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}

		var resp listResponse[struct {
			ID             string         `json:"id"`
			Snippet        snippet        `json:"snippet"`
			ContentDetails contentDetails `json:"contentDetails"`
		}]
		if err := c.get("/playlists", q, &resp); err != nil {
			return out, err
		}
		for i := range resp.Items {
			it := &resp.Items[i]
			out = append(out, Playlist{
				ID:          it.ID,
				Title:       it.Snippet.Title,
				Description: it.Snippet.Description,
				ItemCount:   it.ContentDetails.ItemCount,
				URL:         PlaylistURL(it.ID),
			})
			if len(out) >= want {
				break
			}
		}
		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

// ListPlaylistItems returns playlistItems.list for a playlist id (PL… / LL…).
func (c *Client) ListPlaylistItems(playlistID string, opts ListOptions) ([]Video, error) {
	playlistID = strings.TrimSpace(playlistID)
	if playlistID == "" {
		return nil, ErrNotFound
	}
	// Accept VL-prefixed Music-style ids by stripping VL when present.
	playlistID = strings.TrimPrefix(playlistID, "VL")

	want := opts.MaxResults
	if want <= 0 {
		want = 25
	}
	var out []Video
	pageToken := opts.PageToken
	for len(out) < want {
		pageSize := clampPageSize(want - len(out))
		q := url.Values{}
		q.Set("part", "snippet,contentDetails")
		q.Set("playlistId", playlistID)
		q.Set("maxResults", strconv.Itoa(pageSize))
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}

		var resp listResponse[struct {
			Snippet struct {
				Title        string `json:"title"`
				ChannelID    string `json:"videoOwnerChannelId"`
				ChannelTitle string `json:"videoOwnerChannelTitle"`
				PublishedAt  string `json:"publishedAt"`
				ResourceID   struct {
					VideoID string `json:"videoId"`
				} `json:"resourceId"`
			} `json:"snippet"`
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		}]
		if err := c.get("/playlistItems", q, &resp); err != nil {
			return out, err
		}
		for _, it := range resp.Items {
			id := it.ContentDetails.VideoID
			if id == "" {
				id = it.Snippet.ResourceID.VideoID
			}
			if id == "" {
				continue
			}
			snip := snippet{
				Title:        it.Snippet.Title,
				ChannelID:    it.Snippet.ChannelID,
				ChannelTitle: it.Snippet.ChannelTitle,
				PublishedAt:  it.Snippet.PublishedAt,
			}
			out = append(out, videoFromSnippet(id, snip, contentDetails{}))
			if len(out) >= want {
				break
			}
		}
		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}
