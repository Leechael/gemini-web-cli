package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// StripResponsePrefix removes Google's XSSI prefix from a batchexecute response.
func StripResponsePrefix(response []byte) []byte {
	return bytes.TrimPrefix(response, []byte(")]}'\n"))
}

// ExtractRPCBody parses a batchexecute response and returns the wrb.fr body for rpcID.
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
func ParseLengthPrefixedFrames(content []byte) [][]byte {
	var frames [][]byte
	runes := []rune(string(content))
	pos := 0
	totalLen := len(runes)

	for pos < totalLen {
		for pos < totalLen && (runes[pos] == ' ' || runes[pos] == '\t' || runes[pos] == '\n' || runes[pos] == '\r') {
			pos++
		}
		if pos >= totalLen {
			break
		}

		numStart := pos
		for pos < totalLen && runes[pos] >= '0' && runes[pos] <= '9' {
			pos++
		}
		if pos == numStart {
			pos++
			continue
		}

		if pos >= totalLen || runes[pos] != '\n' {
			break
		}

		utf16Units := 0
		for _, ch := range string(runes[numStart:pos]) {
			utf16Units = utf16Units*10 + int(ch-'0')
		}

		contentStart := pos
		unitsConsumed := 0
		for pos < totalLen && unitsConsumed < utf16Units {
			r := runes[pos]
			units := 1
			if r > 0xFFFF {
				units = 2
			}
			if unitsConsumed+units > utf16Units {
				break
			}
			unitsConsumed += units
			pos++
		}

		if unitsConsumed < utf16Units {
			break
		}

		chunk := strings.TrimSpace(string(runes[contentStart:pos]))
		if chunk != "" {
			frames = append(frames, []byte(chunk))
		}
	}

	return frames
}

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
