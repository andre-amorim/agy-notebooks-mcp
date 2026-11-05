package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Default Google OAuth2 endpoints
const (
	GoogleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL = "https://oauth2.googleapis.com/token"

	// Default OAuth2 Scopes for NotebookLM and Google Drive/Docs read access
	DefaultScope = "openid email https://www.googleapis.com/auth/drive.readonly"
)

// OAuthConfig holds client credentials and endpoint URLs.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// GetOAuthConfig retrieves the OAuth configuration from environment variables or returns defaults.
func GetOAuthConfig(redirectURL string) *OAuthConfig {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	// If not provided in environment, use a fallback installed client ID or indicate manual configuration
	if clientID == "" {
		clientID = "100234567890-client.apps.googleusercontent.com" // Placeholder for developer override
	}

	return &OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       strings.Fields(DefaultScope),
	}
}

// BuildAuthURL constructs the authorization URL.
func (c *OAuthConfig) BuildAuthURL(state string) string {
	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("redirect_uri", c.RedirectURL)
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(c.Scopes, " "))
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	v.Set("state", state)
	return GoogleAuthURL + "?" + v.Encode()
}

// Exchange exchanges an authorization code for an OAuth2 token.
func (c *OAuthConfig) Exchange(ctx context.Context, code string) (*TokenConfig, error) {
	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("client_secret", c.ClientSecret)
	v.Set("code", code)
	v.Set("grant_type", "authorization_code")
	v.Set("redirect_uri", c.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	var expiry time.Time
	if tokenResp.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &TokenConfig{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       expiry,
	}, nil
}

// RefreshToken exchanges a refresh token for a fresh access token.
func RefreshToken(ctx context.Context, token *TokenConfig) error {
	if token.RefreshToken == "" {
		return fmt.Errorf("cannot refresh: no refresh token available")
	}

	cfg := GetOAuthConfig("")
	v := url.Values{}
	v.Set("client_id", cfg.ClientID)
	v.Set("client_secret", cfg.ClientSecret)
	v.Set("refresh_token", token.RefreshToken)
	v.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", GoogleTokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	token.AccessToken = tokenResp.AccessToken
	if tokenResp.TokenType != "" {
		token.TokenType = tokenResp.TokenType
	}
	if tokenResp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// Persist refreshed token to disk
	return Save(token)
}

// Login initiates an interactive authentication flow.
func Login(ctx context.Context, manualAuth bool, port int) (*TokenConfig, error) {
	if manualAuth {
		return loginManual(ctx)
	}
	return loginLoopback(ctx, port)
}

func loginManual(ctx context.Context) (*TokenConfig, error) {
	redirectURL := "urn:ietf:wg:oauth:2.0:oob"
	cfg := GetOAuthConfig(redirectURL)
	authURL := cfg.BuildAuthURL("manual-state")

	fmt.Fprintf(os.Stderr, "\n👉 Open the following URL in your browser to authenticate:\n\n%s\n\n", authURL)
	fmt.Fprintf(os.Stderr, "Paste the authorization code here: ")

	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("empty authorization code provided")
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := Save(token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}
	return token, nil
}

func loginLoopback(ctx context.Context, port int) (*TokenConfig, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to bind loopback listener on port %d: %w (try --manual-auth)", port, err)
	}
	defer listener.Close()

	assignedPort := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", assignedPort)
	cfg := GetOAuthConfig(redirectURL)
	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	authURL := cfg.BuildAuthURL(state)

	codeChan := make(chan string)
	errChan := make(chan error)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/callback" {
				http.NotFound(w, r)
				return
			}

			if r.URL.Query().Get("state") != state {
				errChan <- fmt.Errorf("OAuth state mismatch")
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				errChan <- fmt.Errorf("no authorization code returned in callback")
				http.Error(w, "Missing code", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<h2>Authentication Successful!</h2><p>You may now close this browser tab and return to the terminal.</p>")
			codeChan <- code
		}),
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	fmt.Fprintf(os.Stderr, "\n🌐 Opening browser for Google Authentication...\n")
	fmt.Fprintf(os.Stderr, "If your browser does not open automatically, visit this URL:\n\n%s\n\n", authURL)

	openBrowser(authURL)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case code := <-codeChan:
		_ = server.Shutdown(context.Background())
		token, err := cfg.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		if err := Save(token); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
		return token, nil
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}
