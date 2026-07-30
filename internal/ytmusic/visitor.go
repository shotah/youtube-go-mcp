package ytmusic

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TokenInfo is identity diagnostics for the current OAuth access token.
type TokenInfo struct {
	Email         string `json:"email,omitempty"`
	EmailVerified string `json:"email_verified,omitempty"`
	Sub           string `json:"sub,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ExpiresIn     string `json:"expires_in,omitempty"`
	Audience      string `json:"aud,omitempty"`
	ChannelTitle  string `json:"channelTitle,omitempty"`
	ChannelID     string `json:"channelId,omitempty"`
}

// WhoAmI returns Google tokeninfo (+ YouTube channel when available).
func (c *Client) WhoAmI() (*TokenInfo, error) {
	if c == nil || c.OAuth == nil || !c.OAuth.Ready() {
		return nil, fmt.Errorf("%w: WhoAmI requires OAuth", ErrAuthRequired)
	}
	if err := c.OAuth.EnsureAccessToken(false); err != nil {
		return nil, err
	}
	access, err := c.OAuth.BearerToken()
	if err != nil {
		return nil, err
	}

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
