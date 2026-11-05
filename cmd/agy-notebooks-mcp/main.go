package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andre-amorim/agy-notebooks-mcp/internal/auth"
	"github.com/andre-amorim/agy-notebooks-mcp/internal/client"
	"github.com/andre-amorim/agy-notebooks-mcp/internal/mcp"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		handleAuthSubcommand(os.Args[2:])
		return
	}

	showVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("agy-notebooks-mcp v%s (arm64/linux/darwin)\n", version)
		return
	}

	// Default behavior: Run MCP stdio server
	runServer()
}

func runServer() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Attempt to load token (or fallback to env/cache)
	token, err := auth.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ℹ️ Notice: %v (running in offline/sandbox mode)\n", err)
	}

	cli, err := client.NewClient(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing client: %v\n", err)
		os.Exit(1)
	}

	server := mcp.NewServer(cli, os.Stdin, os.Stdout)
	if err := server.Serve(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Server terminated with error: %v\n", err)
		os.Exit(1)
	}
}

func handleAuthSubcommand(args []string) {
	if len(args) == 0 {
		printAuthHelp()
		return
	}

	switch args[0] {
	case "login":
		loginCmd := flag.NewFlagSet("login", flag.ExitOnError)
		manualAuth := loginCmd.Bool("manual-auth", false, "Use out-of-band manual code entry (recommended for Crostini/headless environments)")
		port := loginCmd.Int("port", 8085, "Local callback port for OAuth loopback listener")
		_ = loginCmd.Parse(args[1:])

		ctx := context.Background()
		fmt.Fprintln(os.Stderr, "🚀 Initializing Google OAuth2 authentication flow...")
		token, err := auth.Login(ctx, *manualAuth, *port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Authentication failed: %v\n", err)
			os.Exit(1)
		}
		path, _ := auth.GetTokenPath()
		fmt.Fprintf(os.Stderr, "✅ Authentication successful! Token saved to: %s\n", path)
		if !token.Expiry.IsZero() {
			fmt.Fprintf(os.Stderr, "Token expires at: %s\n", token.Expiry.Local().Format("2006-01-02 15:04:05 MST"))
		}

	case "status":
		path, err := auth.GetTokenPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		token, err := auth.Load()
		if err != nil {
			fmt.Printf("Status: Not authenticated (%v)\nToken Path: %s\n", err, path)
			return
		}

		fmt.Printf("Status: Authenticated ✅\nToken Path: %s\n", path)
		if token.IsExpired() {
			fmt.Println("Access Token: Expired (will refresh automatically on demand)")
		} else {
			fmt.Println("Access Token: Valid")
		}
		if !token.Expiry.IsZero() {
			fmt.Printf("Expires: %s\n", token.Expiry.Local().Format("2006-01-02 15:04:05 MST"))
		}
		if token.RefreshToken != "" {
			fmt.Println("Refresh Token: Present")
		}

	case "logout":
		if err := auth.Clear(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to clear token: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Successfully cleared local authentication credentials.")

	case "set-token":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: agy-notebooks-mcp auth set-token <token_string>")
			os.Exit(1)
		}
		token := &auth.TokenConfig{
			AccessToken: args[1],
			TokenType:   "Bearer",
		}
		if err := auth.Save(token); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save token: %v\n", err)
			os.Exit(1)
		}
		path, _ := auth.GetTokenPath()
		fmt.Printf("✅ Static token saved to %s\n", path)

	default:
		printAuthHelp()
	}
}

func printAuthHelp() {
	fmt.Println("Usage: agy-notebooks-mcp auth <subcommand> [flags]")
	fmt.Println("\nSubcommands:")
	fmt.Println("  login      Initiate Google OAuth2 authentication flow")
	fmt.Println("             Flags: --manual-auth, --port=<int>")
	fmt.Println("  status     Display current authentication status and token expiry")
	fmt.Println("  logout     Remove stored token credentials from disk")
	fmt.Println("  set-token  Manually configure a static token without browser interaction")
}
