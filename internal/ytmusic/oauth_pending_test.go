package ytmusic

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPendingPathRelative(t *testing.T) {
	got := PendingPath("oauth.json")
	if got != pendingFileName {
		t.Fatalf("expected %q, got %q", pendingFileName, got)
	}
}

func TestPendingPathSubdir(t *testing.T) {
	got := PendingPath(filepath.Join("data", "oauth.json"))
	want := filepath.Join("data", pendingFileName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSaveAndLoadPending(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "oauth.json")

	session := &PendingDeviceSession{
		DeviceCode: "dev-code",
		Interval:   5,
		ExpiresIn:  900,
		OutPath:    outPath,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}
	if err := SavePending(session, outPath); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPending(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceCode != "dev-code" {
		t.Fatalf("device_code=%q", loaded.DeviceCode)
	}
	if loaded.Interval != 5 {
		t.Fatalf("interval=%d", loaded.Interval)
	}
	if loaded.OutPath != outPath {
		t.Fatalf("out_path=%q", loaded.OutPath)
	}
}

func TestLoadPendingExpired(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "oauth.json")

	session := &PendingDeviceSession{
		DeviceCode: "dev-code",
		Interval:   5,
		ExpiresIn:  900,
		OutPath:    outPath,
		ExpiresAt:  time.Now().Add(-time.Minute),
	}
	if err := SavePending(session, outPath); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPending(outPath)
	if err == nil {
		t.Fatal("expected expired error")
	}
	if _, statErr := os.Stat(PendingPath(outPath)); statErr == nil {
		t.Fatal("pending file should have been removed")
	}
}

func TestLoadPendingNotFound(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "oauth.json")

	_, err := LoadPending(outPath)
	if err == nil {
		t.Fatal("expected error for missing pending file")
	}
}

func TestSavePendingNil(t *testing.T) {
	if err := SavePending(nil, "out.json"); err == nil {
		t.Fatal("expected error for nil session")
	}
}

func TestRemovePending(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "oauth.json")

	session := &PendingDeviceSession{
		DeviceCode: "dev",
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	}
	_ = SavePending(session, outPath)
	RemovePending(outPath)

	if _, err := os.Stat(PendingPath(outPath)); err == nil {
		t.Fatal("file should have been removed")
	}
}
