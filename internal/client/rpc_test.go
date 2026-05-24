package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCallRPC(t *testing.T) {
	var gotPath string
	var gotSourcePath string
	var gotReqID string
	var gotFormRPCID string
	var gotFormPayload string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotSourcePath = r.URL.Query().Get("source-path")
		gotReqID = r.URL.Query().Get("_reqid")
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		var outer [][][]any
		if err := json.Unmarshal([]byte(r.PostForm.Get("f.req")), &outer); err != nil {
			t.Fatal(err)
		}
		gotFormRPCID, _ = outer[0][0][0].(string)
		gotFormPayload, _ = outer[0][0][1].(string)
		_, _ = w.Write(makeTestBatchResponse("rpc", `["ok"]`, 0))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.buildLabel = "bl"
	c.sessionID = "sid"
	c.reqID = 123
	c.accountPath = "/u/1"
	c.httpClient = srv.Client()

	body, rejectCode, err := c.CallRPC(t.Context(), "rpc", "[1]", WithSourceCid("c_1"))
	if err != nil {
		t.Fatalf("CallRPC: %v", err)
	}
	if string(body) != `["ok"]` {
		t.Fatalf("body = %s, want [\"ok\"]", body)
	}
	if rejectCode != 0 {
		t.Fatalf("rejectCode = %d, want 0", rejectCode)
	}
	if gotPath != "/u/1/_/BardChatUi/data/batchexecute" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotSourcePath != "/u/1/app/c_1" {
		t.Fatalf("source-path = %q", gotSourcePath)
	}
	if gotReqID != "123" {
		t.Fatalf("_reqid = %q", gotReqID)
	}
	if gotFormRPCID != "rpc" || gotFormPayload != "[1]" {
		t.Fatalf("form rpc/payload = %q/%q", gotFormRPCID, gotFormPayload)
	}
}

func makeTestBatchResponse(rpcID, body string, rejectCode int) []byte {
	frame := `[["wrb.fr",` + strconv.Quote(rpcID) + `,` + strconv.Quote(body) + `,null,null,[` + strconv.Itoa(rejectCode) + `]]]`
	content := "\n" + frame + "\n"
	return []byte(")]}'\n" + strconv.Itoa(testUTF16Len(content)) + content)
}

func testUTF16Len(s string) int {
	units := 0
	for _, r := range s {
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return units
}

func TestMakeTestBatchResponse(t *testing.T) {
	if !strings.HasPrefix(string(makeTestBatchResponse("rpc", "[]", 0)), ")]}'\n") {
		t.Fatalf("test response missing prefix")
	}
}
