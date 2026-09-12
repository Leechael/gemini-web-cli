package types

import (
	"encoding/json"
	"testing"
)

func TestExtractImages_ImageToImagePath(t *testing.T) {
	t.Skip("waiting for real image-to-image fixture verification")

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
	marker := []any{nil, nil, nil, []any{nil, nil, nil, "nfr_more_trigger_si"}}
	arr := make([]any, 8)
	arr[7] = []any{generated, marker}

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

func TestBuildModelHeaderForSurface(t *testing.T) {
	chat := BuildModelHeader("56fdd199312815e2", 1)
	var chatArr []any
	if err := json.Unmarshal([]byte(chat[ModelHeaderKey]), &chatArr); err != nil {
		t.Fatal(err)
	}
	if chatArr[15] != float64(1) {
		t.Errorf("chat header [15] = %v, want 1", chatArr[15])
	}

	img := BuildModelHeaderForSurface("56fdd199312815e2", 1, 2)
	var imgArr []any
	if err := json.Unmarshal([]byte(img[ModelHeaderKey]), &imgArr); err != nil {
		t.Fatal(err)
	}
	if imgArr[15] != float64(2) {
		t.Errorf("image header [15] = %v, want 2", imgArr[15])
	}
	if imgArr[14] != float64(1) {
		t.Errorf("image header [14] = %v, want selector 1", imgArr[14])
	}
}
