// RPC: k81mDb — MarkChatRead
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<chat_id>"]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[]
//
// Test fixture: testdata/mark_chat_read_basic.txt
//
// Notes:
//   - Empty bodies are accepted because this write RPC only needs reject-code validation.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const markChatReadRPCID = "k81mDb"

// EncodeMarkChatRead returns the MarkChatRead payload.
func EncodeMarkChatRead(cid string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{cid})
	return markChatReadRPCID, string(payloadBytes)
}

// DecodeMarkChatRead validates the MarkChatRead response body.
func DecodeMarkChatRead(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode MarkChatRead JSON: %w", err)
	}
	if len(data) != 0 {
		return fmt.Errorf("MarkChatRead response did not match expected empty array shape")
	}
	return nil
}
