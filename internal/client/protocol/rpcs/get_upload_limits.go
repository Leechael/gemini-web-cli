// RPC: mhs1xe — GetUploadLimits
// Source-path: any Gemini virtual page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[[1,3]]
//	  ↑ ↑
//	  upload capability selector
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[[[500, 300, 500000]]]]
//	    ↑    ↑    ↑
//	    [0]  [1]  [2]
//
// Test fixture: testdata/get_upload_limits_basic.txt
//
// Notes:
//   - MaxTotalBytes keeps the raw server value. The exact unit is not yet confirmed.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getUploadLimitsRPCID = "mhs1xe"

// UploadLimits is the decoded upload capability limit tuple.
type UploadLimits struct {
	MaxFiles      int
	MaxFileMB     int
	MaxTotalBytes int
}

// EncodeGetUploadLimits returns (rpcID, payload JSON string).
func EncodeGetUploadLimits() (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{[]any{1, 3}})
	return getUploadLimitsRPCID, string(payloadBytes)
}

// DecodeGetUploadLimits parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetUploadLimits(body []byte) (*UploadLimits, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("GetUploadLimits body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetUploadLimits JSON: %w", err)
	}

	limitsData, ok := protocol.ArrayAt(data, 0, 0, 0)
	if !ok {
		return nil, fmt.Errorf("GetUploadLimits response missing limit data")
	}

	limits := &UploadLimits{
		MaxFiles:      protocol.IntAt(limitsData, 0),
		MaxFileMB:     protocol.IntAt(limitsData, 1),
		MaxTotalBytes: protocol.IntAt(limitsData, 2),
	}
	if limits.MaxFiles == 0 && limits.MaxFileMB == 0 && limits.MaxTotalBytes == 0 {
		return nil, fmt.Errorf("GetUploadLimits response did not contain limits")
	}
	return limits, nil
}
