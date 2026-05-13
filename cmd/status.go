package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

		// Full diagnostics — always show header + cookie source before init,
		// so users get useful output even when init fails (expired cookies,
		// rate limit, etc.). status is best-effort by design; we never bubble
		// init failures up as exit-code errors here.
		fmt.Printf("=== gemini-web-cli %s (built %s) ===\n", Version, BuildTime)
		fmt.Println()
		fmt.Println("=== Account Diagnostics ===")
		fmt.Printf("  Model: %s\n", modelName)
		if effective := resolveCookiesJSON(); effective != "" {
			fmt.Printf("  Cookie source: %s (%s)\n", effective, cookieSourceOrigin())
		} else {
			fmt.Printf("  Cookie source: <none — using env vars or no cookies>\n")
		}

		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			var rle *client.RateLimitError
			if errors.As(err, &rle) {
				fmt.Printf("  Init: FAILED — HTTP %d (rate limited)\n", rle.StatusCode)
				fmt.Println("  Hints:")
				fmt.Println("    - Try a different proxy or exit node.")
				fmt.Println("    - Wait a few minutes and retry.")
				fmt.Println("    - Verify gemini.google.com/app loads in a browser via the same proxy.")
				if proxy != "" {
					fmt.Printf("  Proxy: %s\n", proxy)
				}
				return nil
			}
			fmt.Printf("  Init: FAILED — %v\n", err)
			fmt.Println("  Hint: cookies may be expired or incomplete; re-export __Secure-1PSID + __Secure-1PSIDTS from your browser.")
			return nil
		}
		defer cleanup(c, jsonCookies)

		fmt.Printf("  Init: OK (access token obtained)\n")

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

		printAccountQuotas(ctx, c)
		printAbuseStatus(ctx, c)

		return nil
	},
}

// cookieSourceOrigin reports which input the cookies path was resolved from,
// matching the priority order in resolveCookiesJSON.
func cookieSourceOrigin() string {
	if cookiesJSON != "" {
		return "--cookies-json flag"
	}
	if os.Getenv("GEMINI_WEB_COOKIES_JSON_PATH") != "" {
		return "$GEMINI_WEB_COOKIES_JSON_PATH"
	}
	return "auto-discovered"
}

func printAbuseStatus(ctx context.Context, c *client.Client) {
	abuse, err := c.FetchAbuseStatus(ctx)
	if err != nil {
		fmt.Printf("  Abuse status: unavailable (%v)\n", err)
		return
	}
	if abuse == nil {
		return
	}
	if abuse.IsClean {
		fmt.Printf("  Abuse status: clean (no flags)\n")
		return
	}
	parts := []string{}
	if abuse.StatusCode != 0 {
		parts = append(parts, fmt.Sprintf("status=%d", abuse.StatusCode))
	}
	if abuse.Signal != "" {
		parts = append(parts, fmt.Sprintf("signal=%s", abuse.Signal))
	}
	detail := ""
	if len(parts) > 0 {
		detail = " (" + strings.Join(parts, ", ") + ")"
	}
	fmt.Printf("  Abuse status: FLAGGED%s — Google has marked this account; expect throttling or rejections\n", detail)
}

func printAccountQuotas(ctx context.Context, c *client.Client) {
	quotas, err := c.FetchQuotas(ctx, true, true)
	if err != nil {
		fmt.Printf("  Quotas: unavailable (%v)\n", err)
	} else if len(quotas) > 0 {
		fmt.Printf("  Quotas (%d):\n", len(quotas))
		for _, q := range quotas {
			usage := ""
			if q.Total > 0 {
				usage = fmt.Sprintf(" %d/%d remaining", q.Remaining, q.Total)
			} else if q.Total == 0 && q.Remaining == 0 {
				usage = " unlimited"
			}
			reset := ""
			if q.ResetTime > 0 {
				reset = fmt.Sprintf(" (resets %s)", time.Unix(q.ResetTime, 0).Local().Format("2006-01-02 15:04 MST"))
			}
			fmt.Printf("    %s [%s]: %.1f%% used%s%s\n", q.Label, q.ID, q.UsagePercent, usage, reset)
		}
	}

	if extra, err := c.FetchExtraQuota(ctx); err == nil && extra != nil {
		state := "ok"
		if extra.IsBlocked {
			state = "BLOCKED"
		}
		reset := ""
		if extra.ResetTime > 0 {
			reset = fmt.Sprintf(" (resets %s)", time.Unix(extra.ResetTime, 0).Local().Format("2006-01-02 15:04 MST"))
		}
		fmt.Printf("  Extra-feature quota: %s, %.1f%% used%s\n", state, extra.UsagePercent, reset)
	}
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
