package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

// PostStreamGenerate sends a StreamGenerate request and returns the response body.
func PostStreamGenerate(ctx context.Context, req StreamGenerateRequest) (io.ReadCloser, error) {
	outerReq := []any{nil, string(req.InnerReq)}
	outerJSON, err := json.Marshal(outerReq)
	if err != nil {
		return nil, fmt.Errorf("marshal stream outer request: %w", err)
	}

	form := url.Values{}
	form.Set("at", req.AccessToken)
	form.Set("f.req", string(outerJSON))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	setBatchHeaders(httpReq, req.URL, req.UserAgent)
	for k, v := range req.ModelHeader {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("x-goog-ext-525005358-jspb", fmt.Sprintf(`["%s",1]`, req.UUID))

	client := req.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, BodySnippet: snippet}
	}
	return resp.Body, nil
}
