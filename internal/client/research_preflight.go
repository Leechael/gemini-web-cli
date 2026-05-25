package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// deepResearchPreflight sends the preflight RPCs needed to enable deep research.
// All calls are best-effort; errors are logged but not fatal.
func (c *Client) deepResearchPreflight(ctx context.Context, cid string, rid string) {
	rpcID, payload := rpcs.EncodeBardSettings("bard_activity_enabled")
	c.bestEffortRPC(ctx, rpcID, payload)

	rpcID, payload = rpcs.EncodePrefsSyncFeatureState(rpcs.PrefsSyncFeatureState{
		FeatureFlags: []string{
			"music_generation_soft",
			"image_generation_soft",
			"music_generation_soft",
			"image_generation_soft",
			"music_generation_soft",
		},
	})
	c.bestEffortRPC(ctx, rpcID, payload)

	rpcID, payload = rpcs.EncodePrefsSyncPopupState(rpcs.PrefsSyncPopupState{Visits: 1})
	c.bestEffortRPC(ctx, rpcID, payload)

	rpcID, payload = rpcs.EncodeDeepResearchBootstrap("en")
	c.bestEffortRPC(ctx, rpcID, payload)

	if cid != "" {
		c.bestEffortRPCBatch(ctx, []RPCCall{
			{ID: "qpEbW", Payload: `[[[1,4],[6,6],[1,15]]]`},
			{ID: "aPya6c", Payload: `[]`},
		}, WithSourceCid(cid))

		if rid != "" {
			rpcID, payload = rpcs.EncodeDeepResearchAck(rid)
			c.bestEffortRPC(ctx, rpcID, payload, WithSourceCid(cid))
		}
	}
}

func (c *Client) bestEffortRPC(ctx context.Context, rpcID, payload string, opts ...RPCOpt) {
	if _, rejectCode, err := c.CallRPC(ctx, rpcID, payload, opts...); err != nil {
		fmt.Fprintf(logWriter, "preflight RPC %s failed: %v\n", rpcID, err)
	} else if rejectCode != 0 {
		fmt.Fprintf(logWriter, "preflight RPC %s rejected with code=%d\n", rpcID, rejectCode)
	}
}

func (c *Client) bestEffortRPCBatch(ctx context.Context, calls []RPCCall, opts ...RPCOpt) {
	_, rejectCodes, err := c.CallRPCBatch(ctx, calls, opts...)
	if err != nil {
		ids := make([]string, len(calls))
		for i, call := range calls {
			ids[i] = call.ID
		}
		fmt.Fprintf(logWriter, "preflight batch %v failed: %v\n", ids, err)
		return
	}
	for _, call := range calls {
		if rejectCode := rejectCodes[call.ID]; rejectCode != 0 {
			fmt.Fprintf(logWriter, "preflight RPC %s rejected with code=%d\n", call.ID, rejectCode)
		}
	}
}
