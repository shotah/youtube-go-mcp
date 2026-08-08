package ytmusic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	pendingFileName   = "oauth_pending.json"
	pendingTTL        = 15 * time.Minute
	PendingSessionTTL = pendingTTL
)

// PendingDeviceSession is persisted so a separate process (e.g. gantry agent)
// can resume polling after the user approves in a browser.
type PendingDeviceSession struct {
	DeviceCode string    `json:"device_code"`
	Interval   int       `json:"interval"`
	ExpiresIn  int       `json:"expires_in"`
	OutPath    string    `json:"out_path"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// PendingPath returns the path to the pending session file, stored
// in the same directory as the configured oauth output path.
func PendingPath(oauthOutPath string) string {
	dir := filepath.Dir(oauthOutPath)
	if dir == "" || dir == "." {
		return pendingFileName
	}
	return filepath.Join(dir, pendingFileName)
}

// SavePending persists a device session so a later process can poll.
func SavePending(session *PendingDeviceSession, oauthOutPath string) error {
	if session == nil {
		return errors.New("nil pending session")
	}
	pPath := PendingPath(oauthOutPath)
	dir := filepath.Dir(pPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create pending dir: %w", err)
		}
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pending: %w", err)
	}
	return os.WriteFile(pPath, data, 0o600)
}

// LoadPending reads a previously saved pending device session.
func LoadPending(oauthOutPath string) (*PendingDeviceSession, error) {
	pPath := PendingPath(oauthOutPath)
	data, err := os.ReadFile(pPath) //nolint:gosec // G703: operator-configured path
	if err != nil {
		return nil, fmt.Errorf("no pending auth session (run 'auth oauth start' first): %w", err)
	}
	var session PendingDeviceSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse pending session: %w", err)
	}
	if time.Now().After(session.ExpiresAt) {
		_ = os.Remove(pPath)
		return nil, errors.New("pending auth session expired — run 'auth oauth start' again")
	}
	return &session, nil
}

// RemovePending deletes the pending session file.
func RemovePending(oauthOutPath string) {
	_ = os.Remove(PendingPath(oauthOutPath))
}
