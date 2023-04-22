package vc

import (
	"fmt"

	"github.com/mobyvb/pull-pal/llm"

	"github.com/go-git/go-git/v5"
)

// Issue represents an issue on a version control server.
type Issue struct {
	ID      string
	Subject string
	Body    string
	URL     string
	Author  Author
}

// Author represents a commit, issue, or code change request author on a version control server.
type Author struct {
	Email  string
	Handle string
	Token  string
}

// Repository represents a version control repository and its local path.
type Repository struct {
	LocalPath  string
	HostDomain string
	Name       string
	Owner      Author
	localRepo  *git.Repository
}

// SSH returns the SSH connection string for the repository.
func (repo Repository) SSH() string {
	return fmt.Sprintf("git%s:%s/%s.git", repo.HostDomain, repo.Owner.Handle, repo.Name)
}

// HTTPS returns the HTTPS representation of the remote repository.
func (repo Repository) HTTPS() string {
	return fmt.Sprintf("https://%s/%s/%s.git", repo.HostDomain, repo.Owner.Handle, repo.Name)
}

// VCClient is an interface for version control server's client, e.g. a Github or Gerrit client.
type VCClient interface {
	// ListOpenIssues lists unresolved issues on the version control server.
	ListOpenIssues() ([]Issue, error)
	// OpenCodeChangeRequest opens a new "code change request" on the version control server (e.g. "pull request" in Github).
	OpenCodeChangeRequest(req llm.CodeChangeRequest, res llm.CodeChangeResponse) (id, url string, err error)
	// UpdateCodeChangeRequest updates an existing code change request on the version control server.
	// UpdateCodeChangeRequest(id string, res llm.CodeChangeResponse)
	// TODO: add/read comments to/from issues and code change requests
}
