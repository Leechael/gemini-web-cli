// RPC: ku4Jyf — DeepResearchBootstrap
// Source-path: any Gemini page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape (verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	["<lang>", null, null, null, 14, null, null, [32], null, []]
//	  language                                capability ids     flags
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[1]
//
// Test fixture: testdata/research_deep_research_bootstrap_basic.txt
//
// Notes:
//   - Current callers ignore the metadata response; decode only validates JSON.
//   - Empty bodies are accepted because preflight responses are best-effort acknowledgements.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const deepResearchBootstrapRPCID = "ku4Jyf"

// EncodeDeepResearchBootstrap returns the DeepResearchBootstrap payload.
func EncodeDeepResearchBootstrap(lang string) (rpcID, payload string) {
	if lang == "" {
		lang = "en"
	}
	payloadBytes, _ := json.Marshal([]any{lang, nil, nil, nil, 14, nil, nil, []any{32}, nil, []any{}})
	return deepResearchBootstrapRPCID, string(payloadBytes)
}

// DecodeDeepResearchBootstrap validates the DeepResearchBootstrap response body.
func DecodeDeepResearchBootstrap(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode DeepResearchBootstrap JSON: %w", err)
	}
	return nil
}
