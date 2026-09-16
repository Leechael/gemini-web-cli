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

func (c *Client) streamGenerate(ctx context.Context, prompt string, metadata []string, uploads []*UploadResult, model *types.Model, deepResearch bool, notebook string, cb StreamCallback) error {
	if model == nil {
		model = &types.Models[0]
	}

	uuid := generateUUID()
	mode := resolveGenerationMode(c.generationModeSnapshot(), prompt, uploads)
	s := c.session()

	modelHeaders := model.Headers
	if mode == "image" {
		modelHeaders = types.BuildModelHeaderForSurface(model.ModelID(), modelSelector(model), 2)
	}

	innerReq := c.buildInnerRequest(prompt, metadata, uploads, model, deepResearch, uuid, s.language, mode, normalizeNotebookResource(notebook))
	innerJSON, err := json.Marshal(innerReq)
	if err != nil {
		return fmt.Errorf("marshaling inner request: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(logWriter, "inner request payload bytes=%d\n", len(innerJSON))
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
	//
	// Mid-stream body timeouts (Client.Timeout / context deadline while reading)
	// are also retried here when no visible content has been delivered yet. On a
	// multi-account serve pool, exhausting these retries still allows failover to
	// the next account for new chats.
	//
	// Metadata-only frames are buffered until visible content arrives or the
	// attempt succeeds, then discarded on retry so callers never see duplicate or
	// stale chat ids from a failed attempt.
	retryableParseError := false
	err = runStreamGenerateAttempts(ctx, func() (bool, error) {
		body, requestErr := c.callStreamGenerate(ctx, transport.StreamGenerateRequest{
			AccessToken: s.accessToken,
			InnerReq:    innerJSON,
			UUID:        uuid,
			ModelHeader: modelHeaders,
		}, s)
		if requestErr != nil {
			retryableParseError = false
			return isRetryableStreamGenerateError(requestErr), requestErr
		}

		var buffered []*types.ModelOutput
		contentEmitted := false
		parseErr := c.parseStreamResponse(body, func(out *types.ModelOutput) {
			if streamOutputHasVisibleContent(out) {
				contentEmitted = true
				for _, b := range buffered {
					cb(b)
				}
				buffered = nil
				cb(out)
				return
			}
			buffered = append(buffered, out)
		})
		transport.FinalizeStreamLog(body, parseErr)
		body.Close()
		if parseErr == nil {
			for _, b := range buffered {
				cb(b)
			}
			return false, nil
		}

		var eerr *rpcs.EnvelopeError
		code13Retry := errors.As(parseErr, &eerr) && eerr.Code == 13 && !contentEmitted
		bodyRetry := isRetryableStreamBodyError(parseErr) && !contentEmitted
		retryableParseError = code13Retry || bodyRetry
		return retryableParseError, parseErr
	}, func(attempt, maxAttempts int, retryErr error) {
		var eerr *rpcs.EnvelopeError
		if errors.As(retryErr, &eerr) && eerr.Code == 13 {
			log.Printf("gemini stream: code 13 (BardErrorInfo 1155) retry attempt %d/%d: %s", attempt, maxAttempts, formatStreamAttemptError(retryErr))
			return
		}
		if isRetryableStreamBodyError(retryErr) {
			log.Printf("gemini stream: body read retry attempt %d/%d: %s", attempt, maxAttempts, formatStreamAttemptError(retryErr))
			return
		}
		log.Printf("gemini stream: request retry attempt %d/%d: %s", attempt, maxAttempts, formatStreamAttemptError(retryErr))
	})
	if err != nil {
		if retryableParseError {
			log.Printf("gemini stream: retries exhausted after %d attempts: %s", maxStreamGenerateAttempts, formatStreamAttemptError(err))
		} else {
			log.Printf("gemini stream: failed: %s", formatStreamAttemptError(err))
		}
	}
	return err
}
