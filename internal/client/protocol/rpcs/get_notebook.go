// RPC: HcT8bb — GetNotebook
// Source-path: /notebooks/view (prefetch, full form) or /notebook/<uuid> (page load)
// Reject codes: none observed in sample fixtures
//
// Payload shape (verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	["notebooks/<uuid>", ["<lang>"], 0]   full form (list-page click / prefetch)
//	["notebooks/<uuid>"]                  short form (notebook page load)
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[["notebooks/<uuid>", <notebook_arr>, []]]
//
//	notebook_arr structure:
//	  [0]: title
//	  [6]: theme — ["", null, null, [color,...], [color,...]]
//	  [10]: sources block — [0] source records, [1] source detail entries
//	        detail entry: [resource_name, [resource_name, filename, [sec, nanos],
//	        null, null, null, status, null, mime, null, null, ""]]
//	  [14]: [emoji, [updated_sec, nanos], 1, null, [created_sec, nanos], ...]
//	        (created/updated mapping is inferred from a single sample)
//
// Test fixture: testdata/get_notebook_basic.txt
//
// Notes:
//   - The encoder always emits the full form; the server also accepts the short form.
//   - Empty bodies decode to nil because notebook reads can be absent.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getNotebookRPCID = "HcT8bb"

// NotebookSource is one source file attached to a notebook.
type NotebookSource struct {
	ResourceName string // "notebooks/<uuid>/sources/<source_uuid>"
	FileName     string
	MimeType     string
	UploadedUnix int64
}

// Notebook is the decoded GetNotebook result.
type Notebook struct {
	ResourceName string // "notebooks/<uuid>"
	Title        string
	Emoji        string
	Sources      []NotebookSource
	CreatedUnix  int64
	UpdatedUnix  int64
}

// EncodeGetNotebook returns the full-form GetNotebook payload. The id may be
// passed with or without the "notebooks/" prefix.
func EncodeGetNotebook(notebookID string, lang string) (rpcID, payload string) {
	if lang == "" {
		lang = "en"
	}
	resource := notebookID
	if !strings.HasPrefix(resource, "notebooks/") {
		resource = "notebooks/" + resource
	}
	payloadBytes, _ := json.Marshal([]any{resource, []any{lang}, 0})
	return getNotebookRPCID, string(payloadBytes)
}

// DecodeGetNotebook parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetNotebook(body []byte) (*Notebook, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetNotebook JSON: %w", err)
	}

	outer, ok := protocol.ArrayAt(data, 0)
	if !ok || len(outer) == 0 {
		return nil, nil
	}

	nb := &Notebook{ResourceName: protocol.StringAt(outer, 0)}
	arr, ok := protocol.ArrayAt(outer, 1)
	if !ok {
		return nb, nil
	}

	nb.Title = protocol.StringAt(arr, 0)

	if meta, ok := protocol.ArrayAt(arr, 14); ok {
		nb.Emoji = protocol.StringAt(meta, 0)
		nb.UpdatedUnix = unixAt(meta, 1)
		nb.CreatedUnix = unixAt(meta, 4)
	}

	sourcesBlock, ok := protocol.ArrayAt(arr, 10)
	if !ok || len(sourcesBlock) < 2 {
		return nb, nil
	}
	details, ok := sourcesBlock[1].([]any)
	if !ok {
		return nb, nil
	}
	for _, raw := range details {
		entry, ok := raw.([]any)
		if !ok || len(entry) < 2 {
			continue
		}
		fields, ok := entry[1].([]any)
		if !ok {
			continue
		}
		src := NotebookSource{
			ResourceName: protocol.StringAt(fields, 0),
			FileName:     protocol.StringAt(fields, 1),
			UploadedUnix: unixAt(fields, 2),
		}
		if len(fields) > 8 {
			src.MimeType = protocol.StringAt(fields, 8)
		}
		if src.ResourceName == "" && src.FileName == "" {
			continue
		}
		nb.Sources = append(nb.Sources, src)
	}
	return nb, nil
}

// unixAt reads a [seconds, nanos] timestamp pair and returns the seconds.
func unixAt(arr []any, idx int) int64 {
	if idx >= len(arr) {
		return 0
	}
	pair, ok := arr[idx].([]any)
	if !ok || len(pair) == 0 {
		return 0
	}
	if f, ok := pair[0].(float64); ok {
		return int64(f)
	}
	return 0
}
