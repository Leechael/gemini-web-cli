// RPC: StreamGenerate (BardFrontendService) — non-batchexecute endpoint
// URL-path: /BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate
// Reject codes: HTTP-level 429 (RateLimit) / envelope code 1052 (ModelUnavailable)
//
// Inner request shape (81-element array):
//
//	[0]: message content
//	     no attachments: [prompt, 0, null, null, null, null, 0]
//	     with attachments: [prompt, 0, null, [[[uploadId, 1, null, mime], filename], ...], null, null, 0]
//	     video mode: message[9] = [null, null, null, null, null, null, [[null, null, null, 1]]]
//	[1]: [language]
//	[2]: metadata 10-element
//	     new chat: ['', '', '', null, null, null, null, null, null, '']
//	     continuation: [cid, rid, rcid, null, null, null, null, null, null, context]
//	[3]: "!" + base64(2600 random bytes) — request entropy
//	[4]: hex(16 random bytes) — request UUID
//	[6]: [0]
//	[7]: 1 — enable snapshot streaming
//	[10]: 1
//	[11]: 0
//	[17]: [[0]] (new) / [[1]] (continuation)
//	[18]: 0
//	[27]: 1
//	[30]: [4]
//	[41]: [1]
//	[45]: 1 if TemporaryChat else unset
//	[49]: mode flag — 11 (video) / 14 (image-to-video) / 21 (music) / 1 (deep research)
//	[53]: 0
//	[54]: [] (video) / [[[[[1]]]]] (deep research)
//	[55]: [[16]] (video) / [[1]] (deep research)
//	[59]: UUID
//	[61]: []
//	[68]: 2 (deep research) / 1 (normal)
//	[79]: modelSelector(model) — 1/2/3/4 by tier
//	[80]: 1
//
// Other slots default to nil (Go json.Marshal nil → null).
//
// Response shape: stream of length-prefixed JSON frames; each frame is a wrb.fr envelope
// containing a nested JSON string at envelope[2]. See DecodeStreamGenerateFrame for parser.
//
// Test fixtures: testdata/stream_generate_basic_*.json + variant fixtures
package rpcs

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// EncodeStreamGenerateOpts collects the inputs needed for the 81-element request.
type EncodeStreamGenerateOpts struct {
	Prompt        string
	Language      string
	Metadata      []string
	Uploads       []FileRef
	Mode          string
	DeepResearch  bool
	ModelSelector int
	UUID          string
	EntropyToken  string
	HexUUID       string
	TemporaryChat bool
}

// FileRef is the protocol-layer upload reference.
type FileRef struct {
	UploadID string
	MimeType string
	FileName string
}

// EnvelopeError reports a server error code embedded in a stream envelope.
type EnvelopeError struct {
	Code int
}

func (e *EnvelopeError) Error() string {
	return fmt.Sprintf("envelope error code %d", e.Code)
}

// EncodeStreamGenerate constructs the full StreamGenerate inner request.
func EncodeStreamGenerate(opts EncodeStreamGenerateOpts) []any {
	req := make([]any, 81)
	if opts.Language == "" {
		opts.Language = "en"
	}
	if opts.ModelSelector == 0 {
		opts.ModelSelector = 1
	}

	var message []any
	if len(opts.Uploads) > 0 {
		fileRefs := make([]any, 0, len(opts.Uploads))
		for _, u := range opts.Uploads {
			fileRefs = append(fileRefs, []any{[]any{u.UploadID, 1, nil, u.MimeType}, u.FileName})
		}
		message = []any{opts.Prompt, 0, nil, fileRefs, nil, nil, 0}
	} else {
		message = []any{opts.Prompt, 0, nil, nil, nil, nil, 0}
	}
	if opts.Mode == "video" {
		for len(message) < 10 {
			message = append(message, nil)
		}
		message[9] = []any{nil, nil, nil, nil, nil, nil, []any{[]any{nil, nil, nil, 1}}}
	}
	req[0] = message
	req[1] = []any{opts.Language}

	if len(opts.Metadata) > 0 {
		meta := make([]any, max(len(opts.Metadata), 10))
		for i, v := range opts.Metadata {
			if v != "" {
				meta[i] = v
			}
		}
		req[2] = meta
	} else {
		meta := make([]any, 10)
		meta[0] = ""
		meta[1] = ""
		meta[2] = ""
		meta[9] = ""
		req[2] = meta
	}

	req[3] = opts.EntropyToken
	req[4] = opts.HexUUID
	req[6] = []any{0}
	req[7] = 1
	req[10] = 1
	req[11] = 0
	req[17] = []any{[]any{0}}
	if len(opts.Metadata) > 0 && opts.Metadata[0] != "" {
		req[17] = []any{[]any{1}}
	}
	req[18] = 0
	req[27] = 1
	req[30] = []any{4}
	req[41] = []any{1}
	if opts.TemporaryChat {
		req[45] = 1
	}
	req[53] = 0
	req[59] = opts.UUID
	req[61] = []any{}
	req[79] = opts.ModelSelector
	req[80] = 1

	if opts.DeepResearch {
		req[49] = 1
		req[54] = []any{[]any{[]any{[]any{[]any{1}}}}}
		req[55] = []any{[]any{1}}
		req[68] = 2
	} else {
		switch opts.Mode {
		case "video":
			req[49] = 11
			req[54] = []any{}
			req[55] = []any{[]any{16}}
		case "image-to-video":
			req[49] = 14
		case "music":
			req[49] = 21
		}
		req[68] = 1
	}

	return req
}

// DecodeStreamGenerateFrame parses one wrb.fr envelope into a model output.
func DecodeStreamGenerateFrame(envelope []any) (*types.ModelOutput, error) {
	if len(envelope) == 0 {
		return nil, nil
	}
	if errCode := ExtractErrorCode(envelope); errCode != 0 {
		return nil, &EnvelopeError{Code: errCode}
	}
	return parseStreamGenerateEnvelope(envelope), nil
}

func parseStreamGenerateEnvelope(envelope []any) *types.ModelOutput {
	for len(envelope) == 1 {
		inner, ok := envelope[0].([]any)
		if !ok {
			break
		}
		envelope = inner
	}
	if len(envelope) < 3 {
		return nil
	}
	contentStr, ok := envelope[2].(string)
	if !ok || contentStr == "" {
		return nil
	}

	var content []any
	if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
		return nil
	}

	out := &types.ModelOutput{}
	if metaArr, ok := protocol.ArrayAt(content, 1); ok {
		for _, v := range metaArr {
			if s, ok := v.(string); ok {
				out.Metadata = append(out.Metadata, s)
			} else {
				out.Metadata = append(out.Metadata, "")
			}
		}
	}

	if candidates, ok := protocol.ArrayAt(content, 4); ok && len(candidates) > 0 {
		if cand, ok := candidates[0].([]any); ok {
			out.RCid = protocol.StringAt(cand, 0)
			if text := protocol.StringAt(cand, 1, 0); text != "" {
				out.Text = html.UnescapeString(text)
			}
			if out.Text == "" || strings.HasPrefix(out.Text, "http://googleusercontent.com/") {
				if alt := protocol.StringAt(cand, 22, 0); alt != "" {
					out.Text = html.UnescapeString(alt)
				}
			}
			if len(cand) > 12 && cand[12] != nil {
				out.Images = ExtractImages(cand[12])
				out.Videos = ExtractVideos(cand[12])
				out.Media = ExtractMedia(cand[12])
			}
			out.DeepResearchPlan = ExtractDeepResearchPlan(cand)
			if strings.HasPrefix(out.Text, "http://googleusercontent.com/") && (len(out.Images) > 0 || len(out.Videos) > 0 || len(out.Media) > 0) {
				out.Text = ""
			}
			out.Text = protocol.StripCardURLLines(out.Text)
		}
	}

	if out.RCid != "" && len(out.Metadata) >= 2 {
		for len(out.Metadata) < 10 {
			out.Metadata = append(out.Metadata, "")
		}
		if out.Metadata[2] == "" {
			out.Metadata[2] = out.RCid
		}
	}

	if contextStr := protocol.StringAt(content, 25); contextStr != "" {
		out.Done = true
		for len(out.Metadata) < 10 {
			out.Metadata = append(out.Metadata, "")
		}
		out.Metadata[9] = contextStr
	}
	if len(content) > 2 {
		if dictVal, ok := content[2].(map[string]any); ok {
			if contextStr, ok := dictVal["26"].(string); ok && contextStr != "" {
				out.Done = true
				for len(out.Metadata) < 10 {
					out.Metadata = append(out.Metadata, "")
				}
				out.Metadata[9] = contextStr
			}
		}
	}
	return out
}

// ExtractErrorCode tries multiple known paths to find an error code in the envelope.
func ExtractErrorCode(envelope []any) int {
	unwrapped := envelope
	for len(unwrapped) == 1 {
		inner, ok := unwrapped[0].([]any)
		if !ok {
			break
		}
		unwrapped = inner
	}
	if code := drillErrorCode(unwrapped, 0, 5, 2, 0, 1, 0); code != 0 {
		return code
	}
	if len(unwrapped) > 5 {
		if arr, ok := unwrapped[5].([]any); ok && len(arr) > 0 {
			if f, ok := arr[0].(float64); ok && f != 0 {
				return int(f)
			}
		}
	}
	return 0
}

func drillErrorCode(arr []any, indices ...int) int {
	current := arr
	for i, idx := range indices {
		if i == len(indices)-1 {
			if idx < len(current) {
				if f, ok := current[idx].(float64); ok {
					return int(f)
				}
			}
			return 0
		}
		next, ok := protocol.ArrayAt(current, idx)
		if !ok {
			return 0
		}
		current = next
	}
	return 0
}
