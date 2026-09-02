package client

import "testing"

func TestResolveGenerationModeImageUsesStandardRequest(t *testing.T) {
	if got := resolveGenerationMode("image", "Draw a sunset", nil); got != "" {
		t.Fatalf("resolveGenerationMode(image) = %q, want empty wire mode", got)
	}
}
