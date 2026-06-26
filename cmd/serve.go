package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/cookies"
	"github.com/Leechael/gemini-web-cli/internal/server"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start HTTP server with OpenAI-compatible API",
	Args:  cobra.NoArgs,
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	var jsonCookies map[string]string
	var extraCookies map[string]string

	effectiveCookies := resolveCookiesJSON()
	if effectiveCookies != "" {
		jar, err := cookies.Load(effectiveCookies)
		if err != nil {
			return fmt.Errorf("loading cookies from %s: %w", effectiveCookies, err)
		}
		jsonCookies = jar.Cookies

		extraCookies = make(map[string]string)
		for k, v := range jar.Cookies {
			if k != "__Secure-1PSID" && k != "__Secure-1PSIDTS" {
				extraCookies[k] = v
			}
		}
	}

	psid := firstNonEmpty(jsonCookies["__Secure-1PSID"], os.Getenv("GEMINI_SECURE_1PSID"))
	psidts := firstNonEmpty(jsonCookies["__Secure-1PSIDTS"], os.Getenv("GEMINI_SECURE_1PSIDTS"))

	if psid == "" {
		return cookiesNotFoundError()
	}

	var acctIdx *int
	if rootCmd.PersistentFlags().Changed("account-index") {
		acctIdx = &accountIndex
	}

	model := types.FindModel(modelName)

	cfg := client.Config{
		Secure1PSID:   psid,
		Secure1PSIDTS: psidts,
		ExtraCookies:  extraCookies,
		Proxy:         proxy,
		AccountIndex:  acctIdx,
		Model:         model,
		Verbose:       verbose,
		Timeout:       time.Duration(requestTimeout) * time.Second,
	}

	srv, err := server.New(cfg)
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
	serveCmd.GroupID = "util"
	rootCmd.AddCommand(serveCmd)
}
