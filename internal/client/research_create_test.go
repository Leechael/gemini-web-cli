package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

func TestDeepResearchPlanFromOutputRejectsNil(t *testing.T) {
	_, _, err := deepResearchPlanFromOutput(nil)
	if err == nil || err.Error() != "deep research plan response was empty" {
		t.Fatalf("err = %v, want empty-response error", err)
	}
}

func TestDeepResearchPlanFromOutputRejectsMissingChatID(t *testing.T) {
	_, _, err := deepResearchPlanFromOutput(&types.ModelOutput{
		DeepResearchPlan: &types.DeepResearchPlanData{Title: "Plan"},
	})
	if err == nil || err.Error() != "no chat ID returned from deep research" {
		t.Fatalf("err = %v, want missing-chat-ID error", err)
	}
}

func TestDeepResearchPlanFromOutputRejectsMissingStructuredPlan(t *testing.T) {
	text := strings.Repeat("界", 1205)
	_, _, err := deepResearchPlanFromOutput(&types.ModelOutput{
		Metadata: []string{"c_valid"},
		Text:     text,
	})
	want := "Gemini did not return a deep research plan. Preview: " + strings.Repeat("界", 1200)
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want Unicode-safe 1200-rune preview", err)
	}
	if !utf8.ValidString(err.Error()) {
		t.Fatal("preview error is not valid UTF-8")
	}
}

func TestDeepResearchPlanFromOutputCopiesPlanAndPreservesPrompt(t *testing.T) {
	data := &types.DeepResearchPlanData{
		Title:         "Research title",
		Steps:         []string{"Find sources", "Compare results"},
		ETAText:       "About five minutes",
		ConfirmPrompt: "Commencer la recherche",
	}
	plan, confirmPrompt, err := deepResearchPlanFromOutput(&types.ModelOutput{
		Metadata:         []string{"c_validated", "r_plan"},
		DeepResearchPlan: data,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Cid != "c_validated" || plan.Title != data.Title || plan.ETAText != data.ETAText {
		t.Fatalf("plan = %#v", plan)
	}
	if len(plan.Steps) != 2 || plan.Steps[0] != "Find sources" || plan.Steps[1] != "Compare results" {
		t.Fatalf("steps = %#v", plan.Steps)
	}
	data.Steps[0] = "mutated"
	if plan.Steps[0] != "Find sources" {
		t.Fatal("returned plan aliases source steps")
	}
	if confirmPrompt != "Commencer la recherche" {
		t.Fatalf("confirm prompt = %q", confirmPrompt)
	}
}

func TestDeepResearchPlanFromOutputFallsBackToEnglishPrompt(t *testing.T) {
	_, confirmPrompt, err := deepResearchPlanFromOutput(&types.ModelOutput{
		Metadata:         []string{"c_validated"},
		DeepResearchPlan: &types.DeepResearchPlanData{Title: "Plan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if confirmPrompt != "Start research" {
		t.Fatalf("confirm prompt = %q, want Start research", confirmPrompt)
	}
}

func TestCreateAndStartDeepResearchStopsWhenPlanMissing(t *testing.T) {
	streamPrompts := make([]string, 0, 1)
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "StreamGenerate") {
			prompt, err := decodeResearchStreamPrompt(r)
			if err != nil {
				t.Errorf("decode stream prompt: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			streamPrompts = append(streamPrompts, prompt)
			_, _ = w.Write(makeResearchPlanStreamBody(t, "ordinary response", "c_created", "", false))
			return
		}
		writeEmptyResearchBatch(w, r)
	}))
	defer srv.Close()

	_, err := c.CreateAndStartDeepResearch(t.Context(), "research prompt", nil)
	if err == nil || !strings.HasPrefix(err.Error(), "Gemini did not return a deep research plan. Preview: ordinary response") {
		t.Fatalf("err = %v, want missing-plan error", err)
	}
	wantPrompt := deepResearchPromptPrefix + "research prompt"
	if len(streamPrompts) != 1 || streamPrompts[0] != wantPrompt {
		t.Fatalf("stream prompts = %#v, want %#v", streamPrompts, []string{wantPrompt})
	}
}

func TestCreateAndStartDeepResearchUsesServerPromptAndReturnsPlan(t *testing.T) {
	streamPrompts := make([]string, 0, 2)
	statusBody := makeResearchStatusBody(t, testResearchRunningTurn())
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "StreamGenerate") {
			prompt, err := decodeResearchStreamPrompt(r)
			if err != nil {
				t.Errorf("decode stream prompt: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			streamPrompts = append(streamPrompts, prompt)
			if len(streamPrompts) == 1 {
				_, _ = w.Write(makeResearchPlanStreamBody(t, "plan ready", "c_validated", "Commencer la recherche", true))
				return
			}
			_, _ = w.Write(makeStreamBody(t, "confirmation accepted", true))
			return
		}
		if r.URL.Query().Get("rpcids") == "hNvQHb" {
			_, _ = w.Write(makeTestBatchResponse("hNvQHb", statusBody, 0))
			return
		}
		writeEmptyResearchBatch(w, r)
	}))
	defer srv.Close()

	plan, err := c.CreateAndStartDeepResearch(t.Context(), "research prompt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Cid != "c_validated" || plan.Title != "Research title" || plan.ETAText != "About five minutes" || len(plan.Steps) != 1 {
		t.Fatalf("plan = %#v", plan)
	}
	if len(streamPrompts) != 2 || streamPrompts[0] != deepResearchPromptPrefix+"research prompt" || streamPrompts[1] != "Commencer la recherche" {
		t.Fatalf("stream prompts = %#v", streamPrompts)
	}
}

func TestCreateAndStartDeepResearchConfirmationErrorIsFatal(t *testing.T) {
	streamRequests := 0
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "StreamGenerate") {
			streamRequests++
			if streamRequests == 1 {
				_, _ = w.Write(makeResearchPlanStreamBody(t, "plan ready", "c_validated", "Start now", true))
				return
			}
			http.Error(w, "confirm rejected", http.StatusBadRequest)
			return
		}
		writeEmptyResearchBatch(w, r)
	}))
	defer srv.Close()

	_, err := c.CreateAndStartDeepResearch(t.Context(), "research prompt", nil)
	if err == nil || !strings.HasPrefix(err.Error(), "deep research confirm step failed: ") {
		t.Fatalf("err = %v, want fatal confirmation error", err)
	}
	if streamRequests != 2 {
		t.Fatalf("StreamGenerate requests = %d, want 2", streamRequests)
	}
}

func TestWaitForDeepResearchStartEventuallyRuns(t *testing.T) {
	checks := 0
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checks++
		turn := testResearchPendingTurn("56")
		if checks == 2 {
			turn = testResearchRunningTurn()
		}
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", makeResearchStatusBody(t, turn), 0))
	}))
	defer srv.Close()

	if err := c.waitForDeepResearchStart(t.Context(), "c_research"); err != nil {
		t.Fatal(err)
	}
	if checks != 2 {
		t.Fatalf("checks = %d, want 2", checks)
	}
}

func TestWaitForDeepResearchStartRejectsNotResearch(t *testing.T) {
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", makeResearchStatusBody(t, testResearchTextTurn("ordinary response")), 0))
	}))
	defer srv.Close()

	err := c.waitForDeepResearchStart(t.Context(), "c_research")
	if err == nil || err.Error() != "deep research did not start (state=not_research)" {
		t.Fatalf("err = %v", err)
	}
}

func TestWaitForDeepResearchStartRejectsUnknownState(t *testing.T) {
	started, retry, err := deepResearchStartState("mystery")
	if started || retry || err == nil || err.Error() != "deep research did not start (state=mystery)" {
		t.Fatalf("started=%t retry=%t err=%v", started, retry, err)
	}
}

func TestWaitForDeepResearchStartExhaustsEmptyState(t *testing.T) {
	checks := 0
	emptyBody := makeResearchStatusBody(t)
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checks++
		_, _ = w.Write(makeTestBatchResponse("hNvQHb", emptyBody, 0))
	}))
	defer srv.Close()

	err := c.waitForDeepResearchStart(t.Context(), "c_research")
	if err == nil || err.Error() != "deep research did not start (state=empty)" {
		t.Fatalf("err = %v", err)
	}
	if checks != 5 {
		t.Fatalf("checks = %d, want 5", checks)
	}
}

func TestWaitForDeepResearchStartWrapsStatusReadErrorImmediately(t *testing.T) {
	requests := 0
	c, srv := newResearchCreateTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "status unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	started := time.Now()
	err := c.waitForDeepResearchStart(t.Context(), "c_research")
	if err == nil || !strings.HasPrefix(err.Error(), "deep research status check failed: ") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(started) >= 2*time.Second {
		t.Fatalf("status error was retried or slept: elapsed=%s", time.Since(started))
	}
	if requests != 2 {
		t.Fatalf("HTTP requests = %d, want raw read plus fallback", requests)
	}
}

func newResearchCreateTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	origBase := baseURL
	baseURL = srv.URL
	t.Cleanup(func() { baseURL = origBase })

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.httpClient = srv.Client()
	return c, srv
}

func writeEmptyResearchBatch(w http.ResponseWriter, r *http.Request) {
	rpcIDs := strings.Split(r.URL.Query().Get("rpcids"), ",")
	bodies := make(map[string]string, len(rpcIDs))
	for _, rpcID := range rpcIDs {
		if rpcID != "" {
			bodies[rpcID] = "[]"
		}
	}
	_, _ = w.Write(makeTestMultiBatchResponse(bodies, nil))
}

func makeResearchPlanStreamBody(t *testing.T, text, cid, confirmPrompt string, includePlan bool) []byte {
	t.Helper()
	candidate := make([]any, 3)
	candidate[0] = "rc_plan"
	candidate[1] = []any{text}
	if includePlan {
		candidate[2] = map[string]any{"56": []any{
			"Research title",
			[]any{[]any{nil, "Search", "Find official sources"}},
			"About five minutes",
			[]any{confirmPrompt},
		}}
	}

	content := make([]any, 26)
	content[1] = []any{cid, "r_plan"}
	content[4] = []any{candidate}
	content[25] = "context"
	contentJSON, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	frameJSON, err := json.Marshal([]any{[]any{"wrb.fr", nil, string(contentJSON)}})
	if err != nil {
		t.Fatal(err)
	}
	framed := "\n" + string(frameJSON) + "\n"
	return []byte(")]}'\n" + fmt.Sprintf("%d", utf16Units(framed)) + framed)
}

func decodeResearchStreamPrompt(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	var outer []any
	if err := json.Unmarshal([]byte(r.Form.Get("f.req")), &outer); err != nil {
		return "", err
	}
	if len(outer) < 2 {
		return "", fmt.Errorf("outer request has %d elements", len(outer))
	}
	innerJSON, ok := outer[1].(string)
	if !ok {
		return "", fmt.Errorf("outer request inner payload is %T", outer[1])
	}
	var inner []any
	if err := json.Unmarshal([]byte(innerJSON), &inner); err != nil {
		return "", err
	}
	if len(inner) == 0 {
		return "", fmt.Errorf("inner request is empty")
	}
	message, ok := inner[0].([]any)
	if !ok || len(message) == 0 {
		return "", fmt.Errorf("inner request message is %#v", inner[0])
	}
	prompt, ok := message[0].(string)
	if !ok {
		return "", fmt.Errorf("inner request prompt is %T", message[0])
	}
	return prompt, nil
}

func makeResearchStatusBody(t *testing.T, turns ...json.RawMessage) string {
	t.Helper()
	body, err := json.Marshal([]any{turns})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
