package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// CreateAndStartDeepResearch creates a research plan and starts execution.
// Mirrors the Python flow: preflight, send prompt, then confirm execution.
func (c *Client) CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error) {
	if model == nil {
		model = &types.Models[0]
	}

	// Step 0: Preflight RPCs (no cid yet)
	c.deepResearchPreflight(ctx, "", "")

	// Step 1: Send the prompt with deep research flags; this creates a plan internally.
	step1, err := c.deepResearchGenerate(ctx, prompt, nil, model)
	if err != nil {
		return nil, fmt.Errorf("deep research plan request failed: %w", err)
	}

	plan := &types.DeepResearchPlan{}
	if len(step1.Metadata) > 0 {
		plan.Cid = step1.Metadata[0]
	}
	if plan.Cid == "" {
		return nil, fmt.Errorf("no chat ID returned from deep research")
	}

	// Extract plan from step 1 response.
	if step1.DeepResearchPlan != nil {
		plan.Title = step1.DeepResearchPlan.Title
		plan.Steps = step1.DeepResearchPlan.Steps
		plan.ETAText = step1.DeepResearchPlan.ETAText
	}

	// Step 2: Send confirm prompt to start research execution.
	var rid string
	if len(step1.Metadata) > 1 {
		rid = step1.Metadata[1]
	}
	confirmPrompt := "开始研究"
	if step1.DeepResearchPlan != nil && step1.DeepResearchPlan.ConfirmPrompt != "" {
		confirmPrompt = step1.DeepResearchPlan.ConfirmPrompt
	}
	c.deepResearchPreflight(ctx, plan.Cid, rid)
	_, err = c.deepResearchGenerate(ctx, confirmPrompt, step1.Metadata, model)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("deep research confirm step failed: %w", err)
		}
		// Non-fatal: research may still proceed.
		fmt.Fprintf(logWriter, "Warning: confirm step failed: %v\n", err)
	}

	return plan, nil
}

func (c *Client) deepResearchGenerate(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, nil, model, true, nil)
	return best, err
}
