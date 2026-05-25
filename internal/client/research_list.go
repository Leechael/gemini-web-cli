package client

import (
	"context"
	"fmt"

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
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/library"))
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("list_research_reports rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListResearchReports(body)
}
