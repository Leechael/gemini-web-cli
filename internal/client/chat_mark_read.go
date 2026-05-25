package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// MarkChatRead marks a chat as read.
func (c *Client) MarkChatRead(ctx context.Context, cid string) error {
	rpcID, payload := rpcs.EncodeMarkChatRead(cid)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(cid))
	if err != nil {
		return err
	}
	if rejectCode != 0 {
		return fmt.Errorf("MarkChatRead rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeMarkChatRead(body)
}
