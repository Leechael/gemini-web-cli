package rpcs

import "testing"

func TestEncodeBulkLogCounter_WireParity(t *testing.T) {
	rpcID, payload := EncodeBulkLogCounter([]LogCounter{{ID: "1", Value: 447}})
	if rpcID != "ozz5Z" || payload != `[[[[null,"1",447]]]]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}

func TestEncodeBulkLogCounter_MultipleCounters(t *testing.T) {
	rpcID, payload := EncodeBulkLogCounter([]LogCounter{{ID: "1", Value: 447}, {ID: "2", Value: 9}})
	if rpcID != "ozz5Z" || payload != `[[[[null,"1",447],[null,"2",9]]]]` {
		t.Fatalf("%q %q", rpcID, payload)
	}
}
