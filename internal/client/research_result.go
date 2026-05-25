package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GetDeepResearchResult fetches the full research result text.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, sources := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return text, sources, nil
		}
	}

	turns, err := c.ReadChat(ctx, cid, 5)
	if err != nil {
		return "", nil, err
	}
	var bestText string
	for _, turn := range turns {
		resp := turn.AssistantResponse
		if strings.HasPrefix(resp, "http://googleusercontent.com/") {
			continue
		}
		if len(resp) > len(bestText) {
			bestText = resp
		}
	}
	if bestText == "" {
		return "", nil, fmt.Errorf("no research result found for chat %s — research may still be running", cid)
	}
	return bestText, nil, nil
}
