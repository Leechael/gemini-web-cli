package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	var readErr *streamReadError
	if !errors.As(err, &readErr) || !errors.Is(readErr, boom) {
		t.Fatalf("err = %v, want streamReadError wrapping boom", err)
	}
	if readErr.HTTPStatus != 200 || readErr.BytesRead == 0 || readErr.Frames == 0 {
		t.Fatalf("diagnostics = http=%d bytes=%d frames=%d, want http=200 with progress", readErr.HTTPStatus, readErr.BytesRead, readErr.Frames)
	}
	if !strings.Contains(err.Error(), "reading stream:") || !strings.Contains(err.Error(), "http=200") {
		t.Fatalf("err = %v, want reading stream prefix with http=200", err)
	}
}

func TestStreamGenerateRetriesBodyTimeoutBeforeOutput(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(&errReader{err: fmt.Errorf(
					"%w (Client.Timeout or context cancellation while reading body)",
					context.DeadlineExceeded,
				)}),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(makeStreamBody(t, "complete", true))),
		}, nil
	})}

	callbacks := 0
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {
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

func TestStreamGenerateDoesNotRetryBodyTimeoutAfterOutput(t *testing.T) {
	c := newTestClient()
	requests := 0
	timeoutErr := fmt.Errorf("%w (Client.Timeout or context cancellation while reading body)", context.DeadlineExceeded)
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		body := makeStreamBody(t, "partial", false)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(&chunkReader{chunks: [][]byte{body}, errs: []error{timeoutErr}}),
		}, nil
	})}

	callbacks := 0
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {
		callbacks++
	})
	var readErr *streamReadError
	if !errors.As(err, &readErr) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want streamReadError wrapping deadline exceeded", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if callbacks != 1 {
		t.Fatalf("callbacks = %d, want 1", callbacks)
	}
}

func TestStreamGenerateRetriesBodyTimeoutAfterMetadataOnly(t *testing.T) {
	c := newTestClient()
	requests := 0
	timeoutErr := fmt.Errorf("%w (Client.Timeout or context cancellation while reading body)", context.DeadlineExceeded)
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			metaOnly := makeStreamBodyMetadataOnly(t)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(&chunkReader{chunks: [][]byte{metaOnly}, errs: []error{timeoutErr}}),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(makeStreamBody(t, "complete", true))),
		}, nil
	})}

	var texts []string
	var metaCallbacks int
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(out *types.ModelOutput) {
		if out.TextDelta != "" {
			texts = append(texts, out.TextDelta)
		}
		if len(out.Metadata) > 0 && out.TextDelta == "" && out.ThoughtsDelta == "" {
			metaCallbacks++
		}
	})
	if err != nil {
		t.Fatalf("streamGenerate: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	// Failed attempt's metadata must be discarded; only the successful attempt
	// may deliver metadata (flushed when text arrives) plus the text frame.
	if metaCallbacks > 1 {
		t.Fatalf("metadata callbacks = %d, want at most 1 (no stale metadata from failed attempt)", metaCallbacks)
	}
	if len(texts) != 1 || texts[0] != "complete" {
		t.Fatalf("texts = %#v, want [complete]", texts)
	}
}

func TestStreamGenerateRetriesCode13AfterMetadataOnly(t *testing.T) {
	c := newTestClient()
	requests := 0
	c.httpClient = &http.Client{Transport: streamRoundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			metaThen13 := append([]byte(nil), makeStreamBodyMetadataOnly(t)...)
			metaThen13 = append(metaThen13, makeCode13StreamBody(t)[len(")]}'\n"):]...)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(metaThen13))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(makeStreamBody(t, "complete", true)))}, nil
	})}

	callbacks := 0
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {
		callbacks++
	})
	if err != nil {
		t.Fatalf("streamGenerate: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if callbacks < 1 {
		t.Fatalf("callbacks = %d, want at least 1", callbacks)
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
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {
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
	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {
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

	err := c.streamGenerate(t.Context(), "prompt", nil, nil, &types.Models[0], false, "", func(*types.ModelOutput) {})
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

func TestIsRetryableStreamBodyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "client timeout while reading body",
			err: &streamReadError{Err: fmt.Errorf(
				"%w (Client.Timeout or context cancellation while reading body)",
				context.DeadlineExceeded,
			)},
			want: true,
		},
		{
			name: "unexpected eof while reading",
			err:  &streamReadError{Err: io.ErrUnexpectedEOF},
			want: true,
		},
		{
			name: "user canceled",
			err:  &streamReadError{Err: context.Canceled},
			want: false,
		},
		{
			name: "plain parse error",
			err:  errors.New("no valid response frames parsed"),
			want: false,
		},
		{
			name: "non-timeout read error",
			err:  &streamReadError{Err: errors.New("boom")},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableStreamBodyError(tc.err); got != tc.want {
				t.Fatalf("isRetryableStreamBodyError() = %t, want %t", got, tc.want)
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

func makeStreamBodyMetadataOnly(t *testing.T) []byte {
	t.Helper()
	content := make([]any, 5)
	content[1] = []any{"c_abc", "r_def"}
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

type errReader struct {
	err error
}

func (r *errReader) Read([]byte) (int, error) {
	return 0, r.err
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
