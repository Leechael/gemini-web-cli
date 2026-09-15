package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Leechael/gemini-web-cli/internal/server"
)

const (
	envServeHost     = "GEMINI_WEB_CLI_HOST"
	envServePort     = "GEMINI_WEB_CLI_PORT"
	envServeStateDir = "GEMINI_WEB_CLI_STATE_DIR"
)

var (
	servePort            int
	serveHost            string
	serveAPIKey          string
	serveExposeThoughts  bool
	serveStateDir        string
	serveMCPDefaultModel string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server with OpenAI-compatible API",
	Args:  cobra.NoArgs,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	stateDir := serveBindStateDir(serveStateDir, cmd.Flags().Changed("state-dir"))
	cfgs, cookieSources, err := clientConfigsWithStateDir(stateDir)
	if err != nil {
		return err
	}

	apiKey := firstNonEmpty(serveAPIKey, os.Getenv("GEMINI_WEB_CLI_API_KEY"))

	exposeThoughts := serveExposeThoughts || os.Getenv("GEMINI_WEB_CLI_EXPOSE_THOUGHTS") == "1"
	stateInfo := server.StateInfo{
		StateDir:        stateDir,
		CookieSource:    strings.Join(cookieSources, ", "),
		ChatMappingMode: "memory only",
	}
	if stateDir != "" {
		stateInfo.ChatMappingPath = filepath.Join(stateDir, "chat-map.pb")
		stateInfo.ChatMappingMode = stateInfo.ChatMappingPath
	}
	stateInfo.MCPDefaultModel = serveMCPDefaultModel

	srv, err := server.New(cfgs, cookieSources, apiKey, exposeThoughts, serveMCPDefaultModel, stateInfo)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}
	defer srv.Close()

	if err := srv.Init(ctx); err != nil {
		return fmt.Errorf("initializing server: %w", err)
	}

	host := serveBindHost(serveHost, cmd.Flags().Changed("host"))
	port, err := serveBindPort(servePort, cmd.Flags().Changed("port"))
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	return srv.ListenAndServe(addr)
}

// serveBindHost uses --host when the flag was set, otherwise GEMINI_WEB_CLI_HOST,
// otherwise the flag default (127.0.0.1).
func serveBindHost(flagValue string, flagSet bool) string {
	return flagOrEnv(flagValue, flagSet, envServeHost)
}

// serveBindStateDir uses --state-dir when the flag was set, otherwise
// GEMINI_WEB_CLI_STATE_DIR, otherwise the flag default (unset).
func serveBindStateDir(flagValue string, flagSet bool) string {
	return flagOrEnv(flagValue, flagSet, envServeStateDir)
}

func flagOrEnv(flagValue string, flagSet bool, envName string) string {
	if flagSet {
		return flagValue
	}
	if v := os.Getenv(envName); v != "" {
		return v
	}
	return flagValue
}

// serveBindPort uses --port when the flag was set, otherwise GEMINI_WEB_CLI_PORT,
// otherwise the flag default (8080).
func serveBindPort(flagValue int, flagSet bool) (int, error) {
	if flagSet {
		return flagValue, nil
	}
	if v := os.Getenv(envServePort); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid $%s %q", envServePort, v)
		}
		return p, nil
	}
	return flagValue, nil
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Port to listen on (or $GEMINI_WEB_CLI_PORT)")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "Host to bind to (or $GEMINI_WEB_CLI_HOST)")
	serveCmd.Flags().StringVar(&serveAPIKey, "api-key", "", "API key for /v1 endpoints (or $GEMINI_WEB_CLI_API_KEY)")
	serveCmd.Flags().BoolVar(&serveExposeThoughts, "expose-thoughts", false, "Expose model thoughts/reasoning in API responses")
	serveCmd.Flags().StringVar(&serveStateDir, "state-dir", "", "Directory for serve state (cookies.json lookup and chat-map.pb persistence; or $GEMINI_WEB_CLI_STATE_DIR)")
	serveCmd.Flags().StringVar(&serveMCPDefaultModel, "mcp-default-model", "", "Default model for MCP tools (used when a tool call omits model)")
	serveCmd.GroupID = "util"
	rootCmd.AddCommand(serveCmd)
}
