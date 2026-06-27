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
	path, _ := resolveCookiesJSONWithStateDir("")
	return path
}

func resolveCookiesJSONWithStateDir(stateDir string) (string, string) {
	if cookiesJSON != "" {
		return cookiesJSON, "--cookies-json"
	}
	if stateDir != "" {
		p := filepath.Join(stateDir, "cookies.json")
		if _, err := os.Stat(p); err == nil {
			return p, "state-dir"
		}
	}
	if p := os.Getenv(envCookiesPath); p != "" {
		return p, "$" + envCookiesPath
	}
	// Auto-discover from search paths
	for _, p := range cookiesSearchPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, "auto-discover"
		}
	}
	return "", ""
}

// clientConfigFromFlags builds client configuration from CLI flags and environment.
func clientConfigFromFlags() (client.Config, map[string]string, error) {
	cfg, jsonCookies, _, err := clientConfigFromFlagsWithStateDir("")
	return cfg, jsonCookies, err
}

func clientConfigFromFlagsWithStateDir(stateDir string) (client.Config, map[string]string, string, error) {
	var jsonCookies map[string]string
	var extraCookies map[string]string

	effectiveCookies, cookieSource := resolveCookiesJSONWithStateDir(stateDir)
	if effectiveCookies != "" {
		jar, err := cookies.Load(effectiveCookies)
		if err != nil {
			return client.Config{}, nil, "", fmt.Errorf("loading cookies from %s: %w", effectiveCookies, err)
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
		return client.Config{}, nil, "", cookiesNotFoundError()
	}
	if psidts == "" {
		fmt.Fprintln(os.Stderr, "Warning: __Secure-1PSIDTS not found. Session may still work with long-lived cookies.")
	}

	var acctIdx *int
	if rootCmd.PersistentFlags().Changed("account-index") {
		acctIdx = &accountIndex
	}

	model := types.FindModel(modelName)

	if effectiveCookies == "" {
		cookieSource = "env vars"
	}

	return client.Config{
		Secure1PSID:   psid,
		Secure1PSIDTS: psidts,
		ExtraCookies:  extraCookies,
		Proxy:         proxy,
		AccountIndex:  acctIdx,
		Model:         model,
		Verbose:       verbose,
		Timeout:       time.Duration(requestTimeout * float64(time.Second)),
	}, jsonCookies, cookieSourceName(effectiveCookies, cookieSource), nil
}

func cookieSourceName(path, source string) string {
	if path == "" {
		return source
	}
	if source == "" {
		return path
	}
	return fmt.Sprintf("%s (%s)", path, source)
}

// initClient creates and initializes a GeminiClient from CLI flags.
func initClient(ctx context.Context) (*client.Client, map[string]string, error) {
	cfg, jsonCookies, err := clientConfigFromFlags()
	if err != nil {
		return nil, nil, err
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

func setGenerationMode(c *client.Client, mode string) error {
	switch mode {
	case "", "auto", "text", "video", "image-to-video", "music":
		c.SetGenerationMode(mode)
		return nil
	default:
		return fmt.Errorf("invalid generation mode %q — use auto, text, video, image-to-video, or music", mode)
	}
}

func resolveModelForClient(ctx context.Context, c *client.Client, preferred ...string) *types.Model {
	if modelName == "" || modelName == "unspecified" {
		for _, name := range preferred {
			if name == "" {
				continue
			}
			if model := types.FindModel(name); model != nil {
				return model
			}
		}
		return types.FindModel("unspecified")
	}
	if model := types.FindModel(modelName); model != nil {
		return model
	}
	if c != nil {
		_ = c.FetchAndCacheModels(ctx)
		if model := c.ResolveModel(modelName); model != nil {
			return model
		}
	}
	fmt.Fprintf(os.Stderr, "Warning: model %q not found; using Gemini auto-select.\n", modelName)
	return types.FindModel("unspecified")
}

func preferredModelForGenerationMode(mode string, prompt string, hasUploads bool) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "auto" || mode == "" {
		lower := strings.ToLower(prompt)
		switch {
		case hasUploads && (strings.Contains(lower, "video") || strings.Contains(lower, "视频")):
			mode = "image-to-video"
		case strings.Contains(lower, "music") || strings.Contains(lower, "song") || strings.Contains(lower, "audio") || strings.Contains(lower, "音乐") || strings.Contains(lower, "歌曲"):
			mode = "music"
		case strings.Contains(lower, "video") || strings.Contains(lower, "视频"):
			mode = "video"
		}
	}
	switch mode {
	case "video", "image-to-video", "music":
		return "gemini-3.5-flash"
	default:
		return ""
	}
}
