package rpcs

import (
	"encoding/json"
	"syscall"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

func rpcFixtureBody(t *testing.T, filename, rpcID string) []byte {
	t.Helper()
	raw, err := readRPCFixture("../testdata/" + filename)
	if err != nil {
		t.Fatal(err)
	}
	body, rejectCode, err := protocol.ExtractRPCBody(protocol.StripResponsePrefix(raw), rpcID)
	if err != nil {
		t.Fatalf("ExtractRPCBody: %v", err)
	}
	if rejectCode != 0 {
		t.Fatalf("rejectCode = %d, want 0", rejectCode)
	}
	return body
}

func readRPCFixture(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	var out []byte
	buf := make([]byte, 4096)
	for {
		n, err := syscall.Read(fd, buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return out, nil
		}
	}
}

func TestEncodeGetUserLocation_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetUserLocation()
	if rpcID != "K4WWud" {
		t.Fatalf("rpcID = %q, want K4WWud", rpcID)
	}
	if payload != `[[0],["en"]]` {
		t.Fatalf("payload = %s", payload)
	}
	var decoded []any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not JSON: %v", err)
	}
}

func TestDecodeGetUserLocation_FromSampleFixture(t *testing.T) {
	location, err := DecodeGetUserLocation(rpcFixtureBody(t, "get_user_location_basic.txt", "K4WWud"))
	if err != nil {
		t.Fatalf("DecodeGetUserLocation: %v", err)
	}
	if location.Region != "Sample Region, US" {
		t.Fatalf("Region = %q", location.Region)
	}
	if location.Source == "" {
		t.Fatalf("Source is empty")
	}
	if location.IsPrecise {
		t.Fatalf("IsPrecise = true, want false")
	}
	if location.MapTileURL == "" {
		t.Fatalf("MapTileURL is empty")
	}
}

func TestDecodeGetUserLocation_EmptyBody(t *testing.T) {
	if _, err := DecodeGetUserLocation(nil); err == nil {
		t.Fatalf("DecodeGetUserLocation(nil) error = nil")
	}
}

func TestDecodeGetUserLocation_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetUserLocation([]byte(`{not-json`)); err == nil {
		t.Fatalf("DecodeGetUserLocation(malformed) error = nil")
	}
}
