package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var googNoStream bool

var googCmd = &cobra.Command{
	Use:   "goog [query]",
	Short: "Google search via Gemini (shortcut for: ask \"@Google ...\")",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		prompt := "@Google " + strings.Join(args, " ")
		model := resolveModel()

		if googNoStream {
			output, err := c.GenerateContent(ctx, prompt, model)
			if err != nil {
				return err
			}
			fmt.Println(output.Text)
			printImages(output)
		} else {
			output, err := c.GenerateContentStream(ctx, prompt, model, func(out *types.ModelOutput) {
				if out.TextDelta != "" {
					fmt.Print(out.TextDelta)
				}
			})
			if err != nil {
				return err
			}
			if output != nil {
				fmt.Println()
				printImages(output)
			}
		}
		return nil
	},
}

func init() {
	googCmd.Flags().BoolVar(&googNoStream, "no-stream", false, "Wait for complete response")
}
