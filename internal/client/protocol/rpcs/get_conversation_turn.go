// RPC: EqPOKe — GetConversationTurn
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["<chat_id>", "<request_id>"]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	turn_arr or [turn_arr]
//
// Test fixture: testdata/get_conversation_turn_basic.txt
//
// Notes:
//   - Empty bodies are errors because this RPC fetches one exact turn.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

const getConversationTurnRPCID = "EqPOKe"

// EncodeGetConversationTurn returns the GetConversationTurn payload.
func EncodeGetConversationTurn(cid, requestID string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{cid, requestID})
	return getConversationTurnRPCID, string(payloadBytes)
}

// DecodeGetConversationTurn parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetConversationTurn(body []byte) (*types.ChatTurn, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, fmt.Errorf("GetConversationTurn body is empty")
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetConversationTurn JSON: %w", err)
	}
	turnArr, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("GetConversationTurn response is not an array")
	}
	if len(turnArr) == 1 {
		if inner, ok := turnArr[0].([]any); ok {
			turnArr = inner
		}
	}
	turn := decodeChatTurnArray(turnArr)
	if turn.UserPrompt == "" && turn.AssistantResponse == "" {
		return nil, fmt.Errorf("GetConversationTurn response did not contain a turn")
	}
	return &turn, nil
}
