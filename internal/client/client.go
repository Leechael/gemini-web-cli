package client

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/AIO-Starter/gemini-web-cli/internal/types"
)

const (
	baseURL      = "https://gemini.google.com"
	userAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
	cookieDomain = ".google.com"
)

// Client communicates with the Gemini web API.
type Client struct {
	httpClient   *http.Client
	accessToken  string
	buildLabel   string
	sessionID    string
	reqID        int
	accountIndex *int
	accountPath  string // "" or "/u/N"
	model        *types.Model
	proxy        string
	verbose      bool
	timeout      time.Duration

	// Cookies for persistence tracking
	ExtraCookies map[string]string
}

// Config holds client construction parameters.
type Config struct {
	Secure1PSID   string
	Secure1PSIDTS string
	ExtraCookies  map[string]string
	Proxy         string
	AccountIndex  *int
	Model         *types.Model
	Verbose       bool
	Timeout       time.Duration
}

// New creates a new Client (not yet initialized).
func New(cfg Config) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(baseURL)

	// Set required cookies
	cookies := []*http.Cookie{
		{Name: "__Secure-1PSID", Value: cfg.Secure1PSID, Domain: cookieDomain, Path: "/", Secure: true, HttpOnly: true},
	}
	if cfg.Secure1PSIDTS != "" {
		cookies = append(cookies, &http.Cookie{Name: "__Secure-1PSIDTS", Value: cfg.Secure1PSIDTS, Domain: cookieDomain, Path: "/", Secure: true, HttpOnly: true})
	}
	for k, v := range cfg.ExtraCookies {
		cookies = append(cookies, &http.Cookie{Name: k, Value: v, Domain: cookieDomain, Path: "/"})
	}
	jar.SetCookies(u, cookies)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	// Generate random reqID
	n, _ := rand.Int(rand.Reader, big.NewInt(90000))
	reqID := int(n.Int64()) + 10000

	// Account path for multi-account support
	var accountPath string
	if cfg.AccountIndex != nil {
		accountPath = fmt.Sprintf("/u/%d", *cfg.AccountIndex)
	}

	return &Client{
		httpClient: &http.Client{
			Jar:       jar,
			Transport: transport,
			Timeout:   timeout,
		},
		reqID:        reqID,
		accountIndex: cfg.AccountIndex,
		accountPath:  accountPath,
		model:        cfg.Model,
		proxy:        cfg.Proxy,
		verbose:      cfg.Verbose,
		timeout:      timeout,
		ExtraCookies: cfg.ExtraCookies,
	}, nil
}

// Init fetches the access token and session data from the Gemini app page.
func (c *Client) Init(ctx context.Context) error {
	appURL := c.appPath()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+appURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("init request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("init returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading init response: %w", err)
	}

	htmlBody := string(body)

	token := extractRegex(htmlBody, `"SNlM0e"\s*:\s*"([^"]*)"`)
	if token == "" {
		return fmt.Errorf("failed to extract access token (SNlM0e) — cookies may be invalid")
	}
	c.accessToken = token

	c.buildLabel = extractRegex(htmlBody, `"cfb2h"\s*:\s*"([^"]*)"`)
	c.sessionID = extractRegex(htmlBody, `"FdrFJe"\s*:\s*"([^"]*)"`)

	if c.verbose {
		fmt.Fprintf(logWriter, "Init OK: token=%s... bl=%s sid=%s\n", token[:min(8, len(token))], c.buildLabel, c.sessionID)
	}

	return nil
}

// Close releases resources.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}

// appPath returns the path for this account: "/app" or "/u/N/app".
func (c *Client) appPath() string {
	return c.accountPath + "/app"
}

func (c *Client) nextReqID() int {
	id := c.reqID
	c.reqID += 100000
	return id
}

func (c *Client) commonHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	h.Set("Host", "gemini.google.com")
	h.Set("Origin", baseURL)
	h.Set("Referer", baseURL+"/")
	h.Set("User-Agent", userAgent)
	h.Set("X-Same-Domain", "1")
	return h
}

// streamURL constructs the StreamGenerate URL with account path prefix.
func (c *Client) streamURL() string {
	params := url.Values{}
	params.Set("_reqid", fmt.Sprintf("%d", c.nextReqID()))
	params.Set("rt", "c")
	params.Set("hl", "en")
	params.Set("pageId", "none")
	if c.buildLabel != "" {
		params.Set("bl", c.buildLabel)
	}
	if c.sessionID != "" {
		params.Set("f.sid", c.sessionID)
	}

	// Account path prefix goes into the URL path, not just params
	path := c.accountPath + "/_/BardChatUi/data/assistant.lamda.BardFrontendService/StreamGenerate"
	return baseURL + path + "?" + params.Encode()
}

// batchURL constructs the batchexecute URL with account path prefix.
func (c *Client) batchURL(rpcIDs []string, sourcePath string) string {
	params := url.Values{}
	params.Set("rpcids", strings.Join(rpcIDs, ","))
	params.Set("_reqid", fmt.Sprintf("%d", c.nextReqID()))
	params.Set("rt", "c")
	params.Set("hl", "en")
	params.Set("pageId", "none")
	if sourcePath != "" {
		params.Set("source-path", sourcePath)
	} else {
		params.Set("source-path", c.appPath())
	}
	if c.buildLabel != "" {
		params.Set("bl", c.buildLabel)
	}
	if c.sessionID != "" {
		params.Set("f.sid", c.sessionID)
	}

	// Account path prefix goes into the URL path
	return baseURL + c.accountPath + "/_/BardChatUi/data/batchexecute?" + params.Encode()
}

// logWriter is stderr for debug output.
var logWriter io.Writer = io.Discard

// SetVerbose enables debug logging to stderr.
func SetVerbose(w io.Writer) {
	logWriter = w
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
