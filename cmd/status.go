package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/cookies"
	"github.com/spf13/cobra"
)

var statusCookiesOnly bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check login status and account diagnostics",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusCookiesOnly {
			if cookiesJSON == "" {
				return fmt.Errorf("--cookies-json is required for --cookies-only")
			}
			jar, err := cookies.Load(cookiesJSON)
			if err != nil {
				return err
			}

			report := map[string]any{
				"cookies_json": cookiesJSON,
				"required": map[string]any{
					"__Secure-1PSID": map[string]any{
						"present": jar.Cookies["__Secure-1PSID"] != "",
					},
					"__Secure-1PSIDTS": map[string]any{
						"present": jar.Cookies["__Secure-1PSIDTS"] != "",
					},
				},
				"all_cookie_keys_in_json": sortedKeys(jar.Cookies),
			}

			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		// Full check requires client init
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			var rle *client.RateLimitError
			if errors.As(err, &rle) {
				report := map[string]any{
					"status":  "rate_limited",
					"code":    rle.StatusCode,
					"message": "Google returned HTTP 429 — your exit node is likely rate-limited.",
					"hints": []string{
						"Try a different proxy or exit node.",
						"Wait a few minutes and retry.",
						"Verify you can load gemini.google.com/app in a browser through the same proxy.",
					},
				}
				if proxy != "" {
					report["proxy"] = proxy
				}
				data, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(data))
				return nil
			}
			return err
		}
		defer cleanup(c, jsonCookies)

		fmt.Println("=== Account Diagnostics ===")
		fmt.Printf("  Init: OK (access token obtained)\n")
		fmt.Printf("  Model: %s\n", modelName)
		if cookiesJSON != "" {
			fmt.Printf("  Cookie source: %s\n", cookiesJSON)
		}

		// Fetch account status
		status, dynamicModels, fetchErr := c.FetchUserStatus(ctx)
		if fetchErr != nil {
			fmt.Printf("  Account status: unknown (fetch failed: %v)\n", fetchErr)
		} else {
			fmt.Printf("  Account status: %s — %s\n", status.Name, status.Description)
			if len(dynamicModels) > 0 {
				fmt.Printf("  Available models (%d):\n", len(dynamicModels))
				for _, m := range dynamicModels {
					suffix := ""
					if m.AdvancedOnly {
						suffix = " [advanced]"
					}
					fmt.Printf("    %s (%s)%s\n", m.Name, m.DisplayName, suffix)
				}
			}
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusCookiesOnly, "cookies-only", false, "Cookie diagnostics only")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
