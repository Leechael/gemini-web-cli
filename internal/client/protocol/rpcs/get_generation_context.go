// RPC: kwDCne — GetGenerationContext
// Source-path: any Gemini chat page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<uuid>"]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	["<chat_id>", "<prompt>", "<request_id>"]
//
// Test fixture: testdata/get_generation_context_basic.txt
//
// Notes:
//   - Empty bodies are errors because this RPC fetches one exact generation context.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getGenerationContextRPCID = "kwDCne"

// GenerationContext is the decoded context for a generation request.
type GenerationContext struct {
	ChatID    string `json:"chatId"`
	Prompt    string `json:"prompt"`
	RequestID string `json:"requestId"`
}

// EncodeGetGenerationContext returns the GetGenerationContext payload.
func EncodeGetGenerationContext(uuid string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{uuid})
	return getGenerationContextRPCID, string(payloadBytes)
}

// DecodeGetGenerationContext parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetGenerationContext(body []byte) (*GenerationContext, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, fmt.Errorf("GetGenerationContext body is empty")
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetGenerationContext JSON: %w", err)
	}
	ctx := &GenerationContext{
		ChatID:    protocol.StringAt(data, 0),
		Prompt:    protocol.StringAt(data, 1),
		RequestID: protocol.StringAt(data, 2),
	}
	if !strings.HasPrefix(ctx.ChatID, "c_") || ctx.Prompt == "" || !strings.HasPrefix(ctx.RequestID, "r_") {
		return nil, fmt.Errorf("GetGenerationContext response did not match expected context shape")
	}
	return ctx, nil
}
