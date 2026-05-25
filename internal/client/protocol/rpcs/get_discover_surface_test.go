package rpcs

import "testing"

func TestEncodeGetDiscoverSurface_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeGetDiscoverSurface("en", []int{390, 391, 392})
	if rpcID != "V8rlHe" {
		t.Fatalf("rpcID = %q, want V8rlHe", rpcID)
	}
	if payload != `[["en"],[[390,391,392]]]` {
		t.Fatalf("payload = %q", payload)
	}
}

func TestDecodeGetDiscoverSurface_FromSampleFixture(t *testing.T) {
	cards, err := DecodeGetDiscoverSurface(rpcFixtureBody(t, "get_discover_surface_basic.txt", "V8rlHe"))
	if err != nil {
		t.Fatalf("DecodeGetDiscoverSurface: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("cards = %d, want 3", len(cards))
	}
	if cards[0].ID != "image-gen-sommelier" {
		t.Fatalf("ID = %q", cards[0].ID)
	}
	if cards[0].Title != "Sommelier" {
		t.Fatalf("Title = %q", cards[0].Title)
	}
	if cards[0].PreviewURL != "https://lh3.googleusercontent.com/sample-discover-1" {
		t.Fatalf("PreviewURL = %q", cards[0].PreviewURL)
	}
	if cards[0].Description == "" {
		t.Fatal("Description is empty")
	}
}

func TestDecodeGetDiscoverSurface_EmptyBody(t *testing.T) {
	cards, err := DecodeGetDiscoverSurface(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("cards = %d, want 0", len(cards))
	}
}

func TestDecodeGetDiscoverSurface_MalformedJSON(t *testing.T) {
	if _, err := DecodeGetDiscoverSurface([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
