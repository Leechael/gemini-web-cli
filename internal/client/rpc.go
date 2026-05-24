package client

import (
	"context"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol"
	"github.com/Leechael/gemini-web-cli/internal/client/transport"
)

type rpcConfig struct {
	sourcePath string
	sourceCid  string
}

// RPCOpt configures a single CallRPC invocation.
type RPCOpt func(*rpcConfig)

// WithSourcePath overrides the source-path query parameter.
func WithSourcePath(sp string) RPCOpt {
	return func(cfg *rpcConfig) {
		cfg.sourcePath = sp
	}
}

// WithSourceCid sets source-path to c.appPath() + "/" + cid.
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
	if cfg.sourceCid != "" {
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
