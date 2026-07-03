package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GetDeepResearchResult fetches the full research result text.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err != nil {
		return "", nil, err
	}
	if len(rawTurns) > 0 {
		text, sources := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return text, sources, nil
		}
		if status := inspectResearchStatusFromRaw(rawTurns); status != nil {
			return "", nil, fmt.Errorf("research result is not ready for chat %s: state=%s", cid, status.State)
		}
	}

	return "", nil, fmt.Errorf("no research result found for chat %s", cid)
}
