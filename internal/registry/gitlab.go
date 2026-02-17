package registry

import (
	"fmt"
	"net/url"
	"strings"
)

// GitLabURLBuilder constructs raw file API URLs for GitLab-hosted registries.
type GitLabURLBuilder struct {
	BaseURL   string
	ProjectID string
	Branch    string
}

// NewGitLabURLBuilder creates a builder for a GitLab project.
func NewGitLabURLBuilder(baseURL, projectID, branch string) *GitLabURLBuilder {
	return &GitLabURLBuilder{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		ProjectID: projectID,
		Branch:    branch,
	}
}

// RawFileURL returns the GitLab raw file API URL for a given path.
// Uses: GET /api/v4/projects/:id/repository/files/:file_path/raw?ref=:branch
func (b *GitLabURLBuilder) RawFileURL(filePath string) string {
	encodedPath := url.PathEscape(filePath)
	return fmt.Sprintf(
		"%s/api/v4/projects/%s/repository/files/%s/raw?ref=%s",
		b.BaseURL,
		url.PathEscape(b.ProjectID),
		encodedPath,
		url.QueryEscape(b.Branch),
	)
}

// RegistryJSONURL returns the URL for registry.json.
func (b *GitLabURLBuilder) RegistryJSONURL() string {
	return b.RawFileURL("registry.json")
}

// StackManifestURL returns the URL for a stack's stack.json.
func (b *GitLabURLBuilder) StackManifestURL(stackID string) string {
	return b.RawFileURL(fmt.Sprintf("stacks/%s/stack.json", stackID))
}

// StackFileURL returns the URL for a specific file in a stack.
func (b *GitLabURLBuilder) StackFileURL(stackID, filename string) string {
	return b.RawFileURL(fmt.Sprintf("stacks/%s/%s", stackID, filename))
}
