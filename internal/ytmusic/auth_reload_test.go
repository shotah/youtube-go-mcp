package ytmusic

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSapisidPrefersSecure3PAPISID(t *testing.T) {
	cookie := "SAPISID=wrong; __Secure-3PAPISID=right; other=1"
	if got := sapisidFromCookie(cookie); got != "right" {
		t.Fatalf("got %q want right", got)
	}
}

func TestAuthReloadsWhenHeadersFileChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headers.json")
	writeHeaders := func(sapisid string) {
		t.Helper()
		raw := []byte(`{"cookie":"VISITOR=1; __Secure-3PAPISID=` + sapisid + `","x-goog-authuser":"0"}`)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		// Ensure mtime advances on filesystems with coarse resolution.
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		_ = os.Chtimes(path, info.ModTime().Add(time.Second), info.ModTime().Add(time.Second))
	}

	writeHeaders("first")
	c := NewClient()
	if err := c.SetAuthPath(path); err != nil {
		t.Fatal(err)
	}
	if c.Auth.SAPISID != "first" {
		t.Fatalf("initial SAPISID=%q", c.Auth.SAPISID)
	}

	writeHeaders("second")
	c.maybeReloadAuth()
	if c.Auth.SAPISID != "second" {
		t.Fatalf("reloaded SAPISID=%q want second", c.Auth.SAPISID)
	}
}

func TestReloadAuthIfChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headers.json")
	raw := []byte(`{"cookie":"VISITOR=1; __Secure-3PAPISID=one","x-goog-authuser":"0"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	c := NewClient()
	if err := c.SetAuthPath(path); err != nil {
		t.Fatal(err)
	}
	if c.reloadAuthIfChanged() {
		t.Fatal("expected no change when mtime unchanged")
	}

	raw = []byte(`{"cookie":"VISITOR=1; __Secure-3PAPISID=two","x-goog-authuser":"0"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	_ = os.Chtimes(path, info.ModTime().Add(2*time.Second), info.ModTime().Add(2*time.Second))

	if !c.reloadAuthIfChanged() {
		t.Fatal("expected reload when mtime advanced")
	}
	if c.Auth.SAPISID != "two" {
		t.Fatalf("SAPISID=%q", c.Auth.SAPISID)
	}
}
