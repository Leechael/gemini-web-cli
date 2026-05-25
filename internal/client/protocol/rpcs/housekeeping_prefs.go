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
