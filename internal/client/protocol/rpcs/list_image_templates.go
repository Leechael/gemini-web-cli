// RPC: XhaU0b — ListImageTemplates
// Source-path: /images
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[4,[2],3]
//	↑  ↑   ↑
//	observed fixed selector values for the images template surface
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[], [[[template, ...]]]]
//	       ↑
//	       data[1][0][0]
//
//	template structure:
//	  [0]: template id
//	  [1]: display name
//	  [2][0][0]: preview image URL
//
// Test fixture: testdata/list_image_templates_basic.txt
//
// Notes:
//   - The observed payload is stable and has no pagination inputs.
//   - Template ids are kept for future StreamGenerate template wiring.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const listImageTemplatesRPCID = "XhaU0b"

// ImageTemplate is one image generation style template.
type ImageTemplate struct {
	ID         string
	Name       string
	PreviewURL string
}

// EncodeListImageTemplates returns the observed fixed payload.
func EncodeListImageTemplates() (rpcID, payload string) {
	return listImageTemplatesRPCID, "[4,[2],3]"
}

// DecodeListImageTemplates parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListImageTemplates(body []byte) ([]ImageTemplate, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ListImageTemplates JSON: %w", err)
	}

	items, ok := protocol.ArrayAt(data, 1, 0, 0)
	if !ok {
		return nil, nil
	}

	templates := make([]ImageTemplate, 0, len(items))
	for idx := range items {
		item, ok := protocol.ArrayAt(items, idx)
		if !ok {
			continue
		}
		tmpl := ImageTemplate{
			ID:         protocol.StringAt(item, 0),
			Name:       protocol.StringAt(item, 1),
			PreviewURL: protocol.StringAt(item, 2, 0, 0),
		}
		if tmpl.ID == "" && tmpl.Name == "" {
			continue
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}
