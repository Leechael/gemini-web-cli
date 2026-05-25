// RPC: CNgdBe — ListGems (stub)
// Source-path: /app
// Reject codes: none observed in no-gems samples
//
// Payload shape: [<flag>, [<lang>], <count>]
//
//	<flag>: 1
//	<lang>: ISO language code, for example "en"
//	<count>: 0 in observed bootstrap call
//
// Response shape: [] when the account has no gems
//
// Notes:
//   - This is not a full gems-domain implementation.
//   - Library code does not call it automatically; debug commands expose it for protocol verification.
package rpcs

import "encoding/json"

// EncodeListGems returns RPC CNgdBe — ListGems stub used only for debug verification.
func EncodeListGems(lang string) (rpcID, payload string) {
	arr := []any{1, []any{lang}, 0}
	bytes, _ := json.Marshal(arr)
	return "CNgdBe", string(bytes)
}
