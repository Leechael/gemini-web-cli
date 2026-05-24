// RPC: MyzX6c — ListFeatureFlags
// Source-path: any Gemini virtual page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[false, [[id, enabled, value, ...], ...]]
//	        ↑
//	        feature flag tuples
//
//	feature flag tuple:
//	  [0]: numeric id
//	  [1]: enabled flag
//	  [2]: value, with RPC-specific type
//
// Test fixture: testdata/list_feature_flags_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const listFeatureFlagsRPCID = "MyzX6c"

// FeatureFlag is one decoded feature flag tuple.
type FeatureFlag struct {
	ID      string
	Enabled bool
	Value   any
}

// EncodeListFeatureFlags returns (rpcID, payload JSON string).
func EncodeListFeatureFlags() (rpcID, payload string) {
	return listFeatureFlagsRPCID, "[]"
}

// DecodeListFeatureFlags parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListFeatureFlags(body []byte) ([]FeatureFlag, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("ListFeatureFlags body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ListFeatureFlags JSON: %w", err)
	}

	flagData, ok := protocol.ArrayAt(data, 1)
	if !ok {
		return nil, fmt.Errorf("ListFeatureFlags response missing flag data")
	}

	flags := []FeatureFlag{}
	for idx := range flagData {
		item, ok := protocol.ArrayAt(flagData, idx)
		if !ok {
			continue
		}
		id := strconv.Itoa(protocol.IntAt(item, 0))
		if id == "0" {
			continue
		}
		value, _ := protocol.ValueAt(item, 2)
		flags = append(flags, FeatureFlag{
			ID:      id,
			Enabled: protocol.BoolAt(item, 1),
			Value:   value,
		})
	}
	if len(flags) == 0 {
		return nil, fmt.Errorf("ListFeatureFlags response did not contain flags")
	}
	return flags, nil
}
