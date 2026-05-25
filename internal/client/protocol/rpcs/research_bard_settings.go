// RPC: ESY5D — BardSettings
// Source-path: any Gemini page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[[["<setting_key>"]]]
//	  setting key array
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[1]
//
// Test fixture: testdata/research_bard_settings_basic.txt
//
// Notes:
//   - Deep research preflight queries "bard_activity_enabled".
//   - The same RPC is also used for general account settings reads.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const bardSettingsRPCID = "ESY5D"

// EncodeBardSettings returns the BardSettings payload.
func EncodeBardSettings(settingKey string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{[]any{[]any{settingKey}}})
	return bardSettingsRPCID, string(payloadBytes)
}

// DecodeBardSettings validates the BardSettings response body.
func DecodeBardSettings(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode BardSettings JSON: %w", err)
	}
	return nil
}
