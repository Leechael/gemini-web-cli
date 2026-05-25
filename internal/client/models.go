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
	"github.com/Leechael/gemini-web-cli/internal/types"
)

const rpcGetUserStatus = "otAQ7b"

// FetchUserStatus calls the GetUserStatus RPC to discover account status
// and dynamically available models.
func (c *Client) FetchUserStatus(ctx context.Context) (types.AccountStatus, []types.Model, error) {
	payload := "[]"
	rpcReq := []any{
		[]any{
			[]any{rpcGetUserStatus, payload, nil, "generic"},
		},
	}
	reqJSON, _ := json.Marshal(rpcReq)

	form := url.Values{}
	form.Set("at", c.accessToken)
	form.Set("f.req", string(reqJSON))

	reqURL := c.batchURL([]string{rpcGetUserStatus}, c.appPath())

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return types.StatusAvailable, nil, err
	}
	headers := c.commonHeaders()
	for k, v := range headers {
		httpReq.Header[k] = v
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return types.StatusAvailable, nil, fmt.Errorf("user status request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.StatusAvailable, nil, err
	}

	responseBody := protocol.StripResponsePrefix(body)
	rpcBody, rejectCode, err := protocol.ExtractRPCBody(responseBody, rpcGetUserStatus)
	if err != nil {
		return types.StatusAvailable, nil, fmt.Errorf("extracting user status RPC body: %w", err)
	}
	if rejectCode != 0 {
		return types.StatusAvailable, nil, fmt.Errorf("user status RPC rejected with code=%d", rejectCode)
	}

	return parseUserStatus(string(rpcBody))
}

// FetchAndCacheModels calls FetchUserStatus and caches the results.
func (c *Client) FetchAndCacheModels(ctx context.Context) error {
	status, models, err := c.FetchUserStatus(ctx)
	if err != nil {
		if c.verbose {
			fmt.Fprintf(logWriter, "Warning: failed to fetch user status: %v\n", err)
		}
		return nil // Non-fatal: fall back to hardcoded models
	}

	c.AccountStatus = status
	if c.verbose {
		fmt.Fprintf(logWriter, "Account status: %s - %s\n", status.Name, status.Description)
	}

	if len(models) > 0 {
		c.modelRegistry = make(map[string]*types.Model)
		for i := range models {
			c.modelRegistry[models[i].ModelID()] = &models[i]
			// Also index by name for quick lookup
			c.modelRegistry[models[i].Name] = &models[i]
		}
		if c.verbose {
			fmt.Fprintf(logWriter, "Discovered %d models dynamically\n", len(models))
		}
	}

	return nil
}

// AvailableModels returns the dynamically discovered models, or nil if not fetched.
func (c *Client) AvailableModels() []types.Model {
	if c.modelRegistry == nil {
		return nil
	}
	seen := make(map[string]bool)
	var result []types.Model
	for _, m := range c.modelRegistry {
		if !seen[m.Name] {
			seen[m.Name] = true
			result = append(result, *m)
		}
	}
	return result
}

// ResolveModel returns a dynamically discovered model by name, display name, or model ID.
func (c *Client) ResolveModel(name string) *types.Model {
	if c == nil || c.modelRegistry == nil || name == "" {
		return nil
	}
	if m := c.modelRegistry[name]; m != nil {
		return m
	}
	for _, m := range c.modelRegistry {
		if m.Name == name || m.DisplayName == name || m.ModelID() == name {
			return m
		}
	}
	return nil
}

func parseUserStatus(body string) (types.AccountStatus, []types.Model, error) {
	if body == "" || body == "[]" {
		return types.StatusAvailable, nil, nil
	}

	var data []any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return types.StatusAvailable, nil, fmt.Errorf("parsing user status response: %w", err)
	}

	// Status code at [14]
	statusCode := 1000
	if len(data) > 14 {
		if f, ok := data[14].(float64); ok {
			statusCode = int(f)
		}
	}
	accountStatus := types.AccountStatusFromCode(statusCode)

	if accountStatus.IsHardBlock() {
		return accountStatus, nil, nil
	}

	// Models list at [15]
	var models []types.Model
	if len(data) > 15 {
		modelsList, ok := data[15].([]any)
		if !ok {
			return accountStatus, nil, nil
		}

		// Tier flags at [16], capability flags at [17]
		var tierFlags, capFlags []float64
		if len(data) > 16 {
			if arr, ok := data[16].([]any); ok {
				for _, v := range arr {
					if f, ok := v.(float64); ok {
						tierFlags = append(tierFlags, f)
					}
				}
			}
		}
		if len(data) > 17 {
			if arr, ok := data[17].([]any); ok {
				for _, v := range arr {
					if f, ok := v.(float64); ok {
						capFlags = append(capFlags, f)
					}
				}
			}
		}

		capacity, capacityField := computeCapacity(tierFlags, capFlags)
		idNameMapping := buildModelIDNameMapping(capacity, capacityField)

		for _, modelData := range modelsList {
			md, ok := modelData.([]any)
			if !ok {
				continue
			}
			modelID := ""
			displayName := ""
			description := ""
			if len(md) > 0 {
				if s, ok := md[0].(string); ok {
					modelID = s
				}
			}
			if len(md) > 1 {
				if s, ok := md[1].(string); ok {
					displayName = s
				}
			}
			if len(md) > 2 {
				if s, ok := md[2].(string); ok {
					description = s
				}
			}

			if modelID == "" || displayName == "" {
				continue
			}

			// Check availability for unauthenticated accounts
			if accountStatus.Code == 1016 {
				flashModel := types.FindModel("gemini-3-flash")
				if flashModel != nil && modelID != flashModel.ModelID() {
					continue
				}
			}

			selector := capacity
			if len(md) > 17 {
				if f, ok := md[17].(float64); ok && f != 0 {
					selector = int(f)
				}
			}

			name := dynamicModelName(md, modelID, displayName)
			if name == "" {
				name = idNameMapping[modelID]
			}
			if name == "" {
				name = modelID
			}

			// Determine advancedOnly from the dynamic selector when available, with
			// hardcoded definitions as a fallback.
			advancedOnly := selector == 3 || strings.Contains(name, "pro")
			if known := types.FindModel(name); known != nil {
				advancedOnly = known.AdvancedOnly
			}

			models = append(models, types.Model{
				Name:         name,
				DisplayName:  displayName,
				AdvancedOnly: advancedOnly,
				Headers:      types.BuildModelHeader(modelID, selector),
				Description:  description,
			})
		}
	}

	return accountStatus, models, nil
}

func dynamicModelName(modelData []any, modelID string, displayName string) string {
	candidates := []string{}
	for _, idx := range []int{19, 11, 10, 1} {
		if idx < len(modelData) {
			if s, ok := modelData[idx].(string); ok && s != "" {
				candidates = append(candidates, s)
			}
		}
	}
	for _, s := range candidates {
		name := strings.ToLower(strings.TrimSpace(s))
		name = strings.ReplaceAll(name, " ", "-")
		name = strings.ReplaceAll(name, "_", "-")
		name = strings.Trim(name, "-")
		if name == "" {
			continue
		}
		if !strings.HasPrefix(name, "gemini-") {
			name = "gemini-" + name
		}
		return name
	}
	if displayName != "" {
		name := strings.ToLower(strings.TrimSpace(displayName))
		name = strings.ReplaceAll(name, " ", "-")
		if !strings.HasPrefix(name, "gemini-") {
			name = "gemini-" + name
		}
		return name
	}
	return modelID
}

func computeCapacity(tierFlags, capFlags []float64) (int, int) {
	tierSet := make(map[int]bool)
	for _, f := range tierFlags {
		tierSet[int(f)] = true
	}
	capSet := make(map[int]bool)
	for _, f := range capFlags {
		capSet[int(f)] = true
	}

	// Highest priority: override capacity_field = 13
	if tierSet[21] {
		return 1, 13
	}
	if tierSet[22] {
		return 2, 13
	}

	// Priority order: capacity_field = 12
	if capSet[115] {
		return 4, 12 // Plus accounts
	}
	if tierSet[16] || capSet[106] {
		return 3, 12 // Pro accounts (uncommon)
	}
	if tierSet[8] || (!capSet[106] && capSet[19]) {
		return 2, 12 // Pro accounts
	}

	return 1, 12 // Free accounts
}

// buildModelIDNameMapping returns a mapping from server model_id to the name
// that should surface in `gemini models list` and the registry, picked against
// the account's tier. Plus accounts see `-plus` names; Advanced accounts see
// `-advanced`; Free accounts see the bare names. IDs unique to the BASIC tier
// (e.g. gemini-3-pro) fall back to the bare name regardless of tier so paid
// accounts can still reference and discover them.
//
// capacityField is currently accepted for future tier extensions; the suffix
// only depends on capacity (4 = Plus, 2 = Advanced/Pro, otherwise Basic).
func buildModelIDNameMapping(capacity, capacityField int) map[string]string {
	_ = capacityField
	primarySuffix := tierSuffixForCapacity(capacity)

	result := make(map[string]string)

	// First pass: register IDs whose canonical name matches the primary tier.
	for _, m := range types.Models {
		if m.Name == "unspecified" {
			continue
		}
		id := m.ModelID()
		if id == "" {
			continue
		}
		if !nameMatchesTierSuffix(m.Name, primarySuffix) {
			continue
		}
		result[id] = m.Name
	}

	// Second pass: fill any IDs the primary tier didn't cover (BASIC-only ids
	// such as `gemini-3-pro` on a Plus/Advanced account).
	for _, m := range types.Models {
		if m.Name == "unspecified" {
			continue
		}
		id := m.ModelID()
		if id == "" {
			continue
		}
		if _, exists := result[id]; exists {
			continue
		}
		baseName := m.Name
		for _, suffix := range []string{"-plus", "-advanced"} {
			baseName = strings.TrimSuffix(baseName, suffix)
		}
		result[id] = baseName
	}
	return result
}

func tierSuffixForCapacity(capacity int) string {
	switch capacity {
	case 4:
		return "-plus"
	case 2:
		return "-advanced"
	default:
		return ""
	}
}

// nameMatchesTierSuffix reports whether a model Name belongs to the tier
// identified by suffix. Empty suffix means "BASIC tier" — i.e. names without
// any tier suffix.
func nameMatchesTierSuffix(name, suffix string) bool {
	if suffix == "" {
		return !strings.HasSuffix(name, "-plus") && !strings.HasSuffix(name, "-advanced")
	}
	return strings.HasSuffix(name, suffix)
}
