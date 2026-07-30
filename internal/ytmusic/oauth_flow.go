package ytmusic

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RunDeviceAuthFlow completes the TV/device OAuth flow.
// It obtains a device code, optionally calls waitForUser (e.g. to print the URL),
// then polls Google until the user approves or ctx is done.
func RunDeviceAuthFlow(ctx context.Context, creds OAuthCredentials, waitForUser func(code *DeviceCode) error) (*OAuthToken, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	code, err := creds.GetDeviceCode()
	if err != nil {
		return nil, err
	}
	if waitForUser != nil {
		if err := waitForUser(code); err != nil {
			return nil, err
		}
	}
	return PollDeviceToken(ctx, creds, code)
}

// PollDeviceToken polls token exchange for an existing device code.
func PollDeviceToken(ctx context.Context, creds OAuthCredentials, code *DeviceCode) (*OAuthToken, error) {
	if code == nil {
		return nil, fmt.Errorf("%w: nil device code", ErrInvalidAuth)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	interval := time.Duration(code.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	if code.ExpiresIn <= 0 {
		deadline = time.Now().Add(15 * time.Minute)
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: device code expired before authorization completed", ErrSessionExpired)
		}

		tok, err := creds.TokenFromDeviceCode(code.DeviceCode)
		if err == nil {
			return tok, nil
		}
		switch {
		case errors.Is(err, ErrOAuthPending):
			// keep polling
		case errors.Is(err, ErrOAuthSlowDown):
			interval += 5 * time.Second
		default:
			return nil, err
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
