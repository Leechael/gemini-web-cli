package cmd

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		fmt.Println("Available models for --model:")

		printed := map[string]bool{}
		printedIDs := map[string]bool{}
		if resolveCookiesJSON() != "" {
			c, jsonCookies, err := initClient(ctx)
			if err == nil {
				defer cleanup(c, jsonCookies)
				_ = c.FetchAndCacheModels(ctx)
				for _, m := range c.AvailableModels() {
					printModelLine(m)
					printed[m.Name] = true
					if id := m.ModelID(); id != "" {
						printedIDs[id] = true
					}
				}
			}
		}

		for _, m := range types.Models {
			if printed[m.Name] || (m.ModelID() != "" && printedIDs[m.ModelID()]) {
				continue
			}
			printModelLine(m)
		}
		fmt.Println("\nNote: dynamic models come from the current Gemini account when cookies are available.")
		fmt.Println("Use 'unspecified' to let Gemini auto-select.")
		return nil
	},
}

func printModelLine(m types.Model) {
	suffix := ""
	if m.Name == "unspecified" {
		suffix = " (default)"
	}
	if m.AdvancedOnly {
		suffix += " [advanced]"
	}
	if m.DisplayName != "" && m.DisplayName != m.Name && m.Name != "unspecified" {
		suffix += fmt.Sprintf(" (%s)", m.DisplayName)
	}
	fmt.Printf("  %s%s\n", m.Name, suffix)
}
