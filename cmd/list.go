package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var listCursor string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List chat history",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		items, cursor, err := c.ListChats(ctx, listCursor)
		if err != nil {
			return err
		}

		if len(items) == 0 {
			fmt.Println("No chats found.")
			return nil
		}

		// Calculate column widths
		idW := len("ID")
		titleW := len("Title")
		for _, it := range items {
			if len(it.Cid) > idW {
				idW = len(it.Cid)
			}
			title := it.Title
			if len(title) > 50 {
				title = title[:50]
			}
			if len(title) > titleW {
				titleW = len(title)
			}
		}

		// Print header
		header := fmt.Sprintf("%-*s  %-*s  %s", idW, "ID", titleW, "Title", "Updated")
		fmt.Println(header)
		for i := 0; i < len(header); i++ {
			fmt.Print("-")
		}
		fmt.Println()

		// Print rows
		for _, it := range items {
			title := it.Title
			if len(title) > 50 {
				title = title[:50]
			}
			updated := it.UpdatedAt
			if len(updated) > 16 {
				updated = updated[:16]
			}
			fmt.Printf("%-*s  %-*s  %s\n", idW, it.Cid, titleW, title, updated)
		}

		if cursor != "" {
			fmt.Printf("\n(next page: --cursor %s)\n", cursor)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listCursor, "cursor", "", "Pagination cursor")
}
