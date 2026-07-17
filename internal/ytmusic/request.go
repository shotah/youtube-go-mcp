package ytmusic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) makeRequest(endpoint string, body map[string]any, params url.Values) (any, error) {
	syncDefaultGlobals()
	if c == nil {
		c = Default
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Language == "" {
		c.Language = defaultLanguage
	}
	if c.Region == "" {
		c.Region = defaultRegion
	}
	if c.ClientName == "" {
		c.ClientName = defaultClientName
	}
	if c.ClientVersion == "" {
		c.ClientVersion = defaultClientVersion
	}

	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    c.ClientName,
				"clientVersion": c.ClientVersion,
				"hl":            c.Language,
				"gl":            c.Region,
			},
			"user": map[string]any{
				"lockedSafetyMode": false,
			},
		},
	}
	maps.Copy(payload, body)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	if params == nil {
		params = url.Values{}
	} else {
		// Copy so retries don't accumulate duplicate query keys.
		params = cloneValues(params)
	}
	if params.Get("prettyPrint") == "" {
		params.Set("prettyPrint", "false")
	}
	params.Set("key", searchKey)

	reqURL := fmt.Sprintf("https://music.youtube.com/youtubei/v1/%s?%s", endpoint, params.Encode())

	var lastStatus int
	var lastSnippet string
	maxAttempts := c.maxRetries() + 1
	for attempt := range maxAttempts {
		c.throttle()

		request, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(jsonData))
		if err != nil {
			return nil, err
		}
		for k, vals := range defaultRequestHeader {
			for _, v := range vals {
				request.Header.Set(k, v)
			}
		}
		c.maybeReloadAuth()
		if c.Auth != nil {
			c.Auth.Apply(request, c.Now())
		}

		response, err := c.HTTPClient.Do(request)
		if err != nil {
			return nil, err
		}
		respBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			var result any
			if err := json.Unmarshal(respBody, &result); err != nil {
				return nil, err
			}
			return result, nil
		}

		lastStatus = response.StatusCode
		lastSnippet = string(respBody)
		if len(lastSnippet) > 300 {
			lastSnippet = lastSnippet[:300] + "…"
		}

		if isAuthFailureStatus(response.StatusCode) {
			// If the operator replaced headers.json after our last load, retry once with fresh cookies.
			if c.reloadAuthIfChanged() && attempt+1 < maxAttempts {
				continue
			}
			if c.Authenticated() {
				return nil, fmt.Errorf("%w: %s HTTP %d: %s — %s", ErrSessionExpired, endpoint, response.StatusCode, lastSnippet, AuthRefreshHint)
			}
		}

		if !shouldRetryStatus(response.StatusCode) || attempt+1 >= maxAttempts {
			break
		}

		delay := retryDelay(response, attempt, c.Now())
		c.sleepFunc()(delay)
	}

	if shouldRetryStatus(lastStatus) {
		return nil, fmt.Errorf("%w: %s HTTP %d: %s", ErrRateLimited, endpoint, lastStatus, lastSnippet)
	}
	return nil, fmt.Errorf("ytmusic: %s returned HTTP %d: %s", endpoint, lastStatus, lastSnippet)
}

func cloneValues(in url.Values) url.Values {
	out := make(url.Values, len(in))
	for k, vals := range in {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}
