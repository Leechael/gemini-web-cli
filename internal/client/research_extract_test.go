package client

import (
	"encoding/json"
	"testing"
)

func TestExtractResearchResultFromRaw(t *testing.T) {
	report := testResearchReport()
	rawTurns := []json.RawMessage{testResearchDoneTurn(report)}

	text, sources := extractResearchResultFromRaw(rawTurns)
	if text != report {
		t.Errorf("text len = %d, want %d", len(text), len(report))
	}
	if sources[1].URL != "https://example.com/source" || sources[1].Title != "Sample source" {
		t.Fatalf("sources = %+v", sources)
	}
}

func TestExtractResearchResultFromRawStopsAtNewerRunningTurn(t *testing.T) {
	rawTurns := []json.RawMessage{
		testResearchRunningTurn(),
		testResearchDoneTurn(testResearchReport()),
	}

	text, sources := extractResearchResultFromRaw(rawTurns)
	if text != "" || sources != nil {
		t.Fatalf("got text len=%d sources=%+v, want no result while latest research turn is running", len(text), sources)
	}

	status := inspectResearchStatusFromRaw(rawTurns)
	if status == nil || status.State != "running" {
		t.Fatalf("status = %+v, want running", status)
	}
}

func TestInspectResearchStatusFromRawPendingConfirm(t *testing.T) {
	status := inspectResearchStatusFromRaw([]json.RawMessage{testResearchPendingTurn()})
	if status == nil || status.State != "pending_confirm" {
		t.Fatalf("status = %+v, want pending_confirm", status)
	}
}

func testResearchReport() string {
	report := "# Deep Research Report\n\nThis is a long report about the topic..."
	for len(report) < 250 {
		report += " More content here."
	}
	return report
}

func testResearchDoneTurn(report string) json.RawMessage {
	drData := make([]any, 6)
	drData[4] = report
	drData[5] = []any{map[string]any{
		"44": []any{[]any{nil, []any{[]any{nil, nil, nil, []any{[]any{nil, "https://example.com/source", "Sample source"}, float64(1)}}}}},
	}}

	cand := make([]any, 31)
	cand[30] = []any{drData}
	return testResearchTurn(cand)
}

func testResearchRunningTurn() json.RawMessage {
	cand := make([]any, 13)
	cand[1] = []any{"我这就开始。研究完成后，我会告诉你。\nhttp://googleusercontent.com/immersive_entry_chip/0\n"}
	cand[12] = []any{nil, nil, nil, nil, nil, nil, nil, nil, map[string]any{"70": float64(3)}}
	return testResearchTurn(cand)
}

func testResearchPendingTurn() json.RawMessage {
	cand := make([]any, 13)
	cand[1] = []any{"这是我拟定的方案。\nhttp://googleusercontent.com/deep_research_confirmation_content/0\n"}
	cand[12] = []any{nil, nil, nil, nil, nil, nil, nil, nil, map[string]any{"56": []any{"Plan"}, "70": float64(2)}}
	return testResearchTurn(cand)
}

func testResearchTurn(cand []any) json.RawMessage {
	turn := make([]any, 4)
	turn[3] = []any{[]any{cand}}
	turnJSON, _ := json.Marshal(turn)
	return turnJSON
}
