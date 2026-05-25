// RPC: hNvQHb — ReadChat
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<chat_id>", <max_turns>, null, 1, [1], [4], null, 1]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[turn_arr, ...]]
//
//	turn_arr structure:
//	  [0]: metadata; request id is [0][1]
//	  [2][0][0]: user prompt
//	  [3][0][0]: candidate array
//	  candidate[0]: response id
//	  candidate[1][0]: assistant text
//	  candidate[12]: generated media metadata
//
// Test fixture: testdata/read_chat_basic.txt
//
// Notes:
//   - Assistant text can contain googleusercontent card URL lines; they are stripped during decode.
package rpcs

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const readChatRPCID = "hNvQHb"

// EncodeReadChat returns the ReadChat payload.
func EncodeReadChat(chatID string, maxTurns int) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{chatID, maxTurns, nil, 1, []any{1}, []any{4}, nil, 1})
	return readChatRPCID, string(payloadBytes)
}

// DecodeReadChat parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeReadChat(body []byte) ([]types.ChatTurn, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ReadChat JSON: %w", err)
	}

	turnList, ok := protocol.ArrayAt(data, 0)
	if !ok {
		return nil, nil
	}

	turns := make([]types.ChatTurn, 0, len(turnList))
	for _, turn := range turnList {
		turnArr, ok := turn.([]any)
		if !ok || len(turnArr) < 4 {
			continue
		}
		ct := decodeChatTurnArray(turnArr)
		if ct.UserPrompt != "" || ct.AssistantResponse != "" {
			turns = append(turns, ct)
		}
	}

	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	return turns, nil
}

func decodeChatTurnArray(turnArr []any) types.ChatTurn {
	ct := types.ChatTurn{}
	if userPrompt := protocol.StringAt(turnArr, 2, 0, 0); userPrompt != "" {
		ct.UserPrompt = html.UnescapeString(userPrompt)
	}
	if rid := protocol.StringAt(turnArr, 0, 1); rid != "" {
		ct.Rid = rid
	}

	cand := nestedArray(turnArr, 3, 0, 0)
	if cand == nil {
		return ct
	}
	ct.RCid = protocol.StringAt(cand, 0)
	ct.AssistantResponse = html.UnescapeString(protocol.StringAt(cand, 1, 0))
	if ct.AssistantResponse == "" || strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") {
		if alt := protocol.StringAt(cand, 22, 0); alt != "" {
			ct.AssistantResponse = html.UnescapeString(alt)
		}
	}
	if mediaData, ok := protocol.ValueAt(cand, 12); ok && mediaData != nil {
		ct.Images = decodeImages(mediaData)
		ct.Videos = decodeVideos(mediaData)
		ct.Media = decodeMedia(mediaData)
	}
	if strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") && (len(ct.Images) > 0 || len(ct.Videos) > 0 || len(ct.Media) > 0) {
		ct.AssistantResponse = ""
	}
	ct.AssistantResponse = protocol.StripCardURLLines(ct.AssistantResponse)
	return ct
}

func nestedArray(root any, path ...int) []any {
	arr, ok := protocol.ArrayAt(root, path...)
	if !ok {
		return nil
	}
	return arr
}

func childArray(arr []any, idx int) []any {
	if idx < 0 || idx >= len(arr) {
		return nil
	}
	out, _ := arr[idx].([]any)
	return out
}

func decodeImages(imageData any) []types.Image {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	var images []types.Image
	if webImgs := childArray(arr, 1); webImgs != nil {
		for _, wi := range webImgs {
			wiArr, ok := wi.([]any)
			if !ok {
				continue
			}
			img := types.Image{URL: protocol.StringAt(wiArr, 0, 0, 0), Title: protocol.StringAt(wiArr, 7, 0)}
			if img.URL != "" {
				images = append(images, img)
			}
		}
	}
	if len(arr) > 7 && arr[7] != nil {
		for _, item := range findGeneratedImageItems(arr[7]) {
			if u := protocol.StringAt(item, 3, 3); u != "" {
				images = append(images, types.Image{URL: u, Generated: true})
			}
		}
	}
	return images
}

func findGeneratedImageItems(root any) [][]any {
	var items [][]any
	var walk func(any)
	walk = func(v any) {
		arr, ok := v.([]any)
		if !ok {
			return
		}
		if protocol.StringAt(arr, 3, 3) != "" {
			items = append(items, arr)
			return
		}
		for _, child := range arr {
			walk(child)
		}
	}
	walk(root)
	return items
}

func decodeVideos(imageData any) []types.Video {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	var videoRoot any
	if len(arr) > 59 && arr[59] != nil {
		videoRoot = arr[59]
	}
	if videoRoot == nil {
		for _, elem := range arr {
			if m, ok := elem.(map[string]any); ok {
				if v, exists := m["60"]; exists {
					videoRoot = v
					break
				}
			}
		}
	}
	if videoRoot == nil {
		return nil
	}
	current, ok := videoRoot.([]any)
	if !ok || len(current) == 0 {
		return nil
	}
	for range 4 {
		next := childArray(current, 0)
		if next == nil {
			return nil
		}
		current = next
	}
	urls := childArray(current, 7)
	if len(urls) < 2 {
		return nil
	}
	thumbnail, _ := urls[0].(string)
	videoURL, _ := urls[1].(string)
	if videoURL == "" {
		return nil
	}
	return []types.Video{{URL: videoURL, Thumbnail: thumbnail}}
}

func decodeMedia(imageData any) []types.GeneratedMedia {
	arr, ok := imageData.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	var mediaData []any
	if len(arr) > 86 && arr[86] != nil {
		mediaData, _ = arr[86].([]any)
	}
	if mediaData == nil {
		for _, elem := range arr {
			if m, ok := elem.(map[string]any); ok {
				for _, key := range []string{"86", "87"} {
					mediaData, _ = m[key].([]any)
					if mediaData != nil {
						break
					}
				}
			}
		}
	}
	if mediaData == nil {
		return nil
	}
	media := types.GeneratedMedia{}
	if urls := childArray(childArray(mediaData, 0), 1); urls != nil {
		if u := childArray(urls, 7); len(u) >= 2 {
			media.MP3Thumbnail, _ = u[0].(string)
			media.MP3URL, _ = u[1].(string)
		}
	}
	if mp4Part := childArray(mediaData, 1); mp4Part != nil {
		if urls := childArray(childArray(mp4Part, 1), 7); len(urls) >= 2 {
			media.MP4Thumbnail, _ = urls[0].(string)
			media.MP4URL, _ = urls[1].(string)
		}
		if urls := childArray(childArray(mp4Part, 3), 7); len(urls) >= 2 {
			media.VTTURL, _ = urls[1].(string)
		}
	}
	if meta := childArray(mediaData, 2); meta != nil {
		if len(meta) > 0 {
			media.Title, _ = meta[0].(string)
		}
		if len(meta) > 2 {
			media.Artist, _ = meta[2].(string)
		}
		if len(meta) > 4 {
			media.Genre, _ = meta[4].(string)
		}
		if len(meta) > 5 {
			if moods, ok := meta[5].([]any); ok {
				for _, item := range moods {
					if s, ok := item.(string); ok && s != "" {
						media.Moods = append(media.Moods, s)
					}
				}
			}
		}
	}
	if media.MP3URL == "" && media.MP4URL == "" {
		return nil
	}
	return []types.GeneratedMedia{media}
}
