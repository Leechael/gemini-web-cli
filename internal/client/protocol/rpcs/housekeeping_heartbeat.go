// RPC: Ub3MPb — Heartbeat
// Source-path: / (root bootstrap)
// Reject codes: none observed
//
// Payload shape: []
// Response shape: typically empty or [null, null, null, 0]
//
// RPC: VxUbXb — UIHeartbeat
// Source-path: any Gemini page
// Reject codes: none observed
//
// Payload shape: []
// Response shape: [null, null, null, 0]
//
// Notes:
//   - Browser sends these during bootstrap and UI state transitions.
//   - Library code does not call them automatically; debug commands expose them for protocol verification.
package rpcs

const (
	heartbeatRPCID   = "Ub3MPb"
	uiHeartbeatRPCID = "VxUbXb"
)

// EncodeHeartbeat returns RPC Ub3MPb — Heartbeat.
func EncodeHeartbeat() (rpcID, payload string) {
	return heartbeatRPCID, "[]"
}

// EncodeUIHeartbeat returns RPC VxUbXb — high-frequency UI heartbeat.
func EncodeUIHeartbeat() (rpcID, payload string) {
	return uiHeartbeatRPCID, "[]"
}
