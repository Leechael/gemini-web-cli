package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/client/transport"
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

func TestStreamGenerateRetriesCode13BeforeOutput(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		body := makeCode13StreamBody(t)
		if requests > 1 {
			body = makeStreamBody(t, "complete", true)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}

	callbacks := 0
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, func(*types.ModelOutput) {
		callbacks++
	})
	if err != nil {
		t.Fatalf("streamGenerate: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if callbacks != 1 {
		t.Fatalf("callbacks = %d, want 1", callbacks)
	}
}

func TestStreamGenerateDoesNotRetryCode13AfterOutput(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		body := makeStreamBodyWithCode13AfterText(t, "partial")
		if requests > 1 {
			body = makeStreamBody(t, "complete", true)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
	})}

	callbacks := 0
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, func(*types.ModelOutput) {
		callbacks++
	})
	assertEnvelopeErrorCode(t, err, 13)
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if callbacks != 1 {
		t.Fatalf("callbacks = %d, want 1", callbacks)
	}
}

func TestCallStreamGenerateRetriesTransientFailures(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		if requests < 3 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("stream")),
		}, nil
	})}

	body, err := c.CallStreamGenerate(t.Context(), transport.StreamGenerateRequest{InnerReq: []byte("[]")})
	if err != nil {
		t.Fatalf("CallStreamGenerate: %v", err)
	}
	defer body.Close()
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestStreamGenerateUsesOneThreeAttemptBudget(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		if requests%3 != 0 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(makeCode13StreamBody(t))),
		}, nil
	})}

	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, func(*types.ModelOutput) {})
	assertEnvelopeErrorCode(t, err, 13)
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func assertEnvelopeErrorCode(t *testing.T, err error, code int) {
	t.Helper()
	var envelopeErr *rpcs.EnvelopeError
	if !errors.As(err, &envelopeErr) || envelopeErr.Code != code {
		t.Fatalf("err = %v, want EnvelopeError code %d", err, code)
	}
}

type streamRoundTripFunc func(*http.Request) (*http.Response, error)

func (f streamRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func makeCode13StreamBody(t *testing.T) []byte {
	t.Helper()
	errorFrame, err := json.Marshal([]any{nil, nil, nil, nil, nil, []any{13}})
	if err != nil {
		t.Fatalf("Marshal error frame: %v", err)
	}
	framed := "\n" + string(errorFrame) + "\n"
	return []byte(")]}'\n" + strconv.Itoa(utf16Units(framed)) + framed)
}

func makeStreamBodyWithCode13AfterText(t *testing.T, text string) []byte {
	t.Helper()
	body := append([]byte(nil), makeStreamBody(t, text, false)...)
	errorFrame, err := json.Marshal([]any{nil, nil, nil, nil, nil, []any{13}})
	if err != nil {
		t.Fatalf("Marshal error frame: %v", err)
	}
	framed := "\n" + string(errorFrame) + "\n"
	body = append(body, []byte(strconv.Itoa(utf16Units(framed))+framed)...)
	return body
}

func TestIsRetryableStreamGenerateError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unexpected EOF before response",
			err:  &transport.StreamRequestError{Err: io.ErrUnexpectedEOF},
			want: true,
		},
		{
			name: "wrapped EOF before response",
			err:  &transport.StreamRequestError{Err: errors.New(`Post "https://gemini.google.com/...": unexpected EOF`)},
			want: true,
		},
		{
			name: "bad gateway",
			err:  &transport.HTTPStatusError{StatusCode: 502},
			want: true,
		},
		{
			name: "bad request",
			err:  &transport.HTTPStatusError{StatusCode: 400},
			want: false,
		},
		{
			name: "non stream request error",
			err:  errors.New("unexpected EOF"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableStreamGenerateError(tc.err); got != tc.want {
				t.Fatalf("isRetryableStreamGenerateError() = %t, want %t", got, tc.want)
			}
		})
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
