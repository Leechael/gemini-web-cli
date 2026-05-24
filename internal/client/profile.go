package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// GetUserProfile returns the current account's user profile.
func (c *Client) GetUserProfile(ctx context.Context) (*rpcs.UserProfile, error) {
	rpcID, payload := rpcs.EncodeGetUserProfile()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetUserProfile rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetUserProfile(body)
}
