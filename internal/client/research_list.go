package client

import (
	"context"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// ListResearchReports fetches completed deep research reports from the library page.
func (c *Client) ListResearchReports(ctx context.Context, count int) ([]rpcs.ResearchReport, error) {
	if count <= 0 {
		count = 4
	}
	rpcID, payload := rpcs.EncodeListResearchReports(rpcs.ListReportsFilter{
		Flags: []int{0, 0, 0, 1, 1, 0, 0, 1, 0},
		Count: count,
	})
	body, _, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/library"))
	if err != nil {
		return nil, err
	}
	return rpcs.DecodeListResearchReports(body)
}
