package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client/transport/rpclog"
)

// StreamURLConfig contains the query and path inputs for a StreamGenerate URL.
type StreamURLConfig struct {
	BaseURL     string
	AccountPath string
	ReqID       int
	Language    string
	BuildLabel  string
	SessionID   string
}

// BuildStreamGenerateURL constructs the Gemini StreamGenerate URL.
func BuildStreamGenerateURL(cfg StreamURLConfig) string {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://gemini.google.com"
	}
	language := cfg.Language
	if language == "" {
		language = "en"
	}
	params := url.Values{}
	params.Set("_reqid", fmt.Sprintf("%d", cfg.ReqID))
	params.Set("rt", "c")
	params.Set("hl", language)
	params.Set("pageId", "none")
	if cfg.BuildLabel != "" {
		params.Set("bl", cfg.BuildLabel)
	}
	if cfg.SessionID != "" {
		params.Set("f.sid", cfg.SessionID)
	}
	path := cfg.AccountPath + "/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	return base + path + "?" + params.Encode()
}

// StreamGenerateRequest contains the inputs for a StreamGenerate POST.
type StreamGenerateRequest struct {
	Client      *http.Client
	URL         string
	AccessToken string
	InnerReq    []byte
	UUID        string
	ModelHeader map[string]string
	UserAgent   string
}

// HTTPStatusError reports a non-200 StreamGenerate response.
type HTTPStatusError struct {
	StatusCode  int
	BodySnippet string
}

func (e *HTTPStatusError) Error() string {
	if e.BodySnippet == "" {
		return fmt.Sprintf("stream returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("stream returned HTTP %d: %s", e.StatusCode, e.BodySnippet)
}

// StreamRequestError wraps failures that happen before Gemini returns an HTTP response.
type StreamRequestError struct {
	Err error
}

func (e *StreamRequestError) Error() string {
	return fmt.Sprintf("stream request failed: %v", e.Err)
}

func (e *StreamRequestError) Unwrap() error {
	return e.Err
}

// PostStreamGenerate sends a StreamGenerate request and returns the response body.
func PostStreamGenerate(ctx context.Context, req StreamGenerateRequest) (io.ReadCloser, error) {
	start := time.Now()

	outerReq := []any{nil, string(req.InnerReq)}
	outerJSON, err := json.Marshal(outerReq)
	if err != nil {
		return nil, fmt.Errorf("marshal stream outer request: %w", err)
	}

	form := url.Values{}
	form.Set("at", req.AccessToken)
	form.Set("f.req", string(outerJSON))
	formEncoded := form.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, strings.NewReader(formEncoded))
	if err != nil {
		return nil, err
	}
	setBatchHeaders(httpReq, req.URL, req.UserAgent)
	for k, v := range req.ModelHeader {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("x-goog-ext-525005358-jspb", fmt.Sprintf(`["%s",1]`, req.UUID))

	entry := rpclog.Entry{
		Method:     httpReq.Method,
		URL:        httpReq.URL.String(),
		Kind:       rpclog.KindStream,
		ReqHeaders: httpReq.Header.Clone(),
		ReqBody:    rpclog.StringBody(rpclog.RedactAT(formEncoded)),
	}

	client := req.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		entry.Status = 0
		entry.Error = err.Error()
		entry.DurMS = time.Since(start).Milliseconds()
		rpclog.Log(ctx, entry)
		return nil, &StreamRequestError{Err: err}
	}
	entry.RespHeaders = resp.Header.Clone()
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		entry.Status = resp.StatusCode
		entry.RespBody = rpclog.BytesBody(body)
		entry.Error = fmt.Sprintf("stream returned HTTP %d", resp.StatusCode)
		entry.DurMS = time.Since(start).Milliseconds()
		rpclog.Log(ctx, entry)
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, BodySnippet: snippet}
	}

	entry.Status = http.StatusOK
	if !rpclog.Enabled() {
		return resp.Body, nil
	}
	capture := rpclog.StartBodyCapture("stream-response")
	wrapped := rpclog.WrapStreamReadCloser(resp.Body, capture, func(body *rpclog.Body, readErr error) {
		entry.RespBody = body
		entry.DurMS = time.Since(start).Milliseconds()
		if readErr != nil && readErr != io.EOF {
			entry.Error = readErr.Error()
		}
		rpclog.Log(ctx, entry)
	})
	return wrapped, nil
}
