package client

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

func TestParseStreamResponse_ChunkedFrame(t *testing.T) {
	body := makeStreamBody(t, "hello", true)
	reader := &chunkReader{chunks: [][]byte{body[:3], body[3:9], body[9:27], body[27:]}}

	var got []*types.ModelOutput
	err := (&Client{}).parseStreamResponse(reader, func(out *types.ModelOutput) {
		got = append(got, out)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("outputs = %d, want 1", len(got))
	}
	if got[0].Text != "hello" || !got[0].Done {
		t.Fatalf("output = %#v", got[0])
	}
}

func TestParseStreamResponse_ProtocolFixture(t *testing.T) {
	body, err := os.ReadFile("protocol/testdata/stream_generate_basic_response.txt")
	if err != nil {
		t.Fatal(err)
	}
	var got []*types.ModelOutput
	err = (&Client{}).parseStreamResponse(strings.NewReader(string(body)), func(out *types.ModelOutput) {
		got = append(got, out)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "Sample assistant response." || !got[0].Done {
		t.Fatalf("outputs = %#v", got)
	}
}

func TestParseStreamResponse_ReturnsNonEOFReadErrorAfterOutput(t *testing.T) {
	body := makeStreamBody(t, "partial", false)
	boom := errors.New("boom")
	reader := &chunkReader{chunks: [][]byte{body}, errs: []error{boom}}

	err := (&Client{}).parseStreamResponse(reader, func(out *types.ModelOutput) {})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want boom", err)
	}
}

func makeStreamBody(t *testing.T, text string, done bool) []byte {
	t.Helper()
	content := make([]any, 5)
	content[1] = []any{"c_abc", "r_def"}
	content[4] = []any{[]any{"rc_ghi", []any{text}}}
	if done {
		for len(content) <= 25 {
			content = append(content, nil)
		}
		content[25] = "ctx"
	}
	contentJSON, _ := json.Marshal(content)
	frameJSON, _ := json.Marshal([]any{[]any{"wrb.fr", nil, string(contentJSON)}})
	framed := "\n" + string(frameJSON) + "\n"
	return []byte(")]}'\n" + strconv.Itoa(utf16Units(framed)) + framed)
}

func utf16Units(s string) int {
	units := 0
	for _, r := range s {
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return units
}

type chunkReader struct {
	chunks [][]byte
	errs   []error
	idx    int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.chunks) {
		return 0, io.EOF
	}
	chunk := r.chunks[r.idx]
	err := error(nil)
	if r.idx < len(r.errs) {
		err = r.errs[r.idx]
	}
	r.idx++
	return copy(p, chunk), err
}
