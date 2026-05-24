// RPC: K4WWud — GetUserLocation
// Source-path: any Gemini virtual page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[[0],["en"]]
//	  ↑   ↑
//	 mode language
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[["<region>", "<source>", false, null, "<map tile URL>"]]
//	  ↑           ↑          ↑              ↑
//	  [0]         [1]        [2]            [4]
//
// Test fixture: testdata/get_user_location_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getUserLocationRPCID = "K4WWud"

// UserLocation is the decoded account location signal.
type UserLocation struct {
	Region     string
	Source     string
	IsPrecise  bool
	MapTileURL string
}

// EncodeGetUserLocation returns (rpcID, payload JSON string).
func EncodeGetUserLocation() (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{[]any{0}, []any{"en"}})
	return getUserLocationRPCID, string(payloadBytes)
}

// DecodeGetUserLocation parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetUserLocation(body []byte) (*UserLocation, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("GetUserLocation body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetUserLocation JSON: %w", err)
	}

	locationData, ok := protocol.ArrayAt(data, 0)
	if !ok {
		return nil, fmt.Errorf("GetUserLocation response missing location data")
	}

	location := &UserLocation{
		Region:     protocol.StringAt(locationData, 0),
		Source:     protocol.StringAt(locationData, 1),
		IsPrecise:  protocol.BoolAt(locationData, 2),
		MapTileURL: protocol.StringAt(locationData, 4),
	}
	if location.Region == "" && location.Source == "" && location.MapTileURL == "" {
		return nil, fmt.Errorf("GetUserLocation response did not contain location fields")
	}
	return location, nil
}
