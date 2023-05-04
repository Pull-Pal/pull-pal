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
	"time"

	"github.com/mobyvb/pull-pal/llm"

	"github.com/go-git/go-git/v5"
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
	log.Info("Creating new Github client...")
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

	if repo.LocalPath != "" {
		// remove local repo if it exists already
		err := os.RemoveAll(repo.LocalPath)
		if err != nil {
			return nil, err
		}
	}

	log.Info("Cloning repository locally...", zap.String("local repo path", repo.LocalPath), zap.String("url", repo.SSH()))
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
		log.Info("failed")
		return nil, err
	}
	repo.localRepo = localRepo

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
	/*
		branchName := randomBranchName()
		branchRefName := plumbing.NewBranchReferenceName(branchName)
		baseBranch := "main"
		remoteName := "origin"
	*/
	body := res.Notes
	body += fmt.Sprintf("\n\nResolves #%s", req.IssueID)

	// Create new local branch
	/*
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
	*/

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

func randomBranchName() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
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

// GetLocalFile gets the current representation of the file at the provided path from the local git repo.
func (gc *GithubClient) GetLocalFile(path string) (llm.File, error) {
	fullPath := filepath.Join(gc.repo.LocalPath, path)

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		// if file doesn't exist, just return an empty file
		// this means we want to prompt the llm to populate it for the first time
		if errors.Is(err, os.ErrNotExist) {
			return llm.File{
				Path:     path,
				Contents: "",
			}, nil
		}
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
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// set worktree to nil so a new commit can be started
	gc.worktree = nil

	return nil
}
