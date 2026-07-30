package youtube

import (
	"net/url"
	"strconv"
	"strings"
)

// GetVideos fetches videos.list by id (comma-separated ok, up to 50).
func (c *Client) GetVideos(ids ...string) ([]Video, error) {
	clean := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		return nil, nil
	}
	if len(clean) > 50 {
		clean = clean[:50]
	}
	q := url.Values{}
	q.Set("part", "snippet,contentDetails")
	q.Set("id", strings.Join(clean, ","))

	var resp listResponse[struct {
		ID             string         `json:"id"`
		Snippet        snippet        `json:"snippet"`
		ContentDetails contentDetails `json:"contentDetails"`
	}]
	if err := c.get("/videos", q, &resp); err != nil {
		return nil, err
	}
	out := make([]Video, 0, len(resp.Items))
	for i := range resp.Items {
		it := &resp.Items[i]
		out = append(out, videoFromSnippet(it.ID, it.Snippet, it.ContentDetails))
	}
	return out, nil
}

// GetVideo returns a single video or ErrNotFound.
func (c *Client) GetVideo(id string) (*Video, error) {
	items, err := c.GetVideos(id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	v := items[0]
	return &v, nil
}

// ListLikedVideos returns videos.list?myRating=like (YouTube thumbs-up, not Music Liked Songs).
// When opts.MusicOnly is set, pages until enough music-leaning rows or the likes list ends.
func (c *Client) ListLikedVideos(opts ListOptions) ([]Video, error) {
	want := opts.MaxResults
	if want <= 0 {
		want = 25
	}
	var out []Video
	pageToken := opts.PageToken
	for len(out) < want {
		pageSize := clampPageSize(want - len(out))
		if opts.MusicOnly {
			pageSize = maxPageSize // over-fetch; many likes are non-music
		}
		q := url.Values{}
		q.Set("part", "snippet,contentDetails")
		q.Set("myRating", "like")
		q.Set("maxResults", strconv.Itoa(pageSize))
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}

		var resp listResponse[struct {
			ID             string         `json:"id"`
			Snippet        snippet        `json:"snippet"`
			ContentDetails contentDetails `json:"contentDetails"`
		}]
		if err := c.get("/videos", q, &resp); err != nil {
			return out, err
		}
		for i := range resp.Items {
			it := &resp.Items[i]
			v := videoFromSnippet(it.ID, it.Snippet, it.ContentDetails)
			if opts.MusicOnly && !LooksLikeMusic(v) {
				continue
			}
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
