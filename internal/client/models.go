package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	responseBody := stripResponsePrefix(string(body))
	rpcBody, rejectCode, err := extractRPCBody(responseBody, rpcGetUserStatus)
	if err != nil {
		return types.StatusAvailable, nil, fmt.Errorf("extracting user status RPC body: %w", err)
	}
	if rejectCode != 0 {
		return types.StatusAvailable, nil, fmt.Errorf("user status RPC rejected with code=%d", rejectCode)
	}

	return parseUserStatus(rpcBody)
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
		idNameMapping := buildModelIDNameMapping()

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

			// Build capacity tail for header construction
			var capacityTail string
			if capacityField == 13 {
				capacityTail = fmt.Sprintf("null,%d", capacity)
			} else {
				capacityTail = fmt.Sprintf("%d", capacity)
			}

			name := idNameMapping[modelID]
			if name == "" {
				name = modelID // fallback to hex ID
			}

			// Determine advancedOnly from hardcoded model definitions (per-model property),
			// not from account-level capacity (which would mark ALL models as advanced on paid accounts).
			advancedOnly := false
			if known := types.FindModel(name); known != nil {
				advancedOnly = known.AdvancedOnly
			}

			models = append(models, types.Model{
				Name:         name,
				DisplayName:  displayName,
				AdvancedOnly: advancedOnly,
				Headers:      types.BuildModelHeader(modelID, capacityTail),
				Description:  description,
			})
		}
	}

	return accountStatus, models, nil
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

func buildModelIDNameMapping() map[string]string {
	result := make(map[string]string)
	for _, m := range types.Models {
		if m.Name == "unspecified" {
			continue
		}
		id := m.ModelID()
		if id == "" {
			continue
		}
		// Use the base (non-tier) name: strip -plus, -advanced suffix
		baseName := m.Name
		for _, suffix := range []string{"-plus", "-advanced"} {
			baseName = strings.TrimSuffix(baseName, suffix)
		}
		if _, exists := result[id]; !exists {
			result[id] = baseName
		}
	}
	return result
}
