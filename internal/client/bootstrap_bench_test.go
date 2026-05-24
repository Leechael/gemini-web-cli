package client

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

const benchPerRPCDelay = 50 * time.Millisecond

func BenchmarkPrefetchBootstrap_Goroutine(b *testing.B) {
	srv := newBootstrapBenchServer(benchPerRPCDelay)
	defer srv.Close()
	c := newBenchClient(b, srv)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.prefetchViaGoroutine(context.Background())
	}
}

func BenchmarkPrefetchBootstrap_Batch(b *testing.B) {
	srv := newBootstrapBenchServer(benchPerRPCDelay)
	defer srv.Close()
	c := newBenchClient(b, srv)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.prefetchViaBatch(context.Background())
	}
}

func newBootstrapBenchServer(delay time.Duration) *httptest.Server {
	return httptest.NewServer(newBootstrapTestHandler(delay, ""))
}

func newBenchClient(b *testing.B, srv *httptest.Server) *Client {
	b.Helper()
	origBase := baseURL
	baseURL = srv.URL
	b.Cleanup(func() { baseURL = origBase })
	c := newTestClient()
	c.accessToken = "token"
	c.language = "en"
	c.reqID = 1
	c.httpClient = srv.Client()
	return c
}
