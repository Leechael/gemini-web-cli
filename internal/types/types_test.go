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
	if imgArr[4] != "56fdd199312815e2" {
		t.Errorf("image header [4] = %v, want model id", imgArr[4])
	}
	run, ok := imgArr[8].([]any)
	if !ok {
		t.Fatalf("image header [8] type %T, want array", imgArr[8])
	}
	wantRun := []float64{4, 5, 6, 8, 4, 5, 6, 8}
	if len(run) != len(wantRun) {
		t.Fatalf("image header [8] len = %d, want %d", len(run), len(wantRun))
	}
	for i, v := range wantRun {
		if run[i] != v {
			t.Errorf("image header [8][%d] = %v, want %v", i, run[i], v)
		}
	}
}

func TestFindModelAliases(t *testing.T) {
	cases := map[string]string{
		"gemini-3.1-flash-lite":   "gemini-3.5-flash-lite",
		"gemini-3.5-flash":        "gemini-3.8-flash",
		"gemini-3-flash-plus":     "gemini-3.8-flash-plus",
		"gemini-3-flash-advanced": "gemini-3.8-flash-advanced",
	}
	for oldName, current := range cases {
		alias := FindModel(oldName)
		canonical := FindModel(current)
		if alias == nil || canonical == nil {
			t.Fatalf("FindModel(%q)=%v FindModel(%q)=%v", oldName, alias, current, canonical)
		}
		if alias.Name != canonical.Name || alias.ModelID() != canonical.ModelID() {
			t.Errorf("alias %q resolved to %+v, want %+v", oldName, alias, canonical)
		}
	}
}
