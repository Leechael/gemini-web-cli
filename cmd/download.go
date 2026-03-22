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
	"time"

	"github.com/Leechael/gemini-web-cli/internal/cookies"
	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	downloadOutput string
	downloadPoll   bool
)

// downloadable is a unified item that can be downloaded from a chat.
type downloadable struct {
	URL     string
	Label   string // e.g. "image", "video", "mp3", "mp4"
	DefExt  string // default extension
	Poll206 bool   // whether to poll on HTTP 206
}

var downloadCmd = &cobra.Command{
	Use:   "download [url_or_chat_id] [N]",
	Short: "Download generated images, videos, or media by URL or chat ID",
	Long: `Download generated images, videos, or media.

Examples:
  download https://lh3.googleusercontent.com/...       # Direct URL (image)
  download --poll https://...                           # Direct URL with 206 polling (video)
  download c_abc123                                    # All media from chat
  download c_abc123 -o output.png                      # All media (with prefix)
  download c_abc123 2                                  # Only the 2nd item`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		itemIndex := 0
		hasIndex := false

		// Check for N selector in second arg (e.g. "download c_xxx 2")
		if len(args) > 1 {
			sel := strings.TrimPrefix(args[1], "#") // also accept #N for compat
			n, err := strconv.Atoi(sel)
			if err != nil || n < 1 {
				return fmt.Errorf("invalid index %q — use 1, 2, etc.", args[1])
			}
			itemIndex = n - 1
			hasIndex = true
		}

		if strings.HasPrefix(target, "c_") {
			return downloadFromChat(target, itemIndex, hasIndex)
		} else if strings.HasPrefix(target, "http") {
			defExt := ""
			if downloadPoll {
				defExt = ".mp4"
			}
			return downloadFile(target, defExt, downloadPoll)
		}
		return fmt.Errorf("expected a URL or chat ID (c_...), got %q", target)
	},
}

func collectDownloadables(turns []types.ChatTurn) []downloadable {
	var items []downloadable
	for _, turn := range turns {
		for _, img := range turn.Images {
			items = append(items, downloadable{
				URL:    img.URL,
				Label:  "image",
				DefExt: ".png",
			})
		}
		for _, vid := range turn.Videos {
			items = append(items, downloadable{
				URL:     vid.URL,
				Label:   "video",
				DefExt:  ".mp4",
				Poll206: true,
			})
		}
		for _, m := range turn.Media {
			if m.MP3URL != "" {
				items = append(items, downloadable{
					URL:     m.MP3URL,
					Label:   "mp3",
					DefExt:  ".mp3",
					Poll206: true,
				})
			}
			if m.MP4URL != "" {
				items = append(items, downloadable{
					URL:     m.MP4URL,
					Label:   "mp4",
					DefExt:  ".mp4",
					Poll206: true,
				})
			}
		}
	}
	return items
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

	items := collectDownloadables(turns)

	if len(items) == 0 {
		return fmt.Errorf("no downloadable media found in chat %s", chatID)
	}

	if singleMode {
		if index >= len(items) {
			return fmt.Errorf("item #%d not found — chat has %d item(s)", index+1, len(items))
		}
		item := items[index]
		fmt.Fprintf(os.Stderr, "Found %d item(s) in chat, downloading #%d (%s)\n", len(items), index+1, item.Label)
		return downloadFile(item.URL, item.DefExt, item.Poll206)
	}

	// Download all items
	fmt.Fprintf(os.Stderr, "Found %d item(s) in chat, downloading all\n", len(items))
	for i, item := range items {
		saved := downloadOutput
		if saved != "" {
			ext := filepath.Ext(saved)
			base := strings.TrimSuffix(saved, ext)
			if len(items) > 1 {
				ext = item.DefExt
			} else if ext == "" {
				ext = item.DefExt
			}
			if len(items) > 1 {
				saved = fmt.Sprintf("%s_%d%s", base, i+1, ext)
			}
		}
		old := downloadOutput
		downloadOutput = saved
		if err := downloadFile(item.URL, item.DefExt, item.Poll206); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download #%d (%s): %v\n", i+1, item.Label, err)
		}
		downloadOutput = old
	}
	return nil
}

// downloadFile downloads a file from a URL. If defaultExt is non-empty, it's used as the
// default extension when generating filenames. If poll206 is true, retries on HTTP 206
// (used for in-progress video/media generation).
func downloadFile(fileURL string, defaultExt string, poll206 bool) error {
	var jsonCookies map[string]string
	effectiveCookies := resolveCookiesJSON()
	if effectiveCookies != "" {
		jar, err := cookies.Load(effectiveCookies)
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
		hash := md5.Sum([]byte(fileURL))
		ext := ".png"
		if defaultExt != "" {
			ext = defaultExt
		}
		output = fmt.Sprintf("gemini-%x%s", hash[:4], ext)
	}

	// Append size param for full-size images
	dlURL := fileURL
	if defaultExt == "" && strings.Contains(fileURL, "googleusercontent.com") {
		parts := strings.Split(fileURL, "/")
		if !strings.Contains(parts[len(parts)-1], "=") {
			dlURL = fileURL + "=s2048"
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

	httpClient := &http.Client{
		Jar:       cookieJar,
		Transport: transport,
	}

	for {
		req, err := http.NewRequestWithContext(context.Background(), "GET", dlURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Origin", "https://gemini.google.com")
		req.Header.Set("Referer", "https://gemini.google.com/")

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		if resp.StatusCode == 206 && poll206 {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "Still generating (HTTP 206), retrying in 10s...\n")
			time.Sleep(10 * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
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
}

func init() {
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Output file path (default: auto from URL)")
	downloadCmd.Flags().BoolVar(&downloadPoll, "poll", false, "Poll on HTTP 206 (for in-progress video/media generation)")
}
