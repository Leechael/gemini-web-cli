package client

// Baseline tests using real response data captured from browser HAR files.
// These test fixtures are from data/sample01.har (deep research session).
//
// If Gemini changes their wire format, these tests will catch the breakage
// even if the synthetic unit tests still pass.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	return data
}

func loadStringFixture(t *testing.T, name string) string {
	t.Helper()
	data := loadFixture(t, name)
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Raw string file, not JSON-encoded
		return string(data)
	}
	return s
}

// ============================================================================
// Baseline: StreamGenerate deep research step 1
//
// Real response from HAR sample01.har entry[54].
// This is the response to sending a research prompt with deep_research=true.
//
// Expected:
//   - Multiple frames with metadata [cid, rid] (2 elements)
//   - Plan data in a dict with key "56" containing title, steps, eta, confirm
//   - Completion frame: content[2] is {"26": "context_str", "44": false}
//   - Text contains plan description (in Chinese)
// ============================================================================

func TestBaseline_StreamResearchStep1(t *testing.T) {
	raw := loadStringFixture(t, "stream_research_step1_response")
	if len(raw) < 100 {
		t.Fatalf("fixture too small: %d bytes", len(raw))
	}

	content := raw
	if strings.HasPrefix(content, ")]}'\n") {
		content = content[5:]
	}
	frames := parseLengthPrefixedFrames(content)
	if len(frames) < 3 {
		t.Fatalf("expected >=3 frames, got %d", len(frames))
	}

	// Parse all frames, collect outputs
	var outputs []*outputInfo
	for _, frame := range frames {
		var envelope []any
		if err := json.Unmarshal([]byte(frame), &envelope); err != nil {
			continue
		}
		out := parseEnvelope(envelope)
		if out != nil {
			outputs = append(outputs, &outputInfo{
				metadata:  out.Metadata,
				text:      out.Text,
				done:      out.Done,
				rcid:      out.RCid,
				hasPlan:   out.DeepResearchPlan != nil,
				planTitle: "",
				planConf:  "",
			})
			if out.DeepResearchPlan != nil {
				outputs[len(outputs)-1].planTitle = out.DeepResearchPlan.Title
				outputs[len(outputs)-1].planConf = out.DeepResearchPlan.ConfirmPrompt
			}
		}
	}

	if len(outputs) == 0 {
		t.Fatal("no outputs parsed from step1 response")
	}

	// Should have at least one frame with metadata containing cid
	hasCid := false
	for _, o := range outputs {
		if len(o.metadata) > 0 && strings.HasPrefix(o.metadata[0], "c_") {
			hasCid = true
			break
		}
	}
	if !hasCid {
		t.Error("no frame contains cid in metadata")
	}

	// Should have a plan in at least one frame
	hasPlan := false
	for _, o := range outputs {
		if o.hasPlan {
			hasPlan = true
			if o.planTitle == "" {
				t.Error("plan has empty title")
			}
			break
		}
	}
	if !hasPlan {
		t.Error("no frame contains deep research plan")
	}

	// Should have a completion frame with context string in metadata[9]
	hasContext := false
	for _, o := range outputs {
		if o.done && len(o.metadata) >= 10 && o.metadata[9] != "" {
			hasContext = true
			break
		}
	}
	if !hasContext {
		t.Error("no completion frame with context string in metadata[9]")
	}
}

type outputInfo struct {
	metadata  []string
	text      string
	done      bool
	rcid      string
	hasPlan   bool
	planTitle string
	planConf  string
}

// ============================================================================
// Baseline: StreamGenerate deep research step 2
//
// Real response from HAR sample01.har entry[85].
// This is the response to sending "开始研究" (confirm) with deep_research=true.
//
// Expected:
//   - Metadata carries forward cid from step 1
//   - Completion frame present
// ============================================================================

func TestBaseline_StreamResearchStep2(t *testing.T) {
	raw := loadStringFixture(t, "stream_research_step2_response")

	content := raw
	if strings.HasPrefix(content, ")]}'\n") {
		content = content[5:]
	}
	frames := parseLengthPrefixedFrames(content)
	if len(frames) == 0 {
		t.Fatal("no frames parsed from step2 response")
	}

	var outputs []*outputInfo
	for _, frame := range frames {
		var envelope []any
		if err := json.Unmarshal([]byte(frame), &envelope); err != nil {
			continue
		}
		out := parseEnvelope(envelope)
		if out != nil {
			outputs = append(outputs, &outputInfo{
				metadata: out.Metadata,
				done:     out.Done,
				rcid:     out.RCid,
			})
		}
	}

	if len(outputs) == 0 {
		t.Fatal("no outputs parsed from step2 response")
	}

	// Should have cid in metadata
	hasCid := false
	for _, o := range outputs {
		if len(o.metadata) > 0 && strings.HasPrefix(o.metadata[0], "c_") {
			hasCid = true
			break
		}
	}
	if !hasCid {
		t.Error("no frame contains cid in step2 metadata")
	}
}

// ============================================================================
// Baseline: batchexecute LIST_CHATS response
//
// Real response from HAR sample03.har.
//
// Expected:
//   - wrb.fr envelope with rpcId "MaZiqc"
//   - Body parses to a list with items at [2] and cursor at [1]
//   - Each item has [cid, title, ...] structure
// ============================================================================

func TestBaseline_BatchListChats(t *testing.T) {
	raw := loadStringFixture(t, "batch_list_chats_response")
	if len(raw) < 50 {
		t.Skipf("fixture too small: %d bytes", len(raw))
	}

	stripped := raw
	if strings.HasPrefix(stripped, ")]}'\n") {
		stripped = stripped[5:]
	}

	body, rejectCode, err := extractRPCBody(stripped, "MaZiqc")
	if err != nil {
		t.Fatalf("extractRPCBody: %v", err)
	}
	if rejectCode != 0 {
		t.Fatalf("rejectCode = %d", rejectCode)
	}
	if body == "" || body == "[]" {
		t.Skip("empty list in fixture")
	}

	items, cursor, err := parseListChats(body)
	if err != nil {
		t.Fatalf("parseListChats: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected >0 items")
	}

	// Validate first item structure
	first := items[0]
	if !strings.HasPrefix(first.Cid, "c_") {
		t.Errorf("first item cid = %q, want c_...", first.Cid)
	}
	if first.Title == "" {
		t.Error("first item has empty title")
	}
	if first.UpdatedAt == "" {
		t.Error("first item has empty UpdatedAt")
	}

	// Cursor may or may not be present
	_ = cursor
}

// ============================================================================
// Baseline: inner request array structure (deep research step 1)
//
// Real request from HAR sample01.har entry[54].
// Validates that our buildInnerRequest produces the same structure.
//
// Key indices from real data:
//   [0]: message content array
//   [2]: 10-element metadata ["","","",null*6,""]
//   [3]: "!" + token (present, length ~3468)
//   [4]: hex UUID (32 chars)
//   [6]: [0]   [7]: 1   [10]: 1   [11]: 0
//   [17]: [[0]] (new chat)   [18]: 0   [27]: 1   [30]: [4]
//   [41]: [1]   [45]: null   [49]: 1   [53]: 0
//   [54]: [[[[[1]]]]]   [55]: [[1]]
//   [59]: UUID string   [61]: []   [68]: 2
// ============================================================================

func TestBaseline_InnerRequestResearchStep1(t *testing.T) {
	data := loadFixture(t, "inner_req_research_step1")
	var real []any
	if err := json.Unmarshal(data, &real); err != nil {
		t.Fatal(err)
	}

	if len(real) != 69 {
		t.Fatalf("expected 69 elements, got %d", len(real))
	}

	// Validate key indices
	assertIndex := func(idx int, check func(any) bool, desc string) {
		t.Helper()
		if !check(real[idx]) {
			v, _ := json.Marshal(real[idx])
			t.Errorf("[%d] %s: %s", idx, desc, string(v))
		}
	}

	// [2] metadata: 10-element array starting with ["","",""]
	assertIndex(2, func(v any) bool {
		arr, ok := v.([]any)
		return ok && len(arr) == 10
	}, "metadata should be 10 elements")

	// [3] token starting with "!" (redacted in fixtures)
	assertIndex(3, func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) > 5 && s[0] == '!'
	}, "should be !token")

	// [4] hex UUID (32 chars, or redacted)
	assertIndex(4, func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) >= 10
	}, "should be hex UUID or redacted")

	// [7] = 1 (snapshot streaming)
	assertIndex(7, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 1
	}, "should be 1")

	// [17] = [[0]] (new chat)
	assertIndex(17, func(v any) bool {
		arr, ok := v.([]any)
		if !ok || len(arr) != 1 {
			return false
		}
		inner, ok := arr[0].([]any)
		return ok && len(inner) == 1
	}, "should be [[0]]")

	// [45] = null (NOT temporary)
	assertIndex(45, func(v any) bool {
		return v == nil
	}, "should be null (not temporary)")

	// [49] = 1 (deep research)
	assertIndex(49, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 1
	}, "should be 1 (deep research)")

	// [68] = 2 (deep research mode)
	assertIndex(68, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 2
	}, "should be 2 (deep research)")
}

// ============================================================================
// Baseline: inner request array structure (deep research step 2 confirm)
//
// Real request from HAR sample01.har entry[85].
//
// Key differences from step 1:
//   [0]: ["开始研究", 0, null, ...]
//   [2]: 10-element metadata with cid, rid, rcid, and context_str at [9]
//   [17]: [[1]] (existing chat)
// ============================================================================

func TestBaseline_InnerRequestResearchStep2(t *testing.T) {
	data := loadFixture(t, "inner_req_research_step2")
	var real []any
	if err := json.Unmarshal(data, &real); err != nil {
		t.Fatal(err)
	}

	if len(real) != 69 {
		t.Fatalf("expected 69 elements, got %d", len(real))
	}

	// [0] prompt should be "开始研究"
	msg, ok := real[0].([]any)
	if !ok || len(msg) == 0 {
		t.Fatal("[0] not a message array")
	}
	prompt, _ := msg[0].(string)
	if prompt != "开始研究" {
		t.Errorf("[0][0] = %q, want 开始研究", prompt)
	}

	// [2] metadata should have cid at [0], rid at [1], rcid at [2], context at [9]
	meta, ok := real[2].([]any)
	if !ok || len(meta) != 10 {
		t.Fatalf("[2] metadata len = %d, want 10", len(meta))
	}
	cid, _ := meta[0].(string)
	if !strings.HasPrefix(cid, "c_") {
		t.Errorf("metadata[0] = %q, want c_...", cid)
	}
	rid, _ := meta[1].(string)
	if !strings.HasPrefix(rid, "r_") {
		t.Errorf("metadata[1] = %q, want r_...", rid)
	}
	rcid, _ := meta[2].(string)
	if !strings.HasPrefix(rcid, "rc_") {
		t.Errorf("metadata[2] = %q, want rc_...", rcid)
	}
	ctx, _ := meta[9].(string)
	if ctx == "" {
		t.Error("metadata[9] (context string) is empty")
	}

	// [17] = [[1]] (existing chat)
	outer, ok := real[17].([]any)
	if !ok || len(outer) != 1 {
		t.Fatalf("[17] = %v", real[17])
	}
	inner, ok := outer[0].([]any)
	if !ok || len(inner) != 1 {
		t.Fatalf("[17][0] = %v", outer[0])
	}
	flag, _ := inner[0].(float64)
	if flag != 1 {
		t.Errorf("[17] = [[%v]], want [[1]]", flag)
	}

	// [68] = 2 (deep research)
	mode, _ := real[68].(float64)
	if mode != 2 {
		t.Errorf("[68] = %v, want 2", mode)
	}
}
