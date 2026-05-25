package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// GetChatMetadata returns metadata for a single chat.
func (c *Client) GetChatMetadata(ctx context.Context, cid string) (*rpcs.ChatMetadata, error) {
	rpcID, payload := rpcs.EncodeGetChatMetadata(cid)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(cid))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetChatMetadata rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetChatMetadata(body)
}
