// RPC: jGArJ — ListResearchReports
// Source-path: /library
// Reject codes: none observed in sample fixtures
//
// Payload shape:
//
//	[<filter_flags_array>, <count>]
//	  9-slot filter mask   max reports to return
//
// Response shape (after StripResponsePrefix + ExtractRPCBody):
//
//	[[report_arr, ...], <cursor or metadata>]
//
//	report_arr structure:
//	  [0]: ["<chat_id>", "<request_id>"]
//	  [1]: [<unix_seconds_created>, <nanos>]
//	  [2]: status or type flag
//	  [3]: "<title>"
//	  [4]: ["<content_snippet>"]
//	  [5]: "<report_id>"
//
// Test fixture: testdata/research_list_reports_basic.txt
//
// Notes:
//   - Filter flag meanings are not fully decoded; callers pass through the browser mask.
//   - The decoder searches nested arrays for report entries by ID/title/snippet/timestamp signature.
//   - This keeps decoding tolerant of future wrapper nesting changes while rejecting incomplete entries.
package rpcs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const listResearchReportsRPCID = "jGArJ"

// ResearchReport is one completed deep research report listed in the library.
type ResearchReport struct {
	Cid       string `json:"cid"`
	RequestID string `json:"requestId"`
	ReportID  string `json:"reportId"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	CreatedAt int64  `json:"createdAt"`
}

// ListReportsFilter controls the ListResearchReports query.
type ListReportsFilter struct {
	Flags []int
	Count int
}

// EncodeListResearchReports returns the ListResearchReports payload.
func EncodeListResearchReports(f ListReportsFilter) (rpcID, payload string) {
	flags := f.Flags
	if len(flags) == 0 {
		flags = []int{0, 0, 0, 1, 1, 0, 0, 1, 0}
	}
	count := f.Count
	if count <= 0 {
		count = 4
	}
	payloadBytes, _ := json.Marshal([]any{flags, count})
	return listResearchReportsRPCID, string(payloadBytes)
}

// DecodeListResearchReports parses the wrb.fr body JSON returned by ExtractRPCBody.
func DecodeListResearchReports(body []byte) ([]ResearchReport, error) {
	if strings.TrimSpace(string(body)) == "" || strings.TrimSpace(string(body)) == "[]" {
		return nil, nil
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode ListResearchReports JSON: %w", err)
	}
	var reports []ResearchReport
	collectResearchReports(data, &reports)
	return reports, nil
}

func collectResearchReports(value any, reports *[]ResearchReport) {
	arr, ok := value.([]any)
	if !ok {
		return
	}
	if report, ok := decodeResearchReportArray(arr); ok {
		*reports = append(*reports, report)
		return
	}
	for _, item := range arr {
		collectResearchReports(item, reports)
	}
}

func decodeResearchReportArray(arr []any) (ResearchReport, bool) {
	if len(arr) < 6 {
		return ResearchReport{}, false
	}

	ids, ok := arr[0].([]any)
	if !ok || len(ids) < 2 {
		return decodeLegacyResearchReportArray(arr)
	}
	cid, ok := ids[0].(string)
	if !ok || !strings.HasPrefix(cid, "c_") {
		return ResearchReport{}, false
	}
	requestID, ok := ids[1].(string)
	if !ok || !strings.HasPrefix(requestID, "r_") {
		return ResearchReport{}, false
	}

	createdParts, ok := arr[1].([]any)
	if !ok || len(createdParts) == 0 {
		return ResearchReport{}, false
	}
	created, ok := createdParts[0].(float64)
	if !ok {
		return ResearchReport{}, false
	}

	title, _ := arr[3].(string)
	if title == "" {
		return ResearchReport{}, false
	}
	snippetParts, ok := arr[4].([]any)
	if !ok || len(snippetParts) == 0 {
		return ResearchReport{}, false
	}
	snippet, _ := snippetParts[0].(string)
	if snippet == "" {
		return ResearchReport{}, false
	}
	reportID, ok := arr[5].(string)
	if !ok || !strings.HasPrefix(reportID, "rc_") {
		return ResearchReport{}, false
	}

	return ResearchReport{
		Cid:       cid,
		RequestID: requestID,
		ReportID:  reportID,
		Title:     title,
		Snippet:   snippet,
		CreatedAt: int64(created),
	}, true
}

func decodeLegacyResearchReportArray(arr []any) (ResearchReport, bool) {
	cid, ok := arr[0].(string)
	if !ok || !strings.HasPrefix(cid, "c_") {
		return ResearchReport{}, false
	}
	requestID, ok := arr[1].(string)
	if !ok || !strings.HasPrefix(requestID, "r_") {
		return ResearchReport{}, false
	}
	reportID, ok := arr[2].(string)
	if !ok || !strings.HasPrefix(reportID, "rc_") {
		return ResearchReport{}, false
	}
	title, _ := arr[3].(string)
	if title == "" {
		return ResearchReport{}, false
	}
	snippet, _ := arr[4].(string)
	if snippet == "" {
		return ResearchReport{}, false
	}
	created, ok := arr[5].(float64)
	if !ok {
		return ResearchReport{}, false
	}
	return ResearchReport{
		Cid:       cid,
		RequestID: requestID,
		ReportID:  reportID,
		Title:     title,
		Snippet:   snippet,
		CreatedAt: int64(created),
	}, true
}
