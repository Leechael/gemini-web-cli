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

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/client/transport/rpclog"
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
	return postBatch(ctx, req.Client, req.URL, req.AccessToken, req.UserAgent, rpclog.KindBatch, []RPCCall{{
		ID: req.RPCID, Payload: req.Payload,
	}})
}

// PostBatchMulti sends multiple RPCs in one batchexecute request and returns the raw response body.
func PostBatchMulti(ctx context.Context, req PostBatchMultiRequest) ([]byte, error) {
	return postBatch(ctx, req.Client, req.URL, req.AccessToken, req.UserAgent, rpclog.KindBatchMulti, req.Calls)
}

func postBatch(ctx context.Context, client *http.Client, rawURL, accessToken, userAgent, kind string, rpcCalls []RPCCall) ([]byte, error) {
	start := time.Now()
	calls := make([]any, 0, len(rpcCalls))
	rpcIDs := make([]string, 0, len(rpcCalls))
	for _, call := range rpcCalls {
		calls = append(calls, []any{call.ID, call.Payload, nil, "generic"})
		rpcIDs = append(rpcIDs, call.ID)
	}
	reqJSON, err := json.Marshal([]any{calls})
	if err != nil {
		return nil, fmt.Errorf("marshal batch request: %w", err)
	}

	form := url.Values{}
	form.Set("at", accessToken)
	form.Set("f.req", string(reqJSON))
	formEncoded := form.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", rawURL, strings.NewReader(formEncoded))
	if err != nil {
		return nil, err
	}
	setBatchHeaders(httpReq, rawURL, userAgent)
	entry := rpclog.Entry{
		Method:     httpReq.Method,
		URL:        httpReq.URL.String(),
		Kind:       kind,
		ReqHeaders: httpReq.Header.Clone(),
		ReqBody:    rpclog.StringBody(rpclog.RedactAT(formEncoded)),
		RPCIDs:     rpcIDs,
	}

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		entry.Error = err.Error()
		entry.DurMS = time.Since(start).Milliseconds()
		rpclog.Log(ctx, entry)
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	entry.Status = resp.StatusCode
	entry.RespHeaders = resp.Header.Clone()
	entry.RespBody = rpclog.BytesBody(body)
	entry.DurMS = time.Since(start).Milliseconds()
	if readErr != nil {
		entry.Error = readErr.Error()
	} else if resp.StatusCode != http.StatusOK {
		entry.Error = fmt.Sprintf("batchexecute returned HTTP %d", resp.StatusCode)
	} else {
		populateBatchRejectCodes(&entry, body, kind, rpcIDs)
	}
	rpclog.Log(ctx, entry)

	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("batchexecute returned HTTP %d", resp.StatusCode)
	}
	return body, nil
}

func populateBatchRejectCodes(entry *rpclog.Entry, body []byte, kind string, rpcIDs []string) {
	_, codes, _ := protocol.ExtractRPCBodies(protocol.StripResponsePrefix(body), rpcIDs)
	for _, id := range rpcIDs {
		code := codes[id]
		if code == 0 {
			continue
		}
		if entry.RejectCode == nil {
			entry.RejectCode = &code
		}
		if kind == rpclog.KindBatchMulti {
			if entry.RejectCodes == nil {
				entry.RejectCodes = make(map[string]int, len(codes))
			}
			entry.RejectCodes[id] = code
		}
	}
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
