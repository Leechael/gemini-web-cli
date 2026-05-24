package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/Leechael/gemini-web-cli/internal/client/protocol/rpcs"
)

// Bootstrap contains best-effort bootstrap account and capability data.
type Bootstrap struct {
	Profile    *rpcs.UserProfile
	Location   *rpcs.UserLocation
	Tools      []rpcs.EnabledTool
	Extensions []rpcs.Extension
	Flags      []rpcs.FeatureFlag
	Limits     *rpcs.UploadLimits
	Errors     map[string]error
}

// GetUserLocation returns the current account's coarse location signal.
func (c *Client) GetUserLocation(ctx context.Context) (*rpcs.UserLocation, error) {
	rpcID, payload := rpcs.EncodeGetUserLocation()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetUserLocation rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetUserLocation(body)
}

// ListEnabledTools returns the tools enabled for the current account.
func (c *Client) ListEnabledTools(ctx context.Context) ([]rpcs.EnabledTool, error) {
	rpcID, payload := rpcs.EncodeListEnabledTools()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ListEnabledTools rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListEnabledTools(body)
}

// ListExtensionCatalog returns the complete extension catalog.
func (c *Client) ListExtensionCatalog(ctx context.Context) ([]rpcs.Extension, error) {
	rpcID, payload := rpcs.EncodeListExtensionCatalog()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ListExtensionCatalog rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListExtensionCatalog(body)
}

// ListFeatureFlags returns feature flag tuples for the current account.
func (c *Client) ListFeatureFlags(ctx context.Context) ([]rpcs.FeatureFlag, error) {
	rpcID, payload := rpcs.EncodeListFeatureFlags()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("ListFeatureFlags rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeListFeatureFlags(body)
}

// GetUploadLimits returns the current upload capability limits.
func (c *Client) GetUploadLimits(ctx context.Context) (*rpcs.UploadLimits, error) {
	rpcID, payload := rpcs.EncodeGetUploadLimits()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, payload)
	if err != nil {
		return nil, err
	}
	if rejectCode != 0 {
		return nil, fmt.Errorf("GetUploadLimits rejected with code=%d", rejectCode)
	}
	return rpcs.DecodeGetUploadLimits(body)
}

// PrefetchBootstrap fetches all bootstrap RPCs concurrently with best-effort errors.
func (c *Client) PrefetchBootstrap(ctx context.Context) *Bootstrap {
	return c.prefetchViaGoroutine(ctx)
}

func (c *Client) prefetchViaGoroutine(ctx context.Context) *Bootstrap {
	bs := &Bootstrap{Errors: map[string]error{}}
	var wg sync.WaitGroup
	var mu sync.Mutex
	recordError := func(key string, err error) {
		if err == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		bs.Errors[key] = err
	}

	wg.Add(6)
	go func() {
		defer wg.Done()
		profile, err := c.GetUserProfile(ctx)
		if err != nil {
			recordError("profile", err)
			return
		}
		bs.Profile = profile
	}()
	go func() {
		defer wg.Done()
		location, err := c.GetUserLocation(ctx)
		if err != nil {
			recordError("location", err)
			return
		}
		bs.Location = location
	}()
	go func() {
		defer wg.Done()
		tools, err := c.ListEnabledTools(ctx)
		if err != nil {
			recordError("tools", err)
			return
		}
		bs.Tools = tools
	}()
	go func() {
		defer wg.Done()
		extensions, err := c.ListExtensionCatalog(ctx)
		if err != nil {
			recordError("extensions", err)
			return
		}
		bs.Extensions = extensions
	}()
	go func() {
		defer wg.Done()
		flags, err := c.ListFeatureFlags(ctx)
		if err != nil {
			recordError("flags", err)
			return
		}
		bs.Flags = flags
	}()
	go func() {
		defer wg.Done()
		limits, err := c.GetUploadLimits(ctx)
		if err != nil {
			recordError("limits", err)
			return
		}
		bs.Limits = limits
	}()
	wg.Wait()
	return bs
}

func (c *Client) prefetchViaBatch(ctx context.Context) *Bootstrap {
	bs := &Bootstrap{Errors: map[string]error{}}
	calls := []RPCCall{
		newBootstrapCall(rpcs.EncodeGetUserProfile),
		newBootstrapCall(rpcs.EncodeGetUserLocation),
		newBootstrapCall(rpcs.EncodeListEnabledTools),
		newBootstrapCall(rpcs.EncodeListExtensionCatalog),
		newBootstrapCall(rpcs.EncodeListFeatureFlags),
		newBootstrapCall(rpcs.EncodeGetUploadLimits),
	}
	bodies, rejectCodes, err := c.CallRPCBatch(ctx, calls)
	if err != nil {
		bs.Errors["batch"] = err
		return bs
	}

	decodeBatchBody(bs, "profile", rpcs.EncodeGetUserProfile, bodies, rejectCodes, func(body []byte) error {
		profile, err := rpcs.DecodeGetUserProfile(body)
		bs.Profile = profile
		return err
	})
	decodeBatchBody(bs, "location", rpcs.EncodeGetUserLocation, bodies, rejectCodes, func(body []byte) error {
		location, err := rpcs.DecodeGetUserLocation(body)
		bs.Location = location
		return err
	})
	decodeBatchBody(bs, "tools", rpcs.EncodeListEnabledTools, bodies, rejectCodes, func(body []byte) error {
		tools, err := rpcs.DecodeListEnabledTools(body)
		bs.Tools = tools
		return err
	})
	decodeBatchBody(bs, "extensions", rpcs.EncodeListExtensionCatalog, bodies, rejectCodes, func(body []byte) error {
		extensions, err := rpcs.DecodeListExtensionCatalog(body)
		bs.Extensions = extensions
		return err
	})
	decodeBatchBody(bs, "flags", rpcs.EncodeListFeatureFlags, bodies, rejectCodes, func(body []byte) error {
		flags, err := rpcs.DecodeListFeatureFlags(body)
		bs.Flags = flags
		return err
	})
	decodeBatchBody(bs, "limits", rpcs.EncodeGetUploadLimits, bodies, rejectCodes, func(body []byte) error {
		limits, err := rpcs.DecodeGetUploadLimits(body)
		bs.Limits = limits
		return err
	})
	return bs
}

func newBootstrapCall(encode func() (string, string)) RPCCall {
	rpcID, payload := encode()
	return RPCCall{ID: rpcID, Payload: payload}
}

func decodeBatchBody(bs *Bootstrap, key string, encode func() (string, string), bodies map[string][]byte, rejectCodes map[string]int, decode func([]byte) error) {
	rpcID, _ := encode()
	if code := rejectCodes[rpcID]; code != 0 {
		bs.Errors[key] = fmt.Errorf("%s rejected with code=%d", key, code)
		return
	}
	body, ok := bodies[rpcID]
	if !ok {
		bs.Errors[key] = fmt.Errorf("%s response missing", key)
		return
	}
	if err := decode(body); err != nil {
		bs.Errors[key] = err
	}
}
