package client

import (
	"encoding/json"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// extractResearchResultFromRaw extracts the deep research report from raw turn data.
func extractResearchResultFromRaw(rawTurns []json.RawMessage) (string, map[int]types.GroundingSource) {
	for _, rawTurn := range rawTurns {
		var turn []any
		if err := json.Unmarshal(rawTurn, &turn); err != nil {
			continue
		}

		cand, _ := protocol.ArrayAt(turn, 3, 0, 0)
		if cand == nil {
			continue
		}

		drData, _ := protocol.ArrayAt(cand, 30, 0)
		if drData == nil || len(drData) < 5 {
			continue
		}

		candidateText, ok := drData[4].(string)
		if !ok || len(candidateText) < 200 {
			continue
		}

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

	if len(drData) > 5 {
		containerArr, _ := protocol.ArrayAt(drData, 5, 0)
		if containerArr == nil {
			return sources
		}
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
