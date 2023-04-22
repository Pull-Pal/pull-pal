package vc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mobyvb/pull-pal/llm"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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

	worktree *git.Worktree
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

// OpenCodeChangeRequest pushes to a new remote branch and opens a PR on Github.
func (gc *GithubClient) OpenCodeChangeRequest(req llm.CodeChangeRequest, res llm.CodeChangeResponse) (id, url string, err error) {
	// TODO handle gc.ctx canceled

	title := req.Subject
	branchName := randomBranchName()
	branchRefName := plumbing.NewBranchReferenceName(branchName)
	baseBranch := "main"
	remoteName := "origin"
	body := res.Notes
	body += fmt.Sprintf("\n\nResolves #%s", req.IssueID)
	issue, err := strconv.Atoi(req.IssueID)
	if err != nil {
		return "", "", err
	}

	// Create new local branch
	headRef, err := gc.repo.localRepo.Head()
	if err != nil {
		return "", "", err
	}
	err = gc.repo.localRepo.CreateBranch(&config.Branch{
		Name:   branchName,
		Remote: remoteName,
		Merge:  branchRefName,
	})
	if err != nil {
		return "", "", err
	}

	// Update the branch to point to the new commit
	err = gc.repo.localRepo.Storer.SetReference(plumbing.NewHashReference(branchRefName, headRef.Hash()))
	if err != nil {
		return "", "", err
	}

	// Push the new branch to the remote repository
	remote, err := gc.repo.localRepo.Remote(remoteName)
	if err != nil {
		return "", "", err
	}

	err = remote.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:refs/heads/%s", branchRefName, branchName))},
		Auth: &http.BasicAuth{
			Username: gc.self.Handle,
			Password: gc.self.Token,
		},
	})
	if err != nil {
		return "", "", err
	}

	// Finally, open a pull request from the new branch.
	pr, _, err := gc.client.PullRequests.Create(gc.ctx, gc.repo.Owner.Handle, gc.repo.Name, &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &baseBranch,
		Body:  &body,
		Issue: &issue,
	})
	if err != nil {
		return "", "", err
	}

	url = pr.GetHTMLURL()
	id = strconv.Itoa(int(pr.GetID()))

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
		return nil, err
	}

	toReturn := []Issue{}
	for _, issue := range issues {
		// TODO make this filtering configurable from outside
		if issue.GetUser().GetLogin() != gc.repo.Owner.Handle {
			continue
		}

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

// GetLocalFile gets the current representation of the file at the provided path from the local git repo.
func (gc *GithubClient) GetLocalFile(path string) (llm.File, error) {
	fullPath := filepath.Join(gc.repo.LocalPath, path)

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return llm.File{}, err
	}

	return llm.File{
		Path:     path,
		Contents: string(data),
	}, nil
}

// StartCommit creates a new worktree associated with this Github client.
func (gc *GithubClient) StartCommit() error {
	if gc.worktree != nil {
		return errors.New("worktree is not nil - cannot start a new commit")
	}

	worktree, err := gc.repo.localRepo.Worktree()
	if err != nil {
		return err
	}

	gc.worktree = worktree

	return nil
}

// ReplaceOrAddLocalFile updates or adds a file in the locally cloned repo, and applies these changes to the current git worktree.
func (gc *GithubClient) ReplaceOrAddLocalFile(newFile llm.File) error {
	if gc.worktree == nil {
		return errors.New("worktree is nil - StartCommit must be called")
	}

	// TODO format non-go files as well
	if strings.HasSuffix(newFile.Path, ".go") {
		newContents, err := format.Source([]byte(newFile.Contents))
		if err != nil {
			return err
		}
		newFile.Contents = string(newContents)
	}

	fullPath := filepath.Join(gc.repo.LocalPath, newFile.Path)

	err := ioutil.WriteFile(fullPath, []byte(newFile.Contents), 0644)
	if err != nil {
		return err
	}

	_, err = gc.worktree.Add(newFile.Path)

	return err
}

// FinishCommit completes a commit, after which a code change request can be opened or updated.
func (gc *GithubClient) FinishCommit(message string) error {
	if gc.worktree == nil {
		return errors.New("worktree is nil - StartCommit must be called")
	}
	_, err := gc.worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  gc.self.Handle,
			Email: gc.self.Email,
		},
	})

	return err
}

func (gc *GithubClient) Close() error {
	// Remove local repository
	if gc.repo.LocalPath != "" {
		err := os.RemoveAll(gc.repo.LocalPath)

		return err
	}

	return nil
}