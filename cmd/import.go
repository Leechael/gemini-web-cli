package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var importOutput string

var importCmd = &cobra.Command{
	Use:   "import [raw_cookies]",
	Short: "Parse raw browser cookies and save as JSON",
	Long: `Parse a raw cookie string (from browser DevTools) and save as a structured JSON file.

Example:
  gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...; SID=abc...'
  gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...' -o cookies.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		raw := args[0]
		parsed := parseRawCookies(raw)

		if len(parsed) == 0 {
			return fmt.Errorf("no cookies parsed from input")
		}

		// Check for required cookie
		if _, ok := parsed["__Secure-1PSID"]; !ok {
			fmt.Fprintln(os.Stderr, "Warning: __Secure-1PSID not found in cookies — this may not work with Gemini")
		}

		// Sort keys for stable output
		keys := make([]string, 0, len(parsed))
		for k := range parsed {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sorted := make(map[string]string, len(parsed))
		for _, k := range keys {
			sorted[k] = parsed[k]
		}

		payload := map[string]any{
			"cookies": sorted,
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}

		output := importOutput
		if output == "" {
			output = defaultCookiesPath()
		}

		dir := filepath.Dir(output)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		if err := os.WriteFile(output, append(data, '\n'), 0600); err != nil {
			return err
		}

		fmt.Printf("Saved %d cookies to %s\n", len(parsed), output)
		return nil
	},
}

// parseRawCookies parses a raw cookie header string like "name1=value1; name2=value2; ..."
func parseRawCookies(raw string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(raw, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(pair[:idx])
		value := strings.TrimSpace(pair[idx+1:])
		if name != "" && value != "" {
			result[name] = value
		}
	}
	return result
}

func init() {
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file path (default: cookies.json or $GEMINI_WEB_COOKIES_JSON_PATH)")
}
