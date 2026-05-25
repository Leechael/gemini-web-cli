package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// ExpandMediaPrompt returns alternate descriptions for a media prompt.
func (c *Client) ExpandMediaPrompt(ctx context.Context, prompt string) ([]string, error) {
	rpcID, payload := rpcs.EncodeExpandMediaPrompt(prompt)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/app"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ExpandMediaPrompt rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeExpandMediaPrompt(body)
}
