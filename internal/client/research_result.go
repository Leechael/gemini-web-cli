package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GetDeepResearchResult fetches the full research result text.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	rawTurns, rawErr := c.ReadChatRaw(ctx, cid, 5)
	if rawErr == nil && len(rawTurns) > 0 {
		text, sources := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return text, sources, nil
		}
		if status := inspectResearchStatusFromRaw(rawTurns); status != nil {
			return "", nil, fmt.Errorf("research result is not ready for chat %s: state=%s", cid, status.State)
		}
		if text := latestAssistantTextFromRaw(rawTurns); text != "" && classifyResearchText(text).State == "done" {
			return text, nil, nil
		}
	}

	turns, err := c.ReadChat(ctx, cid, 5)
	if err != nil {
		return "", nil, err
	}
	var bestText string
	for _, turn := range turns {
		text := turn.AssistantResponse
		if strings.HasPrefix(text, "http://googleusercontent.com/") {
			continue
		}
		if len(text) > len(bestText) {
			bestText = text
		}
	}
	if bestText == "" {
		return "", nil, fmt.Errorf("no research result found for chat %s", cid)
	}
	return bestText, nil, nil
}
