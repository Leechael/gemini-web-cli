package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/cookies"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const envCookiesPath = "GEMINI_WEB_COOKIES_JSON_PATH"

// defaultCookiesPath returns the cookies file path from env or "cookies.json".
func defaultCookiesPath() string {
	if p := os.Getenv(envCookiesPath); p != "" {
		return p
	}
	return "cookies.json"
}

// resolveCookiesJSON returns the effective cookies path: flag > env > "".
func resolveCookiesJSON() string {
	if cookiesJSON != "" {
		return cookiesJSON
	}
	if p := os.Getenv(envCookiesPath); p != "" {
		return p
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
			return nil, nil, fmt.Errorf("loading cookies: %w", err)
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
		return nil, nil, fmt.Errorf("missing required cookie: __Secure-1PSID — export cookies from browser and provide via --cookies-json")
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
