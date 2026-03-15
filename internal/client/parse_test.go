package client

import (
	"encoding/json"
	"testing"
)

// ============================================================================
// Frame parsing: Google's length-prefixed framing protocol
//
// Format: <digit_count><content_of_N_utf16_units> repeated.
// Length counts UTF-16 code units starting immediately AFTER the digits
// (includes the \n after digits and the trailing \n).
//
// Example raw response:
//   )]}'
//
//   107
//   [["wrb.fr","MaZiqc","[]",null,null,null,"generic"],["di",105],["af.httprm",104,"-75...",1]]
//   25
//   [["e",4,null,null,143]]
//
// The "107" counts: \n (1) + JSON (105) + \n (1) = 107 UTF-16 units.
// ============================================================================

func TestParseLengthPrefixedFrames_Basic(t *testing.T) {
	// Simulates a stripped response (after removing )]}'  prefix)
	content := "\n107\n" +
		`[["wrb.fr","MaZiqc","[]",null,null,null,"generic"],["di",105],["af.httprm",104,"-7512073092958521608",1]]` +
		"\n25\n" +
		`[["e",4,null,null,143]]` +
		"\n"

	frames := parseLengthPrefixedFrames(content)
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	// First frame should be the wrb.fr JSON
	if frames[0][0] != '[' {
		t.Errorf("frame 0 should start with [, got %q", frames[0][:1])
	}
	// Second frame
	if frames[1][0] != '[' {
		t.Errorf("frame 1 should start with [, got %q", frames[1][:1])
	}
}

func TestParseLengthPrefixedFrames_UTF16Counting(t *testing.T) {
	// CJK characters are 1 UTF-16 unit each (BMP range)
	// Emoji above U+FFFF are 2 UTF-16 units (surrogate pair)
	jsonContent := `["你好"]` // 6 chars = 6 UTF-16 units
	// Frame: \n + "你好" content + \n = 6 + 2 = 8 UTF-16 units
	content := "8\n" + jsonContent + "\n"

	frames := parseLengthPrefixedFrames(content)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0] != jsonContent {
		t.Errorf("expected %q, got %q", jsonContent, frames[0])
	}
}

// ============================================================================
// Envelope parsing: stream response envelope structure
//
// Each frame from StreamGenerate is a JSON array wrapped in extra nesting:
//   [["wrb.fr", null, "<inner_json_string>", ...]]
//
// The actual content is at envelope[0][2] after unwrapping:
//   inner = JSON.parse(envelope[0][2])
//   inner[1] = metadata [cid, rid]
//   inner[4] = candidates array
//   inner[25] = context string (completion marker for normal chats)
//   inner[2] = dict with {"26": "context_string"} (completion for deep research)
// ============================================================================

func TestParseEnvelope_UnwrapsSingleElement(t *testing.T) {
	// Stream frames come wrapped: [["wrb.fr", null, "..."]]
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
	// content[25] is a string → marks completion for normal chats
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
	// Deep research completion: content[2] is a dict with key "26"
	// This is different from normal chats (content[25] as string).
	//
	// HAR evidence (sample01.har entry[54], last frame):
	//   [null, ["cid", "rid"], {"26": "AwAAA...", "44": false}]
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

// ============================================================================
// Metadata structure
//
// Python ChatSession metadata is 10 elements with merge-on-update semantics:
//   [cid, rid, rcid, null, null, null, null, null, null, context_str]
//
// New chat default: ["", "", "", null, null, null, null, null, null, ""]
// After streaming:  ["c_xxx", "r_xxx", "rc_xxx", null*6, "ctx"]
//
// rcid comes from candidate[0], NOT from the metadata array in stream frames.
// context_str comes from content[2]["26"] for deep research.
// ============================================================================

func TestParseEnvelope_RCidInMetadata(t *testing.T) {
	// rcid is at candidate[0], should be injected into metadata[2]
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
	// metadata should be extended to 10 elements with rcid at [2]
	if len(out.Metadata) < 10 {
		t.Fatalf("metadata len = %d, want >=10", len(out.Metadata))
	}
	if out.Metadata[2] != "rc_ghi" {
		t.Errorf("metadata[2] = %q, want rc_ghi", out.Metadata[2])
	}
}

// ============================================================================
// Image extraction from candidate[12]
//
// Web images:     candidate[12][1][i] → URL at [0][0][0], title at [7][0]
// Generated images: candidate[12][7][0][0][i] → URL at [3][3]
//
// Generated image nesting (from HAR):
//   arr[7] = [[[ [item1], [item2] ]]]
//   Navigate: arr[7][0][0] → items list
//   Each item: [null, null, null, [null, 1, "filename", "url", ...], ...]
// ============================================================================

func TestExtractImages_GeneratedImage(t *testing.T) {
	// Simulate candidate[12] structure for generated images
	// arr[7][0][0] = [item]
	// item[3] = [null, 1, "file.png", "https://lh3.googleusercontent.com/xxx"]
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
	// Web images at arr[1][i]
	// URL at [0][0][0], title at [7][0]
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

// ============================================================================
// Deep research plan extraction
//
// Plan data lives in a dict at key "56" or "57" inside the candidate array.
// Must search recursively since the dict can be at any nesting depth.
//
// Dict structure (from HAR):
//   {"56": [title, [[?,label,body], ...], eta, [confirm], [url], modify_payload], "70": state_int}
//
// Fields:
//   payload[0]    = title string
//   payload[1]    = steps array, each step: [?, label_string, body_string]
//   payload[2]    = eta_text string
//   payload[3][0] = confirm_prompt string
// ============================================================================

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

	// Candidate data with dict at some nesting depth
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
	// Plan can also be at key "57" instead of "56"
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

// ============================================================================
// Chat list parsing
//
// RPC response body for LIST_CHATS (MaZiqc):
//   [null, "cursor_string", [[cid, title, null, null, null, [epoch_s, ns], ...], ...]]
//
// Note: items are at [2], NOT [0]. Cursor at [1].
// ============================================================================

func TestParseListChats(t *testing.T) {
	body := `[null,"next_cursor",` +
		`[["c_abc","Chat Title",null,null,null,[1710000000,0],null,null,null,2],` +
		`["c_def","Another",null,null,null,[1710000100,0],null,null,null,2]]]`

	items, cursor, err := parseListChats(body)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "next_cursor" {
		t.Errorf("cursor = %q", cursor)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].Cid != "c_abc" {
		t.Errorf("[0].Cid = %q", items[0].Cid)
	}
	if items[0].Title != "Chat Title" {
		t.Errorf("[0].Title = %q", items[0].Title)
	}
}

func TestParseListChats_Empty(t *testing.T) {
	items, _, err := parseListChats("[]")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// ============================================================================
// Chat turn parsing
//
// RPC response body for READ_CHAT (hNvQHb):
//   [[[turn1, turn2, ...]]]
//
// Each turn:
//   turn[0]    = metadata [cid, rid]
//   turn[2][0][0] = user_prompt
//   turn[3][0][0] = candidate
//     candidate[0]    = rcid
//     candidate[1][0] = text (fallback: candidate[22][0] for card URLs)
//     candidate[12]   = images
//   turn[0][1] = rid
//
// Server returns newest-first; parseReadChat reverses to chronological order.
// Card URLs (http://googleusercontent.com/...) are cleared when images exist.
// ============================================================================

func TestParseReadChat_ReverseOrder(t *testing.T) {
	// Server returns newest first; data[0] is the turns list
	body := `[[` +
		`[["c1","r2"],null,[["prompt2"]],[[["rc2",["response2"]]]]],` +
		`[["c1","r1"],null,[["prompt1"]],[[["rc1",["response1"]]]]]` +
		`]]`

	turns, err := parseReadChat(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("turns = %d", len(turns))
	}
	// Should be chronological: prompt1 first (reversed from server order)
	if turns[0].UserPrompt != "prompt1" {
		t.Errorf("[0].UserPrompt = %q, want prompt1", turns[0].UserPrompt)
	}
	if turns[1].UserPrompt != "prompt2" {
		t.Errorf("[1].UserPrompt = %q, want prompt2", turns[1].UserPrompt)
	}
}

func TestParseReadChat_CardURLFallback(t *testing.T) {
	// When text is a googleusercontent card URL and images exist,
	// the text should be cleared.
	imgData := make([]any, 8)
	imgData[7] = []any{[]any{[]any{
		[]any{nil, nil, nil, []any{nil, 1, "f.png", "https://lh3.googleusercontent.com/img"}, nil, "$s"},
	}}}
	candidate := []any{
		"rc_1",
		[]any{"http://googleusercontent.com/image_generation_content/0"},
		nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, imgData,
	}

	turn := []any{
		[]any{"c_1", "r_1"},
		nil,
		[]any{[]any{"draw something"}},
		[]any{[]any{candidate}},
	}

	body, _ := json.Marshal([]any{[]any{turn}})
	turns, err := parseReadChat(string(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("turns = %d", len(turns))
	}
	// Text should be cleared (was card URL, has images)
	if turns[0].AssistantResponse != "" {
		t.Errorf("AssistantResponse should be empty, got %q", turns[0].AssistantResponse)
	}
	if len(turns[0].Images) != 1 {
		t.Fatalf("Images = %d, want 1", len(turns[0].Images))
	}
}

// ============================================================================
// Deep research result extraction
//
// The deep research report is stored at cand[30][0][4] in raw turn data,
// NOT in the normal text field (which shows a card URL placeholder).
//
// Sources are at cand[30][0][5][0] as a dict with key "44" containing
// citation groups.
// ============================================================================

func TestExtractResearchResultFromRaw(t *testing.T) {
	// Build a minimal raw turn with deep research data at [30][0][4]
	report := "# Deep Research Report\n\nThis is a long report about the topic..."
	// Pad to >200 chars
	for len(report) < 250 {
		report += " More content here."
	}

	drData := make([]any, 6)
	drData[4] = report

	cand := make([]any, 31)
	cand[30] = []any{drData}

	turn := make([]any, 4)
	turn[3] = []any{[]any{cand}}

	turnJSON, _ := json.Marshal(turn)
	rawTurns := []json.RawMessage{turnJSON}

	text, _ := extractResearchResultFromRaw(rawTurns)
	if text != report {
		t.Errorf("text len = %d, want %d", len(text), len(report))
	}
}

// ============================================================================
// Inner request array construction
//
// The request to StreamGenerate uses a 69-element array.
// Key indices for deep research (from HAR analysis):
//
//   [0]  = [prompt, 0, null, null, null, null, 0]  # message content
//   [1]  = ["en"]                                   # language
//   [2]  = metadata (10 elements for new chat, preserved for continuations)
//   [3]  = "!" + url_safe_token(2600)               # deep research only
//   [4]  = hex_uuid                                  # deep research only
//   [6]  = [0]
//   [7]  = 1                                        # enable snapshot streaming
//   [10] = 1
//   [11] = 0
//   [17] = [[0]] new chat / [[1]] existing chat
//   [18] = 0
//   [27] = 1
//   [30] = [4]
//   [41] = [1]
//   [45] = null (NOT 1 — temporary flag only when explicitly set)
//   [49] = 1                                        # deep research only
//   [53] = 0
//   [54] = [[[[[1]]]]]                              # deep research only
//   [55] = [[1]]                                    # deep research only
//   [59] = "UUID"                                   # per-request UUID
//   [61] = []
//   [68] = 2 for deep research / 1 for normal
// ============================================================================

func TestBuildInnerRequest_NewChat(t *testing.T) {
	c := &Client{}
	req := c.buildInnerRequest("hello", nil, false, false, "TEST-UUID")

	if len(req) != 69 {
		t.Fatalf("len = %d, want 69", len(req))
	}

	// Metadata should be 10 elements for new chat
	meta, ok := req[2].([]any)
	if !ok {
		t.Fatal("req[2] not []any")
	}
	if len(meta) != 10 {
		t.Errorf("metadata len = %d, want 10", len(meta))
	}

	// [17] should be [[0]] for new chat
	if req[17] == nil {
		t.Fatal("[17] is nil")
	}

	// [45] should be nil (NOT temporary)
	if req[45] != nil {
		t.Errorf("[45] = %v, want nil", req[45])
	}

	// [68] should be 1 for normal
	if req[68] != 1 {
		t.Errorf("[68] = %v, want 1", req[68])
	}

	// UUID
	if req[59] != "TEST-UUID" {
		t.Errorf("[59] = %v", req[59])
	}
}

func TestBuildInnerRequest_DeepResearch(t *testing.T) {
	c := &Client{}
	req := c.buildInnerRequest("research topic", nil, true, false, "UUID")

	// Deep research flags
	if req[49] != 1 {
		t.Errorf("[49] = %v, want 1", req[49])
	}
	if req[68] != 2 {
		t.Errorf("[68] = %v, want 2", req[68])
	}
	// [3] should be "!" + token
	if s, ok := req[3].(string); !ok || s[0] != '!' {
		t.Errorf("[3] should start with !, got %v", req[3])
	}
	// [4] should be hex UUID
	if s, ok := req[4].(string); !ok || len(s) != 32 {
		t.Errorf("[4] should be 32-char hex, got %v", req[4])
	}
	// [54] = [[[[[1]]]]]
	if req[54] == nil {
		t.Error("[54] is nil")
	}
}

func TestBuildInnerRequest_ContinuationMetadata(t *testing.T) {
	c := &Client{}
	meta := []string{"c_abc", "r_def", "rc_ghi", "", "", "", "", "", "", "ctx"}
	req := c.buildInnerRequest("follow-up", meta, false, true, "UUID")

	// [17] should be [[1]] for existing chat
	outer, ok := req[17].([]any)
	if !ok || len(outer) != 1 {
		t.Fatalf("[17] = %v", req[17])
	}
	inner, ok := outer[0].([]any)
	if !ok || len(inner) != 1 || inner[0] != 1 {
		t.Errorf("[17] = %v, want [[1]]", req[17])
	}

	// Metadata should preserve all 10 elements
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

// ============================================================================
// wrb.fr extraction from batchexecute responses
//
// batchexecute responses use same length-prefixed framing. The wrb.fr items
// can be nested 1-2 levels deep in the parsed JSON:
//   [[["wrb.fr", "rpcId", "body_json", ...]]]
//
// Reject code is at item[5][0] (e.g., 7 = permission denied).
// ============================================================================

func TestFindWrbFrItems_Nested(t *testing.T) {
	// Items nested one level: [[["wrb.fr", ...]]]
	inner := []any{"wrb.fr", "MaZiqc", "[]", nil, nil, nil, "generic"}
	arr := []any{[]any{inner}}

	items := findWrbFrItems(arr)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0][1] != "MaZiqc" {
		t.Errorf("rpcId = %v", items[0][1])
	}
}

func TestExtractRPCBody_RejectCode(t *testing.T) {
	// wrb.fr with reject code at [5][0]
	inner := []any{"wrb.fr", "MaZiqc", "[]", nil, nil, []any{float64(7)}, "generic"}
	frame, _ := json.Marshal([]any{[]any{inner}})
	response := "20\n" + string(frame) + "\n"

	// Need to match the actual length
	response = adjustFrameLength(string(frame))

	_, rejectCode, err := extractRPCBody(response, "MaZiqc")
	if err != nil {
		t.Fatal(err)
	}
	if rejectCode != 7 {
		t.Errorf("rejectCode = %d, want 7", rejectCode)
	}
}

// helper to create a properly length-prefixed frame
func adjustFrameLength(json string) string {
	// Count UTF-16 units of \n + json + \n
	runes := []rune(json)
	units := 2 // \n before + \n after
	for _, r := range runes {
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return "\n" + string(rune('0'+units/100)) + string(rune('0'+(units/10)%10)) + string(rune('0'+units%10)) + // rough
		"\n" + json + "\n"
}
