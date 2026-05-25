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
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
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
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetGenerationContext JSON: %w", err)
	}
	ctx := &GenerationContext{}
	fillGenerationContext(ctx, data)
	if ctx.ChatID == "" && ctx.Prompt == "" && ctx.RequestID == "" {
		return nil, fmt.Errorf("GetGenerationContext response did not contain context fields")
	}
	return ctx, nil
}

func fillGenerationContext(ctx *GenerationContext, v any) {
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			fillGenerationContext(ctx, item)
		}
	case string:
		switch {
		case strings.HasPrefix(x, "c_") && ctx.ChatID == "":
			ctx.ChatID = x
		case strings.HasPrefix(x, "r_") && ctx.RequestID == "":
			ctx.RequestID = x
		case x != "" && ctx.Prompt == "" && !strings.HasPrefix(x, "rcid_"):
			ctx.Prompt = x
		}
	}
}
