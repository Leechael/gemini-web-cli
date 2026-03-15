package cmd

import (
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Available models for --model:")
		for _, m := range types.Models {
			suffix := ""
			if m.Name == "unspecified" {
				suffix = " (default)"
			}
			if m.AdvancedOnly {
				suffix += " [advanced]"
			}
			fmt.Printf("  %s%s\n", m.Name, suffix)
		}
		fmt.Println("\nNote: these map to specific request headers in the library.")
		fmt.Println("Use 'unspecified' to let Gemini auto-select.")
		return nil
	},
}
