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
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, _ := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return &ResearchStatus{State: "done", TextLen: len(text)}, nil
		}

		hasConfirmation := false
		hasRunning := false
		for _, raw := range rawTurns {
			s := string(raw)
			if strings.Contains(s, "deep_research_confirmation_content") {
				hasConfirmation = true
			}
			if strings.Contains(s, "immersive_entry_chip") {
				hasRunning = true
			}
			if strings.Contains(s, `"70":3`) {
				hasRunning = true
			}
		}

		if hasRunning {
			return &ResearchStatus{State: "running"}, nil
		}
		if hasConfirmation {
			return &ResearchStatus{State: "pending_confirm"}, nil
		}
	}

	latest, err := c.FetchLatestChatResponse(ctx, cid)
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.Text == "" {
		return &ResearchStatus{State: "empty"}, nil
	}
	text := latest.Text

	lower := strings.ToLower(text)
	done := strings.Contains(text, "我已经完成了研究") ||
		strings.Contains(text, "研究完成") ||
		strings.Contains(lower, "i have completed the research") ||
		strings.Contains(lower, "i've completed the research") ||
		strings.Contains(lower, "research is complete")

	if done || (len(text) > 2000 && (strings.HasPrefix(strings.TrimLeft(text, " \t\n"), "#") || strings.Contains(text, "\n## "))) {
		return &ResearchStatus{State: "done", TextLen: len(text)}, nil
	}

	if strings.Contains(text, "Deep Research") || strings.Contains(text, "深度研究") || strings.Contains(text, "deep_research") {
		return &ResearchStatus{State: "running", TextLen: len(text)}, nil
	}

	return &ResearchStatus{State: "not_research", TextLen: len(text)}, nil
}
