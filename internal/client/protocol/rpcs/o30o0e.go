// RPC: o30O0e — GetUserProfile
// Source-path: /app （或任何页面均可）
// Reject codes: none observed in HAR 20260524
//
// Payload shape:
//
//	[["me"], [[["person.photo","person.name","person.email"]], null, [1,7]]]
//	↑       ↑                                                  ↑    ↑
//	subject field-mask list                                   ?    pagination?
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[["me", 1, ["<userId>", [null,...,[null,["me"]]], [profile_arr]]]]]
//	              ↑           ↑                       ↑
//	              user id    metadata               profile data
//
//	profile_arr structure:
//	  [0]:  [true, 0, true, null, ..., "<userId>", ..., [unix_seconds, nanos]]
//	  [1]:  "<display name>"
//	  [3]:  "<email>"  (may be absent if scope not granted)
//	  ...   photo URL nesting: needs further reverse-engineering from HAR sample
//
// Notes:
//   - "me" can also be a specific account id; HAR only uses "me"
//   - Field [1,7] in payload is unclear; HAR shows it constant; pass-through
//   - Photo URL location not yet decoded — leave PhotoURL empty if not found,
//     don't error; this is acceptable for first iteration
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const getUserProfileRPCID = "o30O0e"

// UserProfile is the decoded current-account user profile.
type UserProfile struct {
	UserID      string
	DisplayName string
	Email       string
	PhotoURL    string
}

// EncodeGetUserProfile returns (rpcID, payload JSON string).
func EncodeGetUserProfile() (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{
		[]any{"me"},
		[]any{
			[]any{[]any{"person.photo", "person.name", "person.email"}},
			nil,
			[]any{1, 7},
		},
	})
	return getUserProfileRPCID, string(payloadBytes)
}

// DecodeGetUserProfile parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetUserProfile(body []byte) (*UserProfile, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("GetUserProfile body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetUserProfile JSON: %w", err)
	}

	userData, ok := arrayAt(data, 0, 0, 2)
	if !ok {
		return nil, fmt.Errorf("GetUserProfile response missing user profile data")
	}

	profile := &UserProfile{
		UserID: stringAt(userData, 0),
	}

	if nameBlock, ok := arrayAt(userData, 2, 0); ok {
		profile.DisplayName = firstString(
			stringAt(nameBlock, 1),
			stringAt(nameBlock, 15),
		)
	}
	if photoBlock, ok := arrayAt(userData, 3, 0); ok {
		profile.PhotoURL = stringAt(photoBlock, 1)
	}
	if emailBlock, ok := arrayAt(userData, 9, 0); ok {
		profile.Email = stringAt(emailBlock, 1)
	}
	if profile.Email == "" {
		profile.Email = findEmail(data)
	}

	if profile.UserID == "" && profile.DisplayName == "" && profile.Email == "" {
		return nil, fmt.Errorf("GetUserProfile response did not contain profile fields")
	}
	return profile, nil
}

func arrayAt(root []any, path ...int) ([]any, bool) {
	var cur any = root
	for _, idx := range path {
		arr, ok := cur.([]any)
		if !ok || idx < 0 || idx >= len(arr) {
			return nil, false
		}
		cur = arr[idx]
	}
	arr, ok := cur.([]any)
	return arr, ok
}

func stringAt(root []any, idx int) string {
	if idx < 0 || idx >= len(root) {
		return ""
	}
	s, _ := root[idx].(string)
	return s
}

func firstString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func findEmail(v any) string {
	s, ok := v.(string)
	if ok && strings.Contains(s, "@") {
		return s
	}
	arr, ok := v.([]any)
	if !ok {
		return ""
	}
	for _, item := range arr {
		if email := findEmail(item); email != "" {
			return email
		}
	}
	return ""
}
