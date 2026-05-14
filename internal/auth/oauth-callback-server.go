package auth

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"sync"

	"github.com/Jack200062/ArguTUI/pkg/logging"
	"golang.org/x/oauth2"
)

// implicitFlowRedirectScript redirects URL fragments to query parameters
const implicitFlowRedirectScript = `<script>window.location.search = window.location.hash.substring(1)</script>`

// successPageHTML is the HTML page shown after successful authentication
const successPageHTML = `<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body style="background-color: #efefef;">
	<div style="background-color: #fff;box-shadow: 0 5px 15px rgba(0, 0, 0, 0.5);min-height: 12em;display:flex;flex-direction: column;justify-content: center;align-items:center;box-sizing: border-box;max-width: 30em;margin: 2em auto;color: #333;font-family: 'Source Sans Pro', Helvetica, sans-serif;">
		<div style="font-size:1.5em">Authentication successful!</div>
        <p style="padding: 1em; font-size:1em; text-align:center">
            Authentication was successful, you can now return to CLI. This page will close automatically.
        </p>
	</div>
	<script>window.onload=function(){setTimeout(function(){window.close()}, 4000)}</script>
</body>
</html>`

// callbackServer handles the OAuth2 callback
type callbackServer struct {
	logger       *logging.Logger
	stateNonce   string
	codeVerifier string
	oauth2conf   *oauth2.Config
	ctx          context.Context

	mu           sync.Mutex
	requestCount int
	result       oauth2Result
	resultChan   chan struct{}
}

func newCallbackServer(
	logger *logging.Logger,
	stateNonce string,
	codeVerifier string,
	oauth2conf *oauth2.Config,
	ctx context.Context,
) *callbackServer {
	return &callbackServer{
		logger:       logger,
		stateNonce:   stateNonce,
		codeVerifier: codeVerifier,
		oauth2conf:   oauth2conf,
		ctx:          ctx,
		resultChan:   make(chan struct{}),
	}
}

func (cs *callbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	cs.logger.Debugf("OAuth callback: %s", r.URL)

	// Check for OAuth error response
	if formErr := r.FormValue("error"); formErr != "" {
		cs.completeWithError(w, fmt.Errorf("%s: %s", formErr, r.FormValue("error_description")))
		return
	}

	// Increment and check request count to prevent redirect loops
	cs.mu.Lock()
	cs.requestCount++
	count := cs.requestCount
	cs.mu.Unlock()

	if count > maxCallbackRequests {
		cs.completeWithError(w, fmt.Errorf("unable to complete login flow: too many redirects"))
		return
	}

	// Handle implicit flow redirect
	if len(r.Form) == 0 {
		fmt.Fprint(w, implicitFlowRedirectScript)
		return
	}

	// Validate state to prevent CSRF
	if state := r.FormValue("state"); state != cs.stateNonce {
		cs.completeWithError(w, fmt.Errorf("invalid state parameter"))
		return
	}

	// Try to get token from implicit flow first
	if tokenString := r.FormValue("id_token"); tokenString != "" {
		cs.completeWithSuccess(w, tokenString, "")
		return
	}

	// Handle authorization code flow
	code := r.FormValue("code")
	if code == "" {
		cs.completeWithError(w, fmt.Errorf("no code in request: %q", r.Form))
		return
	}

	token, refreshToken, err := cs.exchangeCode(code)
	if err != nil {
		cs.completeWithError(w, err)
		return
	}

	cs.completeWithSuccess(w, token, refreshToken)
}

func (cs *callbackServer) exchangeCode(code string) (string, string, error) {
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_verifier", cs.codeVerifier),
	}

	tok, err := cs.oauth2conf.Exchange(cs.ctx, code, opts...)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange code: %w", err)
	}

	tokenString, ok := tok.Extra("id_token").(string)
	if !ok {
		return "", "", fmt.Errorf("no id_token in token response")
	}

	refreshToken, _ := tok.Extra("refresh_token").(string)
	return tokenString, refreshToken, nil
}

func (cs *callbackServer) completeWithSuccess(w http.ResponseWriter, token, refreshToken string) {
	cs.mu.Lock()
	cs.result = oauth2Result{Token: token, RefreshToken: refreshToken}
	cs.mu.Unlock()

	fmt.Fprint(w, successPageHTML)
	close(cs.resultChan)
}

func (cs *callbackServer) completeWithError(w http.ResponseWriter, err error) {
	cs.mu.Lock()
	cs.result = oauth2Result{Error: err}
	cs.mu.Unlock()

	http.Error(w, html.EscapeString(err.Error()), http.StatusBadRequest)
	close(cs.resultChan)
}

func (cs *callbackServer) getResult() oauth2Result {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.result
}
