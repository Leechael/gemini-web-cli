package client

import "testing"

func TestResolveGenerationModeImageUsesExplicitWireMode(t *testing.T) {
	if got := resolveGenerationMode("image", "Draw a sunset", nil); got != "image" {
		t.Fatalf("resolveGenerationMode(image) = %q, want \"image\"", got)
	}
}

func TestResolveGenerationModeTextUsesStandardRequest(t *testing.T) {
	if got := resolveGenerationMode("text", "Draw a sunset", nil); got != "" {
		t.Fatalf("resolveGenerationMode(text) = %q, want empty wire mode", got)
	}
}

func TestResolveGenerationModeAutoImageKeywordsStayStandard(t *testing.T) {
	// Auto mode with image-ish prompts keeps the plain text request shape;
	// only explicit --mode image opts into the /images wire flags.
	if got := resolveGenerationMode("auto", "Draw a sunset", nil); got != "" {
		t.Fatalf("resolveGenerationMode(auto) = %q, want empty wire mode", got)
	}
}
