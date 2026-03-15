package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// GenerateContent sends a prompt and returns the full response (non-streaming).
func (c *Client) GenerateContent(ctx context.Context, prompt string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, model, false, nil)
	return best, err
}

// StreamCallback is called for each streaming chunk.
type StreamCallback func(output *types.ModelOutput)

// GenerateContentStream sends a prompt and calls cb for each streaming chunk.
func (c *Client) GenerateContentStream(ctx context.Context, prompt string, model *types.Model, cb StreamCallback) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, model, false, cb)
	return best, err
}

// SendMessage sends a message in an existing chat.
func (c *Client) SendMessage(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, model, false, nil)
	return best, err
}

// SendMessageStream sends a message in a chat with streaming.
func (c *Client) SendMessageStream(ctx context.Context, prompt string, metadata []string, model *types.Model, cb StreamCallback) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, model, false, cb)
	return best, err
}

// SendMessageDeepResearch sends a message with deep research flags.
func (c *Client) SendMessageDeepResearch(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, model, true, nil)
	return best, err
}

// collectStreamResult is the shared implementation for all generate/send methods.
// It accumulates images across frames (deduped) and keeps the best text output.
func (c *Client) collectStreamResult(ctx context.Context, prompt string, metadata []string, model *types.Model, deepResearch bool, cb StreamCallback) (*types.ModelOutput, []types.Image, error) {
	var best *types.ModelOutput
	var allImages []types.Image
	var plan *types.DeepResearchPlanData
	var bestMetadata []string // track the most complete metadata across frames
	seen := map[string]bool{}
	err := c.streamGenerate(ctx, prompt, metadata, model, deepResearch, func(out *types.ModelOutput) {
		for _, img := range out.Images {
			if !seen[img.URL] {
				seen[img.URL] = true
				allImages = append(allImages, img)
			}
		}
		if out.DeepResearchPlan != nil {
			plan = out.DeepResearchPlan
		}
		// Keep the most complete metadata (longest with non-empty fields)
		if len(out.Metadata) > len(bestMetadata) {
			bestMetadata = out.Metadata
		}
		if best == nil || len(out.Text) >= len(best.Text) {
			best = out
		}
		if cb != nil {
			cb(out)
		}
	})
	if err != nil {
		return nil, nil, err
	}
	if best == nil {
		return nil, nil, fmt.Errorf("no response received")
	}
	if len(allImages) > 0 {
		best.Images = allImages
	}
	if plan != nil {
		best.DeepResearchPlan = plan
	}
	if len(bestMetadata) > len(best.Metadata) {
		best.Metadata = bestMetadata
	}
	return best, allImages, nil
}

func (c *Client) streamGenerate(ctx context.Context, prompt string, metadata []string, model *types.Model, deepResearch bool, cb StreamCallback) error {
	if model == nil {
		model = &types.Models[0] // unspecified
	}

	uuid := generateUUID()
	hasCid := len(metadata) > 0 && metadata[0] != ""
	innerReq := c.buildInnerRequest(prompt, metadata, deepResearch, hasCid, uuid)
	innerJSON, err := json.Marshal(innerReq)
	if err != nil {
		return fmt.Errorf("marshaling inner request: %w", err)
	}

	outerReq := []any{nil, string(innerJSON)}
	outerJSON, err := json.Marshal(outerReq)
	if err != nil {
		return fmt.Errorf("marshaling outer request: %w", err)
	}

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(outerJSON))

	reqURL := c.streamURL()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	headers := c.commonHeaders()
	for k, v := range headers {
		httpReq.Header[k] = v
	}
	// Model-specific headers
	for k, v := range model.Headers {
		httpReq.Header.Set(k, v)
	}
	// Per-request UUID header (required by server)
	httpReq.Header.Set("x-goog-ext-525005358-jspb", fmt.Sprintf(`["%s",1]`, uuid))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stream returned HTTP %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	return c.parseStreamResponse(resp.Body, cb)
}

func (c *Client) buildInnerRequest(prompt string, metadata []string, deepResearch bool, hasCid bool, uuid string) []any {
	// Build a 69-element array matching the Python library format
	req := make([]any, 69)

	// [0] = message content
	req[0] = []any{prompt, 0, nil, nil, nil, nil, 0}

	// [1] = language
	req[1] = []any{"en"}

	// [2] = metadata — Python uses ["", "", ""] for new chats,
	// and preserves the full response metadata array for continuations.
	if len(metadata) > 0 {
		meta := make([]any, len(metadata))
		for i, v := range metadata {
			if v != "" {
				meta[i] = v
			} else {
				meta[i] = ""
			}
		}
		req[2] = meta
	} else {
		req[2] = []any{"", "", ""}
	}

	// Common fields for all requests
	req[6] = []any{0}
	req[7] = 1 // Enable Snapshot Streaming
	req[10] = 1
	req[11] = 0
	req[17] = []any{[]any{0}} // [[0]] for new chats
	if hasCid {
		req[17] = []any{[]any{1}} // [[1]] for existing chats
	}
	req[18] = 0
	req[27] = 1
	req[30] = []any{4}
	req[41] = []any{1}
	// Note: req[45] is the temporary chat flag — NOT set by default.
	// Only set req[45]=1 when temporary mode is explicitly requested.
	req[53] = 0
	req[59] = uuid
	req[61] = []any{}

	// Deep research-specific fields
	if deepResearch {
		req[3] = "!" + generateURLSafeToken(2600)
		req[4] = generateHexUUID()
		req[49] = 1
		req[54] = []any{[]any{[]any{[]any{[]any{1}}}}}
		req[55] = []any{[]any{1}}
		req[68] = 2
	} else {
		req[68] = 1
	}

	return req
}

func (c *Client) parseStreamResponse(body io.Reader, cb StreamCallback) error {
	// Read the stream incrementally and parse frames as they arrive.
	// For deep research, the server may keep the stream open after the plan;
	// we stop as soon as we detect a completion frame (content[25] is string).
	var allData []byte
	buf := make([]byte, 64*1024)
	var lastText string
	var output *types.ModelOutput
	framesProcessed := 0

	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			allData = append(allData, buf[:n]...)

			// Parse all complete frames from accumulated data
			content := string(allData)
			if strings.HasPrefix(content, ")]}'\n") {
				content = content[5:]
			}

			frames := parseLengthPrefixedFrames(content)
			// Only process NEW frames (skip already-processed ones)
			done := false
			for i := framesProcessed; i < len(frames); i++ {
				var envelope []any
				if err := json.Unmarshal([]byte(frames[i]), &envelope); err != nil {
					continue
				}
				parsed := parseEnvelope(envelope)
				if parsed == nil {
					continue
				}
				if parsed.Text != "" {
					parsed.TextDelta = calculateDelta(lastText, parsed.Text)
					lastText = parsed.Text
				}
				output = parsed
				cb(parsed)
				if parsed.Done {
					done = true
					break
				}
			}
			framesProcessed = len(frames)

			if done {
				return nil
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			// On timeout/cancel: return what we have
			if output != nil {
				return nil
			}
			return fmt.Errorf("reading stream: %w", readErr)
		}
	}

	if output == nil {
		return fmt.Errorf("no valid response frames parsed")
	}
	return nil
}

// parseLengthPrefixedFrames parses Google's length-prefixed framing protocol.
// Format: <digits><content_of_N_utf16_units> repeated.
// The length counts UTF-16 code units starting immediately after the digits
// (includes the \n after digits and the trailing \n of the JSON payload).
func parseLengthPrefixedFrames(content string) []string {
	var frames []string
	runes := []rune(content)
	pos := 0
	totalLen := len(runes)

	for pos < totalLen {
		// Skip whitespace before length marker
		for pos < totalLen && (runes[pos] == ' ' || runes[pos] == '\t' || runes[pos] == '\n' || runes[pos] == '\r') {
			pos++
		}
		if pos >= totalLen {
			break
		}

		// Read the length number (digits)
		numStart := pos
		for pos < totalLen && runes[pos] >= '0' && runes[pos] <= '9' {
			pos++
		}
		if pos == numStart {
			// Not a digit — skip
			pos++
			continue
		}
		lengthStr := string(runes[numStart:pos])

		// Parse the UTF-16 unit count
		utf16Units := 0
		for _, ch := range lengthStr {
			utf16Units = utf16Units*10 + int(ch-'0')
		}

		// Content starts immediately after the digits (NOT after the newline).
		// The length includes the \n after digits and the trailing \n.
		contentStart := pos
		unitsConsumed := 0
		for pos < totalLen && unitsConsumed < utf16Units {
			r := runes[pos]
			pos++
			if r > 0xFFFF {
				unitsConsumed += 2
			} else {
				unitsConsumed++
			}
		}

		chunk := strings.TrimSpace(string(runes[contentStart:pos]))
		if chunk != "" {
			frames = append(frames, chunk)
		}
	}

	return frames
}

func parseEnvelope(envelope []any) *types.ModelOutput {
	// Unwrap single-element wrapper: [[...]] -> [...]
	for len(envelope) == 1 {
		if inner, ok := envelope[0].([]any); ok {
			envelope = inner
		} else {
			break
		}
	}

	if len(envelope) < 3 {
		return nil
	}

	// Position [2] contains the nested content JSON string
	contentStr, ok := envelope[2].(string)
	if !ok || contentStr == "" {
		return nil
	}

	var content []any
	if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
		return nil
	}

	out := &types.ModelOutput{}

	// Extract metadata from [1]
	if len(content) > 1 {
		if metaArr, ok := content[1].([]any); ok {
			for _, v := range metaArr {
				if s, ok := v.(string); ok {
					out.Metadata = append(out.Metadata, s)
				} else {
					out.Metadata = append(out.Metadata, "")
				}
			}
		}
	}

	// Extract text from candidates at [4]
	if len(content) > 4 {
		if candidates, ok := content[4].([]any); ok && len(candidates) > 0 {
			if cand, ok := candidates[0].([]any); ok {
				// rcid at [0]
				if len(cand) > 0 {
					if s, ok := cand[0].(string); ok {
						out.RCid = s
					}
				}
				// Text at [1][0] — primary path
				if len(cand) > 1 {
					if textArr, ok := cand[1].([]any); ok && len(textArr) > 0 {
						if s, ok := textArr[0].(string); ok {
							out.Text = html.UnescapeString(s)
						}
					}
				}
				// Fallback text at [22][0] if primary is empty or looks like a card URL
				if out.Text == "" || strings.HasPrefix(out.Text, "http://googleusercontent.com/") {
					if len(cand) > 22 {
						if altArr, ok := cand[22].([]any); ok && len(altArr) > 0 {
							if s, ok := altArr[0].(string); ok && s != "" {
								out.Text = html.UnescapeString(s)
							}
						}
					}
				}
				// Images at [12]
				if len(cand) > 12 && cand[12] != nil {
					imgs := extractImages(cand[12])
					if len(imgs) > 0 {
						out.Images = imgs
					}
				}
				// Deep research plan extraction from candidate structured data
				out.DeepResearchPlan = extractDeepResearchPlan(cand)

				// Clean up: card URL placeholder is not useful text when we have images
				if strings.HasPrefix(out.Text, "http://googleusercontent.com/") && len(out.Images) > 0 {
					out.Text = ""
				}
			}
		}
	}

	// Ensure metadata includes rcid from candidate (Python: chat.metadata + chat.rcid)
	if out.RCid != "" && len(out.Metadata) >= 2 {
		for len(out.Metadata) < 10 {
			out.Metadata = append(out.Metadata, "")
		}
		if out.Metadata[2] == "" {
			out.Metadata[2] = out.RCid
		}
	}

	// Check completion at [25]
	if len(content) > 25 {
		if contextStr, ok := content[25].(string); ok {
			out.Done = true
			// Replace metadata with completion context
			for len(out.Metadata) < 10 {
				out.Metadata = append(out.Metadata, "")
			}
			out.Metadata[9] = contextStr
		}
	}

	return out
}

func extractImages(imageData any) []types.Image {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}

	var images []types.Image

	// Web images at [1]
	if len(arr) > 1 {
		if webImgs, ok := arr[1].([]any); ok {
			for _, wi := range webImgs {
				wiArr, ok := wi.([]any)
				if !ok {
					continue
				}
				img := types.Image{}
				// URL at [0][0][0]
				if src := getNestedString(wiArr, 0, 0, 0); src != "" {
					img.URL = src
				}
				// Title at [7][0]
				if title := getNestedString(wiArr, 7, 0); title != "" {
					img.Title = title
				}
				if img.URL != "" {
					images = append(images, img)
				}
			}
		}
	}

	// Generated images at [7][0][0]
	// Structure: arr[7] = [[[ [item1], [item2], ... ]]]
	// Each item: [null, null, null, [null, 1, "filename", "url", ...], ...]
	// URL at item[3][3]
	if len(arr) > 7 && arr[7] != nil {
		var genItems []any
		// Navigate arr[7] to find the items list.
		// Structure: arr[7] = [[[ [item1], [item2], ... ]]]
		// Items are arrays with len > 3 where [3] is an array containing URL at [3].
		// We drill through single-element wrappers to find the list of items.
		if l1, ok := arr[7].([]any); ok && len(l1) > 0 {
			if l2, ok := l1[0].([]any); ok && len(l2) > 0 {
				// Scan all elements of l2 for image items
				for _, l2elem := range l2 {
					l2arr, ok := l2elem.([]any)
					if !ok || len(l2arr) == 0 {
						continue
					}
					// Check if this is an item (has [3] as array with URL)
					if len(l2arr) > 3 {
						if innerArr, ok := l2arr[3].([]any); ok && len(innerArr) > 3 {
							if _, ok := innerArr[3].(string); ok {
								// This is a single image item
								genItems = append(genItems, l2arr)
								continue
							}
						}
					}
					// Otherwise drill one more level
					for _, inner := range l2arr {
						innerArr, ok := inner.([]any)
						if !ok || len(innerArr) < 4 {
							continue
						}
						if sub, ok := innerArr[3].([]any); ok && len(sub) > 3 {
							if _, ok := sub[3].(string); ok {
								genItems = append(genItems, innerArr)
							}
						}
					}
				}
			}
		}
		for _, gi := range genItems {
			giArr, ok := gi.([]any)
			if !ok || len(giArr) < 4 {
				continue
			}
			img := types.Image{Generated: true}
			if u := getNestedString(giArr, 3, 3); u != "" {
				img.URL = u
			}
			if img.URL != "" {
				images = append(images, img)
			}
		}
	}

	return images
}

// extractDeepResearchPlan searches candidate data for a dict with key "56" or "57"
// containing the research plan payload.
func extractDeepResearchPlan(candidateData []any) *types.DeepResearchPlanData {
	// Recursively search for a map with key "56" or "57"
	var planPayload []any
	findDictKey(candidateData, func(m map[string]any) bool {
		for _, key := range []string{"56", "57"} {
			if v, ok := m[key]; ok {
				if arr, ok := v.([]any); ok {
					planPayload = arr
					return true
				}
			}
		}
		return false
	})

	if planPayload == nil {
		return nil
	}

	plan := &types.DeepResearchPlanData{}

	// title at [0]
	if len(planPayload) > 0 {
		if s, ok := planPayload[0].(string); ok {
			plan.Title = s
		}
	}

	// steps at [1] — each step is [?, label, body, ...]
	if len(planPayload) > 1 {
		if stepsArr, ok := planPayload[1].([]any); ok {
			for _, step := range stepsArr {
				if stepArr, ok := step.([]any); ok {
					label := ""
					body := ""
					if len(stepArr) > 1 {
						if s, ok := stepArr[1].(string); ok {
							label = s
						}
					}
					if len(stepArr) > 2 {
						if s, ok := stepArr[2].(string); ok {
							body = s
						}
					}
					if label != "" && body != "" {
						plan.Steps = append(plan.Steps, label+": "+body)
					} else if body != "" {
						plan.Steps = append(plan.Steps, body)
					} else if label != "" {
						plan.Steps = append(plan.Steps, label)
					}
				}
			}
		}
	}

	// eta at [2]
	if len(planPayload) > 2 {
		if s, ok := planPayload[2].(string); ok {
			plan.ETAText = s
		}
	}

	// confirm_prompt at [3][0]
	if len(planPayload) > 3 {
		if arr, ok := planPayload[3].([]any); ok && len(arr) > 0 {
			if s, ok := arr[0].(string); ok {
				plan.ConfirmPrompt = s
			}
		}
	}

	// Validate: at least one field must be non-empty
	if plan.Title == "" && len(plan.Steps) == 0 && plan.ETAText == "" && plan.ConfirmPrompt == "" {
		return nil
	}

	return plan
}

// findDictKey recursively searches nested data for a map matching the predicate.
func findDictKey(data any, pred func(map[string]any) bool) bool {
	switch v := data.(type) {
	case map[string]any:
		if pred(v) {
			return true
		}
		for _, val := range v {
			if findDictKey(val, pred) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if findDictKey(item, pred) {
				return true
			}
		}
	}
	return false
}

func getNestedString(arr []any, indices ...int) string {
	var current any = arr
	for _, idx := range indices {
		a, ok := current.([]any)
		if !ok || idx >= len(a) {
			return ""
		}
		current = a[idx]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

func calculateDelta(prev, current string) string {
	if prev == "" {
		return current
	}
	if strings.HasPrefix(current, prev) {
		return current[len(prev):]
	}
	// Fallback: return the new content
	return current
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateURLSafeToken(nbytes int) string {
	b := make([]byte, nbytes)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateHexUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
