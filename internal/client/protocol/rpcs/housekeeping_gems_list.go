package rpcs

import "encoding/json"

// EncodeListGems returns RPC CNgdBe — ListGems stub used only for debug verification.
func EncodeListGems(lang string) (rpcID, payload string) {
	arr := []any{1, []any{lang}, 0}
	bytes, _ := json.Marshal(arr)
	return "CNgdBe", string(bytes)
}
