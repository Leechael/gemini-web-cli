package client

import (
	"testing"
)

func TestParseQuotaItems_Advanced(t *testing.T) {
	// One Gemini Pro item: action_id=4, usage=42.5%, reset_ts=1773265309,
	// total=100, remaining=57.
	body := `[[[[1,4],1,42.5,[1773265309,0],100,57]]]`
	out := map[string]*Quota{}
	parseQuotaItems(body, "Thinking/Pro", out)

	q := out["1-4"]
	if q == nil {
		t.Fatalf("missing quota for id 1-4: %#v", out)
	}
	if q.Label != "Gemini Pro" {
		t.Errorf("Label = %q, want Gemini Pro", q.Label)
	}
	if q.ActionID != 4 {
		t.Errorf("ActionID = %d, want 4", q.ActionID)
	}
	if q.UsagePercent != 42.5 {
		t.Errorf("UsagePercent = %v, want 42.5", q.UsagePercent)
	}
	if q.Total != 100 || q.Remaining != 57 {
		t.Errorf("Total/Remaining = %d/%d, want 100/57", q.Total, q.Remaining)
	}
	if q.ResetTime != 1773265309 {
		t.Errorf("ResetTime = %d, want 1773265309", q.ResetTime)
	}
}

func TestParseQuotaItems_Flash(t *testing.T) {
	body := `[[[[2,11],1,10,[1773000000,0],500,450]]]`
	out := map[string]*Quota{}
	parseQuotaItems(body, "Flash", out)

	q := out["2-11"]
	if q == nil {
		t.Fatalf("missing quota for id 2-11: %#v", out)
	}
	if q.Label != "Gemini Flash" {
		t.Errorf("Label = %q, want Gemini Flash", q.Label)
	}
	if q.Remaining != 450 {
		t.Errorf("Remaining = %d, want 450", q.Remaining)
	}
}

func TestParseQuotaItems_UnknownActionFallback(t *testing.T) {
	// Action id 99 is not in actionLabels — should fall back to "Gemini <category>".
	body := `[[[[1,99],1,0,[0,0],0,0]]]`
	out := map[string]*Quota{}
	parseQuotaItems(body, "Custom", out)

	q := out["1-99"]
	if q == nil {
		t.Fatalf("missing quota for id 1-99")
	}
	if q.Label != "Gemini Custom" {
		t.Errorf("Label = %q, want Gemini Custom", q.Label)
	}
}

func TestParseQuotaItems_Empty(t *testing.T) {
	out := map[string]*Quota{}
	parseQuotaItems("[]", "Flash", out)
	if len(out) != 0 {
		t.Errorf("expected empty result, got %#v", out)
	}
}

func TestParseExtraQuota_Blocked(t *testing.T) {
	q := parseExtraQuota(`[true,0.92,[1773265309,0]]`)
	if q == nil {
		t.Fatal("expected non-nil ExtraQuota")
	}
	if !q.IsBlocked {
		t.Error("expected IsBlocked=true")
	}
	if q.UsagePercent != 92 {
		t.Errorf("UsagePercent = %v, want 92", q.UsagePercent)
	}
	if q.ResetTime != 1773265309 {
		t.Errorf("ResetTime = %d, want 1773265309", q.ResetTime)
	}
}

func TestParseExtraQuota_Empty(t *testing.T) {
	if parseExtraQuota("[]") != nil {
		t.Error("expected nil for empty body")
	}
	if parseExtraQuota("") != nil {
		t.Error("expected nil for blank body")
	}
}
