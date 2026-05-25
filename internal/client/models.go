package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

// FetchUserStatus calls the GetUserStatus RPC to discover account status
// and dynamically available models.
func (c *Client) FetchUserStatus(ctx context.Context) (types.AccountStatus, []types.Model, error) {
	rpcID, payload := rpcs.EncodeGetUserStatus()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath(c.appPath()))
	if err != nil {
		return types.StatusAvailable, nil, fmt.Errorf("user status request failed: %w", err)
	}
	if rejectCode != 0 {
		return types.StatusAvailable, nil, fmt.Errorf("user status RPC rejected with code=%d", rejectCode)
	}

	raw, err := rpcs.DecodeGetUserStatus(body)
	if err != nil {
		return types.StatusAvailable, nil, err
	}
	return raw.AccountStatus, userStatusModelsToTypes(raw), nil
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

func userStatusModelsToTypes(raw *rpcs.UserStatusResult) []types.Model {
	if raw == nil || raw.AccountStatus.IsHardBlock() {
		return nil
	}
	capacity, capacityField := computeCapacity(raw.TierFlags, raw.CapFlags)
	idNameMapping := buildModelIDNameMapping(capacity, capacityField)

	models := make([]types.Model, 0, len(raw.Models))
	for _, md := range raw.Models {
		if raw.AccountStatus.Code == 1016 {
			flashModel := types.FindModel("gemini-3-flash")
			if flashModel != nil && md.ModelID != flashModel.ModelID() {
				continue
			}
		}

		selector := capacity
		if md.Selector != 0 {
			selector = md.Selector
		}

		name := dynamicModelName(md.Raw, md.ModelID, md.DisplayName)
		if name == "" {
			name = idNameMapping[md.ModelID]
		}
		if name == "" {
			name = md.ModelID
		}

		advancedOnly := selector == 3 || strings.Contains(name, "pro")
		if known := types.FindModel(name); known != nil {
			advancedOnly = known.AdvancedOnly
		}

		models = append(models, types.Model{
			Name:         name,
			DisplayName:  md.DisplayName,
			AdvancedOnly: advancedOnly,
			Headers:      types.BuildModelHeader(md.ModelID, selector),
			Description:  md.Description,
		})
	}
	return models
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
