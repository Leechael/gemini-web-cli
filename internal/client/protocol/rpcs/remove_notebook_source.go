// RPC: AptDmf — RemoveNotebookSource
// Source-path: /notebook/<uuid>
// Reject codes: none observed in sample fixtures
//
// Payload shape (verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	["notebooks/<uuid>/sources/<source_uuid>", [1, 3]]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[]
//
// Test fixture: testdata/remove_notebook_source_basic.txt
//
// Notes:
//   - Semantics confirmed by the capture author: the URL source was invalid
//     (failed to ingest) and the user removed it — this RPC deletes a source
//     from a notebook.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const removeNotebookSourceRPCID = "AptDmf"

// EncodeRemoveNotebookSource returns the RemoveNotebookSource payload. The
// source must be a full resource name ("notebooks/<uuid>/sources/<sid>").
func EncodeRemoveNotebookSource(sourceResource string) (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{sourceResource, []any{1, 3}})
	return removeNotebookSourceRPCID, string(payloadBytes)
}

// DecodeRemoveNotebookSource validates the RemoveNotebookSource response body.
func DecodeRemoveNotebookSource(body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "[]" {
		return nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("decode RemoveNotebookSource JSON: %w", err)
	}
	return nil
}
