// RPC: oMH3Zd — CreateNotebook
// Source-path: /notebooks/create
// Reject codes: none observed in sample fixtures
//
// Payload shape (verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	[["<title>", "", null, null, null, null, null, null, 0, null, 1,
//	  null, null, null, null, [2, null, null, null, 1]]]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	["notebooks/<uuid>", true]
//
// Test fixture: testdata/create_notebook_basic.txt
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const createNotebookRPCID = "oMH3Zd"

// EncodeCreateNotebook returns the CreateNotebook payload for a title.
func EncodeCreateNotebook(title string) (rpcID, payload string) {
	inner := make([]any, 17)
	inner[0] = title
	inner[1] = ""
	inner[8] = 0
	inner[10] = 1
	inner[16] = []any{2, nil, nil, nil, 1}
	payloadBytes, _ := json.Marshal([]any{inner})
	return createNotebookRPCID, string(payloadBytes)
}

// DecodeCreateNotebook parses the wrb.fr body JSON returned by ExtractRPCBody
// and returns the created notebook's resource name ("notebooks/<uuid>").
func DecodeCreateNotebook(body []byte) (string, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "[]" {
		return "", nil
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("decode CreateNotebook JSON: %w", err)
	}
	if len(data) == 0 {
		return "", nil
	}
	resource, _ := data[0].(string)
	return resource, nil
}
