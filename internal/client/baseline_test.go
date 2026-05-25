package client

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
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
		return string(data)
	}
	return s
}

func TestBaseline_StreamResearchStep1(t *testing.T) {
	raw := loadStringFixture(t, "stream_research_step1_response")
	if len(raw) < 100 {
		t.Fatalf("fixture too small: %d bytes", len(raw))
	}

	frames := protocol.ParseLengthPrefixedFrames(protocol.StripResponsePrefix([]byte(raw)))
	if len(frames) < 3 {
		t.Fatalf("expected >=3 frames, got %d", len(frames))
	}

	var outputs []*outputInfo
	for _, frame := range frames {
		var envelope []any
		if err := json.Unmarshal(frame, &envelope); err != nil {
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

func TestBaseline_StreamResearchStep2(t *testing.T) {
	raw := loadStringFixture(t, "stream_research_step2_response")

	frames := protocol.ParseLengthPrefixedFrames(protocol.StripResponsePrefix([]byte(raw)))
	if len(frames) == 0 {
		t.Fatal("no frames parsed from step2 response")
	}

	var outputs []*outputInfo
	for _, frame := range frames {
		var envelope []any
		if err := json.Unmarshal(frame, &envelope); err != nil {
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

func TestBaseline_BatchListChats(t *testing.T) {
	raw := loadStringFixture(t, "batch_list_chats_response")
	if len(raw) < 50 {
		t.Skipf("fixture too small: %d bytes", len(raw))
	}

	body, rejectCode, err := protocol.ExtractRPCBody(protocol.StripResponsePrefix([]byte(raw)), "MaZiqc")
	if err != nil {
		t.Fatalf("ExtractRPCBody: %v", err)
	}
	if rejectCode != 0 {
		t.Fatalf("rejectCode = %d", rejectCode)
	}
	if len(body) == 0 || string(body) == "[]" {
		t.Skip("empty list in fixture")
	}

	items, cursor, err := rpcs.DecodeListChats(body)
	if err != nil {
		t.Fatalf("DecodeListChats: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("expected >0 items")
	}

	first := items[0]
	if !strings.HasPrefix(first.Cid, "c_") {
		t.Errorf("first item cid = %q, want c_...", first.Cid)
	}
	if first.Title == "" {
		t.Error("first item has empty title")
	}
	if first.UpdatedAtUnix == 0 {
		t.Error("first item has empty UpdatedAtUnix")
	}

	_ = cursor
}

func TestBaseline_InnerRequestResearchStep1(t *testing.T) {
	data := loadFixture(t, "inner_req_research_step1")
	var real []any
	if err := json.Unmarshal(data, &real); err != nil {
		t.Fatal(err)
	}

	if len(real) != 69 {
		t.Fatalf("expected 69 elements, got %d", len(real))
	}

	assertIndex := func(idx int, check func(any) bool, desc string) {
		t.Helper()
		if !check(real[idx]) {
			v, _ := json.Marshal(real[idx])
			t.Errorf("[%d] %s: %s", idx, desc, string(v))
		}
	}

	assertIndex(2, func(v any) bool {
		arr, ok := v.([]any)
		return ok && len(arr) == 10
	}, "metadata should be 10 elements")

	assertIndex(3, func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) > 5 && s[0] == '!'
	}, "should be !token")

	assertIndex(4, func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) >= 10
	}, "should be hex UUID or redacted")

	assertIndex(7, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 1
	}, "should be 1")

	assertIndex(17, func(v any) bool {
		arr, ok := v.([]any)
		if !ok || len(arr) != 1 {
			return false
		}
		inner, ok := arr[0].([]any)
		return ok && len(inner) == 1
	}, "should be [[0]]")

	assertIndex(45, func(v any) bool {
		return v == nil
	}, "should be null (not temporary)")

	assertIndex(49, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 1
	}, "should be 1 (deep research)")

	assertIndex(68, func(v any) bool {
		f, ok := v.(float64)
		return ok && f == 2
	}, "should be 2 (deep research)")
}

func TestBaseline_InnerRequestResearchStep2(t *testing.T) {
	data := loadFixture(t, "inner_req_research_step2")
	var real []any
	if err := json.Unmarshal(data, &real); err != nil {
		t.Fatal(err)
	}

	if len(real) != 69 {
		t.Fatalf("expected 69 elements, got %d", len(real))
	}

	msg, ok := real[0].([]any)
	if !ok || len(msg) == 0 {
		t.Fatal("[0] not a message array")
	}
	prompt, _ := msg[0].(string)
	if prompt != "开始研究" {
		t.Errorf("[0][0] = %q, want 开始研究", prompt)
	}

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

	mode, _ := real[68].(float64)
	if mode != 2 {
		t.Errorf("[68] = %v, want 2", mode)
	}
}
