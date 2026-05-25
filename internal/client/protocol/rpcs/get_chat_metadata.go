// RPC: MUAZcd — GetChatMetadata
// Source-path: any Gemini chat page (defaults to /app/<chat_id>)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[null, [["unread_metadata"]], ["<chat_id>", null, null, [["<chat_id>", ""], 0]]]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[["<chat_id>", "<title>", <updated_unix_seconds>, <unread_bool>]]
//
// Test fixture: testdata/get_chat_metadata_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
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
	payloadBytes, _ := json.Marshal([]any{nil, []any{[]any{"unread_metadata"}}, []any{cid, nil, nil, []any{[]any{cid, ""}, 0}}})
	return getChatMetadataRPCID, string(payloadBytes)
}

// DecodeGetChatMetadata parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetChatMetadata(body []byte) (*ChatMetadata, error) {
	if strings.TrimSpace(string(body)) == "" {
		return nil, fmt.Errorf("GetChatMetadata body is empty")
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetChatMetadata JSON: %w", err)
	}
	meta := &ChatMetadata{}
	fillMetadata(meta, data)
	if meta.Cid == "" && meta.Title == "" && meta.UpdatedAt == 0 {
		return nil, fmt.Errorf("GetChatMetadata response did not contain metadata fields")
	}
	return meta, nil
}

func fillMetadata(meta *ChatMetadata, v any) {
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			fillMetadata(meta, item)
		}
	case string:
		if strings.HasPrefix(x, "c_") && meta.Cid == "" {
			meta.Cid = x
		} else if x != "" && meta.Title == "" && !strings.HasPrefix(x, "r_") && !strings.HasPrefix(x, "rcid_") {
			meta.Title = x
		}
	case float64:
		if meta.UpdatedAt == 0 && x > 0 {
			meta.UpdatedAt = int64(x)
		}
	case bool:
		meta.Unread = meta.Unread || x
	}
}
