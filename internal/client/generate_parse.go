package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// QueueingError indicates a long-running video/music job was queued by the server.
type QueueingError struct {
	Hint string
}

func (e *QueueingError) Error() string { return e.Hint }

func (c *Client) parseStreamResponse(body io.Reader, cb StreamCallback) error {
	parser := newStreamFrameParser()
	buf := make([]byte, 64*1024)
	var lastText string
	var lastThoughts string
	var output *types.ModelOutput

	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			frames := parser.append(buf[:n])
			done := false
			for _, frame := range frames {
				var envelope []any
				if err := json.Unmarshal(frame, &envelope); err != nil {
					continue
				}
				if isQueueingFrame(envelope) {
					return &QueueingError{Hint: "job queued; check progress with: gemini-web-cli progress <cid>"}
				}
				parsed, err := rpcs.DecodeStreamGenerateFrame(envelope)
				if err != nil {
					if eerr, ok := err.(*rpcs.EnvelopeError); ok {
						if eerr.Code == 1052 {
							return &ModelUnavailableError{Code: eerr.Code}
						}
						return fmt.Errorf("server returned error code %d", eerr.Code)
					}
					return err
				}
				if parsed == nil {
					continue
				}
				if parsed.Text != "" {
					parsed.TextDelta = calculateDelta(lastText, parsed.Text)
					lastText = parsed.Text
				}
				if parsed.Thoughts != "" {
					parsed.ThoughtsDelta = calculateDelta(lastThoughts, parsed.Thoughts)
					lastThoughts = parsed.Thoughts
				}
				output = parsed
				cb(parsed)
				if parsed.Done {
					done = true
					break
				}
			}
			if done {
				return nil
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("reading stream: %w", readErr)
		}
	}

	if output == nil {
		snippet := parser.snippet()
		if len(snippet) > 0 {
			return fmt.Errorf("no valid response frames parsed, raw response: %s", snippet)
		}
		return fmt.Errorf("no valid response frames parsed (empty response body)")
	}
	return nil
}

type streamFrameParser struct {
	buf            []byte
	prefixStripped bool
	debug          []byte
}

func newStreamFrameParser() *streamFrameParser {
	return &streamFrameParser{}
}

func (p *streamFrameParser) append(chunk []byte) [][]byte {
	p.appendDebug(chunk)
	p.buf = append(p.buf, chunk...)
	p.stripPrefixIfReady()

	var frames [][]byte
	for {
		p.trimLeadingWhitespace()
		if len(p.buf) == 0 {
			break
		}

		digitEnd := 0
		for digitEnd < len(p.buf) && p.buf[digitEnd] >= '0' && p.buf[digitEnd] <= '9' {
			digitEnd++
		}
		if digitEnd == 0 {
			p.buf = p.buf[1:]
			continue
		}
		if digitEnd == len(p.buf) {
			break
		}
		if p.buf[digitEnd] != '\n' {
			break
		}

		declared := 0
		for _, ch := range p.buf[:digitEnd] {
			declared = declared*10 + int(ch-'0')
		}
		contentStart := digitEnd
		frameBytes, ok := utf16PrefixBytes(p.buf[contentStart:], declared)
		if !ok {
			break
		}

		chunk := bytes.TrimSpace(p.buf[contentStart : contentStart+frameBytes])
		if len(chunk) != 0 {
			frame := make([]byte, len(chunk))
			copy(frame, chunk)
			frames = append(frames, frame)
		}
		p.buf = p.buf[contentStart+frameBytes:]
	}
	return frames
}

func (p *streamFrameParser) stripPrefixIfReady() {
	if p.prefixStripped {
		return
	}
	prefix := []byte(")]}'\n")
	if len(p.buf) < len(prefix) && bytes.HasPrefix(prefix, p.buf) {
		return
	}
	p.buf = bytes.TrimPrefix(p.buf, prefix)
	p.prefixStripped = true
}

func (p *streamFrameParser) trimLeadingWhitespace() {
	for len(p.buf) > 0 && (p.buf[0] == ' ' || p.buf[0] == '\t' || p.buf[0] == '\n' || p.buf[0] == '\r') {
		p.buf = p.buf[1:]
	}
}

func (p *streamFrameParser) appendDebug(chunk []byte) {
	const maxDebugBytes = 500
	if len(p.debug) >= maxDebugBytes {
		return
	}
	remaining := maxDebugBytes - len(p.debug)
	if len(chunk) > remaining {
		chunk = chunk[:remaining]
	}
	p.debug = append(p.debug, chunk...)
}

func (p *streamFrameParser) snippet() string {
	snippet := string(p.debug)
	if len(snippet) > 500 {
		snippet = snippet[:500] + "..."
	}
	return snippet
}

func utf16PrefixBytes(content []byte, wantUnits int) (int, bool) {
	pos := 0
	unitsConsumed := 0
	for pos < len(content) && unitsConsumed < wantUnits {
		r, size := utf8.DecodeRune(content[pos:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		units := 1
		if r > 0xFFFF {
			units = 2
		}
		if unitsConsumed+units > wantUnits {
			break
		}
		unitsConsumed += units
		pos += size
	}
	if unitsConsumed < wantUnits {
		return 0, false
	}
	return pos, true
}

func isQueueingFrame(envelope []any) bool {
	// Stub until a real Veo/Lyria queueing HAR confirms the exact envelope path.
	// Do not substring-match assistant text; normal prompts can mention these words.
	return false
}
