package client

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const rpcReadChat = "hNvQHb"

func stripResponsePrefix(s string) string {
	if strings.HasPrefix(s, ")]}'\n") {
		return s[5:]
	}
	return s
}

func parseLengthPrefixedFrames(content string) []string {
	frames := protocol.ParseLengthPrefixedFrames([]byte(content))
	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		out = append(out, string(frame))
	}
	return out
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

					// Images, videos, media at candidate[12]
					if len(cand) > 12 && cand[12] != nil {
						ct.Images = extractImages(cand[12])
						ct.Videos = extractVideos(cand[12])
						ct.Media = extractMedia(cand[12])
					}

					// Clean up: if text is just a card URL and we have images/videos/media, clear it
					if strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") && (len(ct.Images) > 0 || len(ct.Videos) > 0 || len(ct.Media) > 0) {
						ct.AssistantResponse = ""
					}
					// Strip trailing card URL lines from response text (video_gen_chip, card_content, etc.)
					ct.AssistantResponse = stripCardURLLines(ct.AssistantResponse)
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

// stripCardURLLines removes lines that are just googleusercontent card URL placeholders.
func stripCardURLLines(text string) string {
	lines := strings.Split(text, "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http://googleusercontent.com/card_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/video_gen_chip/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_video_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_media_content/") ||
			strings.HasPrefix(trimmed, "http://googleusercontent.com/generated_music_content/") {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.Join(kept, "\n")
	return strings.TrimRight(result, "\n")
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
