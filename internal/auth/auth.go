package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
	keyring "github.com/zalando/go-keyring"

	"github.com/Jack200062/ArguTUI/config"
	"github.com/Jack200062/ArguTUI/internal/transport/argocd"
	"github.com/Jack200062/ArguTUI/internal/ui"
	"github.com/Jack200062/ArguTUI/pkg/logging"
)

const APP_NAME = "Jack200062.ArgoTUI"

// OAuth2 flow constants
const ()

type AuthTokens struct {
	Type         config.LoginType `json:"type"`
	RefreshToken string           `json:"refresh_token,omitempty"`
}

type BrowserOpener interface {
	Open(url string) error
}

// DefaultBrowserOpener uses the system default browser
type DefaultBrowserOpener struct{}

func (d *DefaultBrowserOpener) Open(url string) error {
	return open.Start(url)
}

type pkceChallenge struct {
	Verifier  string
	Challenge string
}

type oauth2Result struct {
	Token        string
	RefreshToken string
	Error        error
}

type Auth struct {
	name          string
	instanceCfg   *config.Instance
	client        *argocd.ArgoCdClient
	logger        *logging.Logger
	app           *tview.Application
	router        *ui.Router
	keyring       any
	browserOpener BrowserOpener
}

func NewAuth(name string, instanceCfg *config.Instance, client *argocd.ArgoCdClient, logger *logging.Logger, ctx context.Context) *Auth {
	return &Auth{
		name:          name,
		instanceCfg:   instanceCfg,
		client:        client,
		logger:        logger,
		browserOpener: &DefaultBrowserOpener{},
	}
}

// WithBrowserOpener sets a custom browser opener (useful for testing)
func (a *Auth) WithBrowserOpener(opener BrowserOpener) *Auth {
	a.browserOpener = opener
	return a
}

// WithRouter sets the UI router
func (a *Auth) WithRouter(router *ui.Router) *Auth {
	a.router = router
	return a
}

// WithApp sets the tview application
func (a *Auth) WithApp(app *tview.Application) *Auth {
	a.app = app
	return a
}

func (a *Auth) GetToken() (string, error) {
	ctx := context.Background()

	loginType := a.instanceCfg.LoginType
	if loginType == config.LOGIN_TYPE_TOKEN {
		return a.instanceCfg.Token, nil
	}

	// Try to get refresh token from keychain and use it to get a fresh access token
	authTokens, err := a.getTokenFromKeychain()
	if err == nil && authTokens.RefreshToken != "" {
		a.logger.Debugf("Found refresh token in keychain, getting fresh access token")
		newToken, newRefreshToken, _, err := a.refreshToken(ctx, authTokens)
		if err == nil {
			// Save updated refresh token if it changed
			if newRefreshToken != "" && newRefreshToken != authTokens.RefreshToken {
				if err := a.saveTokenToKeychain(authTokens.Type, newRefreshToken); err != nil {
					a.logger.Debugf("Failed to save updated refresh token: %v", err)
				}
			}
			return newToken, nil
		}
		a.logger.Debugf("Failed to refresh token: %v, will perform fresh login", err)
	} else {
		a.logger.Debugf("No refresh token found in keychain: %v", err)
	}

	// No valid refresh token, perform full login
	a.logger.Debugf("Performing fresh login")
	var tokenString, refreshToken string

	switch loginType {
	case config.LOGIN_TYPE_SSO:
		tokenString, refreshToken, _, err = a.performSSOLogin(ctx)
	case config.LOGIN_TYPE_CREDENTIALS:
		tokenString, refreshToken, _, err = a.passwordLogin(ctx)
	default:
		err = fmt.Errorf("LoginType=%s is not supported", loginType)
	}

	if err != nil {
		return "", fmt.Errorf("login failed: %w", err)
	}

	// Save refresh token to keychain
	if refreshToken != "" {
		if err := a.saveTokenToKeychain(loginType, refreshToken); err != nil {
			a.logger.Debugf("Failed to save refresh token to keychain: %v", err)
		}
	}

	return tokenString, nil
}

func (a *Auth) refreshToken(ctx context.Context, token AuthTokens) (string, string, int64, error) {
	switch token.Type {
	case config.LOGIN_TYPE_SSO:
		return a.refreshOauthToken(ctx, token.RefreshToken)
	case config.LOGIN_TYPE_CREDENTIALS:
		return a.refreshPasswordToken(ctx, token.RefreshToken)
	default:
		return "", "", 0, fmt.Errorf("Cannot refresh token with LoginType=%s", token.Type)
	}
}

// getTokenFromKeychain retrieves the refresh token from the system keychain
func (a *Auth) getTokenFromKeychain() (AuthTokens, error) {
	var authTokens AuthTokens

	rawToken, err := keyring.Get(APP_NAME, a.name)
	if err != nil {
		return AuthTokens{}, err
	}

	if err := json.Unmarshal([]byte(rawToken), &authTokens); err != nil {
		return AuthTokens{}, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return authTokens, nil
}

// saveTokenToKeychain saves the refresh token to the system keychain
func (a *Auth) saveTokenToKeychain(loginType config.LoginType, refreshToken string) error {
	authTokens := AuthTokens{
		Type:         loginType,
		RefreshToken: refreshToken,
	}

	data, err := json.Marshal(authTokens)
	if err != nil {
		return fmt.Errorf("failed to marshal tokens: %w", err)
	}

	return keyring.Set(APP_NAME, a.name, string(data))
}

// performSSOLogin performs SSO login and returns token, refresh token, and expiration
func (a *Auth) performSSOLogin(ctx context.Context) (string, string, int64, error) {
	httpClient, err := a.client.HttpClient()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	ctx = oidc.ClientContext(ctx, httpClient)

	argoSettings, err := a.client.GetSettings()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get ArgoCD settings: %w", err)
	}

	oauth2conf, provider, err := a.client.OpenIDConfig(argoSettings)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get OpenID config: %w", err)
	}

	token, refreshToken, err := a.oauth2Login(ctx, argoSettings.GetOIDCConfig(), oauth2conf, provider)
	if err != nil {
		return "", "", 0, err
	}

	var expires int64
	if expiry := a.tryExtractExpiration(token); expiry > 0 {
		expires = expiry
	}

	return token, refreshToken, expires, nil
}

// tryExtractExpiration attempts to extract expiration from a JWT token
// Returns 0 if token is not a JWT or doesn't have expiration claim
// This is a best-effort attempt and failures are not errors
func (a *Auth) tryExtractExpiration(token string) int64 {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsedToken, _, err := parser.ParseUnverified(token, jwt.MapClaims{})
	if err != nil {
		return 0
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return 0
	}

	if exp, ok := claims["exp"].(float64); ok {
		return int64(exp)
	}

	return 0
}
