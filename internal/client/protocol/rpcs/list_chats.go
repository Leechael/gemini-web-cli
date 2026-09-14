// RPC: MaZiqc — ListChats
// Source-path: any Gemini page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape (pagination verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	[<page_size>, <cursor or null>, [<flag1>, null, <flag2>]]
//	↑            ↑                   ↑
//	13 typical   pagination cursor   browser variant flags
//
// Pagination (verified): first page uses [1,null,1] (browser page size 34,
// this client uses 13); subsequent pages use page size 20 with flags
// [0,null,1] and the opaque next_cursor from the previous response.
//
// Notebook-scoped variant (implemented; first page verified): the flags array
// grows to 5 elements — [<10>, <cursor or null>, [null, null, 1, "notebooks/<uuid>", 1]].
// This client lists only the first page (PageSize 10). Later pages can be
// requested by passing the previous response cursor in slot 1.
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[null, <next_cursor or null>, [chat_item_arr, ...]]
//
//	chat_item_arr structure:
//	  [0]: "<chat_id>"
//	  [1]: "<chat_title>"
//	  [5]: [<unix_seconds>, <nanoseconds>]
//	  other slots: snippets, flags, and model metadata decoded lazily
//
// Test fixture: testdata/list_chats_basic.txt
//
// Notes:
//   - Empty bodies decode to an empty result because chat lists can be empty.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const listChatsRPCID = "MaZiqc"

// ListChatsPayload describes the list-chat browser payload variant.
type ListChatsPayload struct {
	PageSize int
	Cursor   string
	Flag1    int
	Flag2    int
	// NotebookResource scopes the list to a notebook ("notebooks/<uuid>").
	// When set, the flags array becomes [null, null, 1, <resource>, 1] and
	// the browser page size is 10 (verified in a live session where the
	// response listed the notebook's chats with the notebook resource name
	// at item index 7).
	NotebookResource string
}

// ChatListItem is the protocol-level representation of one listed chat.
type ChatListItem struct {
	Cid           string
	Title         string
	UpdatedAtUnix int64
}

// EncodeListChats returns the default ListChats payload shape.
func EncodeListChats(pageSize int, cursor string) (rpcID, payload string) {
	p := ListChatsPayload{PageSize: pageSize, Cursor: cursor, Flag1: 1, Flag2: 1}
	if cursor != "" {
		p.Flag1 = 0
	}
	return EncodeListChatsRaw(p)
}

// EncodeListChatsRaw returns a specific ListChats browser payload variant.
func EncodeListChatsRaw(p ListChatsPayload) (rpcID, payload string) {
	if p.NotebookResource != "" {
		var cursor any
		if p.Cursor != "" {
			cursor = p.Cursor
		}
		payloadBytes, _ := json.Marshal([]any{p.PageSize, cursor, []any{nil, nil, 1, p.NotebookResource, 1}})
		return listChatsRPCID, string(payloadBytes)
	}
	payloadArr := []any{p.PageSize, nil, []any{p.Flag1, nil, p.Flag2}}
	if p.Cursor != "" {
		payloadArr[1] = p.Cursor
	}
	payloadBytes, _ := json.Marshal(payloadArr)
	return listChatsRPCID, string(payloadBytes)
}

// DecodeListChats parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListChats(body []byte) ([]ChatListItem, string, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, "", nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, "", fmt.Errorf("decode ListChats JSON: %w", err)
	}

	nextCursor := ""
	if s := protocol.StringAt(data, 1); s != "" {
		nextCursor = s
	}

	chatList, ok := protocol.ArrayAt(data, 2)
	if !ok {
		return nil, nextCursor, nil
	}

	items := make([]ChatListItem, 0, len(chatList))
	for _, chat := range chatList {
		chatArr, ok := chat.([]any)
		if !ok {
			continue
		}
		item := ChatListItem{
			Cid:   protocol.StringAt(chatArr, 0),
			Title: protocol.StringAt(chatArr, 1),
		}
		if ts, ok := protocol.ValueAt(chatArr, 5, 0); ok {
			if epoch, ok := ts.(float64); ok {
				item.UpdatedAtUnix = int64(epoch)
			}
		}
		if item.Cid != "" {
			items = append(items, item)
		}
	}
	return items, nextCursor, nil
}
