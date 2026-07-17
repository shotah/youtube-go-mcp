package ytmusic

import "net/url"

func (c *Client) getSearchSuggestions(input string) ([]string, error) {
	page, err := c.makeRequest(
		"music/get_search_suggestions",
		map[string]any{
			"input": input,
		},
		url.Values{},
	)
	if err != nil {
		return nil, err
	}
	return parseSearchSuggestions(page), nil
}
