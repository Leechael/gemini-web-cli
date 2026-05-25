// RPC: PCck7e — DeepResearchAck
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<request_id>"]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[1]
//
// Test fixture: testdata/research_deep_research_ack_basic.txt
//
// Notes:
//   - Confirms a plan-ready deep research state.
//   - Errors are non-fatal to the research flow.
//   - Empty bodies are accepted because preflight responses are best-effort acknowledgements.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const deepResearchAckRPCID = "PCck7e"

// EncodeDeepResearchAck returns the DeepResearchAck payload.
func EncodeDeepResearchAck(requestID string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{requestID})
	return deepResearchAckRPCID, string(payloadBytes)
}

// DecodeDeepResearchAck validates the DeepResearchAck response body.
func DecodeDeepResearchAck(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode DeepResearchAck JSON: %w", err)
	}
	return nil
}
