// Package protocol contains protocol-level helpers for Gemini batchexecute RPCs.
//
// envelope.go handles the response wrapper around every RPC result: the XSSI prefix,
// Google's length-prefixed frames, and the wrb.fr item that carries one RPC body.
// RPC-specific decoders should receive only the inner body returned by ExtractRPCBody.
package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// StripResponsePrefix removes Google's XSSI prefix from a batchexecute response.
// The prefix is not part of the length-prefixed frame stream.
func StripResponsePrefix(response []byte) []byte {
	return bytes.TrimPrefix(response, []byte(")]}'\n"))
}

// ExtractRPCBody parses a batchexecute response and returns the wrb.fr body for rpcID.
// The returned body is the inner JSON string encoded as bytes; rejectCode is zero for accepted RPCs.
func ExtractRPCBody(response []byte, rpcID string) ([]byte, int, error) {
	frames := ParseLengthPrefixedFrames(response)

	for _, frame := range frames {
		var arr []any
		if err := json.Unmarshal(frame, &arr); err != nil {
			continue
		}

		items := findWrbFrItems(arr)
		for _, itemArr := range items {
			if len(itemArr) < 3 {
				continue
			}
			tag, _ := itemArr[0].(string)
			id, _ := itemArr[1].(string)
			if tag != "wrb.fr" || id != rpcID {
				continue
			}

			body := []byte{}
			if s, ok := itemArr[2].(string); ok {
				body = []byte(s)
			}
			var rejectCode int
			if len(itemArr) > 5 {
				if codeArr, ok := itemArr[5].([]any); ok && len(codeArr) > 0 {
					if f, ok := codeArr[0].(float64); ok {
						rejectCode = int(f)
					}
				}
			}
			return body, rejectCode, nil
		}
	}

	return nil, 0, fmt.Errorf("RPC response for %s not found", rpcID)
}

// ParseLengthPrefixedFrames parses Google's length-prefixed framing protocol.
// Format: <digits>\n<content_of_N_utf16_units> repeated.
// The length counts UTF-16 code units starting immediately after the digits.
// Incomplete frames are omitted so callers can retry after reading more bytes.
func ParseLengthPrefixedFrames(content []byte) [][]byte {
	var frames [][]byte
	pos := 0
	totalLen := len(content)

	for pos < totalLen {
		for pos < totalLen && (content[pos] == ' ' || content[pos] == '\t' || content[pos] == '\n' || content[pos] == '\r') {
			pos++
		}
		if pos >= totalLen {
			break
		}

		numStart := pos
		for pos < totalLen && content[pos] >= '0' && content[pos] <= '9' {
			pos++
		}
		if pos == numStart {
			pos++
			continue
		}

		if pos >= totalLen || content[pos] != '\n' {
			break
		}

		utf16Units := 0
		for _, ch := range content[numStart:pos] {
			utf16Units = utf16Units*10 + int(ch-'0')
		}

		contentStart := pos
		unitsConsumed := 0
		for pos < totalLen && unitsConsumed < utf16Units {
			r, size := utf8.DecodeRune(content[pos:])
			if r == utf8.RuneError && size == 0 {
				break
			}
			units := 1
			if r > 0xFFFF {
				units = 2
			}
			if unitsConsumed+units > utf16Units {
				break
			}
			unitsConsumed += units
			pos += size
		}

		if unitsConsumed < utf16Units {
			break
		}

		chunk := bytes.TrimSpace(content[contentStart:pos])
		if len(chunk) != 0 {
			frames = append(frames, chunk)
		}
	}

	return frames
}

// findWrbFrItems finds wrb.fr arrays at the top level or one level deeper.
// Browser responses commonly wrap wrb.fr items as [["wrb.fr", ...]] or [[["wrb.fr", ...]]].
func findWrbFrItems(arr []any) [][]any {
	var results [][]any
	for _, item := range arr {
		itemArr, ok := item.([]any)
		if !ok {
			continue
		}
		if len(itemArr) >= 2 {
			if tag, ok := itemArr[0].(string); ok && tag == "wrb.fr" {
				results = append(results, itemArr)
				continue
			}
		}
		for _, sub := range itemArr {
			subArr, ok := sub.([]any)
			if !ok {
				continue
			}
			if len(subArr) >= 2 {
				if tag, ok := subArr[0].(string); ok && tag == "wrb.fr" {
					results = append(results, subArr)
				}
			}
		}
	}
	return results
}
