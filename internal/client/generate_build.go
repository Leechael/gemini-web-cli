package client

import (
	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

func (c *Client) buildInnerRequest(prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, uuid string, modeOverride ...string) []any {
	fileRefs := make([]rpcs.FileRef, 0, len(uploads))
	for _, u := range uploads {
		fileRefs = append(fileRefs, rpcs.FileRef{
			UploadID: u.ID,
			MimeType: u.MimeType,
			FileName: u.FileName,
		})
	}
	mode := ""
	if len(modeOverride) > 0 {
		mode = modeOverride[0]
	} else {
		mode = c.resolveGenerationMode(prompt, uploads)
	}
	if model == nil {
		model = &types.Models[0]
	}
	return rpcs.EncodeStreamGenerate(rpcs.EncodeStreamGenerateOpts{
		Prompt:        prompt,
		Language:      c.language,
		Metadata:      metadata,
		Uploads:       fileRefs,
		Mode:          mode,
		DeepResearch:  deepResearch,
		ModelSelector: modelSelector(model),
		UUID:          uuid,
		EntropyToken:  "!" + generateURLSafeToken(2600),
		HexUUID:       generateHexUUID(),
	})
}
