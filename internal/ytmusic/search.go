package ytmusic

import (
	"errors"
	"net/url"
)

type SearchClient struct {
	client *Client

	Query string

	Language string
	Region   string

	SearchFilter SearchFilter

	continuationKey string
	newPage         bool
}

func (search *SearchClient) makeRequest() (any, error) {
	body := map[string]any{}

	if search.continuationKey == "" {
		body["query"] = search.Query
		if search.SearchFilter != NoFilter {
			body["params"] = string(search.SearchFilter)
		}
	}

	params := url.Values{}
	if search.continuationKey != "" {
		params.Add("ctoken", search.continuationKey)
		params.Add("continuation", search.continuationKey)
		params.Add("type", "next")
	}

	c := search.client
	if c == nil {
		syncDefaultGlobals()
		c = Default
	}
	if search.Language != "" {
		c = c.withLanguage(search.Language)
	}
	if search.Region != "" {
		c = c.withRegion(search.Region)
	}

	return c.makeRequest("search", body, params)
}

func (c *Client) withLanguage(lang string) *Client {
	if c == nil {
		return &Client{Language: lang, limiter: &rateLimiter{}}
	}
	cp := *c
	cp.Language = lang
	if cp.limiter == nil {
		cp.limiter = &rateLimiter{}
	}
	return &cp
}

func (c *Client) withRegion(region string) *Client {
	if c == nil {
		return &Client{Region: region, limiter: &rateLimiter{}}
	}
	cp := *c
	cp.Region = region
	if cp.limiter == nil {
		cp.limiter = &rateLimiter{}
	}
	return &cp
}

func (search *SearchClient) NextExists() bool {
	if !search.newPage {
		return true
	}
	if search.continuationKey != "" {
		return true
	}
	return false
}

func (search *SearchClient) Next() (*SearchResult, error) {
	if !search.NextExists() {
		return nil, errors.New("end reached")
	}

	page, err := search.makeRequest()
	if err != nil {
		return nil, err
	}
	result, key := parseSearchPage(page)
	if result == nil {
		return nil, errors.New("couldn't parse page")
	}
	search.continuationKey = key
	search.newPage = true
	return result, nil
}
