// RPC: otAQ7b — GetUserStatus
// Source-path: /app
// Reject codes: HTTP 401 / 403 / envelope code 1016 (unauthenticated)
//
// Payload shape: []
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[<global flags>, <account info>, <14>: status_code int,
//	 <15>: models_list [[modelID, displayName, description, ..., selector_at_17], ...],
//	 <16>: tier_flags [int...], <17>: cap_flags [int...]]
//
// Account status code semantics: 1000 = available, 1016 = unauthenticated;
// see types.AccountStatusFromCode for the full local mapping.
//
// Test fixture: testdata/get_user_status_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const getUserStatusRPCID = "otAQ7b"

// UserStatusModel is one raw model entry decoded from GetUserStatus.
type UserStatusModel struct {
	ModelID     string
	DisplayName string
	Description string
	Selector    int
	Raw         []any
}

// UserStatusResult is the protocol-level GetUserStatus result.
type UserStatusResult struct {
	AccountStatus types.AccountStatus
	Models        []UserStatusModel
	TierFlags     []float64
	CapFlags      []float64
}

func EncodeGetUserStatus() (rpcID, payload string) {
	return getUserStatusRPCID, "[]"
}

func DecodeGetUserStatus(body []byte) (*UserStatusResult, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "[]" {
		return &UserStatusResult{AccountStatus: types.StatusAvailable}, nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetUserStatus JSON: %w", err)
	}

	statusCode := 1000
	if v := protocol.IntAt(data, 14); v != 0 {
		statusCode = v
	}
	result := &UserStatusResult{AccountStatus: types.AccountStatusFromCode(statusCode)}
	if result.AccountStatus.IsHardBlock() {
		return result, nil
	}

	if tierFlags, ok := protocol.ArrayAt(data, 16); ok {
		result.TierFlags = numericFlags(tierFlags)
	}
	if capFlags, ok := protocol.ArrayAt(data, 17); ok {
		result.CapFlags = numericFlags(capFlags)
	}
	modelsList, ok := protocol.ArrayAt(data, 15)
	if !ok {
		return result, nil
	}
	for _, modelData := range modelsList {
		md, ok := modelData.([]any)
		if !ok {
			continue
		}
		modelID := protocol.StringAt(md, 0)
		displayName := protocol.StringAt(md, 1)
		if modelID == "" || displayName == "" {
			continue
		}
		selector := protocol.IntAt(md, 17)
		result.Models = append(result.Models, UserStatusModel{
			ModelID:     modelID,
			DisplayName: displayName,
			Description: protocol.StringAt(md, 2),
			Selector:    selector,
			Raw:         md,
		})
	}
	return result, nil
}

func numericFlags(arr []any) []float64 {
	flags := make([]float64, 0, len(arr))
	for _, v := range arr {
		if f, ok := v.(float64); ok {
			flags = append(flags, f)
		}
	}
	return flags
}
