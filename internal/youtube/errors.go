package youtube

import "errors"

var (
	// ErrAuthRequired means no usable OAuth / Bearer token is available.
	ErrAuthRequired = errors.New("youtube: authentication required (OAuth Bearer)")
	// ErrNotFound means the API returned no items for an id lookup.
	ErrNotFound = errors.New("youtube: not found")
	// ErrAPI means a non-success Data API response.
	ErrAPI = errors.New("youtube: data api error")
)
