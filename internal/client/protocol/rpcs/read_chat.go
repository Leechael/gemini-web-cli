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
//	  [4][0]: created timestamp seconds
//	  candidate[0]: response id
//	  candidate[1][0]: assistant text
//	  candidate[12]: generated media metadata
//
// Test fixture: testdata/read_chat_basic.txt
//
// Notes:
//   - Assistant text can contain googleusercontent card URL lines; they are stripped during decode.
//   - Empty bodies decode to no turns because list-style chat reads can be empty.
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
		if ct.UserPrompt != "" || ct.AssistantResponse != "" || len(ct.Images) > 0 || len(ct.Videos) > 0 || len(ct.Media) > 0 {
			turns = append(turns, ct)
		}
	}

	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	return turns, nil
}

// DecodeReadChatRaw returns raw JSON turns without decoding them into ChatTurn values.
func DecodeReadChatRaw(body []byte) ([]json.RawMessage, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, nil
	}
	var data []json.RawMessage
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode raw ReadChat JSON: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	var turns []json.RawMessage
	if err := json.Unmarshal(data[0], &turns); err != nil {
		return data, nil
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
	if v, ok := protocol.ValueAt(turnArr, 4, 0); ok {
		if epoch, ok := v.(float64); ok {
			ct.CreatedAtUnix = int64(epoch)
		}
	}

	cand, ok := protocol.ArrayAt(turnArr, 3, 0, 0)
	if !ok {
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
		ct.Images = types.ExtractImages(mediaData)
		ct.Videos = types.ExtractVideos(mediaData)
		ct.Media = types.ExtractMedia(mediaData)
	}
	if strings.HasPrefix(ct.AssistantResponse, "http://googleusercontent.com/") && (len(ct.Images) > 0 || len(ct.Videos) > 0 || len(ct.Media) > 0) {
		ct.AssistantResponse = ""
	}
	ct.AssistantResponse = protocol.StripCardURLLines(ct.AssistantResponse)
	return ct
}
