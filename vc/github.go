package vc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/mobyvb/pull-pal/llm"

	"github.com/google/go-github/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// GithubClient implements the VCClient interface.
type GithubClient struct {
	ctx context.Context
	log *zap.Logger

	client *github.Client
	self   Author
	repo   Repository
}

// NewGithubClient initializes a Github client and checks out a repository locally.
func NewGithubClient(ctx context.Context, log *zap.Logger, self Author, repo Repository) (*GithubClient, error) {
	log.Info("Creating new Github client...")
	if self.Token == "" {
		return nil, errors.New("Github access token not provided")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: self.Token},
	)
	// oauth client is used to list issues, open pull requests, etc...
	tc := oauth2.NewClient(ctx, ts)

	log.Info("Success. Github client set up.")

	return &GithubClient{
		ctx:    ctx,
		log:    log,
		client: github.NewClient(tc),
		self:   self,
		repo:   repo,
	}, nil
}

// OpenCodeChangeRequest pushes to a new remote branch and opens a PR on Github.
func (gc *GithubClient) OpenCodeChangeRequest(req llm.CodeChangeRequest, res llm.CodeChangeResponse, fromBranch, toBranch string) (id, url string, err error) {
	// TODO handle gc.ctx canceled

	title := req.Subject
	if title == "" {
		title = "update files"
	}

	body := res.Notes
	body += fmt.Sprintf("\n\nResolves #%s", req.IssueID)

	// Finally, open a pull request from the new branch.
	pr, _, err := gc.client.PullRequests.Create(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, &github.NewPullRequest{
		Title: &title,
		Head:  &fromBranch,
		Base:  &toBranch,
		Body:  &body,
	})
	if err != nil {
		return "", "", err
	}

	url = pr.GetHTMLURL()
	id = strconv.Itoa(int(pr.GetID()))

	return id, url, nil
}

// ListOpenIssues lists unresolved issues in the Github repository.
func (gc *GithubClient) ListOpenIssues(options ListIssueOptions) ([]Issue, error) {
	// List and parse GitHub issues
	opt := &github.IssueListByRepoOptions{
		Labels: options.Labels,
	}
	issues, _, err := gc.client.Issues.ListByRepo(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, opt)
	if err != nil {
		return nil, err
	}

	toReturn := []Issue{}
	for _, issue := range issues {
		issueUser := issue.GetUser().GetLogin()
		allowedUser := false
		for _, u := range options.Handles {
			if issueUser == u {
				allowedUser = true
				break
			}
		}
		if !allowedUser {
			continue
		}

		nextIssue := Issue{
			ID:      strconv.Itoa(issue.GetNumber()),
			Subject: issue.GetTitle(),
			Body:    issue.GetBody(),
			URL:     issue.GetHTMLURL(),
			Author: Author{
				Email:  issue.GetUser().GetEmail(),
				Handle: issue.GetUser().GetLogin(),
			},
		}
		toReturn = append(toReturn, nextIssue)
	}

	return toReturn, nil
}

// CommentOnIssue adds a comment to the issue provided.
func (gc *GithubClient) CommentOnIssue(issueNumber int, comment string) error {
	ghComment := &github.IssueComment{
		Body: github.String(comment),
	}

	_, _, err := gc.client.Issues.CreateComment(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, issueNumber, ghComment)

	return err
}

// RemoveLabelFromIssue removes the provided label from an issue if that label is applied.
func (gc *GithubClient) RemoveLabelFromIssue(issueNumber int, label string) error {
	hasLabel := false
	labels, _, err := gc.client.Issues.ListLabelsByIssue(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, issueNumber, nil)
	if err != nil {
		return err
	}
	for _, l := range labels {
		if l.GetName() == label {
			hasLabel = true
			break
		}
	}

	if hasLabel {
		_, err = gc.client.Issues.RemoveLabelForIssue(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, issueNumber, label)
		return err
	}

	return nil
}

// ListOpenComments lists unresolved comments in the Github repository.
func (gc *GithubClient) ListOpenComments(options ListCommentOptions) ([]Comment, error) {
	toReturn := []Comment{}
	prNumber, err := strconv.Atoi(options.ChangeID)
	if err != nil {
		return nil, err
	}
	comments, _, err := gc.client.PullRequests.ListComments(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, prNumber, nil)
	if err != nil {
		return nil, err
	}

	// TODO: filter out comments that "self" has already replied to
	// TODO: ignore resolved comments
	for _, c := range comments {
		commentUser := c.GetUser().GetLogin()
		allowedUser := false
		for _, u := range options.Handles {
			if commentUser == u {
				allowedUser = true
				break
			}
		}
		if !allowedUser {
			continue
		}

		nextComment := Comment{
			ID:       strconv.FormatInt(c.GetID(), 10),
			ChangeID: options.ChangeID,
			URL:      c.GetHTMLURL(),
			Author: Author{
				Email:  c.GetUser().GetEmail(),
				Handle: c.GetUser().GetLogin(),
			},
			Body:     c.GetBody(),
			Position: c.GetPosition(),
			DiffHunk: c.GetDiffHunk(),
		}
		toReturn = append(toReturn, nextComment)
	}

	return toReturn, nil
}
