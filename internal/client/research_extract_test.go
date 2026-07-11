package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractResearchResultFromRaw(t *testing.T) {
	report := testResearchReport()
	rawTurns := []json.RawMessage{testResearchDoneTurn(report)}

	state := inspectResearchStateFromRaw(rawTurns)
	if state.state != "done" || state.text != report {
		t.Errorf("state = %+v, want completed report", state)
	}
	if state.sources[1].URL != "https://example.com/source" || state.sources[1].Title != "Sample source" {
		t.Fatalf("sources = %+v", state.sources)
	}
}

func TestExtractResearchResultFromRawStopsAtNewerRunningTurn(t *testing.T) {
	rawTurns := []json.RawMessage{
		testResearchRunningTurn(),
		testResearchDoneTurn(testResearchReport()),
	}

	state := inspectResearchStateFromRaw(rawTurns)
	if state.state != "running" || state.text != "" || state.sources != nil {
		t.Fatalf("state = %+v, want running", state)
	}
}

func TestInspectResearchStatusFromRawPendingConfirm(t *testing.T) {
	for _, key := range []string{"56", "57"} {
		t.Run("key "+key, func(t *testing.T) {
			state := inspectResearchStateFromRaw([]json.RawMessage{testResearchPendingTurn(key)})
			if state.state != "pending_confirm" {
				t.Fatalf("state = %+v, want pending_confirm", state)
			}
		})
	}
}

func TestClassifyResearchTextRecognizesCompletedReports(t *testing.T) {
	longMarkdown := "# Report\n\n" + strings.Repeat("research result ", 150)
	for _, text := range []string{
		"我已经完成了研究，结果如下。",
		"I have completed the research. Here is the result.",
		"研究完成后，我们发现该方案可行。",
		longMarkdown,
		"\r\n" + longMarkdown,
		"\ufeff\n" + longMarkdown,
	} {
		status := classifyResearchText(text)
		if status.State != "done" || status.TextLen != len(text) {
			t.Fatalf("classifyResearchText() = %+v, want done", status)
		}
	}
}

func TestGetDeepResearchResultPrefersNewestPlainTextReportOverOlderRunningTurn(t *testing.T) {
	report := "# Report\n\n" + strings.Repeat("completed research result ", 100)
	body, err := json.Marshal([]any{[]json.RawMessage{
		testResearchTextTurn(report),
		testResearchRunningTurn(),
	}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", string(body), 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	text, sources, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err != nil {
		t.Fatalf("GetDeepResearchResult: %v", err)
	}
	if text != report || sources != nil {
		t.Fatalf("text len=%d sources=%+v", len(text), sources)
	}
}

func TestGetDeepResearchResultDoesNotReturnOlderReportWhileNewestTurnIsRunning(t *testing.T) {
	report := "# Report\n\n" + strings.Repeat("completed research result ", 100)
	body, err := json.Marshal([]any{[]json.RawMessage{
		testResearchRunningTurn(),
		testResearchTextTurn(report),
	}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", string(body), 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	text, sources, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err == nil || !strings.Contains(err.Error(), "state=running") {
		t.Fatalf("text = %q, sources = %+v, err = %v; want running error", text, sources, err)
	}
}

func TestGetDeepResearchResultDecodedFallbackDoesNotReturnOlderReportWhileLatestTurnIsRunning(t *testing.T) {
	report := "# Report\n\n" + strings.Repeat("completed research result ", 100)
	body, err := json.Marshal([]any{[]json.RawMessage{
		testResearchRunningTurn(),
		testResearchTextTurn(report),
	}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write(makeTestBatchResponse("hNvQHb", "", 7))
			return
		}
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", string(body), 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	text, sources, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err == nil || !strings.Contains(err.Error(), "state=running") {
		t.Fatalf("text = %q, sources = %+v, err = %v; want running error", text, sources, err)
	}
}

func TestGetDeepResearchResultPreservesRawReadErrorWhenFallbackHasNoResult(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write(makeTestBatchResponse("hNvQHb", "", 7))
			return
		}
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", "", 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	_, _, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err == nil || !strings.Contains(err.Error(), "read_chat rejected with code=7") {
		t.Fatalf("err = %v, want raw read error", err)
	}
}

func TestGetDeepResearchResultDecodedFallbackReturnsShortPlainReport(t *testing.T) {
	report := "简短研究结论：该方案可行。"
	body, err := json.Marshal([]any{[]json.RawMessage{testResearchTextTurn(report)}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write(makeTestBatchResponse("hNvQHb", "", 7))
			return
		}
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", string(body), 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	text, sources, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err != nil {
		t.Fatalf("GetDeepResearchResult: %v", err)
	}
	if text != report || sources != nil {
		t.Fatalf("text = %q, sources = %+v", text, sources)
	}
}

func TestGetDeepResearchResultReturnsCompletedTextFallback(t *testing.T) {
	report := "# Report\n\n" + strings.Repeat("completed research result ", 100)
	turn := testResearchTextTurn(report)
	body, err := json.Marshal([]any{[]json.RawMessage{turn}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", string(body), 0))
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.httpClient = srv.Client()
	text, sources, err := c.GetDeepResearchResult(t.Context(), "c_research")
	if err != nil {
		t.Fatalf("GetDeepResearchResult: %v", err)
	}
	if text != report || sources != nil {
		t.Fatalf("text len=%d sources=%+v", len(text), sources)
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

func testResearchTextTurn(text string) json.RawMessage {
	cand := make([]any, 2)
	cand[1] = []any{text}
	return testResearchTurn(cand)
}

func testResearchRunningTurn() json.RawMessage {
	cand := make([]any, 13)
	cand[1] = []any{"我这就开始。研究完成后，我会告诉你。\nhttp://googleusercontent.com/immersive_entry_chip/0\n"}
	cand[12] = []any{nil, nil, nil, nil, nil, nil, nil, nil, map[string]any{"70": float64(3)}}
	return testResearchTurn(cand)
}

func testResearchPendingTurn(key string) json.RawMessage {
	cand := make([]any, 13)
	cand[1] = []any{"这是我拟定的方案。"}
	cand[12] = []any{nil, nil, nil, nil, nil, nil, nil, nil, map[string]any{key: []any{"Plan"}}}
	return testResearchTurn(cand)
}

func testResearchTurn(cand []any) json.RawMessage {
	turn := make([]any, 4)
	turn[3] = []any{[]any{cand}}
	turnJSON, _ := json.Marshal(turn)
	return turnJSON
}
