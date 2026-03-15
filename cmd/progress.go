package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress [chat_id]",
	Short: "Check deep research progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		cid := args[0]
		status, err := c.CheckDeepResearch(ctx, cid)
		if err != nil {
			return err
		}

		switch status.State {
		case "done":
			fmt.Printf("  Status: done\n")
			fmt.Printf("  Report length: %d chars\n", status.TextLen)
			fmt.Printf("\n  Use 'report %s' to retrieve the full result.\n", cid)
		case "running":
			fmt.Printf("  Status: running\n")
		case "pending_confirm":
			fmt.Printf("  Status: pending confirmation (plan created but not started)\n")
		case "not_research":
			fmt.Printf("  Status: not a deep research chat\n")
			fmt.Printf("  This chat ID does not appear to be a deep research task.\n")
		case "empty":
			fmt.Printf("  Status: waiting (no response yet)\n")
		default:
			fmt.Printf("  Status: %s\n", status.State)
		}
		return nil
	},
}
