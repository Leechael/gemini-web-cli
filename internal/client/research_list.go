package client

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// ListResearchReports fetches completed deep research reports from the library page.
func (c *Client) ListResearchReports(ctx context.Context, count int) ([]rpcs.ResearchReport, error) {
	reports, _, err := c.ListResearchReportsPage(ctx, count, "")
	return reports, err
}

// ListResearchReportsPage fetches completed research reports with optional pagination.
func (c *Client) ListResearchReportsPage(ctx context.Context, count int, cursor string) ([]rpcs.ResearchReport, string, error) {
	rpcID, payload := rpcs.EncodeListResearchReports(rpcs.ListReportsFilter{Count: count, Cursor: cursor})
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload, WithSourcePath("/library"))
	if err != nil {
		return nil, "", err
	}
	if rejectCode != 0 {
		return nil, "", fmt.Errorf("list_research_reports rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListResearchReportsPage(body)
}
