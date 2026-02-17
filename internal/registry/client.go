package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseSize = 10 << 20 // 10 MB

// Option configures a Client.
type Option func(*Client)

// Client fetches data from the registry.
type Client struct {
	baseURL     string // direct base URL for simple path concatenation (testing)
	gitlabHost  string // e.g. https://gitlab.cego.dk
	projectPath string // e.g. cego/ai-marketplace
	branch      string // e.g. master or feature/branch
	token       string
	httpClient  *http.Client
	cache       *Cache
}

// NewClient creates a new registry client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cache:      NewCache(5 * time.Minute),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithBaseURL sets a direct base URL for simple path concatenation.
// Paths like "/company-instructions/registry.json" are appended directly.
// This is primarily useful for testing with httptest servers.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithProjectURL parses a GitLab project URL into host and project path components
// for constructing GitLab API URLs that handle branches with slashes correctly.
func WithProjectURL(projectURL string) Option {
	return func(c *Client) {
		u, err := url.Parse(strings.TrimRight(projectURL, "/"))
		if err != nil {
			c.baseURL = projectURL
			return
		}
		c.gitlabHost = u.Scheme + "://" + u.Host
		c.projectPath = strings.TrimPrefix(u.Path, "/")
	}
}

// WithBranch sets the git branch/ref to fetch files from.
func WithBranch(branch string) Option {
	return func(c *Client) { c.branch = branch }
}

// WithToken sets the auth token.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithHTTPClient sets a custom HTTP client (useful for testing).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// fileURL builds the full URL for a file in the registry.
// If baseURL is set (testing), it uses simple concatenation.
// Otherwise it uses the GitLab API endpoint where the branch is a query parameter.
func (c *Client) fileURL(filePath string) string {
	if c.baseURL != "" {
		return c.baseURL + "/" + filePath
	}
	return fmt.Sprintf("%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		c.gitlabHost,
		url.PathEscape(c.projectPath),
		url.PathEscape(filePath),
		url.QueryEscape(c.branch),
	)
}

// FetchRegistry fetches and parses registry.json.
func (c *Client) FetchRegistry(ctx context.Context) (*Registry, error) {
	if cached, ok := c.cache.GetRegistry(); ok {
		return cached, nil
	}

	fileURL := c.fileURL("company-instructions/registry.json")
	data, err := c.get(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("fetching registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}

	c.cache.SetRegistry(&reg)
	return &reg, nil
}

// FetchStackManifest fetches and parses a stack's stack.json.
func (c *Client) FetchStackManifest(ctx context.Context, stackID string) (*StackManifest, error) {
	if cached, ok := c.cache.GetManifest(stackID); ok {
		return cached, nil
	}

	fileURL := c.fileURL(fmt.Sprintf("company-instructions/%s/stack.json", stackID))
	data, err := c.get(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("fetching stack manifest for %s: %w", stackID, err)
	}

	var manifest StackManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing stack manifest for %s: %w", stackID, err)
	}

	c.cache.SetManifest(stackID, &manifest)
	return &manifest, nil
}

// DownloadFile downloads a single file from a stack.
func (c *Client) DownloadFile(ctx context.Context, stackID, filename string) ([]byte, error) {
	fileURL := c.fileURL(fmt.Sprintf("company-instructions/%s/%s", stackID, filename))
	return c.get(ctx, fileURL)
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		return nil, fmt.Errorf("received HTML response from %s (expected JSON); check the registry URL and branch", url)
	}

	return data, nil
}
