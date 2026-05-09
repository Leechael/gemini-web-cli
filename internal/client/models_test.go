package client

import (
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// Tier-aware name resolution: Plus accounts (capacity=4) see `-plus` variants;
// Advanced accounts (capacity=2) see `-advanced`; Basic (capacity=1) see the
// bare names. IDs that only exist in BASIC tier (e.g. gemini-3-pro) always
// fall back to the bare name regardless of account tier.

func modelID(t *testing.T, name string) string {
	t.Helper()
	m := types.FindModel(name)
	if m == nil {
		t.Fatalf("missing model definition for %q", name)
	}
	id := m.ModelID()
	if id == "" {
		t.Fatalf("model %q has empty ModelID", name)
	}
	return id
}

func TestBuildModelIDNameMapping_Basic(t *testing.T) {
	m := buildModelIDNameMapping(1, 12)

	cases := map[string]string{
		modelID(t, "gemini-3-pro"):                 "gemini-3-pro",
		modelID(t, "gemini-3-flash"):               "gemini-3-flash",
		modelID(t, "gemini-3-flash-thinking"):      "gemini-3-flash-thinking",
		modelID(t, "gemini-3-pro-plus"):            "gemini-3-pro",
		modelID(t, "gemini-3-flash-plus"):          "gemini-3-flash",
		modelID(t, "gemini-3-flash-thinking-plus"): "gemini-3-flash-thinking",
	}
	for id, want := range cases {
		if got := m[id]; got != want {
			t.Errorf("Basic: id %s → %q, want %q", id, got, want)
		}
	}
}

func TestBuildModelIDNameMapping_Plus(t *testing.T) {
	m := buildModelIDNameMapping(4, 12)

	// IDs shared by Plus/Advanced should resolve to the -plus variant.
	cases := map[string]string{
		modelID(t, "gemini-3-pro-plus"):            "gemini-3-pro-plus",
		modelID(t, "gemini-3-flash-plus"):          "gemini-3-flash-plus",
		modelID(t, "gemini-3-flash-thinking-plus"): "gemini-3-flash-thinking-plus",
		// BASIC-only IDs remain the bare name even on Plus tier (Plus doesn't
		// re-register `gemini-3-pro` etc., so the second pass fills them).
		modelID(t, "gemini-3-pro"):            "gemini-3-pro",
		modelID(t, "gemini-3-flash"):          "gemini-3-flash",
		modelID(t, "gemini-3-flash-thinking"): "gemini-3-flash-thinking",
	}
	for id, want := range cases {
		if got := m[id]; got != want {
			t.Errorf("Plus: id %s → %q, want %q", id, got, want)
		}
	}
}

func TestBuildModelIDNameMapping_Advanced(t *testing.T) {
	m := buildModelIDNameMapping(2, 13)

	cases := map[string]string{
		modelID(t, "gemini-3-pro-advanced"):            "gemini-3-pro-advanced",
		modelID(t, "gemini-3-flash-advanced"):          "gemini-3-flash-advanced",
		modelID(t, "gemini-3-flash-thinking-advanced"): "gemini-3-flash-thinking-advanced",
		modelID(t, "gemini-3-pro"):                     "gemini-3-pro",
		modelID(t, "gemini-3-flash"):                   "gemini-3-flash",
		modelID(t, "gemini-3-flash-thinking"):          "gemini-3-flash-thinking",
	}
	for id, want := range cases {
		if got := m[id]; got != want {
			t.Errorf("Advanced: id %s → %q, want %q", id, got, want)
		}
	}
}

func TestTierSuffixForCapacity(t *testing.T) {
	cases := []struct {
		cap  int
		want string
	}{
		{1, ""},
		{2, "-advanced"},
		{3, ""},
		{4, "-plus"},
	}
	for _, c := range cases {
		if got := tierSuffixForCapacity(c.cap); got != c.want {
			t.Errorf("capacity=%d: %q, want %q", c.cap, got, c.want)
		}
	}
}
