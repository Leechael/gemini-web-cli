// RPC: Pty9pd — ExpandMediaPrompt
// Source-path: /app
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<user prompt>", null, null, 1]
//	 ↑                         ↑
//	 prompt                    observed fixed expansion flag
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[["variation 1"], ["variation 2"], ["variation 3"]]]
//	  ↑
//	  data[0]
//
// Test fixtures:
//   - testdata/expand_media_prompt_music_basic.txt
//   - testdata/expand_media_prompt_image_basic.txt
//
// Notes:
//   - This is a media prompt expansion helper used across image, music, and video flows.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const expandMediaPromptRPCID = "Pty9pd"

// EncodeExpandMediaPrompt returns (rpcID, payload JSON string).
func EncodeExpandMediaPrompt(prompt string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{prompt, nil, nil, 1})
	return expandMediaPromptRPCID, string(payloadBytes)
}

// DecodeExpandMediaPrompt parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeExpandMediaPrompt(body []byte) ([]string, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ExpandMediaPrompt JSON: %w", err)
	}

	items, ok := protocol.ArrayAt(data, 0)
	if !ok {
		return nil, nil
	}
	variations := make([]string, 0, len(items))
	for idx := range items {
		if text := protocol.StringAt(items, idx, 0); text != "" {
			variations = append(variations, text)
		}
	}
	return variations, nil
}
