package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GetDeepResearchResult fetches the full research result text.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	rawTurns, rawErr := c.ReadChatRaw(ctx, cid, 5)
	if rawErr == nil && len(rawTurns) > 0 {
		for _, rawTurn := range rawTurns {
			state := inspectResearchRawTurn(rawTurn)
			switch state.state {
			case "done":
				return state.text, state.sources, nil
			case "running", "pending_confirm":
				return "", nil, fmt.Errorf("research result is not ready for chat %s: state=%s", cid, state.state)
			}
			if text := assistantTextFromRawTurn(rawTurn); text != "" && classifyResearchText(text).State == "done" {
				return text, nil, nil
			}
		}
	}

	turns, err := c.ReadChat(ctx, cid, 5)
	if err != nil {
		return "", nil, researchFallbackError(rawErr, err)
	}
	var latestText string
	for i := len(turns) - 1; i >= 0; i-- {
		text := turns[i].AssistantResponse
		if text == "" || strings.HasPrefix(text, "http://googleusercontent.com/") {
			continue
		}
		latestText = text
		break
	}
	if latestText == "" {
		return "", nil, researchFallbackError(rawErr, fmt.Errorf("no research result found for chat %s", cid))
	}
	status := classifyResearchText(latestText)
	if status.State == "running" || status.State == "pending_confirm" {
		return "", nil, fmt.Errorf("research result is not ready for chat %s: state=%s", cid, status.State)
	}
	if status.State != "done" {
		return "", nil, researchFallbackError(rawErr, fmt.Errorf("no research result found for chat %s", cid))
	}
	return latestText, nil, nil
}

func researchFallbackError(rawErr, fallbackErr error) error {
	if rawErr == nil {
		return fallbackErr
	}
	return errors.Join(
		fmt.Errorf("read raw chat: %w", rawErr),
		fmt.Errorf("decoded fallback: %w", fallbackErr),
	)
}
