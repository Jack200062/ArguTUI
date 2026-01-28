package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	loginscreen "github.com/Jack200062/ArguTUI/internal/ui/screens/login"
)

func (a *Auth) passwordLogin(ctx context.Context) (string, string, int64, error) {
	if a.app == nil || a.router == nil {
		return "", "", 0, fmt.Errorf("app and router must be set for credential login")
	}

	a.logger.Debugf("Password login!")

	var credentials loginscreen.Credentials
	var loginError error
	done := make(chan struct{})
	cancelled := false

	// Create login screen with callbacks
	screen := loginscreen.NewLoginScreen(
		a.app,
		func(creds loginscreen.Credentials) {
			credentials = creds
			done <- struct{}{}
		},
		func() {
			cancelled = true
			close(done)
		},
	)

	// Add screen to router
	if err := a.router.AddScreen(screen); err != nil {
		// Screen might already exist, try to switch to it
		a.logger.Debugf("Screen already exists, switching to it: %v", err)
	}

	// Switch to login screen
	if err := a.router.SwitchTo(loginscreen.LoginScreenName); err != nil {
		return "", "", 0, fmt.Errorf("failed to show login screen: %w", err)
	}

	// Login loop - allow retries on failure
	for {
		select {
		case <-done:
			if cancelled {
				return "", "", 0, fmt.Errorf("login cancelled by user")
			}
		case <-ctx.Done():
			return "", "", 0, fmt.Errorf("login timed out: %w", ctx.Err())
		}

		creds := credentials
		a.logger.Debugf("Creating Session with credentials %+v", creds)
		// Attempt to create session with the provided credentials
		createdSession, err := a.client.CreateSession(creds.Username, creds.Password)
		if err != nil {
			a.logger.Debugf("Login failed: %v", err)
			loginError = err

			// Show error on login screen and let user retry
			a.app.QueueUpdateDraw(func() {
				screen.ShowError("Login failed: Invalid username or password")
			})
			continue
		}

		// Success - clear any error message
		a.app.QueueUpdateDraw(func() {
			screen.ClearError()
		})

		// Try to extract expiration from the token (if it's a JWT)
		expires := a.tryExtractExpiration(createdSession.Token)

		// Encode credentials as "refresh token" for re-authentication
		encodedCredentials := encodeCredentials(creds.Username, creds.Password)

		return createdSession.Token, encodedCredentials, expires, nil
	}

	// This should never be reached, but satisfies the compiler
	return "", "", 0, loginError
}

// refreshPasswordToken re-authenticates using stored credentials
func (a *Auth) refreshPasswordToken(ctx context.Context, encodedCredentials string) (string, string, int64, error) {
	username, password, err := decodeCredentials(encodedCredentials)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to decode credentials: %w", err)
	}

	createdSession, err := a.client.CreateSession(username, password)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create session: %w", err)
	}

	expires := a.tryExtractExpiration(createdSession.Token)

	// Return same encoded credentials (they don't change)
	return createdSession.Token, encodedCredentials, expires, nil
}

// encodeCredentials encodes username and password for storage
func encodeCredentials(username, password string) string {
	combined := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(combined))
}

// decodeCredentials decodes stored credentials
func decodeCredentials(encoded string) (string, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credentials format")
	}

	return parts[0], parts[1], nil
}
