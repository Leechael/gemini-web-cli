package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/cookies"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const envCookiesPath = "GEMINI_WEB_COOKIES_JSON_PATH"

// cookiesSearchPaths returns the ordered list of paths to search for cookies.json.
// Project-level (./cookies.json) → User-level (~/.config/) → System-level (/etc/).
func cookiesSearchPaths() []string {
	paths := []string{"cookies.json"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "gemini-web-cli", "cookies.json"))
	}
	paths = append(paths, filepath.Join("/etc", "gemini-web-cli", "cookies.json"))
	return paths
}

// defaultCookiesPath returns the best path for writing cookies:
// env > first writable search path (project-level ./cookies.json).
func defaultCookiesPath() string {
	if p := os.Getenv(envCookiesPath); p != "" {
		return p
	}
	return "cookies.json"
}

// resolveCookiesJSON returns the effective cookies path.
// Priority: --cookies-json flag > $GEMINI_WEB_COOKIES_JSON_PATH > auto-discover.
func resolveCookiesJSON() string {
	if cookiesJSON != "" {
		return cookiesJSON
	}
	if p := os.Getenv(envCookiesPath); p != "" {
		return p
	}
	// Auto-discover from search paths
	for _, p := range cookiesSearchPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// initClient creates and initializes a GeminiClient from CLI flags.
func initClient(ctx context.Context) (*client.Client, map[string]string, error) {
	var jsonCookies map[string]string
	var extraCookies map[string]string

	effectiveCookies := resolveCookiesJSON()
	if effectiveCookies != "" {
		jar, err := cookies.Load(effectiveCookies)
		if err != nil {
			return nil, nil, fmt.Errorf("loading cookies from %s: %w", effectiveCookies, err)
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
		return nil, nil, cookiesNotFoundError()
	}
	if psidts == "" {
		fmt.Fprintln(os.Stderr, "Warning: __Secure-1PSIDTS not found. Session may still work with long-lived cookies.")
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

	c, err := client.New(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("creating client: %w", err)
	}

	if err := c.Init(ctx); err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("initializing client: %w", err)
	}

	return c, jsonCookies, nil
}

// cookiesNotFoundError builds a helpful error message listing what was tried.
func cookiesNotFoundError() error {
	var b strings.Builder
	b.WriteString("no cookies found\n\n")
	b.WriteString("Looked in:\n")
	if cookiesJSON != "" {
		b.WriteString(fmt.Sprintf("  --cookies-json %s (not found or missing __Secure-1PSID)\n", cookiesJSON))
	}
	if p := os.Getenv(envCookiesPath); p != "" {
		b.WriteString(fmt.Sprintf("  $GEMINI_WEB_COOKIES_JSON_PATH=%s\n", p))
	}
	for _, p := range cookiesSearchPaths() {
		b.WriteString(fmt.Sprintf("  %s\n", p))
	}
	b.WriteString("\nTo get started:\n")
	b.WriteString("  1. Open https://gemini.google.com in your browser\n")
	b.WriteString("  2. Copy cookies from DevTools (Application → Cookies → copy all as header string)\n")
	b.WriteString("  3. Run: gemini-web-cli import '<cookie_string>'\n")
	return fmt.Errorf("%s", b.String())
}

// cleanup persists cookies and closes the client.
func cleanup(c *client.Client, jsonCookies map[string]string) {
	effectiveCookies := resolveCookiesJSON()
	if effectiveCookies != "" && !noPersist && jsonCookies != nil {
		_ = cookies.Persist(effectiveCookies, jsonCookies, c.ExtraCookies, verbose)
	}
	c.Close()
}

func resolveModel() *types.Model {
	return types.FindModel(modelName)
}
