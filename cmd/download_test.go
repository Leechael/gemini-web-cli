package cmd

import (
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

func TestFullResolutionDownloadURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		url        string
		defaultExt string
		generated  bool
		want       string
	}{
		{
			name:       "generated chat image",
			url:        "https://lh3.googleusercontent.com/generated/image",
			defaultExt: ".png",
			generated:  true,
			want:       "https://lh3.googleusercontent.com/generated/image=s0",
		},
		{
			name:       "web search image",
			url:        "https://lh3.googleusercontent.com/web/image=s512-c?token=abc",
			defaultExt: ".png",
			generated:  false,
			want:       "https://lh3.googleusercontent.com/web/image=s512-c?token=abc",
		},
		{
			name:       "existing thumbnail transform",
			url:        "https://lh3.googleusercontent.com/generated/image=s512-c?token=abc",
			defaultExt: ".png",
			generated:  true,
			want:       "https://lh3.googleusercontent.com/generated/image=s0?token=abc",
		},
		{
			name:       "direct image URL",
			url:        "https://lh3.googleusercontent.com/generated/image",
			defaultExt: "",
			want:       "https://lh3.googleusercontent.com/generated/image=s0",
		},
		{
			name:       "opaque path suffix",
			url:        "https://lh3.googleusercontent.com/generated/image=token",
			defaultExt: ".png",
			generated:  true,
			want:       "https://lh3.googleusercontent.com/generated/image=token",
		},
		{
			name:       "generated video",
			url:        "https://lh3.googleusercontent.com/generated/video",
			defaultExt: ".mp4",
			generated:  true,
			want:       "https://lh3.googleusercontent.com/generated/video",
		},
		{
			name:       "non Google image",
			url:        "https://example.com/image.png",
			defaultExt: ".png",
			generated:  true,
			want:       "https://example.com/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fullResolutionDownloadURL(tt.url, tt.defaultExt, tt.generated); got != tt.want {
				t.Fatalf("fullResolutionDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectDownloadablesPreservesGeneratedFlag(t *testing.T) {
	t.Parallel()

	items := collectDownloadables([]types.ChatTurn{{
		Images: []types.Image{
			{URL: "https://lh3.googleusercontent.com/generated/image", Generated: true},
			{URL: "https://lh3.googleusercontent.com/web/image", Generated: false},
		},
	}})
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if !items[0].Generated {
		t.Fatal("generated image did not preserve Generated=true")
	}
	if items[1].Generated {
		t.Fatal("web image did not preserve Generated=false")
	}
}
