package client

import (
	"context"
	"strings"
)

// ResearchStatus describes the state of a deep research task.
type ResearchStatus struct {
	State   string // "done", "running", "pending_confirm", "not_research", "empty"
	TextLen int
}

// CheckDeepResearch checks the status of a deep research task.
func (c *Client) CheckDeepResearch(ctx context.Context, cid string) (*ResearchStatus, error) {
	rawTurns, rawErr := c.ReadChatRaw(ctx, cid, 5)
	if rawErr == nil {
		if len(rawTurns) == 0 {
			return &ResearchStatus{State: "empty"}, nil
		}
		if status := inspectResearchStatusFromRaw(rawTurns); status != nil {
			return status, nil
		}
		if text := latestAssistantTextFromRaw(rawTurns); text != "" {
			return classifyResearchText(text), nil
		}
		return &ResearchStatus{State: "not_research"}, nil
	}

	latest, err := c.FetchLatestChatResponse(ctx, cid)
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.Text == "" {
		return &ResearchStatus{State: "empty"}, nil
	}
	return classifyResearchText(latest.Text), nil
}

func classifyResearchText(text string) *ResearchStatus {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "deep research") || strings.Contains(text, "深度研究") || strings.Contains(lower, "deep_research") {
		return &ResearchStatus{State: "running", TextLen: len(text)}
	}
	return &ResearchStatus{State: "not_research", TextLen: len(text)}
}
