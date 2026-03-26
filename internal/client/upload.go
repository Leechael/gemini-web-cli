package client

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var uploadURL = "https://push.clients6.google.com/upload/"

const tenantID = "bard-storage"

// UploadResult holds the upload ID and file metadata needed for the request.
type UploadResult struct {
	ID       string
	FileName string
	MimeType string
}

// UploadFile uploads a file via Google's resumable upload protocol.
func (c *Client) UploadFile(ctx context.Context, filePath string) (*UploadResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	fileName := filepath.Base(filePath)
	mimeType := detectMimeType(fileName)

	id, err := c.resumableUpload(ctx, f, fileName, mimeType, stat.Size())
	if err != nil {
		return nil, err
	}

	return &UploadResult{ID: id, FileName: fileName, MimeType: mimeType}, nil
}

func (c *Client) resumableUpload(ctx context.Context, r io.Reader, fileName, mimeType string, size int64) (string, error) {
	// Step 1: Start — get upload URL
	uploadTarget, err := c.uploadStart(ctx, fileName, size)
	if err != nil {
		return "", fmt.Errorf("upload start: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(logWriter, "Upload target URL: %s\n", uploadTarget[:min(120, len(uploadTarget))])
	}

	// Step 2: Upload + Finalize — send file bytes
	return c.uploadFinalize(ctx, uploadTarget, r, size)
}

func (c *Client) uploadStart(ctx context.Context, fileName string, size int64) (string, error) {
	body := "File name: " + fileName

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Push-Id", c.pushID)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("X-Goog-Upload-Command", "start")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", fmt.Sprintf("%d", size))
	req.Header.Set("X-Goog-Upload-Protocol", "resumable")
	req.Header.Set("X-Tenant-Id", tenantID)

	c.setCookieHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload start request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload start returned HTTP %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	// The upload URL is in the x-goog-upload-url response header
	target := resp.Header.Get("X-Goog-Upload-Url")
	if target == "" {
		return "", fmt.Errorf("upload start: missing x-goog-upload-url in response")
	}

	return target, nil
}

func (c *Client) uploadFinalize(ctx context.Context, targetURL string, r io.Reader, size int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, r)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	req.Header.Set("Push-Id", c.pushID)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	req.Header.Set("X-Goog-Upload-Offset", "0")
	req.Header.Set("X-Tenant-Id", tenantID)
	req.ContentLength = size

	c.setCookieHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload finalize request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload finalize returned HTTP %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	uploadID, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading upload response: %w", err)
	}

	return strings.TrimSpace(string(uploadID)), nil
}

// setCookieHeader manually forwards cookies from the jar to cross-domain requests.
func (c *Client) setCookieHeader(req *http.Request) {
	geminiURL, _ := url.Parse(baseURL)
	var parts []string
	for _, ck := range c.httpClient.Jar.Cookies(geminiURL) {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	if len(parts) > 0 {
		req.Header.Set("Cookie", strings.Join(parts, "; "))
	}
}

func detectMimeType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}
