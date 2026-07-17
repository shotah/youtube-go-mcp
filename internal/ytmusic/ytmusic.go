package ytmusic

func Search(query string) *SearchClient {
	return Default.Search(query)
}

func TrackSearch(query string) *SearchClient {
	return Default.TrackSearch(query)
}

func AlbumSearch(query string) *SearchClient {
	return Default.AlbumSearch(query)
}

func ArtistSearch(query string) *SearchClient {
	return Default.ArtistSearch(query)
}

func PlaylistSearch(query string) *SearchClient {
	return Default.PlaylistSearch(query)
}

func VideoSearch(query string) *SearchClient {
	return Default.VideoSearch(query)
}

func GetWatchPlaylist(videoID string) ([]*TrackItem, error) {
	return Default.GetWatchPlaylist(videoID)
}

func GetSearchSuggestions(input string) ([]string, error) {
	return Default.GetSearchSuggestions(input)
}

func GetLyrics(videoID string) (string, error) {
	return Default.GetLyrics(videoID)
}

func (c *Client) Search(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: NoFilter}
}

func (c *Client) TrackSearch(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: TrackFilter}
}

func (c *Client) AlbumSearch(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: AlbumFilter}
}

func (c *Client) ArtistSearch(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: ArtistFilter}
}

func (c *Client) PlaylistSearch(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: PlaylistFilter}
}

func (c *Client) VideoSearch(query string) *SearchClient {
	return &SearchClient{client: c, Query: query, SearchFilter: VideoFilter}
}

func (c *Client) GetWatchPlaylist(videoID string) ([]*TrackItem, error) {
	return c.getWatchPlaylist(videoID)
}

func (c *Client) GetSearchSuggestions(input string) ([]string, error) {
	return c.getSearchSuggestions(input)
}

func (c *Client) GetLyrics(videoID string) (string, error) {
	return c.getLyrics(videoID)
}
