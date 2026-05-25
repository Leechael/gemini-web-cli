package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

var defaultDiscoverSurfaceFilters = []int{390, 391, 392, 393, 394, 395, 396, 418, 417, 415, 416, 414, 400, 422, 425, 431, 432}

// GetDiscoverSurface returns image Discover surface cards.
func (c *Client) GetDiscoverSurface(ctx context.Context) ([]rpcs.DiscoverCard, error) {
	rpcID, payload := rpcs.EncodeGetDiscoverSurface(c.language, defaultDiscoverSurfaceFilters)
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/images"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetDiscoverSurface rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetDiscoverSurface(body)
}
