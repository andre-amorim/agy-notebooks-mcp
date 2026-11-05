package auth

import (
	"os"
	"testing"
	"time"
)

func TestTokenExpiry(t *testing.T) {
	// 1. Expired token
	expired := &TokenConfig{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(-10 * time.Minute),
	}
	if !expired.IsExpired() {
		t.Errorf("expected token to be expired")
	}

	// 2. Token expiring within 30 seconds (safety margin triggers)
	nearExpiry := &TokenConfig{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(30 * time.Second),
	}
	if !nearExpiry.IsExpired() {
		t.Errorf("expected token near expiry to report expired for proactive refresh")
	}

	// 3. Valid future token
	future := &TokenConfig{
		AccessToken: "test-token",
		Expiry:      time.Now().Add(2 * time.Hour),
	}
	if future.IsExpired() {
		t.Errorf("expected future token to not be expired")
	}

	// 4. Token without expiry
	noExpiry := &TokenConfig{
		AccessToken: "test-token",
	}
	if noExpiry.IsExpired() {
		t.Errorf("expected token without expiry to be valid")
	}
}

func TestTokenSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", origConfigHome)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &TokenConfig{
		AccessToken:  "access-12345",
		RefreshToken: "refresh-67890",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
	}

	if err := Save(token); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	path, err := GetTokenPath()
	if err != nil {
		t.Fatalf("failed to get token path: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat token path: %v", err)
	}
	// Check restricted permissions on UNIX
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("expected AccessToken %s, got %s", token.AccessToken, loaded.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("expected RefreshToken %s, got %s", token.RefreshToken, loaded.RefreshToken)
	}

	if err := Clear(); err != nil {
		t.Fatalf("failed to clear token: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected token file to be deleted")
	}
}

func TestEnvTokenFallback(t *testing.T) {
	orig := os.Getenv("NOTEBOOKLM_TOKEN")
	defer os.Setenv("NOTEBOOKLM_TOKEN", orig)

	os.Setenv("NOTEBOOKLM_TOKEN", "mock-env-token-xyz")

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load token from env: %v", err)
	}
	if loaded.AccessToken != "mock-env-token-xyz" {
		t.Errorf("expected env token mock-env-token-xyz, got %s", loaded.AccessToken)
	}
}
