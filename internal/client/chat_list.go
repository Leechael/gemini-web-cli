package client

import (
	"context"
	"fmt"

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

	for _, sourcePath := range sourcePaths {
		for _, p := range payloads {
			rpcID, payload := rpcs.EncodeListChatsRaw(p)
			body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(sourcePath))
			if err != nil {
				fmt.Fprintf(logWriter, "list_chats attempt failed (path=%s): %v\n", sourcePath, err)
				continue
			}
			if rejectCode != 0 {
				fmt.Fprintf(logWriter, "list_chats attempt rejected (path=%s code=%d)\n", sourcePath, rejectCode)
				continue
			}
			items, nextCursor, err := rpcs.DecodeListChats(body)
			if err != nil {
				fmt.Fprintf(logWriter, "list_chats decode failed (path=%s): %v\n", sourcePath, err)
				continue
			}
			if len(items) > 0 {
				return items, nextCursor, nil
			}
		}
	}

	return nil, "", nil
}
