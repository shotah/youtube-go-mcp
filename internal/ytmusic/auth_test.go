package ytmusic

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestParseAuthHeaders(t *testing.T) {
	raw := []byte(`{
		"cookie": "VISITOR_INFO1_LIVE=abc; __Secure-3PAPISID=sapisid-value; other=1",
		"x-goog-authuser": "0",
		"user-agent": "test-agent"
	}`)
	auth, err := ParseAuthHeaders(raw)
	if err != nil {
		t.Fatal(err)
	}
	if auth.SAPISID != "sapisid-value" {
		t.Fatalf("SAPISID=%q", auth.SAPISID)
	}
	if auth.AuthUser != "0" {
		t.Fatalf("AuthUser=%q", auth.AuthUser)
	}
}

func TestParseAuthHeadersMissingCookie(t *testing.T) {
	_, err := ParseAuthHeaders([]byte(`{"x-goog-authuser":"0"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthorizationHeader(t *testing.T) {
	auth := &BrowserAuth{SAPISID: "sapisid-value"}
	now := time.Unix(1700000000, 0)
	got := auth.AuthorizationHeader(now)

	payload := fmt.Sprintf("%d %s %s", now.Unix(), auth.SAPISID, origin)
	sum := sha1.Sum([]byte(payload))
	want := fmt.Sprintf("SAPISIDHASH %d_%s", now.Unix(), hex.EncodeToString(sum[:]))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBrowserAuthApply(t *testing.T) {
	auth := &BrowserAuth{
		Cookie:   "c=1; __Secure-3PAPISID=sap",
		AuthUser: "0",
		SAPISID:  "sap",
	}
	req, _ := http.NewRequest(http.MethodPost, "https://music.youtube.com/youtubei/v1/browse", http.NoBody)
	auth.Apply(req, time.Unix(1700000000, 0))
	if req.Header.Get("Cookie") == "" {
		t.Fatal("missing Cookie")
	}
	if req.Header.Get("Authorization") == "" {
		t.Fatal("missing Authorization")
	}
	if req.Header.Get("X-Goog-AuthUser") != "0" {
		t.Fatal("missing X-Goog-AuthUser")
	}
}

func TestGetLibraryPlaylistsRequiresAuth(t *testing.T) {
	c := NewClient()
	c.Auth = nil
	_, err := c.GetLibraryPlaylists(10)
	if !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("got %v want %v", err, ErrAuthRequired)
	}
}
