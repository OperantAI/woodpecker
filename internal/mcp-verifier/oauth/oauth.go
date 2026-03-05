// Oauth client implementation based on https://github.com/modelcontextprotocol/go-sdk/blob/main/auth/client.go

package oauth

import (
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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// OauthHandler IMPROVE_ME: Find better ways to optimize the OauthHandler auth flow
func OauthHandler(req *http.Request, res *http.Response) (oauth2.TokenSource, error) {

	oauthFlow := NewOauthFlow()
	authHeader := res.Header.Get("Www-Authenticate")
	oauthClientID := viper.GetString("OAUTH_CLIENT_ID")
	if len(oauthClientID) == 0 {
		return nil, fmt.Errorf("WOODPECKER_OAUTH_CLIENT_ID not set for oauth flow")
	}

	oauthClientSecret := viper.GetString("OAUTH_CLIENT_SECRET")
	if len(oauthClientSecret) == 0 {
		return nil, fmt.Errorf("WOODPECKER_OAUTH_CLIENT_SECRET not set for oauth flow")
	}
	oauthScopes := viper.GetString("OAUTH_SCOPES")
	if len(oauthScopes) == 0 {
		return nil, fmt.Errorf("WOODPECKER_OAUTH_SCOPES not set for oauth flow")
	}
	callBackPort := viper.GetString("OAUTH_CALLBACK_PORT")
	if len(callBackPort) == 0 {
		callBackPort = "6274"
	}

	authValue := res.Header.Get("Authorization")
	if authValue != "" {
		return oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: authValue,
			TokenType:   "Bearer",
		}), nil
	}

	metadataURL, err := oauthFlow.ExtractResourceMetadata(authHeader)
	if err != nil {
		return nil, err
	}

	prm, err := oauthFlow.FetchProtectedResource(metadataURL)
	if err != nil {
		return nil, err
	}

	oauthManager := NewOauthManager()
	tokenSource, err := oauthFlow.LoadCachedTokenSource(prm.AuthorizationServers[0], oauthManager)
	if err == nil {
		return tokenSource, err
	}

	authMeta, err := oauthFlow.FetchAuthServerMetadata(prm.AuthorizationServers[0])
	if err != nil {
		return nil, err
	}

	oauthTokenResponse, err := oauthFlow.GetOauthTokenSource(authMeta, oauthClientID, oauthClientSecret, fmt.Sprintf("http://localhost:%s/oauth/callback", callBackPort), oauthScopes)
	if err != nil {
		return nil, err
	}

	err = oauthManager.SaveCacheInfo(prm.AuthorizationServers[0], oauthTokenResponse)
	if err != nil {
		return nil, err
	}

	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: oauthTokenResponse.AccessToken,
		TokenType:   "Bearer",
	}), nil
}

func (o *OauthFlow) LoadCachedTokenSource(issuer string, oAuthManager IOAuthManager) (oauth2.TokenSource, error) {

	oauthTokenResponse, err := oAuthManager.CheckCurrentToken(issuer)
	if err != nil {
		return nil, err
	}

	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: oauthTokenResponse.AccessToken,
		TokenType:   "Bearer",
	}), nil
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
	clientID,
	clientSecret,
	redirectURI,
	scope string,
) (*TokenResponse, error) {

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
		clientSecret, redirectURI,
		verifier,
	)
	if err != nil {
		return nil, err
	}

	return token, nil
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

// waitForAuthCode spawns a temp web server with the oauth callback endpoint to receive the code
// and state from the oauth provider.
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
	tokenEndpoint,
	clientID,
	code,
	clientSecret,
	redirectURI,
	codeVerifier string,
) (*TokenResponse, error) {

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
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
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "mcp-client")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed. Status code (%d): %s", resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
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

// CheckCurrentToken checks if there is a creds file with an access token already provisioned for the current issuer url
func (o *OauthManager) CheckCurrentToken(issuer string) (*TokenResponse, error) {

	appName := viper.GetString("APP_NAME")
	file, err := checkCredsPath(appName)
	if err != nil {
		return nil, err
	}

	existingServers, err := checkExistingCacheConfig(file)
	if err != nil {
		return nil, err
	}

	tokenSource, ok := existingServers[issuer]
	if !ok {
		return nil, fmt.Errorf("no token found for issuer: %s", issuer)
	}

	return &tokenSource, nil
}

// SaveCacheInfo saves in a creds file the access token retrieved from the oauth flow response
func (o *OauthManager) SaveCacheInfo(issuer string, token *TokenResponse) error {

	appName := viper.GetString("APP_NAME")
	file, err := checkCredsPath(appName)
	if err != nil {
		return err
	}

	existingServers, err := checkExistingCacheConfig(file)
	if err != nil {
		return err
	}

	_, ok := existingServers[issuer]
	if !ok {
		existingServers[issuer] = *token
	}

	data, err := json.MarshalIndent(existingServers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, data, 000)
}

// checkExistingCacheConfig retrieves the creds.json struct with the issuer urls and tokens associated if any
func checkExistingCacheConfig(filePath string) (MCPServerCacheConfig, error) {
	var currentConfig MCPServerCacheConfig
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &currentConfig); err != nil {
		return nil, err
	}
	return currentConfig, nil
}

// checkCredsPath based on the app name we search for a $HOME/.config/$WOODPECKER_APP_NAME/creds/creds.json file
// if not present we created with a json struct that takes the Oauth issuer url as the id for that token
func checkCredsPath(appName string) (configPath string, err error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", nil
	}

	dirPath := filepath.Join(dir, ".config", appName, "creds")
	filePath := filepath.Join(dirPath, "creds.json")
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)

	if errors.Is(err, os.ErrExist) {
		output.WriteInfo("File: %s already exists. Using cached toke.", filePath)
		return filePath, nil

	} else if err != nil {
		output.WriteError("Error opening file: %v\n", err)
		return "", err
	}
	defer file.Close()

	// Define initial basic map structure
	tokenConfig := &MCPServerCacheConfig{}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tokenConfig); err != nil {
		output.WriteError("Error writing JSON: %v\n", err)
		return "", err
	}

	return filePath, nil
}
