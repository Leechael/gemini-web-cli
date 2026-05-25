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

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// PostBatchRequest contains the inputs for one batchexecute POST.
type PostBatchRequest struct {
	Client      *http.Client
	URL         string
	AccessToken string
	RPCID       string
	Payload     string
	UserAgent   string
}

// RPCCall contains one RPC ID and payload pair for a batchexecute request.
type RPCCall struct {
	ID      string
	Payload string
}

// PostBatchMultiRequest contains the inputs for one multi-RPC batchexecute POST.
type PostBatchMultiRequest struct {
	Client      *http.Client
	URL         string
	AccessToken string
	Calls       []RPCCall
	UserAgent   string
}

// PostBatch sends a batchexecute request and returns the raw response body.
func PostBatch(ctx context.Context, req PostBatchRequest) ([]byte, error) {
	rpcReq := []any{
		[]any{
			[]any{req.RPCID, req.Payload, nil, "generic"},
		},
	}
	reqJSON, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal batch request: %w", err)
	}

	form := url.Values{}
	form.Set("at", req.AccessToken)
	form.Set("f.req", string(reqJSON))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	setBatchHeaders(httpReq, req.URL, req.UserAgent)

	client := req.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batchexecute returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}

// PostBatchMulti sends multiple RPCs in one batchexecute request and returns the raw response body.
func PostBatchMulti(ctx context.Context, req PostBatchMultiRequest) ([]byte, error) {
	calls := make([]any, 0, len(req.Calls))
	for _, call := range req.Calls {
		calls = append(calls, []any{call.ID, call.Payload, nil, "generic"})
	}
	rpcReq := []any{calls}
	reqJSON, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal batch request: %w", err)
	}

	form := url.Values{}
	form.Set("at", req.AccessToken)
	form.Set("f.req", string(reqJSON))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	setBatchHeaders(httpReq, req.URL, req.UserAgent)

	client := req.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batchexecute returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func setBatchHeaders(httpReq *http.Request, rawURL, userAgent string) {
	ua := userAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	origin := "https://gemini.google.com"
	if u, err := url.Parse(rawURL); err == nil && u.Scheme != "" && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	httpReq.Header.Set("Origin", origin)
	httpReq.Header.Set("Referer", origin+"/")
	httpReq.Header.Set("User-Agent", ua)
	httpReq.Header.Set("X-Same-Domain", "1")
}
