package cmd

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/AIO-Starter/gemini-web-cli/internal/cookies"
	"github.com/AIO-Starter/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var downloadOutput string

var downloadCmd = &cobra.Command{
	Use:   "download [url_or_chat_id] [N]",
	Short: "Download a generated image by URL or chat ID",
	Long: `Download generated images.

Examples:
  download https://lh3.googleusercontent.com/...       # Direct URL
  download c_abc123                                    # All images from chat
  download c_abc123 -o output.png                      # All images (with prefix)
  download c_abc123 2                                  # Only the 2nd image`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		imageIndex := 0
		hasIndex := false

		// Check for N selector in second arg (e.g. "download c_xxx 2")
		if len(args) > 1 {
			sel := strings.TrimPrefix(args[1], "#") // also accept #N for compat
			n, err := strconv.Atoi(sel)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid image index %q — use 1, 2, etc.", args[1])
			}
			imageIndex = n - 1
			hasIndex = true
		}

		if strings.HasPrefix(target, "c_") {
			return downloadFromChat(target, imageIndex, hasIndex)
		} else if strings.HasPrefix(target, "http") {
			return downloadImage(target)
		}
		return fmt.Errorf("expected a URL or chat ID (c_...), got %q", target)
	},
}

func downloadFromChat(chatID string, index int, singleMode bool) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	turns, err := c.ReadChat(ctx, chatID, 30)
	if err != nil {
		return fmt.Errorf("reading chat: %w", err)
	}

	var allImages []types.Image
	for _, turn := range turns {
		allImages = append(allImages, turn.Images...)
	}

	if len(allImages) == 0 {
		return fmt.Errorf("no images found in chat %s", chatID)
	}

	if singleMode {
		// Download specific image by index
		if index >= len(allImages) {
			return fmt.Errorf("image #%d not found — chat has %d image(s)", index+1, len(allImages))
		}
		fmt.Fprintf(os.Stderr, "Found %d image(s) in chat, downloading #%d\n", len(allImages), index+1)
		return downloadImage(allImages[index].URL)
	}

	// Download all images
	fmt.Fprintf(os.Stderr, "Found %d image(s) in chat, downloading all\n", len(allImages))
	for i, img := range allImages {
		// Generate per-image output filename
		saved := downloadOutput
		if saved != "" {
			ext := filepath.Ext(saved)
			base := strings.TrimSuffix(saved, ext)
			if ext == "" {
				ext = ".png"
			}
			if len(allImages) > 1 {
				saved = fmt.Sprintf("%s_%d%s", base, i+1, ext)
			}
		}
		old := downloadOutput
		downloadOutput = saved
		if err := downloadImage(img.URL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download #%d: %v\n", i+1, err)
		}
		downloadOutput = old
	}
	return nil
}

func downloadImage(imgURL string) error {
	var jsonCookies map[string]string
	if cookiesJSON != "" {
		jar, err := cookies.Load(cookiesJSON)
		if err != nil {
			return fmt.Errorf("loading cookies: %w", err)
		}
		jsonCookies = jar.Cookies
	}

	cookieJar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://google.com")
	var httpCookies []*http.Cookie
	for k, v := range jsonCookies {
		httpCookies = append(httpCookies, &http.Cookie{Name: k, Value: v, Domain: ".google.com", Path: "/"})
	}
	cookieJar.SetCookies(u, httpCookies)

	// Determine output filename
	output := downloadOutput
	if output == "" {
		hash := md5.Sum([]byte(imgURL))
		output = fmt.Sprintf("gemini-%x.png", hash[:4])
	}

	// Append size param for full-size
	dlURL := imgURL
	if strings.Contains(imgURL, "googleusercontent.com") {
		parts := strings.Split(imgURL, "/")
		if !strings.Contains(parts[len(parts)-1], "=") {
			dlURL = imgURL + "=s2048"
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Jar:       cookieJar,
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", dlURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image") && !strings.Contains(contentType, "octet") {
		return fmt.Errorf("unexpected content-type: %s", contentType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, data, 0644); err != nil {
		return err
	}

	sizeKB := float64(len(data)) / 1024
	fmt.Printf("Saved to %s (%.1f KB)\n", output, sizeKB)
	return nil
}

func init() {
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Output file path (default: auto from URL)")
}
