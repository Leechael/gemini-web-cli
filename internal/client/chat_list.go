package client

import (
	"context"
	"fmt"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// ListChats returns a list of chats with optional pagination.
func (c *Client) ListChats(ctx context.Context, cursor string) ([]types.ChatItem, string, error) {
	var payloads []rpcs.ListChatsPayload
	if cursor != "" {
		payloads = []rpcs.ListChatsPayload{{PageSize: 20, Cursor: cursor, Flag1: 0, Flag2: 1}}
	} else {
		payloads = []rpcs.ListChatsPayload{
			{PageSize: 13, Flag1: 1, Flag2: 1},
			{PageSize: 13, Flag1: 0, Flag2: 1},
			{PageSize: 13, Flag1: 0, Flag2: 2},
		}
	}

	sourcePaths := []string{c.appPath()}
	if c.appPath() != "/app" {
		sourcePaths = append(sourcePaths, "/app")
	}

	var lastErr error
	validResponse := false
	lastCursor := ""
	for _, sourcePath := range sourcePaths {
		for _, p := range payloads {
			rpcID, payload := rpcs.EncodeListChatsRaw(p)
			body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(sourcePath))
			if err != nil {
				lastErr = err
				fmt.Fprintf(logWriter, "list_chats attempt failed (path=%s): %v\n", sourcePath, err)
				continue
			}
			if rejectCode != 0 {
				lastErr = fmt.Errorf("list_chats rejected with code=%d", rejectCode)
				fmt.Fprintf(logWriter, "list_chats attempt rejected (path=%s code=%d)\n", sourcePath, rejectCode)
				continue
			}
			listItems, nextCursor, err := rpcs.DecodeListChats(body)
			if err != nil {
				lastErr = err
				fmt.Fprintf(logWriter, "list_chats decode failed (path=%s): %v\n", sourcePath, err)
				continue
			}
			validResponse = true
			if nextCursor != "" {
				lastCursor = nextCursor
			}
			items := chatItemsFromRPC(listItems)
			if len(items) > 0 {
				return items, nextCursor, nil
			}
		}
	}

	if !validResponse && lastErr != nil {
		return nil, "", lastErr
	}
	return nil, lastCursor, nil
}

func chatItemsFromRPC(items []rpcs.ChatListItem) []types.ChatItem {
	out := make([]types.ChatItem, 0, len(items))
	for _, item := range items {
		chatItem := types.ChatItem{
			Cid:   item.Cid,
			Title: item.Title,
		}
		if item.UpdatedAtUnix != 0 {
			chatItem.UpdatedAt = time.Unix(item.UpdatedAtUnix, 0).UTC().Format("2006-01-02T15:04")
		}
		out = append(out, chatItem)
	}
	return out
}
