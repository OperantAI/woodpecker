// Oauth client implementation taken from https://github.com/modelcontextprotocol/go-sdk/blob/main/auth/client.go

package mcpverifier

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// An OAuthHandler conducts an OAuth flow and returns a [oauth2.TokenSource] if the authorization
// is approved, or an error if not.
// The handler receives the HTTP request and response that triggered the authentication flow.
// To obtain the protected resource metadata, call [oauthex.GetProtectedResourceMetadataFromHeader].
type OAuthHandler func(req *http.Request, res *http.Response) (oauth2.TokenSource, error)

// HTTPTransport is an [http.RoundTripper] that follows the MCP
// OAuth protocol when it encounters a 401 Unauthorized response.
type HTTPTransport struct {
	handler OAuthHandler
	mu      sync.Mutex // protects opts.Base
	opts    HTTPTransportOptions
}

// NewHTTPTransport returns a new [*HTTPTransport].
// The handler is invoked when an HTTP request results in a 401 Unauthorized status.
// It is called only once per transport. Once a TokenSource is obtained, it is used
// for the lifetime of the transport; subsequent 401s are not processed.
func NewHTTPTransport(handler OAuthHandler, opts *HTTPTransportOptions) (*HTTPTransport, error) {
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}
	t := &HTTPTransport{
		handler: handler,
	}
	if opts != nil {
		t.opts = *opts
	}
	if t.opts.Base == nil {
		t.opts.Base = http.DefaultTransport
	}
	return t, nil
}

// HTTPTransportOptions are options to [NewHTTPTransport].
type HTTPTransportOptions struct {
	// Base is the [http.RoundTripper] to use.
	// If nil, [http.DefaultTransport] is used.
	Base          http.RoundTripper
	CustomHeaders map[string]string
}

func (t *HTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	base := t.opts.Base
	t.mu.Unlock()

	var (
		// If haveBody is set, the request has a nontrivial body, and we need avoid
		// reading (or closing) it multiple times. In that case, bodyBytes is its
		// content.
		haveBody  bool
		bodyBytes []byte
	)
	if req.Body != nil && req.Body != http.NoBody {
		// if we're setting Body, we must mutate first.
		req = req.Clone(req.Context())
		haveBody = true
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		// Now that we've read the request body, http.RoundTripper requires that we
		// close it.
		req.Body.Close() // ignore error
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	if _, ok := base.(*oauth2.Transport); ok {
		// We failed to authorize even with a token source; give up.
		return resp, nil
	}

	resp.Body.Close()
	// Try to authorize.
	t.mu.Lock()
	defer t.mu.Unlock()
	// If we don't have a token source, get one by following the OAuth flow.
	// (We may have obtained one while t.mu was not held above.)
	// TODO: We hold the lock for the entire OAuth flow. This could be a long
	// time. Is there a better way?
	if _, ok := t.opts.Base.(*oauth2.Transport); !ok {
		ts, err := t.handler(req, resp)
		if err != nil {
			return nil, err
		}
		t.opts.Base = &oauth2.Transport{Base: t.opts.Base, Source: ts}
	}

	// If we don't have a body, the request is reusable, though it will be cloned
	// by the base. However, if we've had to read the body, we must clone.
	if haveBody {
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// Loop over the custom headers and set them
	for key, val := range t.opts.CustomHeaders {
		req.Header.Add(key, val)
	}

	return t.opts.Base.RoundTrip(req)
}

type ProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
}

type AuthServerMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// IMPROVE_ME: Fix the correct implementation of the Oauth flow with optimized code
func oauthHandler(req *http.Request, res *http.Response) (oauth2.TokenSource, error) {

	oauthFlow := NewOauthFlow()
	authHeader := res.Header.Get("Www-Authenticate")
	oauthClientID := viper.GetString("OAUTH_CLIENT_ID")
	if len(oauthClientID) == 0 {
		return nil, fmt.Errorf("WOODPECKER_OAUTH_CLIENT_ID not set for oauth flow")
	}
	oauthScopes := viper.GetString("OAUTH_SCOPES")
	if len(oauthScopes) == 0 {
		return nil, fmt.Errorf("WOODPECKER_OAUTH_SCOPES not set for oauth flow")
	}
	callBackPort := viper.GetString("OAUTH_CALLBACK_PORT")
	if len(callBackPort) == 0 {
		callBackPort = "6274"
	}

	metadataURL, err := oauthFlow.ExtractResourceMetadata(authHeader)
	if err != nil {
		output.WriteFatal("Error: %v", err)
	}

	prm, err := oauthFlow.FetchProtectedResource(metadataURL)
	if err != nil {
		output.WriteFatal("Error: %v", err)
	}

	authMeta, err := oauthFlow.FetchAuthServerMetadata(prm.AuthorizationServers[0])
	if err != nil {
		output.WriteFatal("Error: %v", err)
	}

	oauthToken, err := oauthFlow.GetOauthTokenSource(authMeta, oauthClientID, fmt.Sprintf("http://localhost:%s/oauth/callback", callBackPort), oauthScopes)
	if err != nil {
		output.WriteFatal("Error: %v", err)
	}

	return oauthToken, nil
}

func (o *OauthFlow) ExtractResourceMetadata(authHeader string) (string, error) {
	re := regexp.MustCompile(`resource_metadata="([^"]+)"`)
	m := re.FindStringSubmatch(authHeader)
	if len(m) < 2 {
		return "", errors.New("resource_metadata not found")
	}
	return m[1], nil
}

func (o *OauthFlow) FetchProtectedResource(metadataURL string) (*ProtectedResourceMetadata, error) {
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, err
	}
	var prm ProtectedResourceMetadata
	if err := json.NewDecoder(resp.Body).Decode(&prm); err != nil {
		return nil, err
	}
	return &prm, nil
}

func (o *OauthFlow) FetchAuthServerMetadata(issuer string) (*AuthServerMetadata, error) {
	metaURL, err := oauthAuthorizationServerMetadataURL(issuer)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(metaURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("metadata discovery failed: %s", resp.Status)
	}

	var meta AuthServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}

	if meta.TokenEndpoint == "" {
		return nil, errors.New("token_endpoint missing")
	}

	return &meta, nil
}

func (o *OauthFlow) GetOauthTokenSource(
	meta *AuthServerMetadata,
	clientID string,
	redirectURI string,
	scope string,
) (oauth2.TokenSource, error) {

	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}

	challenge := codeChallengeS256(verifier)
	state := uuid.NewString()

	authURL := buildAuthURL(
		meta.AuthorizationEndpoint,
		clientID,
		redirectURI,
		scope,
		challenge,
		state,
	)

	output.WriteInfo("Opening browser for authentication...")
	err = openBrowser(authURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	code, returnedState, err := waitForAuthCode(ctx, redirectURI)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("login timed out")
		}
		return nil, err
	}

	if returnedState != state {
		return nil, errors.New("state mismatch")
	}

	token, err := exchangeCodeForToken(
		meta.TokenEndpoint,
		clientID,
		code,
		redirectURI,
		verifier,
	)
	if err != nil {
		return nil, err
	}

	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token.AccessToken,
		TokenType:   "Bearer",
	}), nil
}

func oauthAuthorizationServerMetadataURL(issuer string) (string, error) {
	u, err := url.Parse(issuer)
	if err != nil {
		return "", err
	}

	// RFC 8414 path-based issuer handling
	return fmt.Sprintf(
		"%s://%s/.well-known/oauth-authorization-server%s",
		u.Scheme,
		u.Host,
		strings.TrimRight(u.Path, "/"),
	), nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func buildAuthURL(
	authEndpoint string,
	clientID string,
	redirectURI string,
	scope string,
	codeChallenge string,
	state string,
) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", scope)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)

	return authEndpoint + "?" + q.Encode()
}

func waitForAuthCode(
	ctx context.Context,
	redirectURI string,
) (string, string, error) {

	u, err := url.Parse(redirectURI)
	output.WriteInfo("Starting temp server: %s", u)
	if err != nil {
		return "", "", err
	}

	codeCh := make(chan struct {
		code  string
		state string
	}, 1)

	mux := http.NewServeMux()

	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		fmt.Fprintln(w, "Authentication complete. You can close this window.")

		select {
		case codeCh <- struct {
			code  string
			state string
		}{code, state}:
		default:
		}
	})

	srv := &http.Server{
		Addr:    u.Host,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("auth server error: %v", err)
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	select {
	case res := <-codeCh:
		return res.code, res.state, nil

	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

func exchangeCodeForToken(
	tokenEndpoint string,
	clientID string,
	code string,
	redirectURI string,
	codeVerifier string,
) (*TokenResponse, error) {

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest(
		"POST",
		tokenEndpoint,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed. Status code (%d): %s", resp.StatusCode, body)
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}

	return &tok, nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}

	err := cmd.Start()
	return err

}
