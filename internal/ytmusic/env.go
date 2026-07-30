package ytmusic

import (
	"os"
	"strings"
)

// Preferred env names (YouTube Data API pivot). Legacy YTMUSIC_* aliases still work.
const (
	EnvOAuthPath     = "YOUTUBE_OAUTH_PATH"
	EnvOAuthClientID = "YOUTUBE_OAUTH_CLIENT_ID"
	//nolint:gosec // G101: env var name, not a secret value
	EnvOAuthClientSecret = "YOUTUBE_OAUTH_CLIENT_SECRET"

	envOAuthPathLegacy     = "YTMUSIC_OAUTH_PATH"
	envOAuthClientIDLegacy = "YTMUSIC_OAUTH_CLIENT_ID"
	//nolint:gosec // G101: env var name, not a secret value
	envOAuthClientSecretLegacy = "YTMUSIC_OAUTH_CLIENT_SECRET"
)

// EnvFirst returns the first non-empty trimmed environment value among names.
func EnvFirst(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func oauthPathFromEnv() string {
	return EnvFirst(EnvOAuthPath, envOAuthPathLegacy)
}

func oauthClientIDFromEnv() string {
	return EnvFirst(EnvOAuthClientID, envOAuthClientIDLegacy)
}

func oauthClientSecretFromEnv() string {
	return EnvFirst(EnvOAuthClientSecret, envOAuthClientSecretLegacy)
}
