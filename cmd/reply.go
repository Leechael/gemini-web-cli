package cmd

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var replyNoStream bool

var replyCmd = &cobra.Command{
	Use:   "reply [chat_id] [prompt]",
	Short: "Continue an existing chat",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		chatID := args[0]
		prompt := args[1]
		model := resolveModel()

		// Fetch latest turn to get rid/rcid for proper conversation continuation
		// (matching Python's cmd_reply: metadata=[cid, rid, rcid])
		metadata := make([]string, 10)
		metadata[0] = chatID

		latest, err := c.FetchLatestChatResponse(ctx, chatID)
		if err == nil && latest != nil {
			if latest.Rid != "" {
				metadata[1] = latest.Rid
			}
			if latest.RCid != "" {
				metadata[2] = latest.RCid
			}
		}

		if replyNoStream {
			output, err := c.SendMessage(ctx, prompt, metadata, model)
			if err != nil {
				return err
			}
			fmt.Println(output.Text)
			printImages(output)
			printVideos(output)
			printMedia(output)
		} else {
			output, err := c.SendMessageStream(ctx, prompt, metadata, model, func(out *types.ModelOutput) {
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
				printVideos(output)
				printMedia(output)
			}
		}

		fmt.Printf("\n---\nChat ID: %s\n", chatID)
		return nil
	},
}

func init() {
	replyCmd.Flags().BoolVar(&replyNoStream, "no-stream", false, "Wait for complete response")
}
