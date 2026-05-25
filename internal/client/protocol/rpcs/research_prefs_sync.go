// RPC: L5adhe — PrefsSync
// Source-path: any Gemini page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[<sparse_state_array>, [["<pref_namespace>"]]]
//	  sparse state array   preference namespace
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[1]
//
// Test fixture: testdata/research_prefs_sync_basic.txt
//
// Notes:
//   - Feature state uses a 193-slot array with feature flags in slot 192.
//   - Popup state uses an 87-slot array with the visit count in slot 86.
//   - Empty bodies are accepted because preflight responses are best-effort acknowledgements.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const prefsSyncRPCID = "L5adhe"

// PrefsSyncFeatureState contains the feature flags stored in the sparse feature-state payload.
type PrefsSyncFeatureState struct {
	FeatureFlags []string
}

// PrefsSyncPopupState contains the popup visit count stored in the sparse popup-state payload.
type PrefsSyncPopupState struct {
	Visits int
}

// EncodePrefsSyncFeatureState returns the feature-state PrefsSync payload.
func EncodePrefsSyncFeatureState(s PrefsSyncFeatureState) (rpcID, payload string) {
	featureState := make([]any, 193)
	flags := make([]any, len(s.FeatureFlags))
	for i, flag := range s.FeatureFlags {
		flags[i] = flag
	}
	featureState[192] = []any{flags}
	payloadBytes, _ := json.Marshal([]any{featureState, []any{[]any{"tool_menu_soft_badge_disabled_ids"}}})
	return prefsSyncRPCID, string(payloadBytes)
}

// EncodePrefsSyncPopupState returns the popup-state PrefsSync payload.
func EncodePrefsSyncPopupState(s PrefsSyncPopupState) (rpcID, payload string) {
	popupState := make([]any, 87)
	popupState[86] = s.Visits
	payloadBytes, _ := json.Marshal([]any{popupState, []any{[]any{"popup_zs_visits_cooldown"}}})
	return prefsSyncRPCID, string(payloadBytes)
}

// DecodePrefsSync validates the PrefsSync response body.
func DecodePrefsSync(body []byte) error {
	if strings.TrimSpace(string(body)) == "" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode PrefsSync JSON: %w", err)
	}
	return nil
}
