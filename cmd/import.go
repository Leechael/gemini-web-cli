package cmd

import (
	"encoding/json"
	"fmt"
	"io"
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

Read from an argument, from stdin when the argument is omitted or "-", or from a pipe:

  gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...; SID=abc...'
  gemini-web-cli import '_ga=GA1.1.123; __Secure-1PSID=g.a000...' -o cookies.json
  pbpaste | gemini-web-cli import
  gemini-web-cli import - -o cookies.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readImportInput(args)
		if err != nil {
			return err
		}
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

// readImportInput returns the raw cookie string from the argument, or stdin
// when the argument is omitted or `-`. A missing argument on a TTY is an
// error so the command does not hang waiting for keyboard input.
func readImportInput(args []string) (string, error) {
	if len(args) == 1 && args[0] != "-" {
		return args[0], nil
	}
	if len(args) == 0 {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return "", err
		}
		if fi.Mode()&os.ModeCharDevice != 0 {
			return "", fmt.Errorf("cookie string required (pass as argument or pipe via stdin)")
		}
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return "", fmt.Errorf("no cookies parsed from stdin")
	}
	return raw, nil
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
