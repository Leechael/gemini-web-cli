package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// rpcGetAbuseStatus is the RPC ID for the account abuse / restriction probe
// (GPRiHf). Returns whether Google has flagged the account and any
// associated status code / signal.
const rpcGetAbuseStatus = "GPRiHf"

// AbuseStatus reports whether the server flagged any abuse markers on the
// current account. A nil return from FetchAbuseStatus means "no data" — only
// trust IsClean when the struct is non-nil.
type AbuseStatus struct {
	IsClean    bool
	StatusCode int    // 0 if unset; raw_status / 1_000_000
	Signal     string // free-form signal string, may be empty
}

// FetchAbuseStatus calls GET_ABUSE_STATUS and parses the response.
func (c *Client) FetchAbuseStatus(ctx context.Context) (*AbuseStatus, error) {
	body, err := c.batchExecuteSingle(ctx, rpcGetAbuseStatus, "[]")
	if err != nil {
		return nil, fmt.Errorf("GetAbuseStatus: %w", err)
	}
	return parseAbuseStatus(body), nil
}

func parseAbuseStatus(body string) *AbuseStatus {
	if body == "" || body == "[]" {
		return &AbuseStatus{IsClean: true}
	}
	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil
	}
	if len(data) <= 1 {
		return &AbuseStatus{IsClean: true}
	}
	abuseInfo, ok := data[1].([]any)
	if !ok || len(abuseInfo) == 0 {
		return &AbuseStatus{IsClean: true}
	}

	out := &AbuseStatus{}
	if len(abuseInfo) > 1 {
		if f, ok := abuseInfo[1].(float64); ok {
			out.StatusCode = int(f) / 1_000_000
		}
	}
	if len(abuseInfo) > 3 {
		if sub, ok := abuseInfo[3].([]any); ok && len(sub) > 1 {
			if s, ok := sub[1].(string); ok {
				out.Signal = s
			}
		}
	}
	return out
}
