package cmd

import "testing"

func TestFullResolutionDownloadURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		url        string
		defaultExt string
		want       string
	}{
		{
			name:       "generated chat image",
			url:        "https://lh3.googleusercontent.com/generated/image",
			defaultExt: ".png",
			want:       "https://lh3.googleusercontent.com/generated/image=s0",
		},
		{
			name:       "existing thumbnail transform",
			url:        "https://lh3.googleusercontent.com/generated/image=s512-c?token=abc",
			defaultExt: ".png",
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
			want:       "https://lh3.googleusercontent.com/generated/image=token",
		},
		{
			name:       "generated video",
			url:        "https://lh3.googleusercontent.com/generated/video",
			defaultExt: ".mp4",
			want:       "https://lh3.googleusercontent.com/generated/video",
		},
		{
			name:       "non Google image",
			url:        "https://example.com/image.png",
			defaultExt: ".png",
			want:       "https://example.com/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fullResolutionDownloadURL(tt.url, tt.defaultExt); got != tt.want {
				t.Fatalf("fullResolutionDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
