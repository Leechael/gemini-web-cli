package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// GetGenerationContext returns the request context for a generation id.
func (c *Client) GetGenerationContext(ctx context.Context, uuid string) (*rpcs.GenerationContext, error) {
	rpcID, payload := rpcs.EncodeGetGenerationContext(uuid)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetGenerationContext rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetGenerationContext(body)
}
