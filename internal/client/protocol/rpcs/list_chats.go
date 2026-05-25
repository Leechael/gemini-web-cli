// RPC: MaZiqc — ListChats
// Source-path: any Gemini page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[<page_size>, <cursor or null>, [<flag1>, null, <flag2>]]
//	↑            ↑                   ↑
//	13 typical   pagination cursor   browser variant flags
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
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const listChatsRPCID = "MaZiqc"

// ListChatsPayload describes the list-chat browser payload variant.
type ListChatsPayload struct {
	PageSize int
	Cursor   string
	Flag1    int
	Flag2    int
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
	payloadArr := []any{p.PageSize, nil, []any{p.Flag1, nil, p.Flag2}}
	if p.Cursor != "" {
		payloadArr[1] = p.Cursor
	}
	payloadBytes, _ := json.Marshal(payloadArr)
	return listChatsRPCID, string(payloadBytes)
}

// DecodeListChats parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListChats(body []byte) ([]types.ChatItem, string, error) {
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

	items := make([]types.ChatItem, 0, len(chatList))
	for _, chat := range chatList {
		chatArr, ok := chat.([]any)
		if !ok {
			continue
		}
		item := types.ChatItem{
			Cid:   protocol.StringAt(chatArr, 0),
			Title: protocol.StringAt(chatArr, 1),
		}
		if ts, ok := protocol.ValueAt(chatArr, 5, 0); ok {
			if epoch, ok := ts.(float64); ok {
				item.UpdatedAt = formatUnixMinuteUTC(int64(epoch))
			}
		}
		if item.Cid != "" {
			items = append(items, item)
		}
	}
	return items, nextCursor, nil
}

func formatUnixMinuteUTC(epoch int64) string {
	days := epoch / 86400
	rem := epoch % 86400
	if rem < 0 {
		rem += 86400
		days--
	}
	year, month, day := civilFromDays(days)
	hour := rem / 3600
	minute := (rem % 3600) / 60
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d", year, month, day, hour, minute)
}

func civilFromDays(days int64) (int64, int64, int64) {
	z := days + 719468
	era := z / 146097
	if z < 0 {
		era = (z - 146096) / 146097
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	year := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day := doy - (153*mp+2)/5 + 1
	month := mp + 3
	if mp >= 10 {
		month = mp - 9
		year++
	}
	return year, month, day
}
