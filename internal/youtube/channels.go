package youtube

import "net/url"

// ChannelMine returns the authenticated user's channel (channels.list?mine=true).
func (c *Client) ChannelMine() (*Channel, error) {
	q := url.Values{}
	q.Set("part", "snippet,contentDetails")
	q.Set("mine", "true")
	q.Set("maxResults", "1")

	var resp listResponse[struct {
		ID             string `json:"id"`
		Snippet        snippet
		ContentDetails struct {
			RelatedPlaylists relatedPlaylists `json:"relatedPlaylists"`
		} `json:"contentDetails"`
	}]
	if err := c.get("/channels", q, &resp); err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, ErrNotFound
	}
	it := resp.Items[0]
	return &Channel{
		ID:            it.ID,
		Title:         it.Snippet.Title,
		LikesPlaylist: it.ContentDetails.RelatedPlaylists.Likes,
		Uploads:       it.ContentDetails.RelatedPlaylists.Uploads,
	}, nil
}
