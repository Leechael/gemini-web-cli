package client

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AIO-Starter/gemini-web-cli/internal/types"
)

const (
	rpcListChats = "MaZiqc"
	rpcReadChat  = "hNvQHb"
)

// ListChats returns a list of chats with optional pagination.
func (c *Client) ListChats(ctx context.Context, cursor string) ([]types.ChatItem, string, error) {
	// Build payload variants (Python tries multiple to handle browser differences)
	var payloads [][]any
	if cursor != "" {
		payloads = [][]any{{20, cursor, []any{0, nil, 1}}}
	} else {
		payloads = [][]any{
			{13, nil, []any{1, nil, 1}},
			{13, nil, []any{0, nil, 1}},
			{13, nil, []any{0, nil, 2}},
		}
	}

	// Try multiple source paths
	sourcePaths := []string{c.appPath()}
	if c.appPath() != "/app" {
		sourcePaths = append(sourcePaths, "/app")
	}

	for _, sourcePath := range sourcePaths {
		for _, payload := range payloads {
			items, nextCursor, err := c.tryListChats(ctx, payload, sourcePath)
			if err != nil {
				fmt.Fprintf(logWriter, "list_chats attempt failed (path=%s): %v\n", sourcePath, err)
				continue
			}
			if len(items) > 0 {
				return items, nextCursor, nil
			}
		}
	}

	return nil, "", nil
}

func (c *Client) tryListChats(ctx context.Context, payload []any, sourcePath string) ([]types.ChatItem, string, error) {
	payloadJSON, _ := json.Marshal(payload)
	fmt.Fprintf(logWriter, "list_chats trying payload=%s path=%s\n", string(payloadJSON), sourcePath)

	rpcReq := []any{
		[]any{
			[]any{rpcListChats, string(payloadJSON), nil, "generic"},
		},
	}
	reqJSON, _ := json.Marshal(rpcReq)

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(reqJSON))

	reqURL := c.batchURL([]string{rpcListChats}, sourcePath)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, "", err
	}
	headers := c.commonHeaders()
	for k, v := range headers {
		httpReq.Header[k] = v
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("list_chats request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	responseBody := stripResponsePrefix(string(body))

	rpcBody, rejectCode, err := extractRPCBody(responseBody, rpcListChats)
	if err != nil {
		return nil, "", err
	}
	if rejectCode != 0 {
		return nil, "", fmt.Errorf("list_chats rejected with code=%d", rejectCode)
	}

	return parseListChats(rpcBody)
}

// ReadChat reads conversation turns from a chat.
func (c *Client) ReadChat(ctx context.Context, cid string, maxTurns int) ([]types.ChatTurn, error) {
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
		return nil, fmt.Errorf("read_chat request failed: %w", err)
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

	return parseReadChat(rpcBody)
}

// LatestResponse holds the result of FetchLatestChatResponse.
type LatestResponse struct {
	Text string
	RCid string
	Rid  string
}

// FetchLatestChatResponse returns the latest assistant response for a chat.
func (c *Client) FetchLatestChatResponse(ctx context.Context, cid string) (*LatestResponse, error) {
	turns, err := c.ReadChat(ctx, cid, 10)
	if err != nil {
		return nil, err
	}
	if len(turns) == 0 {
		return nil, nil
	}
	// Turns are already in chronological order; pick the last one with an assistant response
	for i := len(turns) - 1; i >= 0; i-- {
		if turns[i].AssistantResponse != "" {
			return &LatestResponse{
				Text: turns[i].AssistantResponse,
				RCid: turns[i].RCid,
				Rid:  turns[i].Rid,
			}, nil
		}
	}
	return nil, nil
}

func stripResponsePrefix(s string) string {
	if strings.HasPrefix(s, ")]}'\n") {
		return s[5:]
	}
	return s
}

func extractRPCBody(response, rpcID string) (string, int, error) {
	// Parse length-prefixed frames and find the one with our RPC ID
	frames := parseLengthPrefixedFrames(response)

	for _, frame := range frames {
		var arr []any
		if err := json.Unmarshal([]byte(frame), &arr); err != nil {
			continue
		}

		// Look for wrb.fr envelope — may be nested one level deep
		items := findWrbFrItems(arr)
		for _, itemArr := range items {
			if len(itemArr) >= 3 {
				tag, _ := itemArr[0].(string)
				id, _ := itemArr[1].(string)
				if tag == "wrb.fr" && id == rpcID {
					body := ""
					if s, ok := itemArr[2].(string); ok {
						body = s
					}
					var rejectCode int
					if len(itemArr) > 5 {
						if codeArr, ok := itemArr[5].([]any); ok && len(codeArr) > 0 {
							if f, ok := codeArr[0].(float64); ok {
								rejectCode = int(f)
							}
						}
					}
					return body, rejectCode, nil
				}
			}
		}
	}

	return "", 0, fmt.Errorf("RPC response for %s not found", rpcID)
}

// findWrbFrItems searches for ["wrb.fr", ...] arrays in the parsed response.
// They can appear at the top level or nested one level deep.
func findWrbFrItems(arr []any) [][]any {
	var results [][]any
	for _, item := range arr {
		itemArr, ok := item.([]any)
		if !ok {
			continue
		}
		if len(itemArr) >= 2 {
			if tag, ok := itemArr[0].(string); ok && tag == "wrb.fr" {
				results = append(results, itemArr)
				continue
			}
		}
		// Check one level deeper (the response is often [[["wrb.fr", ...], ...]])
		for _, sub := range itemArr {
			subArr, ok := sub.([]any)
			if !ok {
				continue
			}
			if len(subArr) >= 2 {
				if tag, ok := subArr[0].(string); ok && tag == "wrb.fr" {
					results = append(results, subArr)
				}
			}
		}
	}
	return results
}

func parseListChats(body string) ([]types.ChatItem, string, error) {
	if body == "" || body == "[]" {
		return nil, "", nil
	}

	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, "", fmt.Errorf("parsing list response: %w", err)
	}

	var items []types.ChatItem
	var nextCursor string

	// Cursor at [1]
	if len(data) > 1 {
		if s, ok := data[1].(string); ok {
			nextCursor = s
		}
	}

	// Chat items at [2] — structure: [[cid, title, null, null, null, [epoch_s, ns], ...], ...]
	if len(data) > 2 {
		if chatList, ok := data[2].([]any); ok {
			for _, chat := range chatList {
				chatArr, ok := chat.([]any)
				if !ok {
					continue
				}
				item := types.ChatItem{}
				if len(chatArr) > 0 {
					if s, ok := chatArr[0].(string); ok {
						item.Cid = s
					}
				}
				if len(chatArr) > 1 {
					if s, ok := chatArr[1].(string); ok {
						item.Title = s
					}
				}
				// Timestamp at [5] as [epoch_seconds, nanoseconds]
				if len(chatArr) > 5 {
					if ts, ok := chatArr[5].([]any); ok && len(ts) > 0 {
						if epoch, ok := ts[0].(float64); ok {
							t := time.Unix(int64(epoch), 0).UTC()
							item.UpdatedAt = t.Format("2006-01-02T15:04")
						}
					}
				}
				if item.Cid != "" {
					items = append(items, item)
				}
			}
		}
	}

	return items, nextCursor, nil
}

func parseReadChat(body string) ([]types.ChatTurn, error) {
	if body == "" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, fmt.Errorf("parsing read response: %w", err)
	}

	var turns []types.ChatTurn

	// Turns are at [0] in the response array
	if len(data) > 0 {
		if turnList, ok := data[0].([]any); ok {
			for _, turn := range turnList {
				turnArr, ok := turn.([]any)
				if !ok || len(turnArr) < 4 {
					continue
				}
				ct := types.ChatTurn{}

				// User prompt at [2][0][0]
				if userPrompt := getNestedStringFromAny(turnArr, 2, 0, 0); userPrompt != "" {
					ct.UserPrompt = html.UnescapeString(userPrompt)
				}

				// Assistant response: candidate at [3][0][0]
				cand := getNestedArrayFromAny(turnArr, 3, 0, 0)
				if cand != nil {
					// Text at candidate[1][0], fallback to [22][0] for card URLs
					if len(cand) > 1 {
						if textArr, ok := cand[1].([]any); ok && len(textArr) > 0 {
							if s, ok := textArr[0].(string); ok {
								ct.AssistantResponse = html.UnescapeString(s)
							}
						}
					}
					if strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") {
						if len(cand) > 22 {
							if altArr, ok := cand[22].([]any); ok && len(altArr) > 0 {
								if s, ok := altArr[0].(string); ok && s != "" {
									ct.AssistantResponse = html.UnescapeString(s)
								}
							}
						}
					}

					// rcid at candidate[0]
					if len(cand) > 0 {
						if s, ok := cand[0].(string); ok {
							ct.RCid = s
						}
					}

					// Images at candidate[12]
					if len(cand) > 12 && cand[12] != nil {
						ct.Images = extractImages(cand[12])
					}

					// Clean up: if text is just a card URL and we have images, clear it
					if strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") && len(ct.Images) > 0 {
						ct.AssistantResponse = ""
					}
				}

				// rid from turn metadata at [0][1]
				if rid := getNestedStringFromAny(turnArr, 0, 1); rid != "" {
					ct.Rid = rid
				}

				if ct.UserPrompt != "" || ct.AssistantResponse != "" {
					turns = append(turns, ct)
				}
			}
		}
	}

	// Server returns newest-first; reverse to chronological order
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}

	return turns, nil
}

func getNestedStringFromAny(data any, indices ...int) string {
	current := data
	for _, idx := range indices {
		arr, ok := current.([]any)
		if !ok || idx >= len(arr) {
			return ""
		}
		current = arr[idx]
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}
