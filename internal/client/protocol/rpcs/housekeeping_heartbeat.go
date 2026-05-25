// Housekeeping heartbeat RPCs. Library code does not call these automatically;
// debug commands expose them for protocol verification.
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
