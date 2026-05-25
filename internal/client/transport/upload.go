package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// UploadRequest contains the inputs for Google's resumable upload flow.
type UploadRequest struct {
	Client       *http.Client
	PushURL      string
	PushID       string
	TenantID     string
	Origin       string
	Referer      string
	UserAgent    string
	CookieHeader string
	FileName     string
	Size         int64
	Body         io.Reader
}

// PostUpload runs the start and finalize requests and returns the upload id.
func PostUpload(ctx context.Context, req UploadRequest) (string, error) {
	sessionURL, err := uploadStart(ctx, req)
	if err != nil {
		return "", fmt.Errorf("upload start: %w", err)
	}
	uploadID, err := uploadFinalize(ctx, sessionURL, req)
	if err != nil {
		return "", fmt.Errorf("upload finalize: %w", err)
	}
	return uploadID, nil
}

func uploadStart(ctx context.Context, req UploadRequest) (string, error) {
	body := "File name: " + req.FileName
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.PushURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	setUploadCommonHeaders(httpReq, req)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	httpReq.Header.Set("X-Goog-Upload-Command", "start")
	httpReq.Header.Set("X-Goog-Upload-Header-Content-Length", strconv.FormatInt(req.Size, 10))
	httpReq.Header.Set("X-Goog-Upload-Protocol", "resumable")

	resp, err := uploadHTTPClient(req).Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("upload start request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload start returned HTTP %d: %s", resp.StatusCode, snippet(body, 200))
	}
	sessionURL := resp.Header.Get("X-Goog-Upload-Url")
	if sessionURL == "" {
		return "", fmt.Errorf("upload start: missing x-goog-upload-url in response")
	}
	return sessionURL, nil
}

func uploadFinalize(ctx context.Context, sessionURL string, req UploadRequest) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", sessionURL, req.Body)
	if err != nil {
		return "", err
	}
	setUploadCommonHeaders(httpReq, req)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	httpReq.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	httpReq.Header.Set("X-Goog-Upload-Offset", "0")
	httpReq.ContentLength = req.Size

	resp, err := uploadHTTPClient(req).Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("upload finalize request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload finalize returned HTTP %d: %s", resp.StatusCode, snippet(body, 200))
	}
	uploadID, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading upload response: %w", err)
	}
	return strings.TrimSpace(string(uploadID)), nil
}

func setUploadCommonHeaders(httpReq *http.Request, req UploadRequest) {
	if req.PushID != "" {
		httpReq.Header.Set("Push-Id", req.PushID)
	}
	if req.UserAgent != "" {
		httpReq.Header.Set("User-Agent", req.UserAgent)
	}
	if req.Origin != "" {
		httpReq.Header.Set("Origin", req.Origin)
	}
	if req.Referer != "" {
		httpReq.Header.Set("Referer", req.Referer)
	}
	if req.TenantID != "" {
		httpReq.Header.Set("X-Tenant-Id", req.TenantID)
	}
	if req.CookieHeader != "" {
		httpReq.Header.Set("Cookie", req.CookieHeader)
	}
}

func uploadHTTPClient(req UploadRequest) *http.Client {
	if req.Client != nil {
		return req.Client
	}
	return http.DefaultClient
}

func snippet(body []byte, limit int) string {
	if len(body) > limit {
		body = body[:limit]
	}
	return string(body)
}
