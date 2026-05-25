package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	latexInlineMathPattern = regexp.MustCompile(`\$([^$\n]+)\$`)
	latexTextPattern       = regexp.MustCompile(`\\text\{([^}]*)\}`)
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
			if turn.UserPrompt != "" {
				lines = append(lines, formatMessageBlock("user", i+1, turn.Rid, turn.CreatedAtUnix, formatChatText(turn.UserPrompt)), "")
			}
			if turn.AssistantResponse != "" {
				lines = append(lines, formatMessageBlock("agent", i+1, turn.RCid, turn.CreatedAtUnix, formatChatText(turn.AssistantResponse)), "")
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
				label := fmt.Sprintf("[Generated Media %d]", mediaSeq)
				if m.Title != "" {
					label += " " + m.Title
				}
				if m.MP3URL != "" {
					lines = append(lines, fmt.Sprintf("%s MP3: %s", label, m.MP3URL))
				}
				if m.MP4URL != "" {
					lines = append(lines, fmt.Sprintf("%s MP4: %s", label, m.MP4URL))
				}
				if m.VTTURL != "" {
					lines = append(lines, fmt.Sprintf("%s VTT: %s", label, m.VTTURL))
				}
				if m.Genre != "" || len(m.Moods) > 0 {
					lines = append(lines, fmt.Sprintf("%s Style: %s %s", label, m.Genre, strings.Join(m.Moods, ", ")))
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

func formatMessageBlock(role string, index int, id string, createdAt int64, body string) string {
	label := fmt.Sprintf("--- %s #%d", role, index)
	if id != "" {
		label += " " + id
	}
	if createdAt != 0 {
		label += " (" + time.Unix(createdAt, 0).Local().Format("2006-01-02 15:04 MST") + ")"
	}
	label += " ---"
	return label + "\n" + body
}

func formatChatText(text string) string {
	text = latexTextPattern.ReplaceAllString(text, "$1")
	replacements := map[string]string{
		"$$":              "",
		`\longrightarrow`: "→",
		`\rightarrow`:     "→",
	}
	for old, newText := range replacements {
		text = strings.ReplaceAll(text, old, newText)
	}
	text = latexInlineMathPattern.ReplaceAllString(text, "$1")
	return strings.TrimSpace(text)
}

func init() {
	getCmd.Flags().IntVar(&getMaxTurns, "max-turns", 30, "Max turns to fetch")
	getCmd.Flags().StringVar(&getOutput, "output", "", "Write to file instead of stdout")
}
