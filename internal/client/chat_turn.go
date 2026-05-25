package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GetConversationTurn returns a single conversation turn by request id.
func (c *Client) GetConversationTurn(ctx context.Context, cid, requestID string) (*types.ChatTurn, error) {
	rpcID, payload := rpcs.EncodeGetConversationTurn(cid, requestID)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(cid))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetConversationTurn rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetConversationTurn(body)
}
