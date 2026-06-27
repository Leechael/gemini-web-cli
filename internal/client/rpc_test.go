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
			t.Errorf("ParseForm: %v", err)
			return
		}
		var outer [][][]any
		if err := json.Unmarshal([]byte(r.PostForm.Get("f.req")), &outer); err != nil {
			t.Errorf("Unmarshal f.req: %v", err)
			return
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
	c.reqID.Store(123)
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

func TestCallRPC_WithSourcePath(t *testing.T) {
	var gotSourcePath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSourcePath = r.URL.Query().Get("source-path")
		_, _ = w.Write(makeTestBatchResponse("rpc", `[]`, 0))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.accountPath = "/u/1"
	c.httpClient = srv.Client()

	_, _, err := c.CallRPC(t.Context(), "rpc", "[]", WithSourcePath("/manual"))
	if err != nil {
		t.Fatalf("CallRPC: %v", err)
	}
	if gotSourcePath != "/manual" {
		t.Fatalf("source-path = %q, want /manual", gotSourcePath)
	}
}

func TestCallRPC_SourcePathOverridesSourceCid(t *testing.T) {
	var gotSourcePath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSourcePath = r.URL.Query().Get("source-path")
		_, _ = w.Write(makeTestBatchResponse("rpc", `[]`, 0))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.accountPath = "/u/1"
	c.httpClient = srv.Client()

	_, _, err := c.CallRPC(t.Context(), "rpc", "[]", WithSourcePath("/manual"), WithSourceCid("c_1"))
	if err != nil {
		t.Fatalf("CallRPC: %v", err)
	}
	if gotSourcePath != "/manual" {
		t.Fatalf("source-path = %q, want /manual", gotSourcePath)
	}
}

func TestCallRPCBatch_HappyPath(t *testing.T) {
	var gotRPCIDs string
	var gotBatchCount int
	var gotCallCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRPCIDs = r.URL.Query().Get("rpcids")
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
			return
		}
		var outer [][][]any
		if err := json.Unmarshal([]byte(r.PostForm.Get("f.req")), &outer); err != nil {
			t.Errorf("Unmarshal f.req: %v", err)
			return
		}
		gotBatchCount = len(outer)
		if len(outer) > 0 {
			gotCallCount = len(outer[0])
		}
		_, _ = w.Write(makeTestMultiBatchResponse(map[string]string{"one": `[1]`, "two": `[2]`}, nil))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.httpClient = srv.Client()

	bodies, rejectCodes, err := c.CallRPCBatch(t.Context(), []RPCCall{{ID: "one", Payload: "[1]"}, {ID: "two", Payload: "[2]"}})
	if err != nil {
		t.Fatalf("CallRPCBatch: %v", err)
	}
	if gotRPCIDs != "one,two" {
		t.Fatalf("rpcids = %q, want one,two", gotRPCIDs)
	}
	if gotBatchCount != 1 {
		t.Fatalf("gotBatchCount = %d, want 1", gotBatchCount)
	}
	if gotCallCount != 2 {
		t.Fatalf("gotCallCount = %d, want 2", gotCallCount)
	}
	if string(bodies["one"]) != `[1]` || string(bodies["two"]) != `[2]` {
		t.Fatalf("bodies = %#v", bodies)
	}
	if len(rejectCodes) != 0 {
		t.Fatalf("rejectCodes = %#v, want empty", rejectCodes)
	}
}

func TestCallRPCBatch_DuplicateID(t *testing.T) {
	c := newTestClient()
	_, _, err := c.CallRPCBatch(t.Context(), []RPCCall{{ID: "dup", Payload: "[1]"}, {ID: "dup", Payload: "[2]"}})
	if err == nil {
		t.Fatalf("CallRPCBatch duplicate ID error = nil")
	}
}

func TestCallRPCBatch_PartialResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeTestMultiBatchResponse(map[string]string{"one": `[1]`, "two": `[2]`}, nil))
	}))
	defer srv.Close()

	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.httpClient = srv.Client()

	bodies, rejectCodes, err := c.CallRPCBatch(t.Context(), []RPCCall{{ID: "one", Payload: "[1]"}, {ID: "two", Payload: "[2]"}, {ID: "missing", Payload: "[]"}})
	if err != nil {
		t.Fatalf("CallRPCBatch: %v", err)
	}
	if _, ok := bodies["missing"]; ok {
		t.Fatalf("missing RPC present in bodies")
	}
	if _, ok := rejectCodes["missing"]; ok {
		t.Fatalf("missing RPC present in rejectCodes")
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

func makeTestMultiBatchResponse(bodies map[string]string, rejectCodes map[string]int) []byte {
	frames := make([]string, 0, len(bodies))
	for rpcID, body := range bodies {
		rejectCode := 0
		if rejectCodes != nil {
			rejectCode = rejectCodes[rpcID]
		}
		frames = append(frames, `[["wrb.fr",`+strconv.Quote(rpcID)+`,`+strconv.Quote(body)+`,null,null,[`+strconv.Itoa(rejectCode)+`]]]`)
	}
	var b strings.Builder
	b.WriteString(")]}'\n")
	for _, frame := range frames {
		content := "\n" + frame + "\n"
		b.WriteString(strconv.Itoa(testUTF16Len(content)))
		b.WriteString(content)
	}
	return []byte(b.String())
}

func TestMakeTestBatchResponse(t *testing.T) {
	if !strings.HasPrefix(string(makeTestBatchResponse("rpc", "[]", 0)), ")]}'\n") {
		t.Fatalf("test response missing prefix")
	}
}
