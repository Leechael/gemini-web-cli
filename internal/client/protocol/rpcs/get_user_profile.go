// RPC: o30O0e — GetUserProfile
// Source-path: /app (or any Gemini page)
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
//	[[["me", 1, userData, null, []]]]
//	              ↑
//	              data[0][0][2]
//
//	userData structure:
//	  [0]: "<userId>"
//	  [1]: metadata, including [21][1] == ["me"]
//	  [2]: name blocks; display name is userData[2][0][1]
//	  [3]: photo blocks; photo URL is userData[3][0][1] when present
//	  [9]: email blocks; email is userData[9][0][1] when present
//
//	name block structure:
//	  [0]: [true, 0, true, null, ..., "<userId>", ..., [unix_seconds, nanos]]
//	  [1]: "<display name>"
//	  [15]: alternate "<display name>"
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

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
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

	userData, ok := protocol.ArrayAt(data, 0, 0, 2)
	if !ok {
		return nil, fmt.Errorf("GetUserProfile response missing user profile data")
	}

	profile := &UserProfile{
		UserID: protocol.StringAt(userData, 0),
	}

	if nameBlock, ok := protocol.ArrayAt(userData, 2, 0); ok {
		profile.DisplayName = protocol.FirstString(
			protocol.StringAt(nameBlock, 1),
			protocol.StringAt(nameBlock, 15),
		)
	}
	if photoBlock, ok := protocol.ArrayAt(userData, 3, 0); ok {
		profile.PhotoURL = protocol.StringAt(photoBlock, 1)
	}
	if emailBlock, ok := protocol.ArrayAt(userData, 9, 0); ok {
		profile.Email = protocol.StringAt(emailBlock, 1)
	}
	if profile.Email == "" {
		profile.Email = findEmail(data)
	}

	if profile.UserID == "" && profile.DisplayName == "" && profile.Email == "" {
		return nil, fmt.Errorf("GetUserProfile response did not contain profile fields")
	}
	return profile, nil
}

func findEmail(v any) string {
	arr, ok := v.([]any)
	if !ok {
		return ""
	}
	if len(arr) > 1 {
		if s, ok := arr[1].(string); ok && isEmailCandidate(s) {
			return s
		}
	}
	for _, item := range arr {
		if email := findEmail(item); email != "" {
			return email
		}
	}
	return ""
}

func isEmailCandidate(s string) bool {
	if strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1 && strings.LastIndexByte(s, '@') == at && strings.Contains(s[at+1:], ".")
}
