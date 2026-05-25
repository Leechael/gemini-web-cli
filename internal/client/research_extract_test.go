package client

import (
	"encoding/json"
	"testing"
)

func TestExtractResearchResultFromRaw(t *testing.T) {
	report := "# Deep Research Report\n\nThis is a long report about the topic..."
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
