package client

import (
	"context"
	"fmt"
	"os"

	"github.com/Leechael/gemini-web-cli/internal/types"
)

// ModelUnavailableError indicates error code 1052 (model not available).
type ModelUnavailableError struct {
	Code int
}

func (e *ModelUnavailableError) Error() string {
	return fmt.Sprintf("model unavailable (error code %d)", e.Code)
}

// GenerateContent sends a prompt and returns the full response (non-streaming).
func (c *Client) GenerateContent(ctx context.Context, prompt string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, nil, model, false, nil)
	return best, err
}

// GenerateContentWithFiles sends a prompt with file attachments.
func (c *Client) GenerateContentWithFiles(ctx context.Context, prompt string, uploads []*UploadResult, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, uploads, model, false, nil)
	return best, err
}

// StreamCallback is called for each streaming chunk.
type StreamCallback func(output *types.ModelOutput)

// GenerateContentStream sends a prompt and calls cb for each streaming chunk.
func (c *Client) GenerateContentStream(ctx context.Context, prompt string, model *types.Model, cb StreamCallback) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, nil, model, false, cb)
	return best, err
}

// GenerateContentStreamWithFiles sends a prompt with files and calls cb for each chunk.
func (c *Client) GenerateContentStreamWithFiles(ctx context.Context, prompt string, uploads []*UploadResult, model *types.Model, cb StreamCallback) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, nil, uploads, model, false, cb)
	return best, err
}

// SendMessage sends a message in an existing chat.
func (c *Client) SendMessage(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, nil, model, false, nil)
	return best, err
}

// SendMessageStream sends a message in a chat with streaming.
func (c *Client) SendMessageStream(ctx context.Context, prompt string, metadata []string, model *types.Model, cb StreamCallback) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, nil, model, false, cb)
	return best, err
}

// SendMessageDeepResearch sends a message with deep research flags.
func (c *Client) SendMessageDeepResearch(ctx context.Context, prompt string, metadata []string, model *types.Model) (*types.ModelOutput, error) {
	best, _, err := c.collectStreamResult(ctx, prompt, metadata, nil, model, true, nil)
	return best, err
}

// collectStreamResult is the shared implementation for all generate/send methods.
func (c *Client) collectStreamResult(ctx context.Context, prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, cb StreamCallback) (*types.ModelOutput, []types.Image, error) {
	best, allImages, err := c.doCollectStream(ctx, prompt, metadata, uploads, model, deepResearch, cb)
	if err != nil {
		if mErr, ok := err.(*ModelUnavailableError); ok {
			fallback := types.FindModel(types.FallbackModelName)
			if fallback != nil && (model == nil || model.Name != fallback.Name) {
				fmt.Fprintf(os.Stderr, "Model unavailable (code %d), retrying with %s...\n", mErr.Code, fallback.DisplayName)
				return c.doCollectStream(ctx, prompt, metadata, uploads, fallback, deepResearch, cb)
			}
		}
		return nil, nil, err
	}
	return best, allImages, nil
}

func (c *Client) doCollectStream(ctx context.Context, prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, cb StreamCallback) (*types.ModelOutput, []types.Image, error) {
	var best *types.ModelOutput
	var allImages []types.Image
	var allVideos []types.Video
	var allMedia []types.GeneratedMedia
	var plan *types.DeepResearchPlanData
	var bestMetadata []string
	seenImg := map[string]bool{}
	seenVid := map[string]bool{}
	seenMedia := map[string]bool{}
	err := c.streamGenerate(ctx, prompt, metadata, uploads, model, deepResearch, func(out *types.ModelOutput) {
		for _, img := range out.Images {
			if !seenImg[img.URL] {
				seenImg[img.URL] = true
				allImages = append(allImages, img)
			}
		}
		for _, vid := range out.Videos {
			if !seenVid[vid.URL] {
				seenVid[vid.URL] = true
				allVideos = append(allVideos, vid)
			}
		}
		for _, m := range out.Media {
			key := m.MP3URL + "|" + m.MP4URL
			if !seenMedia[key] {
				seenMedia[key] = true
				allMedia = append(allMedia, m)
			}
		}
		if out.DeepResearchPlan != nil {
			plan = out.DeepResearchPlan
		}
		if len(out.Metadata) > len(bestMetadata) {
			bestMetadata = out.Metadata
		}
		if best == nil || len(out.Text) >= len(best.Text) {
			best = out
		}
		if cb != nil {
			cb(out)
		}
	})
	if err != nil {
		return nil, nil, err
	}
	if best == nil {
		return nil, nil, fmt.Errorf("no response received")
	}
	if len(allImages) > 0 {
		best.Images = allImages
	}
	if len(allVideos) > 0 {
		best.Videos = allVideos
	}
	if len(allMedia) > 0 {
		best.Media = allMedia
	}
	if plan != nil {
		best.DeepResearchPlan = plan
	}
	if len(bestMetadata) > len(best.Metadata) {
		best.Metadata = bestMetadata
	}
	return best, allImages, nil
}
