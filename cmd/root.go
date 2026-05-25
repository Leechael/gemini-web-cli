package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Leechael/gemini-web-cli/internal/client"
)

var (
	cookiesJSON    string
	proxy          string
	accountIndex   int
	hasAccountIdx  bool
	modelName      string
	verbose        bool
	noPersist      bool
	requestTimeout float64
)

// Version and BuildTime are set at build time via -ldflags. BuildTime should
// be a UTC timestamp in RFC 3339 form (e.g. 2026-05-09T13:45:00Z).
var (
	Version   = "dev"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:     "gemini-web-cli",
	Short:   "CLI for Gemini web API",
	Long:    "Command-line interface for interacting with Google Gemini via web cookies.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Silence usage after Args validation passes — argument errors
		// still show usage, but runtime errors (network, auth, etc.) don't.
		cmd.SilenceUsage = true
		if verbose {
			client.SetVerbose(os.Stderr)
		}
	},
}

func init() {
	// Detect proxy from environment
	defaultProxy := firstNonEmpty(
		os.Getenv("HTTPS_PROXY"),
		os.Getenv("https_proxy"),
		os.Getenv("HTTP_PROXY"),
		os.Getenv("http_proxy"),
	)

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&cookiesJSON, "cookies-json", "", "Path to JSON cookie file (or set $GEMINI_WEB_COOKIES_JSON_PATH)")
	pf.StringVar(&proxy, "proxy", defaultProxy, "HTTP/SOCKS proxy URL")
	pf.IntVar(&accountIndex, "account-index", 0, "Google account index (e.g. 2 => /u/2)")
	pf.StringVar(&modelName, "model", "unspecified", "Model name")
	pf.BoolVar(&verbose, "verbose", false, "Enable debug logging")
	pf.BoolVar(&noPersist, "no-persist", false, "Do not write updated cookies back")
	pf.Float64Var(&requestTimeout, "request-timeout", 300, "Per-request HTTP timeout in seconds")

	rootCmd.AddGroup(
		&cobra.Group{ID: "chat", Title: "Chat:"},
		&cobra.Group{ID: "research", Title: "Deep Research:"},
		&cobra.Group{ID: "util", Title: "Utilities:"},
		&cobra.Group{ID: "debug", Title: "Debugging:"},
	)

	askCmd.GroupID = "chat"
	replyCmd.GroupID = "chat"
	googCmd.GroupID = "chat"
	listCmd.GroupID = "chat"
	getCmd.GroupID = "chat"
	downloadCmd.GroupID = "chat"
	chatCmd.GroupID = "chat"

	researchCmd.GroupID = "research"
	progressCmd.GroupID = "chat"
	reportCmd.GroupID = "research"

	modelsCmd.GroupID = "util"
	statusCmd.GroupID = "util"
	importCmd.GroupID = "util"
	expandPromptCmd.GroupID = "util"

	debugCmd.GroupID = "debug"

	rootCmd.AddCommand(askCmd, replyCmd, googCmd, listCmd, getCmd, downloadCmd, chatCmd)
	rootCmd.AddCommand(researchCmd, progressCmd, reportCmd)
	rootCmd.AddCommand(modelsCmd, statusCmd, importCmd, expandPromptCmd)
	rootCmd.AddCommand(debugCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
