package rpcs

import "testing"

func TestEncodeExpandMediaPrompt_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeExpandMediaPrompt("Sample user prompt")
	if rpcID != "Pty9pd" {
		t.Fatalf("rpcID = %q, want Pty9pd", rpcID)
	}
	if payload != `["Sample user prompt",null,null,1]` {
		t.Fatalf("payload = %q", payload)
	}
}

func TestDecodeExpandMediaPrompt_FromMusicFixture(t *testing.T) {
	variations, err := DecodeExpandMediaPrompt(rpcFixtureBody(t, "expand_media_prompt_music_basic.txt", "Pty9pd"))
	if err != nil {
		t.Fatalf("DecodeExpandMediaPrompt: %v", err)
	}
	assertExpandMediaPromptVariations(t, variations)
}

func TestDecodeExpandMediaPrompt_FromImageFixture(t *testing.T) {
	variations, err := DecodeExpandMediaPrompt(rpcFixtureBody(t, "expand_media_prompt_image_basic.txt", "Pty9pd"))
	if err != nil {
		t.Fatalf("DecodeExpandMediaPrompt: %v", err)
	}
	assertExpandMediaPromptVariations(t, variations)
}

func TestDecodeExpandMediaPrompt_EmptyBody(t *testing.T) {
	variations, err := DecodeExpandMediaPrompt(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(variations) != 0 {
		t.Fatalf("variations = %d, want 0", len(variations))
	}
}

func TestDecodeExpandMediaPrompt_MalformedJSON(t *testing.T) {
	if _, err := DecodeExpandMediaPrompt([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}

func assertExpandMediaPromptVariations(t *testing.T, variations []string) {
	t.Helper()
	if len(variations) != 3 {
		t.Fatalf("variations = %d, want 3", len(variations))
	}
	if variations[0] != "Expanded variation 1" {
		t.Fatalf("variation 1 = %q", variations[0])
	}
	if variations[2] != "Expanded variation 3" {
		t.Fatalf("variation 3 = %q", variations[2])
	}
}
