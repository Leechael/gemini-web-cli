package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodeListResearchReports_PayloadShape(t *testing.T) {
	rpcID, payload := EncodeListResearchReports(ListReportsFilter{})
	if rpcID != "jGArJ" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	if payload != `[[0,0,0,1,1,0,0,1,0],4]` {
		t.Fatalf("payload = %s", payload)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
}

func TestEncodeListResearchReports_CustomFilter(t *testing.T) {
	_, payload := EncodeListResearchReports(ListReportsFilter{Flags: []int{1, 0, 0}, Count: 10})
	if payload != `[[1,0,0],10]` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestEncodeListResearchReports_WithCursor(t *testing.T) {
	_, payload := EncodeListResearchReports(ListReportsFilter{Count: 4, Cursor: "next_cursor"})
	if payload != `[[0,0,0,1,1,0,0,1,0],4,"next_cursor"]` {
		t.Fatalf("payload = %s", payload)
	}
}

func TestDecodeListResearchReports_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "research_list_reports_basic.txt", "jGArJ")
	reports, cursor, err := DecodeListResearchReportsPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "sample_cursor" {
		t.Fatalf("cursor = %q", cursor)
	}
	if len(reports) != 2 {
		t.Fatalf("reports = %d", len(reports))
	}
	if reports[0].Cid != "c_000000000000001" || reports[0].RequestID != "r_000000000000001" || reports[0].ReportID != "rc_000000000000001" {
		t.Fatalf("report[0] ids = %+v", reports[0])
	}
	if reports[0].Title != "Sample research report title" {
		t.Fatalf("Title = %q", reports[0].Title)
	}
	if reports[0].Snippet != "Sample report content snippet." {
		t.Fatalf("Snippet = %q", reports[0].Snippet)
	}
	if reports[0].CreatedAt != 1700000000 {
		t.Fatalf("CreatedAt = %d", reports[0].CreatedAt)
	}
}

func TestDecodeListResearchReports_EmptyBody(t *testing.T) {
	reports, err := DecodeListResearchReports([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 0 {
		t.Fatalf("reports = %d", len(reports))
	}
}

func TestDecodeListResearchReports_MalformedJSON(t *testing.T) {
	if _, err := DecodeListResearchReports([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
