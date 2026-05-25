package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
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

		bs := c.PrefetchBootstrap(ctx)
		if bs.Profile != nil {
			label := bs.Profile.DisplayName
			if bs.Profile.Email != "" {
				label = fmt.Sprintf("%s <%s>", bs.Profile.DisplayName, bs.Profile.Email)
			}
			fmt.Printf("  User: %s\n", label)
		}
		if bs.Location != nil && bs.Location.Region != "" {
			fmt.Printf("  Location: %s\n", bs.Location.Region)
		}
		if len(bs.Tools) > 0 {
			fmt.Printf("  Enabled tools: %d (%s)\n", len(bs.Tools), toolNamesPreview(bs.Tools, 3))
		}
		if len(bs.Extensions) > 0 {
			fmt.Printf("  Extension catalog: %d entries (%s)\n", len(bs.Extensions), extensionIDsPreview(bs.Extensions, 7))
		}
		if len(bs.Flags) > 0 {
			fmt.Printf("  Feature flags: %d active\n", activeFeatureFlagCount(bs.Flags))
		}
		if bs.Limits != nil {
			fmt.Printf("  Upload limits: raw [%d, %d, %d]\n", bs.Limits.Limit0, bs.Limits.Limit1, bs.Limits.Limit2)
		}
		if len(bs.Errors) > 0 {
			if verbose {
				for _, key := range bootstrapErrorKeys(bs.Errors) {
					fmt.Fprintf(os.Stderr, "bootstrap RPC %s failed: %v\n", key, bs.Errors[key])
				}
			} else {
				fmt.Fprintf(os.Stderr, "Bootstrap: %d of 6 RPCs failed (run with --verbose for details)\n", len(bs.Errors))
			}
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

func toolNamesPreview(tools []rpcs.EnabledTool, limit int) string {
	parts := make([]string, 0, len(tools))
	for _, tool := range tools {
		if tool.Name != "" {
			parts = append(parts, tool.Name)
		}
	}
	return previewStrings(parts, limit)
}

func extensionIDsPreview(extensions []rpcs.Extension, limit int) string {
	parts := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		if ext.ID != "" {
			parts = append(parts, ext.ID)
		}
	}
	return previewStrings(parts, limit)
}

func activeFeatureFlagCount(flags []rpcs.FeatureFlag) int {
	count := 0
	for _, flag := range flags {
		if flag.Enabled {
			count++
		}
	}
	return count
}

func previewStrings(values []string, limit int) string {
	if limit <= 0 || len(values) == 0 {
		return ""
	}
	if len(values) <= limit {
		return strings.Join(values, ", ")
	}
	return strings.Join(values[:limit], ", ") + ", ..."
}

func bootstrapErrorKeys(errors map[string]error) []string {
	keys := make([]string, 0, len(errors))
	for key := range errors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	statusCmd.Flags().BoolVar(&statusCookiesOnly, "cookies-only", false, "Cookie diagnostics only")
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
