package youtube

import (
	"net/url"
	"strconv"
)

// SearchVideos runs search.list (type=video). When MusicOnly, sets videoCategoryId=10.
// Collects up to opts.MaxResults items across pages (max 50 per request).
func (c *Client) SearchVideos(opts SearchOptions) ([]Video, error) {
	want := opts.MaxResults
	if want <= 0 {
		want = 25
	}
	var out []Video
	pageToken := opts.PageToken
	for len(out) < want {
		pageSize := clampPageSize(want - len(out))
		q := url.Values{}
		q.Set("part", "snippet")
		q.Set("type", "video")
		q.Set("q", opts.Query)
		q.Set("maxResults", strconv.Itoa(pageSize))
		if opts.MusicOnly {
			q.Set("videoCategoryId", CategoryMusic)
		}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}

		var resp listResponse[struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet snippet `json:"snippet"`
		}]
		if err := c.get("/search", q, &resp); err != nil {
			return out, err
		}
		for _, it := range resp.Items {
			id := it.ID.VideoID
			if id == "" {
				continue
			}
			v := videoFromSnippet(id, it.Snippet, contentDetails{})
			// search.list snippets omit categoryId often; still set musicUrl when Topic channel.
			out = append(out, v)
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
