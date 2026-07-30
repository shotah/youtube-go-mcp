package ytmusic

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var (
	ytcfgRE       = regexp.MustCompile(`ytcfg\.set\s*\(\s*(\{.+?\})\s*\)\s*;`)
	visitorDataRE = regexp.MustCompile(`"VISITOR_DATA"\s*:\s*"([^"]+)"`)
)

// ensureVisitorID fetches and caches X-Goog-Visitor-Id (ytmusicapi parity).
// InnerTube library endpoints often return empty shelves without it.
func (c *Client) ensureVisitorID() string {
	if c == nil {
		return ""
	}
	c.ensureAuthMu()
	c.authMu.Lock()
	if c.visitorID != "" {
		id := c.visitorID
		c.authMu.Unlock()
		return id
	}
	c.authMu.Unlock()

	// Use a plain client — test transports / InnerTube rewrites must not handle the HTML bootstrap.
	id, err := fetchVisitorID(&http.Client{Timeout: 15 * time.Second})
	if err != nil || id == "" {
		return ""
	}

	c.authMu.Lock()
	c.visitorID = id
	c.authMu.Unlock()
	return id
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func fetchVisitorID(hc *http.Client) (string, error) {
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, origin+"/", http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", defaultRequestHeader["User-Agent"][0])
	req.Header.Set("Accept", "*/*")
	req.AddCookie(&http.Cookie{Name: "SOCS", Value: "CAI"})

	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	id := visitorIDFromHTML(string(body))
	if id == "" {
		return "", errors.New("ytmusic: VISITOR_DATA not found in music.youtube.com HTML")
	}
	return id, nil
}

func visitorIDFromHTML(html string) string {
	if m := ytcfgRE.FindStringSubmatch(html); len(m) >= 2 {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(m[1]), &cfg); err == nil {
			if v, ok := cfg["VISITOR_DATA"].(string); ok && v != "" {
				return v
			}
		}
	}
	if mm := visitorDataRE.FindStringSubmatch(html); len(mm) == 2 {
		return mm[1]
	}
	return ""
}

// TokenInfo is identity diagnostics for the current OAuth access token.
type TokenInfo struct {
	Email         string `json:"email,omitempty"`
	EmailVerified string `json:"email_verified,omitempty"`
	Sub           string `json:"sub,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ExpiresIn     string `json:"expires_in,omitempty"`
	Audience      string `json:"aud,omitempty"`
	// ChannelTitle / ChannelID come from YouTube Data API channels.list?mine=true
	// (youtube scope often omits email from tokeninfo).
	ChannelTitle string `json:"channelTitle,omitempty"`
	ChannelID    string `json:"channelId,omitempty"`
}

// WhoAmI returns Google tokeninfo (+ YouTube channel when available).
func (c *Client) WhoAmI() (*TokenInfo, error) {
	if c == nil || c.OAuth == nil || !c.OAuth.Ready() {
		return nil, fmt.Errorf("%w: WhoAmI requires OAuth", ErrAuthRequired)
	}
	if err := c.OAuth.EnsureAccessToken(false); err != nil {
		return nil, err
	}
	c.OAuth.mu.Lock()
	access := c.OAuth.Token.AccessToken
	c.OAuth.mu.Unlock()

	u := "https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=" + url.QueryEscape(access)
	resp, err := c.httpClient().Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tokeninfo HTTP %d: %s", resp.StatusCode, truncate(string(body)))
	}
	var info TokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	_ = c.fillYouTubeChannel(&info, access)
	return &info, nil
}

func (c *Client) fillYouTubeChannel(info *TokenInfo, access string) error {
	if info == nil || access == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true", http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("channels.list HTTP %d: %s", resp.StatusCode, truncate(string(body)))
	}
	var parsed struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return err
	}
	if len(parsed.Items) == 0 {
		return nil
	}
	info.ChannelID = parsed.Items[0].ID
	info.ChannelTitle = parsed.Items[0].Snippet.Title
	return nil
}

// LibraryProbe summarizes a live library fetch for debugging empty Liked Songs.
type LibraryProbe struct {
	AuthMode          string `json:"authMode"`
	OAuthEmail        string `json:"oauthEmail,omitempty"`
	VisitorIDPresent  bool   `json:"visitorIdPresent"`
	LibraryPlaylists  int    `json:"libraryPlaylists"`
	LikedTitle        string `json:"likedTitle,omitempty"`
	LikedTrackCount   int    `json:"likedTrackCount"`
	LikedTracksParsed int    `json:"likedTracksParsed"`
	LikedShelfFound   bool   `json:"likedShelfFound"`
	HistoryItems      int    `json:"historyItems"`
	Hint              string `json:"hint,omitempty"`
}

// ProbeLibrary hits library endpoints and reports parse/identity diagnostics.
func (c *Client) ProbeLibrary() (*LibraryProbe, error) {
	if c == nil {
		return nil, ErrAuthRequired
	}
	if !c.Authenticated() {
		return nil, ErrAuthRequired
	}
	out := &LibraryProbe{AuthMode: "browser"}
	if c.OAuth != nil && c.OAuth.Ready() {
		out.AuthMode = "oauth"
		if info, err := c.WhoAmI(); err == nil && info != nil {
			out.OAuthEmail = info.Email
			if out.OAuthEmail == "" && info.ChannelTitle != "" {
				out.OAuthEmail = "channel:" + info.ChannelTitle
			}
		}
	}
	out.VisitorIDPresent = c.ensureVisitorID() != ""

	playlists, err := c.GetLibraryPlaylists(25)
	if err != nil {
		return out, fmt.Errorf("library playlists: %w", err)
	}
	out.LibraryPlaylists = len(playlists)

	page, err := c.makeRequest("browse", map[string]any{"browseId": "VLLM"}, nil)
	if err != nil {
		return out, fmt.Errorf("liked songs browse: %w", err)
	}
	detail := parsePlaylistDetail(page, LikedSongsPlaylistID)
	out.LikedTitle = detail.Title
	out.LikedTrackCount = detail.TrackCount
	shelf := playlistShelf(page)
	out.LikedShelfFound = shelf != nil
	if shelf != nil {
		contents, _ := getValue(shelf, path{"contents"}).([]any)
		out.LikedTracksParsed = len(parsePlaylistTracks(contents))
	}

	hist, err := c.GetHistory(10)
	if err != nil {
		out.Hint = "history failed: " + err.Error()
	} else {
		out.HistoryItems = len(hist)
	}

	switch {
	case out.LikedTracksParsed == 0 && !out.LikedShelfFound:
		out.Hint = "InnerTube returned Liked Songs without a playlist shelf — often missing visitor id, wrong Google identity, or a Brand Account. Check oauthEmail; try browser headers from music.youtube.com Library."
	case out.LikedTracksParsed == 0 && out.LikedShelfFound:
		out.Hint = "Shelf present but zero tracks parsed — empty Liked Songs for this identity, or track renderer shape changed."
	case out.LibraryPlaylists == 0 && out.LikedTracksParsed == 0:
		out.Hint = "All library surfaces empty — almost certainly wrong Google account / Brand Account for this token."
	}
	return out, nil
}
