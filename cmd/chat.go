package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	chatMetaJSON bool
	chatTurnJSON bool
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat metadata and inspection utilities",
	Long:  "Subcommands for fetching chat metadata, single conversation turns, and related chat inspection data.",
}

var chatMetaCmd = &cobra.Command{
	Use:   "meta <chatId>",
	Short: "Show metadata for a single chat",
	Args:  cobra.ExactArgs(1),
	RunE:  runChatMeta,
}

var chatTurnCmd = &cobra.Command{
	Use:   "turn <chatId> <requestId>",
	Short: "Fetch a single conversation turn by request id",
	Args:  cobra.ExactArgs(2),
	RunE:  runChatTurn,
}

func runChatMeta(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	meta, err := c.GetChatMetadata(ctx, args[0])
	if err != nil {
		return err
	}
	if chatMetaJSON {
		out, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	updated := ""
	if meta.UpdatedAt != 0 {
		updated = time.Unix(meta.UpdatedAt, 0).UTC().Format("2006-01-02 15:04:05 UTC")
	}
	fmt.Printf("Cid:        %s\n", meta.Cid)
	fmt.Printf("Title:      %s\n", meta.Title)
	fmt.Printf("Updated:    %s\n", updated)
	fmt.Printf("Unread:     %t\n", meta.Unread)
	return nil
}

func runChatTurn(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	turn, err := c.GetConversationTurn(ctx, args[0], args[1])
	if err != nil {
		return err
	}
	if chatTurnJSON {
		out, err := json.Marshal(turn)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("User:       %s\n", turn.UserPrompt)
	fmt.Printf("Assistant:  %s\n", turn.AssistantResponse)
	if turn.RCid != "" {
		fmt.Printf("RCid:       %s\n", turn.RCid)
	}
	if turn.Rid != "" {
		fmt.Printf("Rid:        %s\n", turn.Rid)
	}
	return nil
}

func init() {
	chatMetaCmd.Flags().BoolVar(&chatMetaJSON, "json", false, "Output JSON")
	chatTurnCmd.Flags().BoolVar(&chatTurnJSON, "json", false, "Output JSON")
	chatCmd.AddCommand(chatMetaCmd, chatTurnCmd)
}
