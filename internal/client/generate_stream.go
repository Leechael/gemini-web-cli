package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/transport"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

func (c *Client) streamGenerate(ctx context.Context, prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, cb StreamCallback) error {
	if model == nil {
		model = &types.Models[0]
	}

	uuid := generateUUID()
	hasCid := len(metadata) > 0 && metadata[0] != ""
	mode := c.resolveGenerationMode(prompt, uploads)

	innerReq := c.buildInnerRequest(prompt, metadata, uploads, model, deepResearch, hasCid, uuid, mode)
	innerJSON, err := json.Marshal(innerReq)
	if err != nil {
		return fmt.Errorf("marshaling inner request: %w", err)
	}

	if c.verbose {
		outerJSON, _ := json.Marshal([]any{nil, string(innerJSON)})
		fmt.Fprintf(logWriter, "f.req payload: %s\n", string(outerJSON))
	}

	body, err := c.CallStreamGenerate(ctx, transport.StreamGenerateRequest{
		AccessToken: c.accessToken,
		InnerReq:    innerJSON,
		UUID:        uuid,
		ModelHeader: model.Headers,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	return c.parseStreamResponse(body, cb)
}
