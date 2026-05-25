package types

import "testing"

func TestExtractImages_ImageToImagePath(t *testing.T) {
	generated := []any{nil, nil, nil, []any{nil, nil, nil, "https://lh3.googleusercontent.com/sample-i2i"}}
	imageData := []any{map[string]any{"8": []any{generated}}}

	images := ExtractImages(imageData)
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	if images[0].URL != "https://lh3.googleusercontent.com/sample-i2i" {
		t.Fatalf("URL = %q", images[0].URL)
	}
	if !images[0].Generated {
		t.Fatal("Generated = false, want true")
	}
}

func TestExtractImages_PathPreservation(t *testing.T) {
	generated := []any{nil, nil, nil, []any{nil, nil, nil, "https://lh3.googleusercontent.com/sample-old"}}
	arr := make([]any, 8)
	arr[7] = []any{generated}

	images := ExtractImages(arr)
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	if images[0].URL != "https://lh3.googleusercontent.com/sample-old" {
		t.Fatalf("URL = %q", images[0].URL)
	}
	if !images[0].Generated {
		t.Fatal("Generated = false, want true")
	}
}
