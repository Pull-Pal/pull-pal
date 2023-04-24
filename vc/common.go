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

func (i Issue) String() string {
	return fmt.Sprintf("Issue ID: %s\nAuthor: %s\nSubject: %s\nBody:\n%s\nURL: %s\n", i.ID, i.Author.Handle, i.Subject, i.Body, i.URL)
}

// ListIssueOptions defines options for listing issues.
type ListIssueOptions struct {
	// Labels defines the list of labels an issue must have in order to be listed
	// The issue must have *every* label provided.
	Labels []string
	// Handles defines the list of usernames to list issues from
	// The issue can be created by *any* user provided.
	Handles []string
}

// Comment represents a comment on a code change request.
// TODO comments on issue?
type Comment struct {
	// ChangeID is the local identifier for the code change request this comment was left on (e.g. Github PR number)
	ChangeID string
	// Line is the contents of the code on the line where this comment was left
	Line   string
	Body   string
	Author Author
}

// ListCommentOptions defines options for listing comments.
type ListCommentOptions struct {
	// ChangeID is the local identifier for the code change request to list comments from (e.g. Github PR number)
	ChangeID string
	// Handles defines the list of usernames to list comments from
	// The comment can be created by *any* user provided.
	Handles []string
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
	return fmt.Sprintf("git@%s:%s/%s.git", repo.HostDomain, repo.Owner.Handle, repo.Name)
}

// HTTPS returns the HTTPS representation of the remote repository.
func (repo Repository) HTTPS() string {
	return fmt.Sprintf("https://%s/%s/%s.git", repo.HostDomain, repo.Owner.Handle, repo.Name)
}

// VCClient is an interface for version control server's client, e.g. a Github or Gerrit client.
type VCClient interface {
	// ListOpenIssues lists unresolved issues meeting the provided criteria on the version control server.
	ListOpenIssues(opts ListIssueOptions) ([]Issue, error)
	// ListOpenComments lists unresolved comments meeting the provided criteria on the version control server.
	ListOpenComments(opts ListCommentOptions) ([]Comment, error)
	// OpenCodeChangeRequest opens a new "code change request" on the version control server (e.g. "pull request" in Github).
	OpenCodeChangeRequest(req llm.CodeChangeRequest, res llm.CodeChangeResponse) (id, url string, err error)
	// UpdateCodeChangeRequest updates an existing code change request on the version control server.
	// UpdateCodeChangeRequest(id string, res llm.CodeChangeResponse)
	// TODO: add/read comments to/from issues and code change requests
	// GetLocalFile gets the current representation of the file at the provided path from the local git repo.
	GetLocalFile(path string) (llm.File, error)
	// StartCommit initiates a commit process, after which files can be modified and added to the commit.
	StartCommit() error
	// ReplaceOrAddLocalFile updates or adds a file in the locally cloned repo, and applies these changes to the current git worktree.
	ReplaceOrAddLocalFile(newFile llm.File) error
	// FinishCommit completes a commit, after which a code change request can be opened or updated.
	FinishCommit(message string) error
}
