package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// ReadChat reads conversation turns from a chat.
func (c *Client) ReadChat(ctx context.Context, cid string, maxTurns int) ([]types.ChatTurn, error) {
	rpcID, payload := rpcs.EncodeReadChat(cid, maxTurns)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(cid))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("read_chat rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeReadChat(body)
}

// ReadChatRaw returns the raw JSON turns of a chat.
func (c *Client) ReadChatRaw(ctx context.Context, cid string, maxTurns int) ([]json.RawMessage, error) {
	rpcID, payload := rpcs.EncodeReadChat(cid, maxTurns)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourceCid(cid))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("read_chat rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeReadChatRaw(body)
}

// LatestResponse holds the result of FetchLatestChatResponse.
type LatestResponse struct {
	Text string
	RCid string
	Rid  string
}

// FetchLatestChatResponse returns the latest assistant response for a chat.
func (c *Client) FetchLatestChatResponse(ctx context.Context, cid string) (*LatestResponse, error) {
	turns, err := c.ReadChat(ctx, cid, 10)
	if err != nil {
		return nil, err
	}
	if len(turns) == 0 {
		return nil, nil
	}
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].AssistantResponse != "" {
			return &LatestResponse{
				Text: turns[i].AssistantResponse,
				RCid: turns[i].RCid,
				Rid:  turns[i].Rid,
			}, nil
		}
	}
	return nil, nil
}
