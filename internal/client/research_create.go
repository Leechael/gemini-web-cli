package client

import (
	"context"
	"fmt"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

const deepResearchPromptPrefix = "Use Gemini Deep Research to investigate this request and create a research plan before answering: "

// CreateAndStartDeepResearch creates a research plan and starts execution.
// Mirrors the Python flow: preflight, send prompt, then confirm execution.
func (c *Client) CreateAndStartDeepResearch(ctx context.Context, prompt string, model *types.Model) (*types.DeepResearchPlan, error) {
	if model != nil && model.Name != "" && model.Name != "unspecified" {
		return nil, fmt.Errorf("deep research only supports model auto/unspecified")
	}
	if model == nil {
		model = &types.Models[0]
	}

	// Step 0: Preflight RPCs (no cid yet)
	c.deepResearchPreflight(ctx, "", "")

	// Step 1: Send the prompt with deep research flags; this creates a plan internally.
	step1, err := c.deepResearchGenerate(ctx, deepResearchPromptPrefix+prompt, nil, model)
	if err != nil {
		return nil, fmt.Errorf("deep research plan request failed: %w", err)
	}

	plan, confirmPrompt, err := deepResearchPlanFromOutput(step1)
	if err != nil {
		return nil, err
	}

	// Step 2: Send the server-provided confirmation to start research execution.
	var rid string
	if len(step1.Metadata) > 1 {
		rid = step1.Metadata[1]
	}
	c.deepResearchPreflight(ctx, plan.Cid, rid)
	if _, err := c.deepResearchGenerate(ctx, confirmPrompt, step1.Metadata, model); err != nil {
		return nil, fmt.Errorf("deep research confirm step failed: %w", err)
	}

	if err := c.waitForDeepResearchStart(ctx, plan.Cid); err != nil {
		return nil, err
	}
	return plan, nil
}

func deepResearchPlanFromOutput(step1 *types.ModelOutput) (*types.DeepResearchPlan, string, error) {
	if step1 == nil {
		return nil, "", fmt.Errorf("deep research plan response was empty")
	}
	if len(step1.Metadata) == 0 || step1.Metadata[0] == "" {
		return nil, "", fmt.Errorf("no chat ID returned from deep research")
	}
	if step1.DeepResearchPlan == nil {
		preview := []rune(step1.Text)
		if len(preview) > 1200 {
			preview = preview[:1200]
		}
		return nil, "", fmt.Errorf("Gemini did not return a deep research plan. Preview: %s", string(preview))
	}

	plan := &types.DeepResearchPlan{
		Cid:     step1.Metadata[0],
		Title:   step1.DeepResearchPlan.Title,
		Steps:   append([]string(nil), step1.DeepResearchPlan.Steps...),
		ETAText: step1.DeepResearchPlan.ETAText,
	}
	confirmPrompt := step1.DeepResearchPlan.ConfirmPrompt
	if confirmPrompt == "" {
		confirmPrompt = "Start research"
	}
	return plan, confirmPrompt, nil
}

func (c *Client) waitForDeepResearchStart(ctx context.Context, cid string) error {
	const maxChecks = 5
	lastState := ""
	for check := 1; check <= maxChecks; check++ {
		status, err := c.CheckDeepResearch(ctx, cid)
		if err != nil {
			return fmt.Errorf("deep research status check failed: %w", err)
		}
		lastState = ""
		if status != nil {
			lastState = status.State
		}

		started, retry, err := deepResearchStartState(lastState)
		if err != nil {
			return err
		}
		if started {
			return nil
		}
		if !retry || check == maxChecks {
			return fmt.Errorf("deep research did not start (state=%s)", lastState)
		}

		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("deep research status check failed: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return fmt.Errorf("deep research did not start (state=%s)", lastState)
}

func deepResearchStartState(state string) (started bool, retry bool, err error) {
	switch state {
	case "running", "done":
		return true, false, nil
	case "empty", "pending_confirm":
		return false, true, nil
	case "not_research":
		return false, false, fmt.Errorf("deep research did not start (state=%s)", state)
	default:
		return false, false, fmt.Errorf("deep research did not start (state=%s)", state)
	}
}

func (c *Client) deepResearchGenerate(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, nil, model, true, nil)
	return best, err
}
