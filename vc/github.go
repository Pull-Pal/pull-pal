package vc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/github"
	"github.com/mobyvb/pull-pal/llm"
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
	log.Debug("Creating new Github client...")
	if self.Token == "" {
		return nil, errors.New("Github access token not provided")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: self.Token},
	)
	// oauth client is used to list issues, open pull requests, etc...
	tc := oauth2.NewClient(ctx, ts)

	// clone provided repository to local path
	if repo.LocalPath == "" {
		return nil, errors.New("local path to clone repository not provided")
	}
	log.Debug("Cloning repository locally...")
	// TODO this can be done in-memory - see https://pkg.go.dev/github.com/go-git/go-git/v5#readme-in-memory-example
	localRepo, err := git.PlainClone(repo.LocalPath, false, &git.CloneOptions{
		URL: repo.SSH(),
		// URL: repo.HTTPS(),
		Auth: &http.BasicAuth{
			Username: self.Handle,
			Password: self.Token,
		},
	})
	if err != nil {
		log.Error("Failed to clone local repo.", zap.Error(err))
		return nil, err
	}
	repo.localRepo = localRepo

	log.Debug("Success. Github client set up.")

	return &GithubClient{
		ctx:    ctx,
		log:    log,
		client: github.NewClient(tc),
		self:   self,
		repo:   repo,
	}, nil
}

// OpenCodeChangeRequest opens a new PR on Github based on the provided LLM request and response.
func (gc *GithubClient) OpenCodeChangeRequest(req llm.CodeChangeRequest, res llm.CodeChangeResponse) (id, url string, err error) {
	// TODO handle gc.ctx canceled
	gc.log.Debug("Creating a new pull request...")

	title := req.Subject
	branchName := randomBranchName()
	baseBranch := "main"
	body := res.Notes
	body += fmt.Sprintf("\n\nResolves #%s", req.IssueID)
	issue, err := strconv.Atoi(req.IssueID)
	if err != nil {
		gc.log.Error("Failed to parse issue ID from code change request as integer", zap.String("provided issue ID", req.IssueID), zap.Error(err))
		return "", "", err
	}

	pr, _, err := gc.client.PullRequests.Create(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &baseBranch,
		Body:  &body,
		Issue: &issue,
	})
	if err != nil {
		gc.log.Error("Failed to create pull request", zap.Error(err))
		return "", "", err
	}

	url = pr.GetHTMLURL()
	id = strconv.Itoa(int(pr.GetID()))
	gc.log.Info("Successfully created pull request.", zap.String("ID", id), zap.String("URL", url))

	return id, url, nil
}

func randomBranchName() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ListOpenIssues lists unresolved issues in the Github repository.
func (gc *GithubClient) ListOpenIssues() ([]Issue, error) {
	// List and parse GitHub issues
	issues, _, err := gc.client.Issues.ListByRepo(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, nil)
	if err != nil {
		gc.log.Error("Failed to list issues", zap.Error(err))
		return nil, err
	}

	toReturn := make([]Issue, len(issues))
	for _, issue := range issues {
		nextIssue := Issue{
			ID:      strconv.Itoa(int(issue.GetID())),
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
