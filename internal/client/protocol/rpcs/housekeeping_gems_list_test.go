package rpcs

import "testing"

func TestEncodeListGems_WireParity(t *testing.T) {
	rpcID, payload := EncodeListGems("en")
	if rpcID != "CNgdBe" || payload != `[1,["en"],0]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}
