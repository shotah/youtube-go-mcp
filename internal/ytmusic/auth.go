package ytmusic

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const origin = "https://music.youtube.com"

// BrowserAuth holds session credentials exported from music.youtube.com.
type BrowserAuth struct {
	Cookie       string
	AuthUser     string
	SAPISID      string
	ExtraHeaders http.Header
}

// HeadersFile is the JSON shape written by `youtube-go-mcp auth` / ytmusicapi browser setup.
type HeadersFile map[string]string

// LoadAuthFromFile loads browser headers from a JSON file (cookie + x-goog-authuser required).
func LoadAuthFromFile(path string) (*BrowserAuth, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G703: path comes from operator-configured env/CLI
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrInvalidAuth, path, err)
	}
	return ParseAuthHeaders(data)
}

// ParseAuthHeaders parses a headers JSON object into BrowserAuth.
func ParseAuthHeaders(data []byte) (*BrowserAuth, error) {
	var raw HeadersFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: parse headers json: %w", ErrInvalidAuth, err)
	}

	normalized := make(map[string]string, len(raw))
	for k, v := range raw {
		normalized[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}

	cookie := firstNonEmpty(normalized["cookie"], normalized["cookie:"])
	authUser := firstNonEmpty(normalized["x-goog-authuser"], normalized["x-goog-authuser:"])
	if cookie == "" || authUser == "" {
		return nil, fmt.Errorf("%w: headers must include cookie and x-goog-authuser", ErrInvalidAuth)
	}

	sapisid := sapisidFromCookie(cookie)
	if sapisid == "" {
		return nil, fmt.Errorf("%w: cookie missing __Secure-3PAPISID / SAPISID", ErrInvalidAuth)
	}

	extra := make(http.Header)
	for k, v := range normalized {
		switch k {
		case "cookie", "authorization", "content-type", "content-length", "host", "accept-encoding":
			continue
		case "x-goog-authuser", "user-agent", "origin", "x-origin", "referer", "accept":
			extra.Set(k, v)
		default:
			if strings.HasPrefix(k, "sec-") {
				continue
			}
			extra.Set(k, v)
		}
	}

	return &BrowserAuth{
		Cookie:       cookie,
		AuthUser:     authUser,
		SAPISID:      sapisid,
		ExtraHeaders: extra,
	}, nil
}

// AuthorizationHeader returns a fresh SAPISIDHASH Authorization value.
func (a *BrowserAuth) AuthorizationHeader(now time.Time) string {
	if a == nil {
		return ""
	}
	ts := strconv.FormatInt(now.Unix(), 10)
	payload := ts + " " + a.SAPISID + " " + origin
	sum := sha1.Sum([]byte(payload)) //nolint:gosec // G401: SAPISIDHASH protocol requires SHA-1
	return "SAPISIDHASH " + ts + "_" + hex.EncodeToString(sum[:])
}

// Apply sets auth-related headers on an outbound request.
func (a *BrowserAuth) Apply(req *http.Request, now time.Time) {
	if a == nil || req == nil {
		return
	}
	req.Header.Set("Cookie", a.Cookie)
	req.Header.Set("Authorization", a.AuthorizationHeader(now))
	req.Header.Set("X-Goog-AuthUser", a.AuthUser)
	req.Header.Set("X-Origin", origin)
	req.Header.Set("Origin", origin)
	for k, vals := range a.ExtraHeaders {
		if req.Header.Get(k) != "" {
			continue
		}
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}
}

func sapisidFromCookie(raw string) string {
	// Prefer __Secure-3PAPISID (what music.youtube.com / ytmusicapi use for SAPISIDHASH).
	// Falling back to the first matching name can pick the wrong secret when both exist.
	var sapisid string
	for part := range strings.SplitSeq(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(name) {
		case "__Secure-3PAPISID":
			return strings.TrimSpace(value)
		case "SAPISID":
			if sapisid == "" {
				sapisid = strings.TrimSpace(value)
			}
		}
	}
	return sapisid
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
