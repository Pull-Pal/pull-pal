package vc

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Issue represents an issue on a version control server.
type Issue struct {
	Number  int
	Subject string
	Body    string
	URL     string
	Author  Author
}

func (i Issue) String() string {
	return fmt.Sprintf("Issue #: %d\nAuthor: %s\nSubject: %s\nBody:\n%s\nURL: %s\n", i.Number, i.Author.Handle, i.Subject, i.Body, i.URL)
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
	ID int64
	// ChangeID is the local identifier for the code change request this comment was left on (e.g. Github PR number)
	ChangeID string
	Author   Author
	Body     string
	Position int
	FilePath string
	DiffHunk string
	URL      string
	Branch   string
	PRNumber int
}

func (c Comment) String() string {
	return fmt.Sprintf("Comment ID: %d\nAuthor: %s\nBody: %s\nPosition: %d\n\nDiffHunk:\n%s\n\nURL: %s\nBranch:\n%s\n\nFilePath:\n%s\n\n", c.ID, c.Author.Handle, c.Body, c.Position, c.DiffHunk, c.URL, c.Branch, c.FilePath)
}

// ListCommentOptions defines options for listing comments.
type ListCommentOptions struct {
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

type IssueBody struct {
	PromptBody string
	FilePaths  []string
	BaseBranch string
}

func ParseIssueBody(body string) IssueBody {
	issueBody := IssueBody{
		BaseBranch: "main",
	}
	divider := "---"

	parts := strings.Split(body, divider)
	issueBody.PromptBody = strings.TrimSpace(parts[0])
	// if there was nothing to split, no additional configuration was provided
	if len(parts) <= 1 {
		return issueBody
	}

	configStr := parts[1]
	configLines := strings.Split(configStr, "\n")
	for _, line := range configLines {
		lineParts := strings.Split(line, ":")
		if len(lineParts) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(lineParts[0]))
		if key == "base" {
			issueBody.BaseBranch = strings.TrimSpace(lineParts[1])
			continue
		}
		if key == "files" {
			filePaths := strings.Split(lineParts[1], ",")
			for _, p := range filePaths {
				issueBody.FilePaths = append(issueBody.FilePaths, strings.TrimSpace(p))
			}
			continue
		}
	}

	return issueBody
}
