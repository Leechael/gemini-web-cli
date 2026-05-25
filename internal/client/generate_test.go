package client

import (
	"encoding/json"
	"testing"
)

func TestParseEnvelope_UnwrapsSingleElement(t *testing.T) {
	inner := `[null,["c_abc","r_def"],null,null,[[" rc_ghi",[" hello"]]]]`
	envelope := []any{[]any{"wrb.fr", nil, inner}}

	out := parseEnvelope(envelope)
	if out == nil {
		t.Fatal("parseEnvelope returned nil")
	}
	if len(out.Metadata) < 2 {
		t.Fatalf("expected >=2 metadata, got %d", len(out.Metadata))
	}
	if out.Metadata[0] != "c_abc" {
		t.Errorf("metadata[0] = %q, want c_abc", out.Metadata[0])
	}
}

func TestParseEnvelope_NormalCompletion(t *testing.T) {
	content := make([]any, 26)
	content[1] = []any{"c_abc", "r_def"}
	content[4] = []any{[]any{"rc_ghi", []any{"hello world"}}}
	content[25] = "context_token_here"

	contentJSON, _ := json.Marshal(content)
	envelope := []any{[]any{"wrb.fr", nil, string(contentJSON)}}

	out := parseEnvelope(envelope)
	if out == nil {
		t.Fatal("parseEnvelope returned nil")
	}
	if !out.Done {
		t.Error("expected Done=true for content[25] string")
	}
	if len(out.Metadata) < 10 || out.Metadata[9] != "context_token_here" {
		t.Errorf("metadata[9] = %q, want context_token_here", out.Metadata[9])
	}
}

func TestParseEnvelope_DeepResearchCompletion(t *testing.T) {
	content := []any{
		nil,
		[]any{"c_abc", "r_def"},
		map[string]any{"26": "AwAAAAtoken", "44": false},
	}

	contentJSON, _ := json.Marshal(content)
	envelope := []any{[]any{"wrb.fr", nil, string(contentJSON)}}

	out := parseEnvelope(envelope)
	if out == nil {
		t.Fatal("parseEnvelope returned nil")
	}
	if !out.Done {
		t.Error("expected Done=true for deep research completion dict")
	}
	if len(out.Metadata) < 10 || out.Metadata[9] != "AwAAAAtoken" {
		t.Errorf("metadata[9] = %q, want AwAAAAtoken", out.Metadata[9])
	}
}

func TestParseEnvelope_RCidInMetadata(t *testing.T) {
	content := make([]any, 5)
	content[1] = []any{"c_abc", "r_def"}
	content[4] = []any{[]any{"rc_ghi", []any{"text here"}}}

	contentJSON, _ := json.Marshal(content)
	envelope := []any{[]any{"wrb.fr", nil, string(contentJSON)}}

	out := parseEnvelope(envelope)
	if out == nil {
		t.Fatal("parseEnvelope returned nil")
	}
	if out.RCid != "rc_ghi" {
		t.Errorf("RCid = %q, want rc_ghi", out.RCid)
	}
	if len(out.Metadata) < 10 {
		t.Fatalf("metadata len = %d, want >=10", len(out.Metadata))
	}
	if out.Metadata[2] != "rc_ghi" {
		t.Errorf("metadata[2] = %q, want rc_ghi", out.Metadata[2])
	}
}

func TestExtractImages_GeneratedImage(t *testing.T) {
	item := []any{nil, nil, nil, []any{nil, 1, "file.png", "https://lh3.googleusercontent.com/test"}, nil, "$sig"}
	arr := make([]any, 8)
	arr[7] = []any{[]any{[]any{item}}} // arr[7][0][0] = [item]

	images := extractImages(arr)
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].URL != "https://lh3.googleusercontent.com/test" {
		t.Errorf("URL = %q", images[0].URL)
	}
	if !images[0].Generated {
		t.Error("expected Generated=true")
	}
}

func TestExtractImages_WebImage(t *testing.T) {
	webImg := make([]any, 8)
	webImg[0] = []any{[]any{"https://example.com/img.jpg"}}
	webImg[7] = []any{"Example Title"}

	arr := make([]any, 2)
	arr[1] = []any{webImg}

	images := extractImages(arr)
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
	if images[0].URL != "https://example.com/img.jpg" {
		t.Errorf("URL = %q", images[0].URL)
	}
	if images[0].Title != "Example Title" {
		t.Errorf("Title = %q", images[0].Title)
	}
	if images[0].Generated {
		t.Error("expected Generated=false")
	}
}

func TestExtractVideos_ReadChatDictKey60(t *testing.T) {
	videoElem := make([]any, 8)
	videoElem[7] = []any{"https://lh3.googleusercontent.com/thumb", "https://contribution.usercontent.google.com/download?video"}
	imageData := []any{map[string]any{
		"60": []any{[]any{[]any{[]any{videoElem}}}},
	}}

	videos := extractVideos(imageData)
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
	if videos[0].URL != "https://contribution.usercontent.google.com/download?video" {
		t.Errorf("video URL = %q", videos[0].URL)
	}
}

func TestExtractMedia_ReadChatDictKey87Metadata(t *testing.T) {
	mp3 := make([]any, 8)
	mp3[7] = []any{"https://lh3.googleusercontent.com/mp3-thumb", "https://contribution.usercontent.google.com/download?mp3"}
	mp4 := make([]any, 8)
	mp4[7] = []any{"https://lh3.googleusercontent.com/mp4-thumb", "https://contribution.usercontent.google.com/download?mp4"}
	vtt := make([]any, 8)
	vtt[7] = []any{"", "https://contribution.usercontent.google.com/download?vtt"}
	mediaData := []any{
		[]any{nil, mp3},
		[]any{nil, mp4, nil, vtt},
		[]any{"Late July Transit", nil, "Afternoon Horizon", nil, "K-Pop / City Pop", []any{"Nostalgic", "Breezy"}},
	}
	imageData := []any{map[string]any{"87": mediaData}}

	media := extractMedia(imageData)
	if len(media) != 1 {
		t.Fatalf("media = %d, want 1", len(media))
	}
	m := media[0]
	if m.MP3URL == "" || m.MP4URL == "" || m.VTTURL == "" {
		t.Fatalf("missing URLs: %+v", m)
	}
	if m.Title != "Late July Transit" || m.Artist != "Afternoon Horizon" || m.Genre != "K-Pop / City Pop" {
		t.Fatalf("metadata = %+v", m)
	}
	if len(m.Moods) != 2 || m.Moods[0] != "Nostalgic" || m.Moods[1] != "Breezy" {
		t.Fatalf("moods = %#v", m.Moods)
	}
}

func TestExtractDeepResearchPlan(t *testing.T) {
	plan := []any{
		"Research Plan Title",
		[]any{
			[]any{nil, "Research Websites", "Search for X, Y, Z"},
			[]any{nil, "Analyze Results", ""},
		},
		"Ready in a few mins",
		[]any{"Start research"},
		[]any{"http://googleusercontent.com/deep_research_confirmation_content/0"},
		nil,
	}

	candidateData := []any{
		"rc_abc",
		[]any{"some text"},
		nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		map[string]any{"56": plan, "70": float64(2)},
	}

	result := extractDeepResearchPlan(candidateData)
	if result == nil {
		t.Fatal("extractDeepResearchPlan returned nil")
	}
	if result.Title != "Research Plan Title" {
		t.Errorf("Title = %q", result.Title)
	}
	if result.ETAText != "Ready in a few mins" {
		t.Errorf("ETAText = %q", result.ETAText)
	}
	if result.ConfirmPrompt != "Start research" {
		t.Errorf("ConfirmPrompt = %q", result.ConfirmPrompt)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("Steps count = %d, want 2", len(result.Steps))
	}
	if result.Steps[0] != "Research Websites: Search for X, Y, Z" {
		t.Errorf("Steps[0] = %q", result.Steps[0])
	}
}

func TestExtractDeepResearchPlan_Key57(t *testing.T) {
	plan := []any{"Title 57", nil, "ETA", []any{"Confirm"}}
	candidateData := []any{map[string]any{"57": plan}}

	result := extractDeepResearchPlan(candidateData)
	if result == nil {
		t.Fatal("nil")
	}
	if result.Title != "Title 57" {
		t.Errorf("Title = %q", result.Title)
	}
}

func TestExtractDeepResearchPlan_NoPlan(t *testing.T) {
	candidateData := []any{"rc_abc", []any{"text"}, nil}
	result := extractDeepResearchPlan(candidateData)
	if result != nil {
		t.Error("expected nil for candidate without plan dict")
	}
}

func TestBuildInnerRequest_NewChat(t *testing.T) {
	c := &Client{}
	req := c.buildInnerRequest("hello", nil, nil, nil, false, false, "TEST-UUID")

	if len(req) != 81 {
		t.Fatalf("len = %d, want 81", len(req))
	}

	meta, ok := req[2].([]any)
	if !ok {
		t.Fatal("req[2] not []any")
	}
	if len(meta) != 10 {
		t.Errorf("metadata len = %d, want 10", len(meta))
	}

	if req[17] == nil {
		t.Fatal("[17] is nil")
	}

	if req[45] != nil {
		t.Errorf("[45] = %v, want nil", req[45])
	}

	if req[68] != 1 {
		t.Errorf("[68] = %v, want 1", req[68])
	}

	if req[59] != "TEST-UUID" {
		t.Errorf("[59] = %v", req[59])
	}
	if req[79] != 1 {
		t.Errorf("[79] = %v, want 1", req[79])
	}
	if req[80] != 1 {
		t.Errorf("[80] = %v, want 1", req[80])
	}
}

func TestBuildInnerRequest_GenerationModes(t *testing.T) {
	cases := []struct {
		mode string
		want any
	}{
		{"video", 11},
		{"image-to-video", 14},
		{"music", 21},
	}
	for _, tc := range cases {
		c := &Client{generationMode: tc.mode}
		req := c.buildInnerRequest("make something", nil, nil, nil, false, false, "UUID")
		if req[49] != tc.want {
			t.Errorf("mode %s: [49] = %v, want %v", tc.mode, req[49], tc.want)
		}
	}

	c := &Client{generationMode: "video"}
	req := c.buildInnerRequest("make a video", nil, nil, nil, false, false, "UUID")
	msg := req[0].([]any)
	if len(msg) < 10 || msg[9] == nil {
		t.Fatalf("video mode message marker missing: %v", msg)
	}
	if req[54] == nil || req[55] == nil {
		t.Fatalf("video mode slots missing: [54]=%v [55]=%v", req[54], req[55])
	}
}

func TestBuildInnerRequest_DeepResearch(t *testing.T) {
	c := &Client{}
	req := c.buildInnerRequest("research topic", nil, nil, nil, true, false, "UUID")

	if req[49] != 1 {
		t.Errorf("[49] = %v, want 1", req[49])
	}
	if req[68] != 2 {
		t.Errorf("[68] = %v, want 2", req[68])
	}
	if s, ok := req[3].(string); !ok || s[0] != '!' {
		t.Errorf("[3] should start with !, got %v", req[3])
	}
	if s, ok := req[4].(string); !ok || len(s) != 32 {
		t.Errorf("[4] should be 32-char hex, got %v", req[4])
	}
	if req[54] == nil {
		t.Error("[54] is nil")
	}
}

func TestBuildInnerRequest_ContinuationMetadata(t *testing.T) {
	c := &Client{}
	meta := []string{"c_abc", "r_def", "rc_ghi", "", "", "", "", "", "", "ctx"}
	req := c.buildInnerRequest("follow-up", meta, nil, nil, false, true, "UUID")

	outer, ok := req[17].([]any)
	if !ok || len(outer) != 1 {
		t.Fatalf("[17] = %v", req[17])
	}
	inner, ok := outer[0].([]any)
	if !ok || len(inner) != 1 || inner[0] != 1 {
		t.Errorf("[17] = %v, want [[1]]", req[17])
	}

	reqMeta, ok := req[2].([]any)
	if !ok {
		t.Fatal("req[2] not []any")
	}
	if len(reqMeta) != 10 {
		t.Errorf("metadata len = %d, want 10", len(reqMeta))
	}
	if reqMeta[0] != "c_abc" {
		t.Errorf("metadata[0] = %v", reqMeta[0])
	}
	if reqMeta[9] != "ctx" {
		t.Errorf("metadata[9] = %v", reqMeta[9])
	}
}
