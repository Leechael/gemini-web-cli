// RPC: Te6DCf — SetLanguagePreference
// Source-path: /
// Reject codes: none observed
//
// Payload shape: [[<lang>], [<flag>]]
//
//	<lang>: ISO language code, for example "en" or "zh-CN"
//	<flag>: 1 to set, 0 to unset
//
// Response shape: typically empty
//
// RPC: maGuAc — user preference toggle (exact semantic unknown)
// Source-path: /
// Reject codes: none observed
//
// Payload shape: [<flag>]
// Response shape: typically empty
//
// Notes:
//   - Browser sends these during bootstrap/preference sync.
//   - Library code does not call them automatically; debug commands expose them for protocol verification.
package rpcs

import "encoding/json"

// EncodeSetLanguagePreference returns RPC Te6DCf — SetLanguagePreference.
func EncodeSetLanguagePreference(lang string, set bool) (rpcID, payload string) {
	flag := 0
	if set {
		flag = 1
	}
	arr := []any{[]any{lang}, []any{flag}}
	bytes, _ := json.Marshal(arr)
	return "Te6DCf", string(bytes)
}

// EncodeMaGuAcToggle returns RPC maGuAc — single user preference toggle.
func EncodeMaGuAcToggle(value int) (rpcID, payload string) {
	arr := []any{value}
	bytes, _ := json.Marshal(arr)
	return "maGuAc", string(bytes)
}
