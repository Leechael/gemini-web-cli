package rpcs

import "testing"

func TestEncodeGetUserStatus(t *testing.T) {
	rpcID, payload := EncodeGetUserStatus()
	if rpcID != "otAQ7b" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != "[]" {
		t.Fatalf("payload = %q", payload)
	}
}

func TestDecodeGetUserStatus(t *testing.T) {
	body := loadProtocolTestdata(t, "get_user_status_basic.txt")
	result, err := DecodeGetUserStatus(body)
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountStatus.Code != 1000 {
		t.Fatalf("status = %d", result.AccountStatus.Code)
	}
	if len(result.Models) != 1 {
		t.Fatalf("models = %d, want 1", len(result.Models))
	}
	if result.Models[0].ModelID != "fbb127bbb056c959" || result.Models[0].Selector != 1 {
		t.Fatalf("model = %#v", result.Models[0])
	}
	if len(result.CapFlags) != 1 || result.CapFlags[0] != 19 {
		t.Fatalf("cap flags = %#v", result.CapFlags)
	}
}
