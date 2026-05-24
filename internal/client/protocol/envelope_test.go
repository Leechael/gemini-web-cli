package protocol

import (
	"embed"
	"encoding/json"
	"strings"
	"testing"
)

//go:embed testdata/o30o0e_basic.txt
var testdata embed.FS

func TestStripResponsePrefix(t *testing.T) {
	got := StripResponsePrefix([]byte(")]}'\n123"))
	if string(got) != "123" {
		t.Fatalf("StripResponsePrefix = %q, want 123", got)
	}
}

func TestParseLengthPrefixedFrames(t *testing.T) {
	response := makeFramedResponse(`[["wrb.fr","rpc","[]",null,null,[7]]]`, `[["di",1]]`)
	frames := ParseLengthPrefixedFrames(response)
	if len(frames) != 2 {
		t.Fatalf("frames len = %d, want 2", len(frames))
	}
	if string(frames[0]) != `[["wrb.fr","rpc","[]",null,null,[7]]]` {
		t.Fatalf("frame[0] = %s", frames[0])
	}
}

func TestExtractRPCBody(t *testing.T) {
	response := StripResponsePrefix(makeFramedResponse(`[["wrb.fr","target","[1,2,3]",null,null,[4]]]`))
	body, rejectCode, err := ExtractRPCBody(response, "target")
	if err != nil {
		t.Fatalf("ExtractRPCBody: %v", err)
	}
	if string(body) != "[1,2,3]" {
		t.Fatalf("body = %s, want [1,2,3]", body)
	}
	if rejectCode != 4 {
		t.Fatalf("rejectCode = %d, want 4", rejectCode)
	}
}

func TestExtractRPCBody_FromHARSample(t *testing.T) {
	raw, err := testdata.ReadFile("testdata/o30o0e_basic.txt")
	if err != nil {
		t.Fatal(err)
	}
	body, rejectCode, err := ExtractRPCBody(StripResponsePrefix(raw), "o30O0e")
	if err != nil {
		t.Fatalf("ExtractRPCBody: %v", err)
	}
	if rejectCode != 0 {
		t.Fatalf("rejectCode = %d, want 0", rejectCode)
	}
	var decoded []any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("decoded body is empty")
	}
}

func makeFramedResponse(frames ...string) []byte {
	var b strings.Builder
	b.WriteString(")]}'\n")
	for _, frame := range frames {
		content := "\n" + frame + "\n"
		b.WriteString(utf16LenString(content))
		b.WriteString(content)
	}
	return []byte(b.String())
}

func utf16LenString(s string) string {
	units := 0
	for _, r := range s {
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return strconvItoa(units)
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
