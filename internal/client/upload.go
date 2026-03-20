package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	uploadURL = "https://content-push.googleapis.com/upload"
	pushID    = "feeds/mcudyrk2a4khkz"
)

// UploadFile uploads a file to Gemini and returns its upload ID.
func (c *Client) UploadFile(ctx context.Context, filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	fileName := filepath.Base(filePath)
	contentType := detectContentType(fileName)

	return c.uploadReader(ctx, f, fileName, contentType, stat.Size())
}

// UploadReader uploads data from a reader with the given filename.
func (c *Client) UploadReader(ctx context.Context, r io.Reader, fileName string, size int64) (string, error) {
	contentType := detectContentType(fileName)
	return c.uploadReader(ctx, r, fileName, contentType, size)
}

func (c *Client) uploadReader(ctx context.Context, r io.Reader, fileName, contentType string, size int64) (string, error) {
	boundary := generateBoundary()

	// Build multipart body manually (matching pi-web-access behavior)
	var body strings.Builder
	body.WriteString("------" + boundary + "\r\n")
	body.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", fileName))
	body.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentType))
	body.WriteString("\r\n")

	header := body.String()

	footer := "\r\n------" + boundary + "--\r\n"

	// Use a pipe to stream the multipart body
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		pw.Write([]byte(header))
		io.Copy(pw, r)
		pw.Write([]byte(footer))
	}()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, pr)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "multipart/form-data; boundary=----"+boundary)
	req.Header.Set("Push-Id", pushID)
	req.Header.Set("User-Agent", userAgent)

	// Manually forward cookies — the jar has them for .google.com but
	// the upload endpoint is content-push.googleapis.com (different domain).
	geminiURL, _ := url.Parse(baseURL)
	var cookieParts []string
	for _, ck := range c.httpClient.Jar.Cookies(geminiURL) {
		cookieParts = append(cookieParts, ck.Name+"="+ck.Value)
	}
	if len(cookieParts) > 0 {
		req.Header.Set("Cookie", strings.Join(cookieParts, "; "))
	}

	// Content-Length if known
	if size > 0 {
		req.ContentLength = int64(len(header)) + size + int64(len(footer))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload returned HTTP %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	uploadID, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading upload response: %w", err)
	}

	return strings.TrimSpace(string(uploadID)), nil
}

func detectContentType(fileName string) string {
	ext := filepath.Ext(fileName)
	if ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}

func generateBoundary() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1e16))
	return fmt.Sprintf("FormBoundary%016d", n.Int64())
}
