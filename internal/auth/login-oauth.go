package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"time"

	"crypto/rand"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/settings"
	argoOidc "github.com/argoproj/argo-cd/v2/util/oidc"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// For some reason it is REALLY IMPORTANT that redirect address is localhost:8085
// otherwise we'll get "Unregistered redirect_uri" error from dex
// @see https://github.com/argoproj/argo-cd/blob/c2e594c5/util/dex/config.go#L100
const localServerPort = 8085

const (
	// stateNonceLength is the length of the random state parameter for CSRF protection
	stateNonceLength = 24
	// pkceVerifierLength is the length of PKCE code verifier (RFC 7636 recommends 43-128)
	pkceVerifierLength = 43
	// pkceCharset contains allowed characters for PKCE code verifier per RFC 7636
	pkceCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	// alphanumericCharset contains alphanumeric characters for general secure random strings
	alphanumericCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	// oauthFlowTimeout is the maximum time allowed for completing the OAuth flow
	oauthFlowTimeout = 5 * time.Minute
	// serverShutdownTimeout is the time allowed for graceful server shutdown
	serverShutdownTimeout = 2 * time.Second
	// maxCallbackRequests prevents redirect loops in implicit flow
	maxCallbackRequests = 2
	// callbackPath is the OAuth callback endpoint path
	callbackPath = "/auth/callback"
)

// generatePKCE creates a PKCE code verifier and challenge per RFC 7636
func generatePKCE() (*pkceChallenge, error) {
	verifier, err := SecureRandomString(pkceVerifierLength, pkceCharset)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}

	challengeHash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])

	return &pkceChallenge{
		Verifier:  verifier,
		Challenge: challenge,
	}, nil
}

// oauth2Login opens a browser, runs a temporary HTTP server to delegate OAuth2 login flow and
// returns the JWT token and a refresh token (if supported)
func (a *Auth) oauth2Login(
	ctx context.Context,
	oidcSettings *settings.OIDCConfig,
	oauth2conf *oauth2.Config,
	provider *oidc.Provider,
) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthFlowTimeout)
	defer cancel()

	// Setup listener. Setting port = 0 will allow to get random available port
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localServerPort))
	if err != nil {
		return "", "", fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	oauth2conf.RedirectURL = fmt.Sprintf("http://localhost:%d%s", port, callbackPath)

	stateNonce, err := SecureRandomString(stateNonceLength, alphanumericCharset)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state nonce: %w", err)
	}

	pkce, err := generatePKCE()
	if err != nil {
		return "", "", err
	}

	grantType, err := inferGrantType(provider)
	if err != nil {
		return "", "", fmt.Errorf("failed to infer grant type: %w", err)
	}

	authURL, err := buildAuthURL(oauth2conf, oidcSettings, grantType, stateNonce, pkce)
	if err != nil {
		return "", "", err
	}

	callbackSrv := newCallbackServer(a.logger, stateNonce, pkce.Verifier, oauth2conf, ctx)
	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, callbackSrv.handleCallback)

	server := &http.Server{
		Handler: mux,
	}

	serverErrChan := make(chan error, 1)
	go func() {
		a.logger.Debugf("Starting callback server on port %d", port)
		if err := server.Serve(listener); err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("callback server failed: %w", err)
		}
		close(serverErrChan)
	}()

	fmt.Printf("Opening browser for authentication\n")
	fmt.Printf("Performing %s flow login: %s\n", grantType, authURL)

	if err := a.browserOpener.Open(authURL); err != nil {
		return "", "", fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for completion, timeout, or server error
	select {
	case <-callbackSrv.resultChan:
		// Callback completed
	case err := <-serverErrChan:
		if err != nil {
			return "", "", err
		}
	case <-ctx.Done():
		return "", "", fmt.Errorf("oauth2 login timed out: %w", ctx.Err())
	}

	// Shutdown server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		a.logger.Debugf("Server shutdown error: %v", err)
	}

	// Get result
	result := callbackSrv.getResult()
	if result.Error != nil {
		return "", "", fmt.Errorf("oauth2 callback failed: %w", result.Error)
	}

	fmt.Println("Authentication successful")
	a.logger.Debugf("Token: %s", result.Token)
	a.logger.Debugf("Refresh Token: %s", result.RefreshToken)

	return result.Token, result.RefreshToken, nil
}

// refreshToken attempts to refresh an expired token using the refresh token
// Returns: new token, new refresh token, expiration timestamp, error
func (a *Auth) refreshOauthToken(ctx context.Context, refreshToken string) (string, string, int64, error) {
	// Get ArgoCD settings for OAuth config
	argoSettings, err := a.client.GetSettings()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get ArgoCD settings: %w", err)
	}

	oauth2conf, _, err := a.client.OpenIDConfig(argoSettings)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get OpenID config: %w", err)
	}

	// Use the refresh token to get a new access token
	tokenSource := oauth2conf.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to refresh token: %w", err)
	}

	idToken, ok := newToken.Extra("id_token").(string)
	if !ok {
		return "", "", 0, fmt.Errorf("no id_token in refreshed token response")
	}

	newRefreshToken, _ := newToken.Extra("refresh_token").(string)
	if newRefreshToken == "" {
		newRefreshToken = refreshToken // Keep old refresh token if new one not provided
	}

	// Get expiration from the OAuth2 token
	var expires int64
	if !newToken.Expiry.IsZero() {
		expires = newToken.Expiry.Unix()
	}

	return idToken, newRefreshToken, expires, nil
}

// buildAuthURL constructs the authorization URL based on grant type
func buildAuthURL(
	oauth2conf *oauth2.Config,
	oidcSettings *settings.OIDCConfig,
	grantType GrantType,
	stateNonce string,
	pkce *pkceChallenge,
) (string, error) {
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	if claims := oidcSettings.GetIDTokenClaims(); claims != nil {
		opts = argoOidc.AppendClaimsAuthenticationRequestParameter(opts, claims)
	}

	switch grantType {
	case GRANT_TYPE_AUTH_CODE:
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", pkce.Challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
		return oauth2conf.AuthCodeURL(stateNonce, opts...), nil

	case GRANT_TYPE_IMPLICIT:
		return argoOidc.ImplicitFlowURL(oauth2conf, stateNonce, opts...)

	default:
		return "", fmt.Errorf("unsupported grant type: %v", grantType)
	}
}

// GrantType represents OAuth2 grant types
type GrantType string

// See https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata
const (
	// GRANT_TYPE_AUTH_CODE returns code that should be exchanged for access_token and refresh_token
	GRANT_TYPE_AUTH_CODE GrantType = "authorization_code"
	// GRANT_TYPE_IMPLICIT returns tokens directly
	GRANT_TYPE_IMPLICIT GrantType = "implicit"
)

// inferGrantType determines the appropriate OAuth2 grant type from provider metadata
func inferGrantType(provider *oidc.Provider) (GrantType, error) {
	var claims struct {
		ResponseTypesSupported []string `json:"response_types_supported"`
	}

	if err := provider.Claims(&claims); err != nil {
		return "", fmt.Errorf("failed to get provider claims: %w", err)
	}

	for _, supportedType := range claims.ResponseTypesSupported {
		if supportedType == "code" {
			return GRANT_TYPE_AUTH_CODE, nil
		}
	}

	return GRANT_TYPE_IMPLICIT, nil
}

// SecureRandomString generates a cryptographically-secure pseudo-random string
// of length n using characters from the provided charset.
func SecureRandomString(n int, charset string) (string, error) {
	if n <= 0 {
		return "", nil
	}
	if len(charset) == 0 {
		return "", fmt.Errorf("charset cannot be empty")
	}

	result := make([]byte, n)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}

	return string(result), nil
}
