// RPC: MUAZcd — GetChatMetadata
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[null, [["unread_metadata"]], ["<chat_id>", null, null, null, null, null, [["<chat_id>", ""], 0]]]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[null, ["<chat_id>", "<title>", null, null, null, [<updated_unix_seconds>, <nanos>], [["<chat_id>", "<request_id>"], <unread_count>], ...]]
//
// Test fixture: testdata/get_chat_metadata_basic.txt
//
// Notes:
//   - Empty bodies are errors because this RPC fetches one exact chat record.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getChatMetadataRPCID = "MUAZcd"

// ChatMetadata is the decoded metadata for a single chat.
type ChatMetadata struct {
	Cid       string `json:"cid"`
	Title     string `json:"title"`
	UpdatedAt int64  `json:"updatedAt"`
	Unread    bool   `json:"unread"`
}

// EncodeGetChatMetadata returns the GetChatMetadata payload.
func EncodeGetChatMetadata(cid string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{nil, []any{[]any{"unread_metadata"}}, []any{cid, nil, nil, nil, nil, nil, []any{[]any{cid, ""}, 0}}})
	return getChatMetadataRPCID, string(payloadBytes)
}

// DecodeGetChatMetadata parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetChatMetadata(body []byte) (*ChatMetadata, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, fmt.Errorf("GetChatMetadata body is empty")
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetChatMetadata JSON: %w", err)
	}
	row, ok := protocol.ArrayAt(data, 1)
	if !ok {
		return nil, fmt.Errorf("GetChatMetadata response missing metadata row")
	}

	updatedAt := int64(0)
	if v, ok := protocol.ValueAt(row, 5, 0); ok {
		epoch, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("GetChatMetadata updated timestamp is not numeric")
		}
		updatedAt = int64(epoch)
	}
	meta := &ChatMetadata{
		Cid:       protocol.StringAt(row, 0),
		Title:     protocol.StringAt(row, 1),
		UpdatedAt: updatedAt,
		Unread:    protocol.IntAt(row, 6, 1) != 0,
	}
	if !strings.HasPrefix(meta.Cid, "c_") || meta.Title == "" || meta.UpdatedAt == 0 {
		return nil, fmt.Errorf("GetChatMetadata response did not match expected metadata shape")
	}
	return meta, nil
}
