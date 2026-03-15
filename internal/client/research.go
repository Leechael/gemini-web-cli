package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/AIO-Starter/gemini-web-cli/internal/types"
)

const (
	rpcBardActivity          = "ESY5D"
	rpcDeepResearchPrefs     = "L5adhe"
	rpcDeepResearchBootstrap = "ku4Jyf"
	rpcDeepResearchModelSt   = "qpEbW"
	rpcDeepResearchCaps      = "aPya6c"
	rpcDeepResearchAck       = "PCck7e"
)

// CreateAndStartDeepResearch creates a research plan and starts execution.
// Mirrors the Python flow: preflight → send prompt → confirm with "开始研究".
func (c *Client) CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error) {
	if model == nil {
		model = &types.Models[0]
	}

	// Step 0: Preflight RPCs (no cid yet)
	c.deepResearchPreflight(ctx, "", "")

	// Step 1: Send the prompt with deep research flags
	output, err := c.deepResearchGenerate(ctx, prompt, nil, model)
	if err != nil {
		return nil, fmt.Errorf("deep research plan request failed: %w", err)
	}

	plan := &types.DeepResearchPlan{}

	if len(output.Metadata) > 0 {
		plan.Cid = output.Metadata[0]
	}

	// Parse plan details from the response text
	text := output.Text
	if text != "" {
		plan.Title = extractPlanTitle(text)
		plan.Steps = extractPlanSteps(text)
	}

	if plan.Cid == "" {
		return nil, fmt.Errorf("no chat ID returned from deep research")
	}

	// Extract rid from step 1 metadata for preflight ack
	var rid string
	if len(output.Metadata) > 1 {
		rid = output.Metadata[1]
	}

	// Step 2: Preflight with cid, then confirm
	c.deepResearchPreflight(ctx, plan.Cid, rid)
	confirmPrompt := "开始研究"
	_, err = c.deepResearchGenerate(ctx, confirmPrompt, output.Metadata, model)
	if err != nil {
		// Non-fatal: research may still proceed
		fmt.Fprintf(logWriter, "Warning: confirm step failed: %v\n", err)
	}

	return plan, nil
}

func (c *Client) deepResearchGenerate(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	var last *types.ModelOutput
	err := c.streamGenerate(ctx, prompt, metadata, model, true, func(out *types.ModelOutput) {
		last = out
	})
	if err != nil {
		return nil, err
	}
	if last == nil {
		return nil, fmt.Errorf("no response received")
	}
	return last, nil
}

// deepResearchPreflight sends the preflight RPCs needed to enable deep research.
// All calls are best-effort (errors are logged but not fatal).
func (c *Client) deepResearchPreflight(ctx context.Context, cid string, rid string) {
	// 1. BARD_ACTIVITY
	c.bestEffortRPC(ctx, rpcBardActivity, `[[["bard_activity_enabled"]]]`, "")

	// 2. DEEP_RESEARCH_PREFS — feature_state (193 elements, [192] = music/image features)
	featureState := make([]any, 193)
	featureState[192] = []any{[]any{
		"music_generation_soft",
		"image_generation_soft",
		"music_generation_soft",
		"image_generation_soft",
		"music_generation_soft",
	}}
	prefsPayload1, _ := json.Marshal([]any{featureState, []any{[]any{"tool_menu_soft_badge_disabled_ids"}}})
	c.bestEffortRPC(ctx, rpcDeepResearchPrefs, string(prefsPayload1), "")

	// 3. DEEP_RESEARCH_PREFS — popup_state (87 elements, [86] = 1)
	popupState := make([]any, 87)
	popupState[86] = 1
	prefsPayload2, _ := json.Marshal([]any{popupState, []any{[]any{"popup_zs_visits_cooldown"}}})
	c.bestEffortRPC(ctx, rpcDeepResearchPrefs, string(prefsPayload2), "")

	// 4. DEEP_RESEARCH_BOOTSTRAP
	c.bestEffortRPC(ctx, rpcDeepResearchBootstrap,
		`["en",null,null,null,4,null,null,[2,4,7,15],null,[[5]]]`, "")

	// 5. If we have a cid, send model state + caps + ack
	if cid != "" {
		c.bestEffortRPCMulti(ctx, []rpcCall{
			{rpcDeepResearchModelSt, `[[[1,4],[6,6],[1,15]]]`},
			{rpcDeepResearchCaps, `[]`},
		}, cid)

		if rid != "" {
			ackPayload, _ := json.Marshal([]any{rid})
			c.bestEffortRPC(ctx, rpcDeepResearchAck, string(ackPayload), cid)
		}
	}
}

type rpcCall struct {
	rpcID   string
	payload string
}

func (c *Client) bestEffortRPC(ctx context.Context, rpcID, payload, sourceCid string) {
	c.bestEffortRPCMulti(ctx, []rpcCall{{rpcID, payload}}, sourceCid)
}

func (c *Client) bestEffortRPCMulti(ctx context.Context, calls []rpcCall, sourceCid string) {
	var serialized []any
	var rpcIDs []string
	for _, call := range calls {
		serialized = append(serialized, []any{call.rpcID, call.payload, nil, "generic"})
		rpcIDs = append(rpcIDs, call.rpcID)
	}
	rpcReq := []any{serialized}
	reqJSON, _ := json.Marshal(rpcReq)

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(reqJSON))

	sourcePath := c.appPath()
	if sourceCid != "" {
		sourcePath = c.appPath() + "/" + sourceCid
	}
	reqURL := c.batchURL(rpcIDs, sourcePath)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	headers := c.commonHeaders()
	for k, v := range headers {
		httpReq.Header[k] = v
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		fmt.Fprintf(logWriter, "preflight RPC %v failed: %v\n", rpcIDs, err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
}

// CheckDeepResearch checks the status of a deep research task.
func (c *Client) CheckDeepResearch(ctx context.Context, cid string) (bool, int, error) {
	// First try raw turns for structured deep research data
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, _ := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return true, len(text), nil
		}
	}

	// Fall back to regular text check
	latest, err := c.FetchLatestChatResponse(ctx, cid)
	if err != nil {
		return false, 0, err
	}
	if latest == nil || latest.Text == "" {
		return false, 0, nil
	}
	text := latest.Text

	lower := strings.ToLower(text)
	done := strings.Contains(text, "我已经完成了研究") ||
		strings.Contains(text, "研究完成") ||
		strings.Contains(lower, "i have completed the research") ||
		strings.Contains(lower, "i've completed the research") ||
		strings.Contains(lower, "research is complete")

	if !done && len(text) > 2000 {
		trimmed := strings.TrimLeft(text, " \t\n")
		if strings.HasPrefix(trimmed, "#") || strings.Contains(text, "\n## ") {
			done = true
		}
	}

	return done, len(text), nil
}

// GetDeepResearchResult fetches the full research result text.
// Deep research reports are stored at cand[30][0][4] in raw turn data,
// not in the normal text field.
func (c *Client) GetDeepResearchResult(ctx context.Context, cid string) (string, map[int]types.GroundingSource, error) {
	// Try raw turns first for structured deep research data
	rawTurns, err := c.ReadChatRaw(ctx, cid, 5)
	if err == nil && len(rawTurns) > 0 {
		text, sources := extractResearchResultFromRaw(rawTurns)
		if text != "" {
			return text, sources, nil
		}
	}

	// Fallback to regular read
	turns, err := c.ReadChat(ctx, cid, 5)
	if err != nil {
		return "", nil, err
	}
	var bestText string
	for _, turn := range turns {
		resp := turn.AssistantResponse
		// Skip card URLs
		if strings.HasPrefix(resp, "http://googleusercontent.com/") {
			continue
		}
		if len(resp) > len(bestText) {
			bestText = resp
		}
	}
	if bestText == "" {
		return "", nil, fmt.Errorf("no research result found for chat %s — research may still be running", cid)
	}
	return bestText, nil, nil
}

// extractResearchResultFromRaw extracts the deep research report from raw turn data.
// The report lives at cand[30][0][4] in each turn.
func extractResearchResultFromRaw(rawTurns []json.RawMessage) (string, map[int]types.GroundingSource) {
	for _, rawTurn := range rawTurns {
		var turn []any
		if err := json.Unmarshal(rawTurn, &turn); err != nil {
			continue
		}

		// Navigate to candidate: turn[3][0][0]
		cand := getNestedArrayFromAny(turn, 3, 0, 0)
		if cand == nil {
			continue
		}

		// Deep research data at cand[30][0]
		drData := getNestedArrayFromAny(cand, 30, 0)
		if drData == nil || len(drData) < 5 {
			continue
		}

		// Report text at [30][0][4]
		candidateText, ok := drData[4].(string)
		if !ok || len(candidateText) < 200 {
			continue
		}

		// Extract sources from [30][0][5][0]["44"]
		sources := extractResearchSources(drData)
		return candidateText, sources
	}
	return "", nil
}

func extractResearchSources(drData []any) map[int]types.GroundingSource {
	sources := make(map[int]types.GroundingSource)
	if len(drData) < 6 {
		return sources
	}

	citationsContainer := getNestedArrayFromAny(drData, 5, 0)
	if citationsContainer == nil {
		return sources
	}

	// Look for map with key "44"
	// The structure is: drData[5][0] is a dict-like structure
	// In JSON it's a map; navigate to "44" key
	if len(drData) > 5 {
		containerArr := getNestedArrayFromAny(drData, 5, 0)
		if containerArr == nil {
			return sources
		}
		// containerArr might be a JSON object — try to get it as map
		var rawContainer json.RawMessage
		b, err := json.Marshal(drData[5])
		if err != nil {
			return sources
		}
		if err := json.Unmarshal(b, &rawContainer); err != nil {
			return sources
		}

		// Try parsing as [{...}] where inner is {"44": [...]}
		var outerArr []json.RawMessage
		if err := json.Unmarshal(rawContainer, &outerArr); err != nil || len(outerArr) == 0 {
			return sources
		}
		var innerMap map[string]json.RawMessage
		if err := json.Unmarshal(outerArr[0], &innerMap); err != nil {
			return sources
		}
		citationGroupsRaw, ok := innerMap["44"]
		if !ok {
			return sources
		}
		var citationGroups []any
		if err := json.Unmarshal(citationGroupsRaw, &citationGroups); err != nil {
			return sources
		}

		for _, group := range citationGroups {
			groupArr, ok := group.([]any)
			if !ok || len(groupArr) < 2 {
				continue
			}
			for _, sourceEntries := range groupArr[1:] {
				seArr, ok := sourceEntries.([]any)
				if !ok {
					continue
				}
				for _, item := range seArr {
					itemArr, ok := item.([]any)
					if !ok || len(itemArr) < 4 {
						continue
					}
					inner := getNestedArrayFromAny(itemArr, 3)
					if inner == nil || len(inner) < 2 {
						continue
					}
					detail, ok := inner[0].([]any)
					if !ok || len(detail) < 3 {
						continue
					}
					refNum, ok := inner[1].(float64)
					if !ok {
						continue
					}
					urlStr, _ := detail[1].(string)
					title, _ := detail[2].(string)
					if urlStr != "" && strings.HasPrefix(urlStr, "http") {
						sources[int(refNum)] = types.GroundingSource{
							URL:   urlStr,
							Title: title,
						}
					}
				}
			}
		}
	}
	return sources
}

func getNestedArrayFromAny(data any, indices ...int) []any {
	current := data
	for _, idx := range indices {
		arr, ok := current.([]any)
		if !ok || idx >= len(arr) {
			return nil
		}
		current = arr[idx]
	}
	if arr, ok := current.([]any); ok {
		return arr
	}
	return nil
}

func extractPlanTitle(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") {
			return line
		}
	}
	return ""
}

func extractPlanSteps(text string) []string {
	var steps []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			steps = append(steps, line[2:])
		}
	}
	return steps
}

// ReadChatRaw returns the raw JSON turns for advanced parsing.
func (c *Client) ReadChatRaw(ctx context.Context, cid string, maxTurns int) ([]json.RawMessage, error) {
	payload := []any{cid, maxTurns, nil, 1, []any{1}, []any{4}, nil, 1}
	payloadJSON, _ := json.Marshal(payload)

	rpcReq := []any{
		[]any{
			[]any{rpcReadChat, string(payloadJSON), nil, "generic"},
		},
	}
	reqJSON, _ := json.Marshal(rpcReq)

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(reqJSON))

	sourcePath := c.appPath() + "/" + cid
	reqURL := c.batchURL([]string{rpcReadChat}, sourcePath)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	headers := c.commonHeaders()
	for k, v := range headers {
		httpReq.Header[k] = v
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	responseBody := stripResponsePrefix(string(body))
	rpcBody, _, err := extractRPCBody(responseBody, rpcReadChat)
	if err != nil {
		return nil, err
	}

	var data []json.RawMessage
	if err := json.Unmarshal([]byte(rpcBody), &data); err != nil {
		return nil, err
	}

	// Return turns from [0]
	if len(data) > 0 {
		var turns []json.RawMessage
		if err := json.Unmarshal(data[0], &turns); err != nil {
			return data, nil
		}
		return turns, nil
	}

	return nil, nil
}
