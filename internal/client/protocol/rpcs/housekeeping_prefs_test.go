package rpcs

import "testing"

func TestEncodeSetLanguagePreference_WireParity(t *testing.T) {
	rpcID, payload := EncodeSetLanguagePreference("en", true)
	if rpcID != "Te6DCf" || payload != `[["en"],[1]]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}

func TestEncodeMaGuAcToggle_WireParity(t *testing.T) {
	rpcID, payload := EncodeMaGuAcToggle(1)
	if rpcID != "maGuAc" || payload != `[1]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}
