package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Leechael/gemini-web-cli/internal/server"
)

var (
	servePort           int
	serveHost           string
	serveAPIKey         string
	serveExposeThoughts bool
	serveStateDir       string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server with OpenAI-compatible API",
	Args:  cobra.NoArgs,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, _, cookieSource, err := clientConfigFromFlagsWithStateDir(serveStateDir)
	if err != nil {
		return err
	}

	apiKey := firstNonEmpty(serveAPIKey, os.Getenv("GEMINI_WEB_CLI_API_KEY"))
	if apiKey == "" && !isLoopbackHost(serveHost) {
		return fmt.Errorf("--api-key or GEMINI_WEB_CLI_API_KEY is required when binding to non-loopback host %q", serveHost)
	}

	exposeThoughts := serveExposeThoughts || os.Getenv("GEMINI_WEB_CLI_EXPOSE_THOUGHTS") == "1"
	stateInfo := server.StateInfo{
		StateDir:        serveStateDir,
		CookieSource:    cookieSource,
		ChatMappingMode: "memory only",
	}
	if serveStateDir != "" {
		stateInfo.ChatMappingPath = filepath.Join(serveStateDir, "chat-map.pb")
		stateInfo.ChatMappingMode = stateInfo.ChatMappingPath
	}

	srv, err := server.New(cfg, apiKey, exposeThoughts, stateInfo)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	defer srv.Close()

	if err := srv.Init(ctx); err != nil {
		return fmt.Errorf("initializing server: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	return srv.ListenAndServe(addr)
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "Host to bind to")
	serveCmd.Flags().StringVar(&serveAPIKey, "api-key", "", "API key required for /v1 endpoints (or GEMINI_WEB_CLI_API_KEY)")
	serveCmd.Flags().BoolVar(&serveExposeThoughts, "expose-thoughts", false, "Expose model thoughts/reasoning in API responses")
	serveCmd.Flags().StringVar(&serveStateDir, "state-dir", "", "Directory for serve state (cookies.json lookup and chat-map.pb persistence)")
	serveCmd.GroupID = "util"
	rootCmd.AddCommand(serveCmd)
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
