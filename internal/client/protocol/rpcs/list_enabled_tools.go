// RPC: cYRIkd — ListEnabledTools
// Source-path: any Gemini virtual page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["en"]
//	  ↑
//	 language
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[tool, ...], [tool, ...]]
//	  ↑           ↑
//	  built-in    account-enabled service groups
//
//	tool structure:
//	  [0]: identifier array, such as ["<tool id>"] or ["workspace_tool", "<service>"]
//	  [1]: display name
//	  [2]: icon URL
//
// Test fixture: testdata/list_enabled_tools_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const listEnabledToolsRPCID = "cYRIkd"

// EnabledTool is one tool enabled for the current account.
type EnabledTool struct {
	Name    string
	IconURL string
}

// EncodeListEnabledTools returns (rpcID, payload JSON string).
func EncodeListEnabledTools() (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{"en"})
	return listEnabledToolsRPCID, string(payloadBytes)
}

// DecodeListEnabledTools parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListEnabledTools(body []byte) ([]EnabledTool, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("ListEnabledTools body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ListEnabledTools JSON: %w", err)
	}

	tools := []EnabledTool{}
	for groupIdx := range data {
		group, ok := protocol.ArrayAt(data, groupIdx)
		if !ok {
			continue
		}
		for itemIdx := range group {
			item, ok := protocol.ArrayAt(group, itemIdx)
			if !ok {
				continue
			}
			name := enabledToolName(item)
			if name == "" {
				continue
			}
			tools = append(tools, EnabledTool{
				Name:    name,
				IconURL: protocol.StringAt(item, 2),
			})
		}
	}
	if len(tools) == 0 {
		return nil, fmt.Errorf("ListEnabledTools response did not contain tools")
	}
	return tools, nil
}

func enabledToolName(item []any) string {
	ids, ok := protocol.ArrayAt(item, 0)
	if ok {
		first := protocol.StringAt(ids, 0)
		second := protocol.StringAt(ids, 1)
		if first == "workspace_tool" && second != "" {
			return second
		}
		if first != "" {
			return first
		}
	}
	return protocol.StringAt(item, 1)
}
