package cmd

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	askNoStream bool
	askImage    string
)

var askCmd = &cobra.Command{
	Use:   "ask [prompt]",
	Short: "Single-turn question (prefix with @Google to trigger search)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		prompt := args[0]
		model := resolveModel()

		if askNoStream {
			output, err := c.GenerateContent(ctx, prompt, model)
			if err != nil {
				return err
			}
			fmt.Println(output.Text)
			printImages(output)
			printChatID(output)
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
				printChatID(output)
			}
		}
		return nil
	},
}

func init() {
	askCmd.Flags().BoolVar(&askNoStream, "no-stream", false, "Wait for complete response")
	askCmd.Flags().StringVar(&askImage, "image", "", "Attach an image/file")
}

func printImages(output *types.ModelOutput) {
	if output == nil || len(output.Images) == 0 {
		return
	}
	var web, gen []types.Image
	for _, img := range output.Images {
		if img.Generated {
			gen = append(gen, img)
		} else {
			web = append(web, img)
		}
	}
	if len(web) > 0 {
		fmt.Println("\n---\nImages:")
		for i, img := range web {
			title := ""
			if img.Title != "" {
				title = "  " + img.Title
			}
			fmt.Printf("  %d) %s%s\n", i+1, img.URL, title)
		}
	}
	if len(gen) > 0 {
		fmt.Println("\n---\nGenerated images:")
		for i, img := range gen {
			fmt.Printf("  %d) %s\n", i+1, img.URL)
		}
	}
}

func printChatID(output *types.ModelOutput) {
	if output != nil && len(output.Metadata) > 0 && output.Metadata[0] != "" {
		fmt.Printf("\n---\nChat ID: %s\n", output.Metadata[0])
	}
}
