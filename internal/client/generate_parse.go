package client

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// QueueingError indicates a long-running video/music job was queued by the server.
type QueueingError struct {
	Hint string
}

func (e *QueueingError) Error() string { return e.Hint }

func (c *Client) parseStreamResponse(body io.Reader, cb StreamCallback) error {
	var allData []byte
	buf := make([]byte, 64*1024)
	var lastText string
	var output *types.ModelOutput
	framesProcessed := 0

	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			allData = append(allData, buf[:n]...)
			frames := protocol.ParseLengthPrefixedFrames(protocol.StripResponsePrefix(allData))
			done := false
			for i := framesProcessed; i < len(frames); i++ {
				var envelope []any
				if err := json.Unmarshal(frames[i], &envelope); err != nil {
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
				output = parsed
				cb(parsed)
				if parsed.Done {
					done = true
					break
				}
			}
			framesProcessed = len(frames)
			if done {
				return nil
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if output != nil {
				return nil
			}
			return fmt.Errorf("reading stream: %w", readErr)
		}
	}

	if output == nil {
		snippet := string(allData)
		if len(snippet) > 500 {
			snippet = snippet[:500] + "..."
		}
		if len(snippet) > 0 {
			return fmt.Errorf("no valid response frames parsed, raw response: %s", snippet)
		}
		return fmt.Errorf("no valid response frames parsed (empty response body)")
	}
	return nil
}

func isQueueingFrame(envelope []any) bool {
	b, err := json.Marshal(envelope)
	if err != nil {
		return false
	}
	s := string(b)
	return strings.Contains(s, "queueing=True") || strings.Contains(s, "Stream suspended")
}
