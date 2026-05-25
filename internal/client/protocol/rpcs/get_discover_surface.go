// RPC: V8rlHe — GetDiscoverSurface
// Source-path: /images
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[["en"], [[390,391,...]]]
//	  ↑       ↑
//	 language surface filter ids
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[surface_group, ...], ...]
//
//	image card list is observed at data[0][0][4][0][0].
//	card structure:
//	  [0]: card id
//	  [1][0]: title
//	  [1][1]: preview image URL
//	  [1][2][0][19][1]: description
//
// Test fixture: testdata/get_discover_surface_basic.txt
//
// Notes:
//   - Production responses are large; the test fixture keeps only a small card sample.
//   - Cards describe Discover surface image prompts and transformations.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

const getDiscoverSurfaceRPCID = "V8rlHe"

// DiscoverCard is one Discover surface card.
type DiscoverCard struct {
	ID          string
	Title       string
	PreviewURL  string
	Description string
}

// EncodeGetDiscoverSurface returns the payload for a language and filter list.
func EncodeGetDiscoverSurface(language string, filters []int) (rpcID, payload string) {
	if language == "" {
		language = "en"
	}
	filterValues := make([]any, len(filters))
	for i, filter := range filters {
		filterValues[i] = filter
	}
	payloadBytes, _ := json.Marshal([]any{[]any{language}, []any{filterValues}})
	return getDiscoverSurfaceRPCID, string(payloadBytes)
}

// DecodeGetDiscoverSurface parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeGetDiscoverSurface(body []byte) ([]DiscoverCard, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, nil
	}

	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode GetDiscoverSurface JSON: %w", err)
	}

	items, ok := protocol.ArrayAt(data, 0, 0, 4, 0, 0)
	if !ok {
		return nil, nil
	}

	cards := make([]DiscoverCard, 0, len(items))
	for idx := range items {
		item, ok := protocol.ArrayAt(items, idx)
		if !ok {
			continue
		}
		details, _ := protocol.ArrayAt(item, 1)
		card := DiscoverCard{
			ID:          protocol.StringAt(item, 0),
			Title:       protocol.StringAt(details, 0),
			PreviewURL:  protocol.StringAt(details, 1),
			Description: protocol.StringAt(details, 2, 0, 19, 1),
		}
		if card.ID == "" && card.Title == "" {
			continue
		}
		cards = append(cards, card)
	}
	return cards, nil
}
