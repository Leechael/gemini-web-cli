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
	completionPending := strings.Contains(text, "研究完成后")
	done := strings.Contains(text, "我已经完成了研究") ||
		(strings.Contains(text, "研究完成") && !completionPending) ||
		strings.Contains(lower, "i have completed the research") ||
		strings.Contains(lower, "i've completed the research") ||
		strings.Contains(lower, "research is complete")
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "\ufeff"))
	if done || (len(text) > 2000 && (strings.HasPrefix(trimmed, "#") || strings.Contains(text, "\n## "))) {
		return &ResearchStatus{State: "done", TextLen: len(text)}
	}
	if completionPending || strings.Contains(lower, "deep research") || strings.Contains(text, "深度研究") || strings.Contains(lower, "deep_research") {
		return &ResearchStatus{State: "running", TextLen: len(text)}
	}
	return &ResearchStatus{State: "not_research", TextLen: len(text)}
}
