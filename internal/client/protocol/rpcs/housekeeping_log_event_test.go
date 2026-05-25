package rpcs

import "testing"

func TestEncodeLogClientEvent_Stub(t *testing.T) {
	rpcID, payload, err := EncodeLogClientEvent(LogClientEventOpts{DeviceID: "00000000-0000-0000-0000-000000000001", ModelID: 48, EventID: 1})
	if rpcID != "dI8W6e" || payload != "" || err == nil {
		t.Fatalf("rpcID=%q payload=%q err=%v", rpcID, payload, err)
	}
}

func TestEncodeLogModelSelection_WireParity(t *testing.T) {
	rpcID, payload := EncodeLogModelSelection(LogModelSelectionOpts{Selector1: 2, Selector2: 1, ModelID: 48, ExpIDs: []int{76091940, 24}})
	if rpcID != "TFNzk" || payload != `[[2,null,1],null,[[48],[76091940,24]]]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}
