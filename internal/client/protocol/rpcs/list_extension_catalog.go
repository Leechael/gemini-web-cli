// RPC: uPDUsc — ListExtensionCatalog
// Source-path: any Gemini virtual page (defaults to /app)
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	["en"]
//	  ↑
//	 language
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[extension, ...], [category, ...]]
//	  ↑                 ↑
//	  full catalog      grouped presentation metadata
//
//	extension structure:
//	  [0]: extension id
//	  [1]: display name
//	  [2]: description
//	  [5]: icon URL
//	  [9]: enabled flag when present
//
// Test fixture: testdata/list_extension_catalog_basic.txt
//
// Notes:
//   - This RPC returns the complete extension catalog. ListEnabledTools returns
//     the smaller set already enabled for the current account.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const listExtensionCatalogRPCID = "uPDUsc"

// Extension is one entry in the Gemini extension catalog.
type Extension struct {
	ID          string
	Name        string
	Description string
	IconURL     string
	Enabled     bool
}

// EncodeListExtensionCatalog returns (rpcID, payload JSON string).
func EncodeListExtensionCatalog() (rpcID, payload string) {
	payloadBytes, _ := json.Marshal([]any{"en"})
	return listExtensionCatalogRPCID, string(payloadBytes)
}

// DecodeListExtensionCatalog parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListExtensionCatalog(body []byte) ([]Extension, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("ListExtensionCatalog body is empty")
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ListExtensionCatalog JSON: %w", err)
	}

	catalog, ok := protocol.ArrayAt(data, 0)
	if !ok {
		return nil, fmt.Errorf("ListExtensionCatalog response missing catalog data")
	}

	extensions := []Extension{}
	for idx := range catalog {
		item, ok := protocol.ArrayAt(catalog, idx)
		if !ok {
			continue
		}
		ext := Extension{
			ID:          protocol.StringAt(item, 0),
			Name:        protocol.StringAt(item, 1),
			Description: protocol.StringAt(item, 2),
			IconURL:     protocol.StringAt(item, 5),
			Enabled:     protocol.BoolAt(item, 9),
		}
		if ext.ID == "" && ext.Name == "" {
			continue
		}
		extensions = append(extensions, ext)
	}
	if len(extensions) == 0 {
		return nil, fmt.Errorf("ListExtensionCatalog response did not contain extensions")
	}
	return extensions, nil
}
