package client

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client/transport"
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
	s := c.session()
	uploadID, err := c.callUpload(ctx, transport.UploadRequest{
		PushURL:      uploadURL,
		PushID:       s.pushID,
		TenantID:     tenantID,
		Origin:       baseURL,
		Referer:      baseURL + "/",
		UserAgent:    userAgent,
		CookieHeader: c.buildCookieHeader(),
		FileName:     fileName,
		Size:         size,
		Body:         r,
	})
	if err != nil {
		return "", err
	}
	return uploadID, nil
}

func (c *Client) callUpload(ctx context.Context, req transport.UploadRequest) (string, error) {
	req.Client = c.httpClient
	return transport.PostUpload(ctx, req)
}

func (c *Client) buildCookieHeader() string {
	if c.httpClient == nil || c.httpClient.Jar == nil {
		return ""
	}
	geminiURL, _ := url.Parse(baseURL)
	var parts []string
	for _, ck := range c.httpClient.Jar.Cookies(geminiURL) {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	return strings.Join(parts, "; ")
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
