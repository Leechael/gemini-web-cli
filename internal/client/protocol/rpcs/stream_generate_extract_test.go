package rpcs

import "testing"

func TestExtractVideos_PR309_NewPath(t *testing.T) {
	videoElem := make([]any, 8)
	videoElem[7] = []any{"https://lh3.googleusercontent.com/thumb", "https://contribution.usercontent.google.com/download?video"}
	cand12 := []any{nil, nil, nil, nil, nil, nil, nil, nil, map[string]any{"60": []any{[]any{[]any{[]any{videoElem}}}}}}

	videos := ExtractVideos(cand12)
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
	if videos[0].URL == "" {
		t.Fatal("empty video URL")
	}
}

func TestExtractVideos_PR309_OldPath(t *testing.T) {
	videoElem := make([]any, 8)
	videoElem[7] = []any{"https://lh3.googleusercontent.com/thumb", "https://contribution.usercontent.google.com/download?video"}
	cand12 := make([]any, 60)
	cand12[59] = []any{[]any{[]any{[]any{videoElem}}}}

	videos := ExtractVideos(cand12)
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
}

func TestExtractMedia_PR309_NewPath(t *testing.T) {
	mp3 := make([]any, 8)
	mp3[7] = []any{"https://lh3.googleusercontent.com/mp3-thumb", "https://contribution.usercontent.google.com/download?mp3"}
	mediaData := []any{[]any{nil, mp3}}
	cand12 := []any{map[string]any{"87": mediaData}}

	media := ExtractMedia(cand12)
	if len(media) != 1 {
		t.Fatalf("media = %d, want 1", len(media))
	}
	if media[0].MP3URL == "" {
		t.Fatal("empty mp3 URL")
	}
}

func TestExtractMedia_PR309_OldPath(t *testing.T) {
	mp3 := make([]any, 8)
	mp3[7] = []any{"https://lh3.googleusercontent.com/mp3-thumb", "https://contribution.usercontent.google.com/download?mp3"}
	mediaData := []any{[]any{nil, mp3}}
	cand12 := make([]any, 87)
	cand12[86] = mediaData

	media := ExtractMedia(cand12)
	if len(media) != 1 {
		t.Fatalf("media = %d, want 1", len(media))
	}
	if media[0].MP3URL == "" {
		t.Fatal("empty mp3 URL")
	}
}

func TestExtractDeepResearchPlan_Key56(t *testing.T) {
	plan := []any{"Research Plan Title", []any{[]any{nil, "Search", "Find sources"}}, "Soon", []any{"Start"}}
	got := ExtractDeepResearchPlan([]any{map[string]any{"56": plan}})
	if got == nil {
		t.Fatal("nil plan")
	}
	if got.Title != "Research Plan Title" || got.ConfirmPrompt != "Start" || len(got.Steps) != 1 {
		t.Fatalf("unexpected plan: %#v", got)
	}
}
