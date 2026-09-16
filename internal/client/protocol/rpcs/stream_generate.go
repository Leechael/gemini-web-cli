// RPC: StreamGenerate (BardFrontendService) — non-batchexecute endpoint
// URL-path: /BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate
// Notes: this endpoint does not use batchexecute, so it has no source-path query parameter.
// Reject codes: HTTP-level 429 (RateLimit) / envelope code 1052 (ModelUnavailable)
//
// Inner request shape (99-element array, verified against boq_assistant-bard-web-server_20260910.05_p2):
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
//	[6]: [0] — always 0 in the 20260910.05_p2 build, including deep research
//	     plan-generation requests (the old [1] deep-research marker is stale;
//	     no [1] sample exists in current captures).
//	[7]: 1 — enable snapshot streaming
//	[10]: 1
//	[11]: 0
//	[17]: [[0]] (new) / [[1]] (continuation — decided by rid, not cid)
//	[18]: 0
//	[19]: "notebooks/<uuid>" (notebook-scoped chats only)
//	[27]: 1
//	[30]: [4]
//	[40]: notebook scope — 14-element, [13]=[2] (first turn) / [13]=[2,null,null,null,1] (continuation)
//	[41]: [1]
//	[49]: mode flag — 11 (video) / 14 (image generation, verified on /images surface) /
//	      21 (music) / 1 (deep research).
//	      Note: 14 was previously labeled "image-to-video" from upstream docs; the
//	      20260910.05_p2 capture proves 14 is used for plain image generation with no
//	      uploads. The image-to-video semantics of 14 remain unverified.
//	[53]: 0
//	[54]: [] (video) / [[[[[1]]]]] (deep research)
//	[55]: [[16]] (video) / [[1]] (deep research)
//	[59]: UUID
//	[61]: []
//	[68]: 1
//	[79]: modelSelector(model) — 1/2/3/4 by tier
//	[80]: 1 (text) / 2 (image)
//	[91]: 0
//	[96]: 1 (first turn) / 0 (continuation)
//	[98]: 1
//
// Slots 91/96/98 were added by the 20260910.05_p2 web build.
//
// Verified slot rules (5 captures across text/image/notebook surfaces):
//
//	[17]: [[N]] where N is the number of prior turns the request builds on —
//	      [[0]] for a new chat or a parentless continuation, [[1]] after one
//	      prior turn, [[11]] observed in a live session with eleven prior turns.
//	      This client cannot recover the true count from a bare cid/rid/rcid
//	      triple, so it sends [[1]] for any continuation (an approximation the
//	      server has always tolerated).
//	[19]: notebook resource name ("notebooks/<uuid>") for notebook-scoped chats.
//	[40]: notebook scope — 14-element array, [13]=[2] on the first turn and
//	      [13]=[2,null,null,null,1] on continuation turns.
//	[67]: 0 for plain text-chat continuation turns (verified: round-1 capture,
//	      a live 12-turn session, and a plain message in a completed research
//	      chat); null for first turns, /images, notebooks, and while a deep
//	      research is still running.
//	[96]: 1 only for first turns initiated from a dedicated surface landing
//	      page (/images, /notebook); 0 for /app text chats (new or
//	      continuation) and for deep research plan requests.
//
// Mode-dependent slots:
//
//	image mode: [49]=14 [80]=2, model header [15]=2
//	text/other: [80]=1, model header [15]=1
//
// Video/music/deep-research variants of the new slots have no capture evidence yet.
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
	Prompt           string
	Language         string
	Metadata         []string
	Uploads          []FileRef
	Mode             string
	DeepResearch     bool
	ModelSelector    int
	UUID             string
	EntropyToken     string
	HexUUID          string
	NotebookResource string // "notebooks/<uuid>" for notebook-scoped chats
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
	// StreamGenerate rejects arrive after HTTP 200 with a wrb.fr envelope code.
	// Include http=200 so operators do not mistake this for a transport failure.
	return fmt.Sprintf("stream envelope reject code %d (http=200)", e.Code)
}

// RejectCode exposes the protocol reject code to transport diagnostics.
func (e *EnvelopeError) RejectCode() int { return e.Code }

// EncodeStreamGenerate constructs the full StreamGenerate inner request.
func EncodeStreamGenerate(opts EncodeStreamGenerateOpts) []any {
	req := make([]any, 99)
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
	isNewChat := len(opts.Metadata) == 0 || opts.Metadata[0] == ""
	hasRid := len(opts.Metadata) > 1 && opts.Metadata[1] != ""
	req[17] = []any{[]any{0}}
	if hasRid {
		req[17] = []any{[]any{1}}
	}
	req[18] = 0
	if opts.NotebookResource != "" {
		req[19] = opts.NotebookResource
		notebookScope := make([]any, 14)
		if isNewChat {
			notebookScope[13] = []any{2}
		} else {
			notebookScope[13] = []any{2, nil, nil, nil, 1}
		}
		req[40] = notebookScope
	}
	req[27] = 1
	req[30] = []any{4}
	req[41] = []any{1}
	req[53] = 0
	req[59] = opts.UUID
	req[61] = []any{}
	if !isNewChat && opts.Mode != "image" && opts.NotebookResource == "" && !opts.DeepResearch {
		req[67] = 0
	}
	req[91] = 0
	if isNewChat && (opts.Mode == "image" || opts.NotebookResource != "") {
		req[96] = 1
	} else {
		req[96] = 0
	}
	req[98] = 1
	req[79] = opts.ModelSelector
	req[80] = 1

	if opts.DeepResearch {
		req[49] = 1
		req[54] = []any{[]any{[]any{[]any{[]any{1}}}}}
		req[55] = []any{[]any{1}}
		req[68] = 1
	} else {
		switch opts.Mode {
		case "image":
			req[49] = 14
			req[80] = 2
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
			if thoughts := protocol.StringAt(cand, 37, 0, 0); thoughts != "" {
				out.Thoughts = html.UnescapeString(thoughts)
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
