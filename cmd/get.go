package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	getMaxTurns int
	getOutput   string
)

var getCmd = &cobra.Command{
	Use:   "get [chat_id]",
	Short: "Get a chat conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		chatID := args[0]
		turns, err := c.ReadChat(ctx, chatID, getMaxTurns)
		if err != nil {
			return err
		}

		if len(turns) == 0 {
			fmt.Printf("No turns found for chat %s\n", chatID)
			return fmt.Errorf("no turns found")
		}

		var lines []string
		imgSeq := 0
		vidSeq := 0
		mediaSeq := 0
		for i, turn := range turns {
			lines = append(lines, fmt.Sprintf("--- message %d ---", i+1))
			if turn.UserPrompt != "" {
				lines = append(lines, fmt.Sprintf("[User] %s", turn.UserPrompt))
			}
			if turn.AssistantResponse != "" {
				lines = append(lines, fmt.Sprintf("[Gemini] %s", turn.AssistantResponse))
			}
			for _, img := range turn.Images {
				imgSeq++
				if img.Generated {
					lines = append(lines, fmt.Sprintf("[Generated Image %d] %s", imgSeq, img.URL))
				} else {
					title := ""
					if img.Title != "" {
						title = "  " + img.Title
					}
					lines = append(lines, fmt.Sprintf("[Image %d] %s%s", imgSeq, img.URL, title))
				}
			}
			for _, vid := range turn.Videos {
				vidSeq++
				lines = append(lines, fmt.Sprintf("[Generated Video %d] %s", vidSeq, vid.URL))
				if vid.Thumbnail != "" {
					lines = append(lines, fmt.Sprintf("  Thumbnail: %s", vid.Thumbnail))
				}
			}
			for _, m := range turn.Media {
				mediaSeq++
				if m.MP3URL != "" {
					lines = append(lines, fmt.Sprintf("[Generated Media %d] MP3: %s", mediaSeq, m.MP3URL))
				}
				if m.MP4URL != "" {
					lines = append(lines, fmt.Sprintf("[Generated Media %d] MP4: %s", mediaSeq, m.MP4URL))
				}
			}
			if len(turn.Videos) == 0 && len(turn.Media) == 0 && len(turn.Images) == 0 &&
				turn.AssistantResponse == "" && turn.UserPrompt != "" {
				// Likely still generating
			}
			lines = append(lines, "")
		}

		text := strings.Join(lines, "\n")

		if getOutput != "" {
			dir := filepath.Dir(getOutput)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(getOutput, []byte(text), 0644); err != nil {
				return err
			}
			fmt.Printf("Saved chat to %s\n", getOutput)
		} else {
			fmt.Println(text)
		}
		return nil
	},
}

func init() {
	getCmd.Flags().IntVar(&getMaxTurns, "max-turns", 30, "Max turns to fetch")
	getCmd.Flags().StringVar(&getOutput, "output", "", "Write to file instead of stdout")
}
