package youtube

// TokenSource supplies a Bearer access token (refresh as needed).
// ytmusic.OAuthSession implements this via BearerToken().
type TokenSource interface {
	BearerToken() (string, error)
}

// StaticToken is a fixed access token for tests.
type StaticToken string

// BearerToken returns the static token.
func (t StaticToken) BearerToken() (string, error) {
	if t == "" {
		return "", ErrAuthRequired
	}
	return string(t), nil
}
