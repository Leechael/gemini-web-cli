// StreamGenerate media extractors. These functions decode candidate[12] and nested
// candidate data paths from the StreamGenerate wire format into public types.
package rpcs

import "github.com/Leechael/gemini-web-cli/internal/types"

// ExtractImages extracts web and generated image records from candidate media data.
func ExtractImages(candidate12 any) []types.Image {
	return types.ExtractImages(candidate12)
}

// ExtractVideos extracts generated videos from candidate media data.
// It supports both the historical [12][59] path and the newer [12][8]["60"] path.
func ExtractVideos(candidate12 any) []types.Video {
	return types.ExtractVideos(candidate12)
}

// ExtractMedia extracts generated music/audio media from candidate media data.
// It supports both the historical [12][86] path and the newer [12][0]["87"] path.
func ExtractMedia(candidate12 any) []types.GeneratedMedia {
	return types.ExtractMedia(candidate12)
}

// ExtractDeepResearchPlan searches candidate data for a dict with key "56" or "57".
func ExtractDeepResearchPlan(candidateData []any) *types.DeepResearchPlanData {
	var planPayload []any
	findDictKey(candidateData, func(m map[string]any) bool {
		for _, key := range []string{"56", "57"} {
			if v, ok := m[key]; ok {
				if arr, ok := v.([]any); ok {
					planPayload = arr
					return true
				}
			}
		}
		return false
	})
	if planPayload == nil {
		return nil
	}

	plan := &types.DeepResearchPlanData{}
	if len(planPayload) > 0 {
		plan.Title, _ = planPayload[0].(string)
	}
	if len(planPayload) > 1 {
		if stepsArr, ok := planPayload[1].([]any); ok {
			for _, step := range stepsArr {
				stepArr, ok := step.([]any)
				if !ok {
					continue
				}
				label := ""
				body := ""
				if len(stepArr) > 1 {
					label, _ = stepArr[1].(string)
				}
				if len(stepArr) > 2 {
					body, _ = stepArr[2].(string)
				}
				switch {
				case label != "" && body != "":
					plan.Steps = append(plan.Steps, label+": "+body)
				case body != "":
					plan.Steps = append(plan.Steps, body)
				case label != "":
					plan.Steps = append(plan.Steps, label)
				}
			}
		}
	}
	if len(planPayload) > 2 {
		plan.ETAText, _ = planPayload[2].(string)
	}
	if len(planPayload) > 3 {
		if arr, ok := planPayload[3].([]any); ok && len(arr) > 0 {
			plan.ConfirmPrompt, _ = arr[0].(string)
		}
	}
	if plan.Title == "" && len(plan.Steps) == 0 && plan.ETAText == "" && plan.ConfirmPrompt == "" {
		return nil
	}
	return plan
}

func findDictKey(data any, pred func(map[string]any) bool) bool {
	switch v := data.(type) {
	case map[string]any:
		if pred(v) {
			return true
		}
		for _, val := range v {
			if findDictKey(val, pred) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if findDictKey(item, pred) {
				return true
			}
		}
	}
	return false
}
