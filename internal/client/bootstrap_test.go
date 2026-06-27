package client

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

var bootstrapBodies = map[string]string{
	"o30O0e": `[[["me",1,["user-1",[],[[null,"Test User"]],[[null,"https://example.com/photo.jpg"]],null,null,null,null,null,[[null,"test@example.com"]]],null,[]]]]`,
	"K4WWud": `[["Sample Region, US","SWML_DESCRIPTION_FROM_YOUR_INTERNET_ADDRESS",false,null,"https://example.com/map.png"]]`,
	"cYRIkd": `[[[["academic_search"],"OpenStax","https://example.com/openstax.png"]],[[["google_calendar_2"],"Google Calendar","https://example.com/calendar.png"]]]`,
	"uPDUsc": `[[["workspace_tool","Google Workspace","Workspace tools",null,[2,null,"workspace_tool"],"https://example.com/workspace.png",["Gmail"],null,null,true],["gmail","Gmail","Email access",null,[1,null,"gmail"],"https://example.com/gmail.png",["Gmail"],null,null,false]],[]]`,
	"MyzX6c": `[false,[[1,true,1,0,null,1],[2,false,"off",0,null,1]]]`,
	"mhs1xe": `[[[[500,300,500000]]]]`,
}

func TestGetUserLocation(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	location, err := c.GetUserLocation(t.Context())
	if err != nil {
		t.Fatalf("GetUserLocation: %v", err)
	}
	if location.Region != "Sample Region, US" {
		t.Fatalf("Region = %q", location.Region)
	}
}

func TestListEnabledTools(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	tools, err := c.ListEnabledTools(t.Context())
	if err != nil {
		t.Fatalf("ListEnabledTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Name != "OpenStax" {
		t.Fatalf("tools[0].Name = %q, want OpenStax", tools[0].Name)
	}
}

func TestListExtensionCatalog(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	extensions, err := c.ListExtensionCatalog(t.Context())
	if err != nil {
		t.Fatalf("ListExtensionCatalog: %v", err)
	}
	if len(extensions) != 2 || extensions[0].ID != "workspace_tool" {
		t.Fatalf("extensions = %#v", extensions)
	}
}

func TestListFeatureFlags(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	flags, err := c.ListFeatureFlags(t.Context())
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if len(flags) != 2 || flags[0].ID != "1" {
		t.Fatalf("flags = %#v", flags)
	}
}

func TestGetUploadLimits(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	limits, err := c.GetUploadLimits(t.Context())
	if err != nil {
		t.Fatalf("GetUploadLimits: %v", err)
	}
	if limits.Limit0 != 500 || limits.Limit1 != 300 || limits.Limit2 != 500000 {
		t.Fatalf("limits = %#v", limits)
	}
}

func TestPrefetchBootstrap_AllSuccess(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	bs := c.PrefetchBootstrap(t.Context())
	assertBootstrapFilled(t, bs)
	if len(bs.Errors) != 0 {
		t.Fatalf("Errors = %#v, want empty", bs.Errors)
	}
}

func TestPrefetchBootstrap_PartialFailure(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "K4WWud")
	defer srv.Close()
	bs := c.PrefetchBootstrap(t.Context())
	if bs.Location != nil {
		t.Fatalf("Location = %#v, want nil", bs.Location)
	}
	if bs.Profile == nil || len(bs.Tools) == 0 || len(bs.Extensions) == 0 || len(bs.Flags) == 0 || bs.Limits == nil {
		t.Fatalf("unexpected bootstrap partial result: %#v", bs)
	}
	if len(bs.Errors) != 1 || bs.Errors["location"] == nil {
		t.Fatalf("Errors = %#v, want only location", bs.Errors)
	}
}

func TestPrefetchBootstrap_BatchTiming(t *testing.T) {
	delay := 50 * time.Millisecond
	c, srv := newBootstrapTestClient(t, delay, "")
	defer srv.Close()
	start := time.Now()
	bs := c.PrefetchBootstrap(t.Context())
	elapsed := time.Since(start)
	assertBootstrapFilled(t, bs)
	if elapsed >= delay*3 {
		t.Fatalf("elapsed = %s, want less than %s", elapsed, delay*3)
	}
}

func TestPrefetchViaBatch_RequestFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	origBase := baseURL
	baseURL = srv.URL
	t.Cleanup(func() { baseURL = origBase })
	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.httpClient = srv.Client()
	bs := c.prefetchViaBatch(t.Context())
	if len(bs.Errors) != 6 {
		t.Fatalf("len(Errors) = %d, want 6: %#v", len(bs.Errors), bs.Errors)
	}
	for _, key := range []string{"profile", "location", "tools", "extensions", "flags", "limits"} {
		if bs.Errors[key] == nil {
			t.Fatalf("Errors[%s] is nil", key)
		}
	}
}

func TestPrefetchViaBatch_Parity(t *testing.T) {
	c, srv := newBootstrapTestClient(t, 0, "")
	defer srv.Close()
	goroutineResult := c.prefetchViaGoroutine(t.Context())
	batchResult := c.prefetchViaBatch(t.Context())
	if !reflect.DeepEqual(goroutineResult, batchResult) {
		t.Fatalf("batch result mismatch\ngoroutine=%#v\nbatch=%#v", goroutineResult, batchResult)
	}
}

func newBootstrapTestClient(t *testing.T, delay time.Duration, failRPC string) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(newBootstrapTestHandler(delay, failRPC))
	origBase := baseURL
	baseURL = srv.URL
	t.Cleanup(func() { baseURL = origBase })

	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID.Store(1)
	c.httpClient = srv.Client()
	return c, srv
}

func newBootstrapTestHandler(delay time.Duration, failRPC string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		rpcids := r.URL.Query().Get("rpcids")
		if strings.Contains(rpcids, ",") {
			bodies := map[string]string{}
			for rpcID, body := range bootstrapBodies {
				if rpcID != failRPC {
					bodies[rpcID] = body
				}
			}
			_, _ = w.Write(makeTestMultiBatchResponse(bodies, nil))
			return
		}
		if rpcids == failRPC {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, ok := bootstrapBodies[rpcids]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(makeTestBatchResponse(rpcids, body, 0))
	})
}

func assertBootstrapFilled(t *testing.T, bs *Bootstrap) {
	t.Helper()
	if bs.Profile == nil || bs.Location == nil || len(bs.Tools) == 0 || len(bs.Extensions) == 0 || len(bs.Flags) == 0 || bs.Limits == nil {
		t.Fatalf("bootstrap result not filled: %#v", bs)
	}
}
