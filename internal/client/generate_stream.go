package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
	"github.com/Leechael/gemini-web-cli/internal/client/transport"
	"github.com/Leechael/gemini-web-cli/internal/types"
)

func (c *Client) streamGenerate(ctx context.Context, prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, cb StreamCallback) error {
	if model == nil {
		model = &types.Models[0]
	}

	uuid := generateUUID()
	mode := resolveGenerationMode(c.generationModeSnapshot(), prompt, uploads)
	s := c.session()

	innerReq := c.buildInnerRequest(prompt, metadata, uploads, model, deepResearch, uuid, s.language, mode)
	innerJSON, err := json.Marshal(innerReq)
	if err != nil {
		return fmt.Errorf("marshaling inner request: %w", err)
	}

	if c.verbose {
		outerJSON, _ := json.Marshal([]any{nil, string(innerJSON)})
		fmt.Fprintf(logWriter, "f.req payload: %s\n", string(outerJSON))
	}

	// Code 13 protocol note (from data/rpc_logs reverse engineering):
	//
	// Gemini StreamGenerate returns reject code 13 in the wrb.fr envelope at
	// frame position 6, usually accompanied by BardErrorInfo 1155:
	//   [13, null, [["type.googleapis.com/assistant.boq.bard.application.BardErrorInfo",[1155]]]]
	//
	// Observed characteristics in real traffic:
	//   - Intermittent server-side transient failure, not a malformed request.
	//     The same chat/continuation flow succeeds on retry.
	//   - Occurs during stream frame parsing, not connection setup. HTTP 200 is
	//     returned and a response id (r_xxx) is already assigned before the
	//     second frame carries code 13, so the server may have partially processed
	//     the request.
	//   - Tokens are healthy (all init requests return 200); no refresh needed.
	//   - Retrying the same request with identical parameters/metadata succeeds.
	//
	// Therefore we retry code 13 here after parseStreamResponse fails, not in
	// callStreamGenerate, which only handles connection-layer errors. Retry uses a
	// fresh reqID but the same inner request/metadata so the server sees a new
	// protocol attempt. If protocol behavior changes, this block is the central
	// place to adjust code 13 handling.
	retryableProtocolError := false
	err = runStreamGenerateAttempts(ctx, func() (bool, error) {
		body, requestErr := c.callStreamGenerate(ctx, transport.StreamGenerateRequest{
			AccessToken: s.accessToken,
			InnerReq:    innerJSON,
			UUID:        uuid,
			ModelHeader: model.Headers,
		}, s)
		if requestErr != nil {
			retryableProtocolError = false
			return isRetryableStreamGenerateError(requestErr), requestErr
		}

		emitted := false
		parseErr := c.parseStreamResponse(body, func(out *types.ModelOutput) {
			emitted = true
			cb(out)
		})
		transport.FinalizeStreamLog(body, parseErr)
		body.Close()
		if parseErr == nil {
			return false, nil
		}

		var eerr *rpcs.EnvelopeError
		retryableProtocolError = errors.As(parseErr, &eerr) && eerr.Code == 13 && !emitted
		return retryableProtocolError, parseErr
	}, func(attempt, maxAttempts int, retryErr error) {
		var eerr *rpcs.EnvelopeError
		if errors.As(retryErr, &eerr) && eerr.Code == 13 {
			log.Printf("gemini stream: code 13 (BardErrorInfo 1155) retry attempt %d/%d", attempt, maxAttempts)
			return
		}
		log.Printf("gemini stream: request retry attempt %d/%d: %v", attempt, maxAttempts, retryErr)
	})
	if err != nil && retryableProtocolError {
		log.Printf("gemini stream: code 13 retries exhausted after %d attempts", maxStreamGenerateAttempts)
	}
	return err
}
