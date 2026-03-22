package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress [chat_id]",
	Short: "Check generation progress (deep research, video, music)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		cid := args[0]

		// First try deep research
		status, err := c.CheckDeepResearch(ctx, cid)
		if err != nil {
			return err
		}

		if status.State != "not_research" {
			switch status.State {
			case "done":
				fmt.Printf("  Type: deep research\n")
				fmt.Printf("  Status: done\n")
				fmt.Printf("  Report length: %d chars\n", status.TextLen)
				fmt.Printf("\n  Use 'report %s' to retrieve the full result.\n", cid)
			case "running":
				fmt.Printf("  Type: deep research\n")
				fmt.Printf("  Status: running\n")
			case "pending_confirm":
				fmt.Printf("  Type: deep research\n")
				fmt.Printf("  Status: pending confirmation (plan created but not started)\n")
			case "empty":
				fmt.Printf("  Status: waiting (no response yet)\n")
			default:
				fmt.Printf("  Status: %s\n", status.State)
			}
			return nil
		}

		// Not deep research — check for video/music generation
		turns, err := c.ReadChat(ctx, cid, 5)
		if err != nil {
			return err
		}

		if len(turns) == 0 {
			fmt.Printf("  Status: no turns found\n")
			return nil
		}

		// Check last turn for media
		last := turns[len(turns)-1]
		hasVideos := len(last.Videos) > 0
		hasMedia := len(last.Media) > 0

		if hasVideos {
			fmt.Printf("  Type: video generation\n")
			fmt.Printf("  Status: ready\n")
			for i, vid := range last.Videos {
				fmt.Printf("  Video %d: %s\n", i+1, vid.URL)
			}
			fmt.Printf("\n  Use 'download %s' to save.\n", cid)
			return nil
		}

		if hasMedia {
			fmt.Printf("  Type: music generation\n")
			fmt.Printf("  Status: ready\n")
			for i, m := range last.Media {
				if m.MP3URL != "" {
					fmt.Printf("  Media %d MP3: %s\n", i+1, m.MP3URL)
				}
				if m.MP4URL != "" {
					fmt.Printf("  Media %d MP4: %s\n", i+1, m.MP4URL)
				}
			}
			fmt.Printf("\n  Use 'download %s' to save.\n", cid)
			return nil
		}

		// Check if text suggests generation is in progress
		resp := last.AssistantResponse
		if resp == "" {
			fmt.Printf("  Status: generating (no response yet)\n")
			return nil
		}

		fmt.Printf("  Status: completed (regular chat)\n")
		return nil
	},
}
