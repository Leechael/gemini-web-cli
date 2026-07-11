package client

import (
	"encoding/json"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

type researchRawState struct {
	state   string
	text    string
	sources map[int]types.GroundingSource
}

// inspectResearchStateFromRaw returns the newest conclusive research state.
// Newer running or confirmation states prevent older completed reports from
// being returned, while plain-text completed reports are also recognized.
func inspectResearchStateFromRaw(rawTurns []json.RawMessage) researchRawState {
	for _, rawTurn := range rawTurns {
		state := inspectResearchRawTurn(rawTurn)
		if state.state != "" {
			return state
		}
		if text := assistantTextFromRawTurn(rawTurn); text != "" && classifyResearchText(text).State == "done" {
			return researchRawState{state: "done", text: text}
		}
	}
	return researchRawState{}
}

func inspectResearchRawTurn(rawTurn json.RawMessage) researchRawState {
	var turn []any
	if err := json.Unmarshal(rawTurn, &turn); err != nil {
		return researchRawState{}
	}

	cand, _ := protocol.ArrayAt(turn, 3, 0, 0)
	if cand != nil {
		if text, sources := extractResearchResultFromCandidate(cand); text != "" {
			return researchRawState{state: "done", text: text, sources: sources}
		}
	}

	markers := researchMarkers{}
	collectResearchMarkers(turn, &markers)
	if markers.running {
		return researchRawState{state: "running"}
	}
	if markers.pendingConfirm {
		return researchRawState{state: "pending_confirm"}
	}
	return researchRawState{}
}

func latestAssistantTextFromRaw(rawTurns []json.RawMessage) string {
	for _, rawTurn := range rawTurns {
		if text := assistantTextFromRawTurn(rawTurn); text != "" {
			return text
		}
	}
	return ""
}

func assistantTextFromRawTurn(rawTurn json.RawMessage) string {
	var turn []any
	if err := json.Unmarshal(rawTurn, &turn); err != nil {
		return ""
	}
	cand, _ := protocol.ArrayAt(turn, 3, 0, 0)
	if cand == nil {
		return ""
	}
	text := protocol.FirstString(protocol.StringAt(cand, 1, 0), protocol.StringAt(cand, 22, 0))
	return protocol.StripCardURLLines(text)
}

func extractResearchResultFromCandidate(cand []any) (string, map[int]types.GroundingSource) {
	drData, _ := protocol.ArrayAt(cand, 30, 0)
	if drData == nil || len(drData) < 5 {
		return "", nil
	}

	candidateText, ok := drData[4].(string)
	if !ok || len(candidateText) < 200 {
		return "", nil
	}

	sources := extractResearchSources(drData)
	return candidateText, sources
}

type researchMarkers struct {
	running        bool
	pendingConfirm bool
}

func collectResearchMarkers(value any, markers *researchMarkers) {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			collectResearchMarkers(item, markers)
		}
	case map[string]any:
		if rawState, ok := v["70"]; ok {
			switch researchStateNumber(rawState) {
			case 3:
				markers.running = true
			case 2:
				markers.pendingConfirm = true
			}
		}
		for _, key := range []string{"56", "57"} {
			if _, ok := v[key]; ok {
				markers.pendingConfirm = true
			}
		}
		for _, item := range v {
			collectResearchMarkers(item, markers)
		}
	case string:
		if strings.Contains(v, "immersive_entry_chip") {
			markers.running = true
		}
		if strings.Contains(v, "deep_research_confirmation_content") {
			markers.pendingConfirm = true
		}
	}
}

func researchStateNumber(value any) int {
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v)
		}
	case int:
		return v
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	}
	return 0
}

func extractResearchSources(drData []any) map[int]types.GroundingSource {
	sources := make(map[int]types.GroundingSource)
	if len(drData) < 6 {
		return sources
	}

	if len(drData) > 5 {
		var rawContainer json.RawMessage
		b, err := json.Marshal(drData[5])
		if err != nil {
			return sources
		}
		if err := json.Unmarshal(b, &rawContainer); err != nil {
			return sources
		}

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
