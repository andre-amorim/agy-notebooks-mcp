package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenConfig holds OAuth2 access credentials, refresh tokens, and optional session cookies.
type TokenConfig struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	Cookies      string    `json:"cookies,omitempty"`
}

// IsExpired returns true if the token is past expiry (with a 60-second safety buffer).
func (t *TokenConfig) IsExpired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}
	if t.Expiry.IsZero() {
		return false // No expiry specified
	}
	return time.Now().Add(60 * time.Second).After(t.Expiry)
}

// GetTokenDir returns the configuration directory path ($XDG_CONFIG_HOME/agy-notebooks or ~/.config/agy-notebooks).
func GetTokenDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine user home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configHome, "agy-notebooks")
	return dir, nil
}

// GetTokenPath returns the full path to the token.json storage file.
func GetTokenPath() (string, error) {
	dir, err := GetTokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// Save persists the token to disk with restricted permissions (0600).
func Save(token *TokenConfig) error {
	dir, err := GetTokenDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	path, err := GetTokenPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode token: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

// Load loads the token from disk or falls back to environment variables.
func Load() (*TokenConfig, error) {
	// 1. Check environment variables first (Defensive Scope Cut fallback)
	if envToken := os.Getenv("NOTEBOOKLM_TOKEN"); envToken != "" {
		return &TokenConfig{
			AccessToken: envToken,
			TokenType:   "Bearer",
		}, nil
	}
	if envToken := os.Getenv("GOOGLE_OAUTH_TOKEN"); envToken != "" {
		return &TokenConfig{
			AccessToken: envToken,
			TokenType:   "Bearer",
		}, nil
	}
	if envCookies := os.Getenv("NOTEBOOKLM_COOKIES"); envCookies != "" {
		return &TokenConfig{
			Cookies: envCookies,
		}, nil
	}

	// 2. Load from disk
	path, err := GetTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no token found at %s. Run 'agy-notebooks-mcp auth login' to authenticate", path)
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token TokenConfig
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}
	return &token, nil
}

// Clear removes the persisted token file.
func Clear() error {
	path, err := GetTokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	return nil
}
