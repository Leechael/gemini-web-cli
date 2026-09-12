// RPC: ko3zcd — AddNotebookSource
// Source-path: /notebook/<uuid>
// Reject codes: none observed in sample fixtures
//
// Payload shape (verified against boq_assistant-bard-web-server_20260910.05_p2):
//
//	["notebooks/<uuid>",
//	  [null, "<filename>", null, null,
//	    ["/contrib_service/ttl_1d/<upload_token>", "<mime>"], null, null, null, "<mime>"],
//	  [1, 3]]
//
// The upload token comes from the standard resumable upload flow
// (push.clients6.google.com/upload/ — see upload.go); notebooks reuse it
// unchanged.
//
// URL-source variant (verified in the same capture):
//
//	["notebooks/<uuid>",
//	  [null, "<url>", null, ["<url>"], null, null, null, null, "text/html"],
//	  [1, 3]]
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[resource_name, <notebook_arr>]] — same layout as the GetNotebook body;
//	decoded via DecodeGetNotebook.
//
// Test fixture: testdata/add_notebook_source_basic.txt
package rpcs

import (
	"encoding/json"
	"strings"
)

const addNotebookSourceRPCID = "ko3zcd"

// EncodeAddNotebookSource returns the AddNotebookSource payload. The notebook
// id may be passed with or without the "notebooks/" prefix; the upload token
// is the "/contrib_service/..." string returned by the resumable upload.
func EncodeAddNotebookSource(notebookID string, fileName string, mimeType string, uploadToken string) (rpcID, payload string) {
	resource := notebookID
	if !strings.HasPrefix(resource, "notebooks/") {
		resource = "notebooks/" + resource
	}
	source := []any{
		nil, fileName, nil, nil,
		[]any{uploadToken, mimeType},
		nil, nil, nil, mimeType,
	}
	payloadBytes, _ := json.Marshal([]any{resource, source, []any{1, 3}})
	return addNotebookSourceRPCID, string(payloadBytes)
}

// EncodeAddNotebookURLSource returns the ko3zcd payload variant that attaches
// a web URL as a notebook source.
func EncodeAddNotebookURLSource(notebookID string, url string) (rpcID, payload string) {
	resource := notebookID
	if !strings.HasPrefix(resource, "notebooks/") {
		resource = "notebooks/" + resource
	}
	source := []any{
		nil, url, nil,
		[]any{url},
		nil, nil, nil, nil, "text/html",
	}
	payloadBytes, _ := json.Marshal([]any{resource, source, []any{1, 3}})
	return addNotebookSourceRPCID, string(payloadBytes)
}

// DecodeAddNotebookSource parses the ko3zcd response body. The response is a
// full notebook object with the same layout as GetNotebook, so it reuses
// DecodeGetNotebook.
func DecodeAddNotebookSource(body []byte) (*Notebook, error) {
	return DecodeGetNotebook(body)
}
