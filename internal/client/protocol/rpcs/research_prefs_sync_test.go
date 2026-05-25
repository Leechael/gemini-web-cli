package rpcs

import (
	"encoding/json"
	"testing"
)

func TestEncodePrefsSyncFeatureState_PayloadShape(t *testing.T) {
	flags := []string{"music_generation_soft", "image_generation_soft"}
	rpcID, payload := EncodePrefsSyncFeatureState(PrefsSyncFeatureState{FeatureFlags: flags})
	if rpcID != "L5adhe" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	state := got[0].([]any)
	if len(state) != 193 {
		t.Fatalf("state len = %d", len(state))
	}
	stored := state[192].([]any)[0].([]any)
	if stored[0] != flags[0] || stored[1] != flags[1] {
		t.Fatalf("stored flags = %#v", stored)
	}
}

func TestEncodePrefsSyncPopupState_PayloadShape(t *testing.T) {
	rpcID, payload := EncodePrefsSyncPopupState(PrefsSyncPopupState{Visits: 1})
	if rpcID != "L5adhe" {
		t.Fatalf("rpcID = %q", rpcID)
	}
	var got []any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatal(err)
	}
	state := got[0].([]any)
	if len(state) != 87 || state[86] != float64(1) {
		t.Fatalf("state = %#v", state)
	}
}

func TestEncodePrefsSyncFeatureState_WireParity(t *testing.T) {
	flags := []string{"music_generation_soft", "image_generation_soft", "music_generation_soft", "image_generation_soft", "music_generation_soft"}
	_, got := EncodePrefsSyncFeatureState(PrefsSyncFeatureState{FeatureFlags: flags})
	featureState := make([]any, 193)
	featureState[192] = []any{[]any{"music_generation_soft", "image_generation_soft", "music_generation_soft", "image_generation_soft", "music_generation_soft"}}
	wantBytes, _ := json.Marshal([]any{featureState, []any{[]any{"tool_menu_soft_badge_disabled_ids"}}})
	if got != string(wantBytes) {
		t.Fatalf("payload mismatch")
	}
}

func TestEncodePrefsSyncPopupState_WireParity(t *testing.T) {
	_, got := EncodePrefsSyncPopupState(PrefsSyncPopupState{Visits: 1})
	popupState := make([]any, 87)
	popupState[86] = 1
	wantBytes, _ := json.Marshal([]any{popupState, []any{[]any{"popup_zs_visits_cooldown"}}})
	if got != string(wantBytes) {
		t.Fatalf("payload = %s, want %s", got, string(wantBytes))
	}
}

func TestDecodePrefsSync_FromSampleFixture(t *testing.T) {
	body := rpcFixtureBody(t, "research_prefs_sync_basic.txt", "L5adhe")
	if err := DecodePrefsSync(body); err != nil {
		t.Fatal(err)
	}
}

func TestDecodePrefsSync_EmptyBody(t *testing.T) {
	if err := DecodePrefsSync(nil); err != nil {
		t.Fatal(err)
	}
}

func TestDecodePrefsSync_MalformedJSON(t *testing.T) {
	if err := DecodePrefsSync([]byte("[")); err == nil {
		t.Fatal("expected error")
	}
}
