package cmd

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client"
)

func TestSetGenerationModeAcceptsImage(t *testing.T) {
	c := &client.Client{}
	if err := setGenerationMode(c, "image"); err != nil {
		t.Fatalf("setGenerationMode(image): %v", err)
	}
	if err := setGenerationMode(c, "audio-only"); err == nil || !strings.Contains(err.Error(), "image") {
		t.Fatalf("setGenerationMode(audio-only) error = %v, want valid modes including image", err)
	}
}

func TestPreferredModelsForImageGeneration(t *testing.T) {
	want := []string{"gemini-3.6-flash", "gemini-3.5-flash", "gemini-3-flash"}
	cases := []struct {
		name       string
		mode       string
		prompt     string
		hasUploads bool
	}{
		{name: "explicit mode", mode: "image", prompt: "Draw a sunset"},
		{name: "auto generation", mode: "auto", prompt: "Draw a sunset"},
		{name: "auto edit", mode: "auto", prompt: "Make this image photorealistic", hasUploads: true},
		{name: "case insensitive", mode: "AUTO", prompt: "Create an ILLUSTRATION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := preferredModelsForGenerationMode(tc.mode, tc.prompt, tc.hasUploads)
			if !slices.Equal(got, want) {
				t.Fatalf("preferredModelsForGenerationMode() = %v, want %v", got, want)
			}
		})
	}
}

func TestPreferredModelsForTextGeneration(t *testing.T) {
	if got := preferredModelsForGenerationMode("text", "Describe this image", true); got != nil {
		t.Fatalf("preferredModelsForGenerationMode(text) = %v, want nil", got)
	}
}

func TestResolveModelForClientFallsBackToKnownFlash(t *testing.T) {
	previousModel := modelName
	modelName = "unspecified"
	t.Cleanup(func() { modelName = previousModel })

	got := resolveModelForClient(context.Background(), nil, flashGenerationModelPreferences...)
	if got == nil || got.Name != "gemini-3.5-flash" {
		t.Fatalf("resolveModelForClient() = %v, want gemini-3.5-flash fallback", got)
	}
}
