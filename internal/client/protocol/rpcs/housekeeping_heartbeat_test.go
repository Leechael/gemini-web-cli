package rpcs

import "testing"

func TestEncodeHeartbeat_WireParity(t *testing.T) {
	rpcID, payload := EncodeHeartbeat()
	if rpcID != "Ub3MPb" || payload != "[]" {
		t.Fatalf("%q %q", rpcID, payload)
	}
}

func TestEncodeUIHeartbeat_WireParity(t *testing.T) {
	rpcID, payload := EncodeUIHeartbeat()
	if rpcID != "VxUbXb" || payload != "[]" {
		t.Fatalf("%q %q", rpcID, payload)
	}
}
