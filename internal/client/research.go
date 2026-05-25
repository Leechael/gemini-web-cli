package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const (
	rpcBardSettings          = "ESY5D"
	rpcPrefsSync             = "L5adhe"
	rpcDeepResearchBootstrap = "ku4Jyf"
	rpcDeepResearchModelSt   = "qpEbW"
	rpcDeepResearchCaps      = "aPya6c"
	rpcDeepResearchAck       = "PCck7e"
)

// ResearchStatus describes the state of a deep research task.
type ResearchStatus struct {
	State   string // "done", "running", "pending_confirm", "not_research", "empty"
	TextLen int
}

// CheckDeepResearch checks the status of a deep research task.
// It distinguishes between: completed, running, pending confirmation,
// not a research chat, and empty (no response).
func (c *Client) CheckDeepResearch(ctx context.Context, cid string) (*ResearchStatus, error) {
	// 1. Check raw turns for structured deep research result at cand[30][0][4]
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, _ := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return &ResearchStatus{State: "done", TextLen: len(text)}, nil
		}

		// Check for deep research markers in raw turns:
		// - confirmation page: "deep_research_confirmation_content"
		// - running indicator: "immersive_entry_chip"
		// - dict key "70" with state value (2=plan_ready, 3=running)
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

	// 2. Fall back to text-based detection
	latest, err := c.FetchLatestChatResponse(ctx, cid)
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.Text == "" {
		return &ResearchStatus{State: "empty"}, nil
	}
	text := latest.Text

	// Check for completion phrases
	lower := strings.ToLower(text)
	done := strings.Contains(text, "我已经完成了研究") ||
		strings.Contains(text, "研究完成") ||
		strings.Contains(lower, "i have completed the research") ||
		strings.Contains(lower, "i've completed the research") ||
		strings.Contains(lower, "research is complete")

	if done || (len(text) > 2000 && (strings.HasPrefix(strings.TrimLeft(text, " \t\n"), "#") || strings.Contains(text, "\n## "))) {
		return &ResearchStatus{State: "done", TextLen: len(text)}, nil
	}

	// Check if this is actually a deep research chat at all
	if strings.Contains(text, "Deep Research") || strings.Contains(text, "深度研究") || strings.Contains(text, "deep_research") {
		return &ResearchStatus{State: "running", TextLen: len(text)}, nil
	}

	// Doesn't look like a research chat
	return &ResearchStatus{State: "not_research", TextLen: len(text)}, nil
}

// GetDeepResearchResult fetches the full research result text.
// Deep research reports are stored at cand[30][0][4] in raw turn data,
// not in the normal text field.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	// Try raw turns first for structured deep research data
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, sources := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return text, sources, nil
		}
	}

	// Fallback to regular read
	turns, err := c.ReadChat(ctx, cid, 5)
	if err != nil {
		return "", nil, err
	}
	var bestText string
	for _, turn := range turns {
		resp := turn.AssistantResponse
		// Skip card URLs
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

// extractResearchResultFromRaw extracts the deep research report from raw turn data.
// The report lives at cand[30][0][4] in each turn.
func extractResearchResultFromRaw(rawTurns []json.RawMessage) (string, map[int]types.GroundingSource) {
	for _, rawTurn := range rawTurns {
		var turn []any
		if err := json.Unmarshal(rawTurn, &turn); err != nil {
			continue
		}

		// Navigate to candidate: turn[3][0][0]
		cand, _ := protocol.ArrayAt(turn, 3, 0, 0)
		if cand == nil {
			continue
		}

		// Deep research data at cand[30][0]
		drData, _ := protocol.ArrayAt(cand, 30, 0)
		if drData == nil || len(drData) < 5 {
			continue
		}

		// Report text at [30][0][4]
		candidateText, ok := drData[4].(string)
		if !ok || len(candidateText) < 200 {
			continue
		}

		// Extract sources from [30][0][5][0]["44"]
		sources := extractResearchSources(drData)
		return candidateText, sources
	}
	return "", nil
}

func extractResearchSources(drData []any) map[int]types.GroundingSource {
	sources := make(map[int]types.GroundingSource)
	if len(drData) < 6 {
		return sources
	}

	citationsContainer, _ := protocol.ArrayAt(drData, 5, 0)
	if citationsContainer == nil {
		return sources
	}

	// Look for map with key "44"
	// The structure is: drData[5][0] is a dict-like structure
	// In JSON it's a map; navigate to "44" key
	if len(drData) > 5 {
		containerArr, _ := protocol.ArrayAt(drData, 5, 0)
		if containerArr == nil {
			return sources
		}
		// containerArr might be a JSON object — try to get it as map
		var rawContainer json.RawMessage
		b, err := json.Marshal(drData[5])
		if err != nil {
			return sources
		}
		if err := json.Unmarshal(b, &rawContainer); err != nil {
			return sources
		}

		// Try parsing as [{...}] where inner is {"44": [...]}
		var outerArr []json.RawMessage
		if err := json.Unmarshal(rawContainer, &outerArr); err != nil || len(outerArr) == 0 {
			return sources
		}
		var innerMap map[string]json.RawMessage
		if err := json.Unmarshal(outerArr[0], &innerMap); err != nil {
			return sources
		}
		citationGroupsRaw, ok := innerMap["44"]
		if !ok {
			return sources
		}
		var citationGroups []any
		if err := json.Unmarshal(citationGroupsRaw, &citationGroups); err != nil {
			return sources
		}

		for _, group := range citationGroups {
			groupArr, ok := group.([]any)
			if !ok || len(groupArr) < 2 {
				continue
			}
			for _, sourceEntries := range groupArr[1:] {
				seArr, ok := sourceEntries.([]any)
				if !ok {
					continue
				}
				for _, item := range seArr {
					itemArr, ok := item.([]any)
					if !ok || len(itemArr) < 4 {
						continue
					}
					inner, _ := protocol.ArrayAt(itemArr, 3)
					if inner == nil || len(inner) < 2 {
						continue
					}
					detail, ok := inner[0].([]any)
					if !ok || len(detail) < 3 {
						continue
					}
					refNum, ok := inner[1].(float64)
					if !ok {
						continue
					}
					urlStr, _ := detail[1].(string)
					title, _ := detail[2].(string)
					if urlStr != "" && strings.HasPrefix(urlStr, "http") {
						sources[int(refNum)] = types.GroundingSource{
							URL:   urlStr,
							Title: title,
						}
					}
				}
			}
		}
	}
	return sources
}

func extractPlanTitle(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
			return line
		}
	}
	return ""
}

func extractPlanSteps(text string) []string {
	var steps []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			steps = append(steps, line[2:])
		}
	}
	return steps
}
