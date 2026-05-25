package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/client/transport"
)

// RPCCall contains one RPC ID and payload pair for batch calls.
type RPCCall = transport.RPCCall

type rpcConfig struct {
	sourcePath    string
	sourcePathSet bool
	sourceCid     string
}

// RPCOpt configures a single CallRPC invocation.
type RPCOpt func(*rpcConfig)

// WithSourcePath overrides the source-path query parameter.
// If both WithSourcePath and WithSourceCid are passed, WithSourcePath wins.
func WithSourcePath(sp string) RPCOpt {
	return func(cfg *rpcConfig) {
		cfg.sourcePath = sp
		cfg.sourcePathSet = true
	}
}

// WithSourceCid sets source-path to c.appPath() + "/" + cid unless WithSourcePath is also passed.
func WithSourceCid(cid string) RPCOpt {
	return func(cfg *rpcConfig) {
		cfg.sourceCid = cid
	}
}

// CallRPC sends one RPC and returns the inner wrb.fr body bytes.
func (c *Client) CallRPC(ctx context.Context, rpcID, payload string, opts ...RPCOpt) (body []byte, rejectCode int, err error) {
	cfg := rpcConfig{sourcePath: c.appPath()}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.sourceCid != "" && !cfg.sourcePathSet {
		cfg.sourcePath = c.appPath() + "/" + cfg.sourceCid
	}

	raw, err := transport.PostBatch(ctx, transport.PostBatchRequest{
		Client: c.httpClient,
		URL: transport.BuildBatchURL(transport.BatchURLConfig{
			BaseURL:     baseURL,
			AccountPath: c.accountPath,
			RPCIDs:      []string{rpcID},
			ReqID:       c.nextReqID(),
			Language:    c.language,
			BuildLabel:  c.buildLabel,
			SessionID:   c.sessionID,
			SourcePath:  cfg.sourcePath,
		}),
		AccessToken: c.accessToken,
		RPCID:       rpcID,
		Payload:     payload,
		UserAgent:   userAgent,
	})
	if err != nil {
		return nil, 0, err
	}

	stripped := protocol.StripResponsePrefix(raw)
	return protocol.ExtractRPCBody(stripped, rpcID)
}

// CallRPCBatch sends multiple RPCs in one batchexecute request and returns bodies keyed by RPC ID.
func (c *Client) CallRPCBatch(ctx context.Context, calls []RPCCall, opts ...RPCOpt) (map[string][]byte, map[string]int, error) {
	cfg := rpcConfig{sourcePath: c.appPath()}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.sourceCid != "" && !cfg.sourcePathSet {
		cfg.sourcePath = c.appPath() + "/" + cfg.sourceCid
	}

	rpcIDs := make([]string, 0, len(calls))
	seen := map[string]bool{}
	for _, call := range calls {
		if seen[call.ID] {
			return nil, nil, fmt.Errorf("duplicate RPC ID in batch: %s", call.ID)
		}
		seen[call.ID] = true
		rpcIDs = append(rpcIDs, call.ID)
	}
	raw, err := transport.PostBatchMulti(ctx, transport.PostBatchMultiRequest{
		Client: c.httpClient,
		URL: transport.BuildBatchURL(transport.BatchURLConfig{
			BaseURL:     baseURL,
			AccountPath: c.accountPath,
			RPCIDs:      rpcIDs,
			ReqID:       c.nextReqID(),
			Language:    c.language,
			BuildLabel:  c.buildLabel,
			SessionID:   c.sessionID,
			SourcePath:  cfg.sourcePath,
		}),
		AccessToken: c.accessToken,
		Calls:       calls,
		UserAgent:   userAgent,
	})
	if err != nil {
		return nil, nil, err
	}

	stripped := protocol.StripResponsePrefix(raw)
	return protocol.ExtractRPCBodies(stripped, rpcIDs)
}
