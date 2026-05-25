package rpcs

import (
	"encoding/json"
	"testing"
)

const expectedPrefsSyncFeatureStatePayload = `[[null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,[["music_generation_soft","image_generation_soft","music_generation_soft","image_generation_soft","music_generation_soft"]]],[["tool_menu_soft_badge_disabled_ids"]]]`

const expectedPrefsSyncPopupStatePayload = `[[null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,1],[["popup_zs_visits_cooldown"]]]`

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
	if got != expectedPrefsSyncFeatureStatePayload {
		t.Fatalf("payload mismatch")
	}
}

func TestEncodePrefsSyncPopupState_WireParity(t *testing.T) {
	_, got := EncodePrefsSyncPopupState(PrefsSyncPopupState{Visits: 1})
	if got != expectedPrefsSyncPopupStatePayload {
		t.Fatalf("payload = %s, want %s", got, expectedPrefsSyncPopupStatePayload)
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
