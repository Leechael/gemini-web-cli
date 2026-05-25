package client

import (
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

func parseEnvelope(envelope []any) *types.ModelOutput {
	out, _ := rpcs.DecodeStreamGenerateFrame(envelope)
	return out
}

func extractImages(imageData any) []types.Image {
	return rpcs.ExtractImages(imageData)
}

func extractVideos(imageData any) []types.Video {
	return rpcs.ExtractVideos(imageData)
}

func extractMedia(imageData any) []types.GeneratedMedia {
	return rpcs.ExtractMedia(imageData)
}

func extractDeepResearchPlan(candidateData []any) *types.DeepResearchPlanData {
	return rpcs.ExtractDeepResearchPlan(candidateData)
}

func extractErrorCode(envelope []any) int {
	return rpcs.ExtractErrorCode(envelope)
}
