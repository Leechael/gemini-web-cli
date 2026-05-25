package types

import "testing"

func TestExtractVideos_FallbackWhenPrimaryShapeInvalid(t *testing.T) {
	videoElem := make([]any, 8)
	videoElem[7] = []any{"https://example.com/thumb.jpg", "https://example.com/video.mp4"}

	media := make([]any, 60)
	media[59] = "unexpected primary shape"
	media[0] = map[string]any{"60": []any{[]any{[]any{[]any{videoElem}}}}}

	videos := ExtractVideos(media)
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
	if videos[0].URL != "https://example.com/video.mp4" {
		t.Fatalf("video URL = %q", videos[0].URL)
	}
}
