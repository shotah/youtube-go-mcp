package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiRoot = "https://www.googleapis.com/youtube/v3"

// Client talks to YouTube Data API v3 with OAuth Bearer auth.
type Client struct {
	HTTPClient *http.Client
	Tokens     TokenSource
	BaseURL    string // override for tests (httptest); default apiRoot
}

// New returns a Data API client. tokens must be non-nil for authenticated calls.
func New(tokens TokenSource) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Tokens:     tokens,
		BaseURL:    apiRoot,
	}
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) base() string {
	if c != nil && strings.TrimSpace(c.BaseURL) != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return apiRoot
}

func (c *Client) get(path string, query url.Values, dest any) error {
	if c == nil || c.Tokens == nil {
		return ErrAuthRequired
	}
	token, err := c.Tokens.BearerToken()
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return ErrAuthRequired
	}

	u := c.base() + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%w: %s HTTP %d: %s", ErrAPI, path, resp.StatusCode, truncate(string(body), 400))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("youtube: decode %s: %w", path, err)
	}
	return nil
}

const maxPageSize = 50

func clampPageSize(n int) int {
	if n <= 0 || n > maxPageSize {
		return maxPageSize
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func videoFromSnippet(id string, snip snippet, content contentDetails) Video {
	dur := parseISO8601Duration(content.Duration)
	v := Video{
		VideoID:      id,
		Title:        snip.Title,
		ChannelID:    snip.ChannelID,
		ChannelTitle: snip.ChannelTitle,
		CategoryID:   snip.CategoryID,
		Duration:     dur,
		DurationSec:  int(dur / time.Second),
		PublishedAt:  snip.PublishedAt,
		URL:          WatchURL(id),
	}
	if looksLikeMusicFields(snip.CategoryID, snip.ChannelTitle, snip.Title, snip.Description) {
		v.MusicURL = MusicWatchURL(id)
	}
	return v
}

// Wire shapes used across endpoints (subset of Data API JSON).
type snippet struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	ChannelID    string `json:"channelId"`
	ChannelTitle string `json:"channelTitle"`
	CategoryID   string `json:"categoryId"`
	PublishedAt  string `json:"publishedAt"`
}

type contentDetails struct {
	Duration  string `json:"duration"`
	ItemCount int64  `json:"itemCount"`
}

type relatedPlaylists struct {
	Likes   string `json:"likes"`
	Uploads string `json:"uploads"`
}

type listResponse[T any] struct {
	NextPageToken string `json:"nextPageToken"`
	Items         []T    `json:"items"`
}
