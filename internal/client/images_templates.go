package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// ListImageTemplates returns available image generation style templates.
func (c *Client) ListImageTemplates(ctx context.Context) ([]rpcs.ImageTemplate, error) {
	rpcID, payload := rpcs.EncodeListImageTemplates()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/images"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ListImageTemplates rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListImageTemplates(body)
}
