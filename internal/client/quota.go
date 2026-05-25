package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
)

// RPC IDs used for account quota tracking. These are the same RPCs called by
// the deep research preflight (rpcDeepResearchModelSt / rpcDeepResearchCaps);
// the upstream Python rename in PR #310 reflects what they actually do.
const (
	rpcCheckGeminiQuota = rpcDeepResearchModelSt // "qpEbW" — per-tier model quotas
	rpcCheckQuota       = rpcDeepResearchCaps    // "aPya6c" — overall extra-feature quota
)

// Payloads for CheckGeminiQuota that target specific tiers. Mirror the
// upstream Python constants GEMINI_FLASH_QUOTA_PAYLOAD / GEMINI_ADVANCED_QUOTA_PAYLOAD.
const (
	geminiFlashQuotaPayload    = `[[[1,11],[2,11],[6,11]]]`
	geminiAdvancedQuotaPayload = `[[[1,4],[6,6],[1,15]]]`
)

// Quota describes a single per-model usage limit reported by the server.
type Quota struct {
	ID           string  // composite key from quota_id_list (e.g. "1-4")
	Label        string  // human-readable label, e.g. "Gemini Pro"
	ActionID     int     // 4 = Pro, 11 = Flash, 15 = Flash Thinking
	UsagePercent float64 // 0..100
	Total        int     // 0 if unlimited
	Remaining    int
	ResetTime    int64 // unix timestamp; 0 if unset
}

// ExtraQuota is the overall account-level "extra feature" quota reported by
// the CHECK_QUOTA RPC. Unlike per-model quotas this is a single record.
type ExtraQuota struct {
	IsBlocked    bool
	UsagePercent float64 // 0..100 (server returns 0..1 — already scaled here)
	ResetTime    int64
}

// FetchQuotas calls CHECK_GEMINI_QUOTA for the requested tiers and parses
// per-model quota records out of the response. Pass both flags as true to
// query everything (matches Python's default).
func (c *Client) FetchQuotas(ctx context.Context, flash, advanced bool) (map[string]*Quota, error) {
	if !flash && !advanced {
		flash, advanced = true, true
	}

	type call struct {
		payload  string
		category string
	}
	var calls []call
	if flash {
		calls = append(calls, call{geminiFlashQuotaPayload, "Flash"})
	}
	if advanced {
		calls = append(calls, call{geminiAdvancedQuotaPayload, "Thinking/Pro"})
	}

	results := make(map[string]*Quota)
	for _, cc := range calls {
		body, err := c.batchExecuteSingle(ctx, rpcCheckGeminiQuota, cc.payload)
		if err != nil {
			return results, fmt.Errorf("CheckGeminiQuota[%s]: %w", cc.category, err)
		}
		parseQuotaItems(body, cc.category, results)
	}
	return results, nil
}

// FetchExtraQuota calls CHECK_QUOTA for the account-level extra feature quota.
func (c *Client) FetchExtraQuota(ctx context.Context) (*ExtraQuota, error) {
	body, err := c.batchExecuteSingle(ctx, rpcCheckQuota, "[]")
	if err != nil {
		return nil, fmt.Errorf("CheckQuota: %w", err)
	}
	return parseExtraQuota(body), nil
}

// batchExecuteSingle posts a single RPC call to /batchexecute and returns the
// parsed body string for the matching RPC ID.
func (c *Client) batchExecuteSingle(ctx context.Context, rpcID, payload string) (string, error) {
	rpcReq := []any{
		[]any{
			[]any{rpcID, payload, nil, "generic"},
		},
	}
	reqJSON, _ := json.Marshal(rpcReq)

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(reqJSON))

	reqURL := c.batchURL([]string{rpcID}, c.appPath())

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	for k, v := range c.commonHeaders() {
		httpReq.Header[k] = v
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	stripped := protocol.StripResponsePrefix(raw)
	body, rejectCode, err := protocol.ExtractRPCBody(stripped, rpcID)
	if err != nil {
		return "", err
	}
	if rejectCode != 0 {
		return "", fmt.Errorf("rpc rejected with code=%d", rejectCode)
	}
	return string(body), nil
}

func parseQuotaItems(body, category string, out map[string]*Quota) {
	if body == "" || body == "[]" {
		return
	}
	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return
	}
	if len(data) == 0 {
		return
	}
	items, ok := data[0].([]any)
	if !ok {
		return
	}

	actionLabels := map[int]string{
		4:  "Gemini Pro",
		11: "Gemini Flash",
		15: "Gemini Flash Thinking",
	}

	for _, raw := range items {
		item, ok := raw.([]any)
		if !ok {
			continue
		}
		var idList []any
		if len(item) > 0 {
			idList, _ = item[0].([]any)
		}

		actionID := 0
		if len(idList) > 1 {
			if f, ok := idList[1].(float64); ok {
				actionID = int(f)
			}
		}

		usagePercent := 0.0
		if len(item) > 2 {
			if f, ok := item[2].(float64); ok {
				usagePercent = f
			}
		}

		var resetTS int64
		if len(item) > 3 {
			if r, ok := item[3].([]any); ok && len(r) > 0 {
				if f, ok := r[0].(float64); ok {
					resetTS = int64(f)
				}
			}
		}

		total := 0
		if len(item) > 4 {
			if f, ok := item[4].(float64); ok {
				total = int(f)
			}
		}
		remaining := 0
		if len(item) > 5 {
			if f, ok := item[5].(float64); ok {
				remaining = int(f)
			}
		}

		idParts := make([]string, 0, len(idList))
		for _, v := range idList {
			if f, ok := v.(float64); ok {
				idParts = append(idParts, fmt.Sprintf("%d", int(f)))
			}
		}
		quotaID := strings.Join(idParts, "-")
		if quotaID == "" {
			continue
		}

		label, hasLabel := actionLabels[actionID]
		if !hasLabel {
			label = "Gemini " + category
		}

		out[quotaID] = &Quota{
			ID:           quotaID,
			Label:        label,
			ActionID:     actionID,
			UsagePercent: usagePercent,
			Total:        total,
			Remaining:    remaining,
			ResetTime:    resetTS,
		}
	}
}

func parseExtraQuota(body string) *ExtraQuota {
	if body == "" || body == "[]" {
		return nil
	}
	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil
	}
	q := &ExtraQuota{}
	if len(data) > 0 {
		if b, ok := data[0].(bool); ok {
			q.IsBlocked = b
		}
	}
	if len(data) > 1 {
		if f, ok := data[1].(float64); ok {
			q.UsagePercent = f * 100
		}
	}
	if len(data) > 2 {
		if r, ok := data[2].([]any); ok && len(r) > 0 {
			if f, ok := r[0].(float64); ok {
				q.ResetTime = int64(f)
			}
		}
	}
	return q
}
